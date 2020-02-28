package collector

import (
	"context"
	"strconv"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/kubernetes"
)

const (
	labelSubscriptionId  = "subscription_id"
	labelApplicationId   = "application_id"
	labelApplicationName = "application_name"
	labelExpirationTS    = "expiration_ts"
)

var (
	tokenExpirationDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName("azure_operator", "service_principal_token", "expiration"),
		"Expiration date for Azure Access Tokens.",
		[]string{
			labelSubscriptionId,
			labelApplicationId,
			labelApplicationName,
			labelExpirationTS,
		},
		nil,
	)
)

type ServicePrincipalTokenConfig struct {
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger
}

type ServicePrincipalToken struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	environmentName string
}

func NewServicePrincipalToken(config ServicePrincipalTokenConfig) (*ServicePrincipalToken, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	v := &ServicePrincipalToken{
		k8sClient: config.K8sClient,
		logger:    config.Logger,
	}

	return v, nil
}

func (spt *ServicePrincipalToken) Collect(ch chan<- prometheus.Metric) error {
	ctx := context.Background()

	clientSets, err := getClientSets(spt.k8sClient, spt.environmentName)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, item := range clientSets {
		clientSet := item

		applications, err := clientSet.ApplicationsClient.ListComplete(ctx, "")
		if err != nil {
			return microerror.Mask(err)
		}

		for applications.NotDone() {
			app := applications.Value()

			if *app.AppID == clientSet.ClientID {
				for _, pc := range *app.PasswordCredentials {
					ch <- prometheus.MustNewConstMetric(
						tokenExpirationDesc,
						prometheus.GaugeValue,
						gaugeValue,
						clientSet.SubscriptionID,
						*app.AppID,
						*app.DisplayName,
						strconv.FormatInt(pc.EndDate.Unix(), 10),
					)
				}
			}

			if err := applications.Next(); err != nil {
				return microerror.Mask(err)
			}
		}

		return nil
	}

	return nil
}

func (spt *ServicePrincipalToken) Describe(ch chan<- *prometheus.Desc) error {
	ch <- tokenExpirationDesc
	return nil
}
