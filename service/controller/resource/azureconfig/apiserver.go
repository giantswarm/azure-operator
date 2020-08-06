package azureconfig

import (
	"strings"

	"github.com/giantswarm/certs/v2/pkg/certs"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
)

func newAPIServerDomain(cr capzv1alpha3.AzureCluster) (string, error) {
	splitted := strings.Split(cr.Spec.ControlPlaneEndpoint.Host, ".")
	splitted[0] = certs.APICert.String()
	apiServerDomain := strings.Join(splitted, ".")

	return apiServerDomain, nil
}

func newEtcdServerDomain(cr capzv1alpha3.AzureCluster) (string, error) {
	splitted := strings.Split(cr.Spec.ControlPlaneEndpoint.Host, ".")
	splitted[0] = certs.EtcdCert.String()
	etcdServerDomain := strings.Join(splitted, ".")

	return etcdServerDomain, nil
}

func newKubeletDomain(cr capzv1alpha3.AzureCluster) (string, error) {
	splitted := strings.Split(cr.Spec.ControlPlaneEndpoint.Host, ".")
	splitted[0] = certs.WorkerCert.String()
	kubeletDomain := strings.Join(splitted, ".")

	return kubeletDomain, nil
}
