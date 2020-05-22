package azureconfig

import (
	"strings"

	"github.com/giantswarm/certs"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
)

func newAPIServerDomain(cr capzv1alpha3.AzureCluster) (string, error) {
	splitted := strings.Split(cr.Spec.ControlPlaneEndpoint.Host, ".")
	splitted[0] = certs.APICert.String()
	apiServerDomain := strings.Join(splitted, ".")

	return apiServerDomain, nil
}
