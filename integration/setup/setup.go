package setup

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	appv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"k8s.io/helm/pkg/helm"

	"github.com/giantswarm/azure-operator/integration/env"
	"github.com/giantswarm/azure-operator/integration/key"
	"github.com/giantswarm/azure-operator/pkg/project"
)

// WrapTestMain setup and teardown e2e testing environment.
func WrapTestMain(m *testing.M, c Config) {
	var r int

	ctx := context.Background()

	err := Setup(ctx, c)
	if err != nil {
		log.Printf("%#v\n", err)
		r = 1
	} else {
		r = m.Run()
	}

	if env.KeepResources() != "true" {
		err := Teardown(c)
		if err != nil {
			log.Printf("%#v\n", err)
			r = 1
		}
	}

	os.Exit(r)
}

// Setup e2e testing environment.
func Setup(ctx context.Context, c Config) error {
	var err error

	err = common(ctx, c)
	if err != nil {
		return microerror.Mask(err)
	}

	err = installResources(ctx, c)
	if err != nil {
		return microerror.Mask(err)
	}

	err = provider(ctx, c)
	if err != nil {
		return microerror.Mask(err)
	}

	err = bastion(ctx, c)
	if err != nil {
		return microerror.Mask(err)
	}

	err = c.Guest.Setup(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func installResources(ctx context.Context, config Config) error {
	var err error

	var latestOperatorRelease string
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("getting latest %#q release", project.Name())) // nolint: errcheck

		latestOperatorRelease, err = appcatalog.GetLatestVersion(ctx, key.DefaultCatalogStorageURL(), project.Name())
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("latest %#q release is %#q", project.Name(), latestOperatorRelease)) // nolint: errcheck
	}

	var operatorTarballPath string
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "getting tarball URL") // nolint: errcheck

		operatorVersion := fmt.Sprintf("%s-%s", latestOperatorRelease, env.CircleSHA())
		operatorTarballURL, err := appcatalog.NewTarballURL(key.DefaultTestCatalogStorageURL(), project.Name(), operatorVersion)
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball URL is %#q", operatorTarballURL)) // nolint: errcheck

		config.Logger.LogCtx(ctx, "level", "debug", "message", "pulling tarball") // nolint: errcheck

		operatorTarballPath, err = config.HelmClient.PullChartTarball(ctx, operatorTarballURL)
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball path is %#q", operatorTarballPath)) // nolint: errcheck
	}

	{
		secretYaml := fmt.Sprintf(`
service:
  azure:
    clientid: %s
    clientsecret: %s
    subscriptionid: %s
    tenantid: %s
`, env.AzureClientID(), env.AzureClientSecret(), env.AzureSubscriptionID(), env.AzureTenantID())
		base64secretYaml := base64.StdEncoding.EncodeToString([]byte(secretYaml))
		values := `
---
Installation:
  V1:
    Guest:
      Kubernetes:
        API:
          Auth:
            Provider:
              OIDC:
                ClientID: "%s"
                IssuerURL: ""
                UsernameClaim: "email"
                GroupsClaim: "groups"
      SSH:
        SSOPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDPr6Mxx3cdPNm3v4Ufvo5sRfT7jCgDi7z3wwaCufVrw8am+PBW7toRWBQtGddtp7zsdicHy1+FeWHw09txsbzjupO0yynVAtXSxS8HjsWZOcn0ZRQXMtbbikSxWRs9C255yBswPlD7y9OOiUr8OidIHRYq/vMKIPE+066PqVBYIgO4wR9BRhWPz385+Ob+36K+jkSbniiQr4c8Q545Fm+ilCccLCN1KVVj2pYkCyLHSaEJEp57eyU2ZiBqN0ntgqCVo3xery3HQQalin6Uhqaecwla9bpj48Oo22PLYN2yNhxFU66sSN9TkBquP2VGWlHmWRRg3RPnTY1IPBBC4ea3JOurYOEHydHtoMOGQ6irnd8avqFonXKT2cc/UWUsktv5RwI7S+hUbBdy0o/uX6SbecLIyL+iIIcWL5A0loWyjMEPdDdLjdz72EdnuLVeQohFuSeTpVqpiHugzCCZYwItT7N8QRSgx6wF7j8XkTDZYhWTv9nxtjsRwSDfZJbhsPsgjeQh0z1YJEKZ6RMyrHAzsui/6seFzlgvogRH2iJBzzrKui0uNyE7lQVAeRGHfqUN9YX0DgQ/AvT0BBnCyhMQCD7cJsFJ7A4nRTNsvpPR2uJ2n8fSf2kxXCHH2Tz+CbobVLeZqslKSiz5aO5iKCrHPK7fGnDCKKW8CyYG6V974Q=="
    Secret:
      AzureOperator:
        SecretYaml: %s
    Provider:
      Azure:
        HostCluster:
          CIDR: "0.0.0.0/0"
        MSI:
          Enabled: true
`
		defer func() {
			fs := afero.NewOsFs()
			err := fs.Remove(operatorTarballPath)
			if err != nil {
				config.Logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("deletion of %#q failed", operatorTarballPath), "stack", fmt.Sprintf("%#v", err)) // nolint: errcheck
			}
		}()

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing %#q", project.Name())) // nolint: errcheck

		err = config.HelmClient.InstallReleaseFromTarball(ctx,
			operatorTarballPath,
			key.Namespace(),
			helm.ReleaseName(key.ReleaseName()),
			helm.ValueOverrides([]byte(fmt.Sprintf(values, env.AzureClientID(), base64secretYaml))),
			helm.InstallWait(true))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed %#q", project.Name())) // nolint: errcheck
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring chart CRD exists") // nolint: errcheck

		// The operator will install the CRD on boot but we create chart CRs
		// in the tests so this ensures the CRD is present.
		err = config.K8sClients.CRDClient().EnsureCreated(ctx, appv1alpha1.NewChartCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured chart CRD exists") // nolint: errcheck
	}

	return nil
}
