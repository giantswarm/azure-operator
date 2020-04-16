package collector

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/graphrbac/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
)

type azureCredentials struct {
	subscriptionID string
	tenantID       string
	clientID       string
	clientSecret   string
}

const (
	labelClientId        = "client_id"
	labelSubscriptionId  = "subscription_id"
	labelTenantId        = "tenant_id"
	labelApplicationId   = "application_id"
	labelApplicationName = "application_name"
	labelSecretKeyID     = "secret_key_id"
)

var (
	spExpirationDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName("azure_operator", "service_principal_token", "expiration"),
		"Expiration date for Azure Access Tokens.",
		[]string{
			labelClientId,
			labelSubscriptionId,
			labelTenantId,
			labelApplicationId,
			labelApplicationName,
			labelSecretKeyID,
		},
		nil,
	)

	spExpirationFailedScrapeDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName("azure_operator", "service_principal_token", "check_failed"),
		"Unable to retrieve informations about the service principal expiration date.",
		[]string{
			labelClientId,
			labelSubscriptionId,
			labelTenantId,
		},
		nil,
	)
)

type SPExpirationConfig struct {
	K8sClient       kubernetes.Interface
	Logger          micrologger.Logger
	EnvironmentName string
}

type SPExpiration struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	azureSetting             setting.Azure
	hostAzureClientSetConfig client.AzureClientSetConfig

	environment *azure.Environment
}

func NewSPExpiration(config SPExpirationConfig) (*SPExpiration, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	env, err := azure.EnvironmentFromName(config.EnvironmentName)
	if err != nil {
		return nil, err
	}

	v := &SPExpiration{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		environment: &env,
	}

	return v, nil
}

func (v *SPExpiration) Collect(ch chan<- prometheus.Metric) error {
	secretList, err := v.k8sClient.CoreV1().Secrets(credentialNamespace).List(metav1.ListOptions{
		LabelSelector: credentialLabelSelector,
	})

	if err != nil {
		return microerror.Mask(err)
	}

	failedScrapes := make(map[string]azureCredentials)

	for _, secret := range secretList.Items {
		clientID := string(secret.Data["azure.azureoperator.clientid"])
		clientSecret := string(secret.Data["azure.azureoperator.clientsecret"])
		subscriptionID := string(secret.Data["azure.azureoperator.subscriptionid"])
		tenantID := string(secret.Data["azure.azureoperator.tenantid"])

		creds := azureCredentials{
			subscriptionID: subscriptionID,
			tenantID:       tenantID,
			clientID:       clientID,
			clientSecret:   clientSecret,
		}

		ctx := context.Background()

		c, err := v.getApplicationsClient(creds)
		if err != nil {
			// Ignore but log
			v.logger.LogCtx(ctx, "level", "warning", "message", "Unable to create an applications client: ", err.Error())
			failedScrapes[creds.clientID] = creds
			continue
		}

		apps, err := c.ListComplete(ctx, fmt.Sprintf("appId eq '%s'", creds.clientID))
		if err != nil {
			// Ignore but log
			v.logger.LogCtx(ctx, "level", "warning", "message", "Unable to get application: ", err.Error())
			failedScrapes[creds.clientID] = creds
			continue
		}

		for apps.NotDone() {
			app := apps.Value()
			for _, pc := range *app.PasswordCredentials {
				ch <- prometheus.MustNewConstMetric(
					spExpirationDesc,
					prometheus.GaugeValue,
					float64(pc.EndDate.Unix()),
					creds.clientID,
					creds.subscriptionID,
					creds.tenantID,
					*app.AppID,
					*app.DisplayName,
					*pc.KeyID,
				)
			}

			if err := apps.NextWithContext(ctx); err != nil {
				return microerror.Mask(err)
			}
		}
	}

	// Send metrics for failed scrapes as well
	for _, creds := range failedScrapes {
		ch <- prometheus.MustNewConstMetric(
			spExpirationFailedScrapeDesc,
			prometheus.GaugeValue,
			float64(1),
			creds.clientID,
			creds.subscriptionID,
			creds.tenantID,
		)
	}

	return nil
}

func (v *SPExpiration) Describe(ch chan<- *prometheus.Desc) error {
	ch <- spExpirationDesc
	ch <- spExpirationFailedScrapeDesc
	return nil
}

func (v *SPExpiration) getApplicationsClient(creds azureCredentials) (*graphrbac.ApplicationsClient, error) {
	c := graphrbac.NewApplicationsClient(creds.tenantID)
	a, err := v.getGraphAuthorizer(creds)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	c.Authorizer = a

	return &c, nil
}

func (v *SPExpiration) getAuthorizerForResource(creds azureCredentials, resource string) (autorest.Authorizer, error) {
	var a autorest.Authorizer
	var err error

	oauthConfig, err := adal.NewOAuthConfig(v.environment.ActiveDirectoryEndpoint, creds.tenantID)
	if err != nil {
		return nil, err
	}

	token, err := adal.NewServicePrincipalToken(
		*oauthConfig, creds.clientID, creds.clientSecret, resource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	a = autorest.NewBearerAuthorizer(token)

	return a, err
}

func (v *SPExpiration) getGraphAuthorizer(creds azureCredentials) (autorest.Authorizer, error) {
	var a autorest.Authorizer
	var err error

	a, err = v.getAuthorizerForResource(creds, v.environment.GraphEndpoint)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return a, err
}
