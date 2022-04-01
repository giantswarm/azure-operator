package azureconfig

import (
	"strings"

	"github.com/giantswarm/certs/v4/pkg/certs"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
)

func newAPIServerDomain(cr capz.AzureCluster) (string, error) {
	splitted := strings.Split(cr.Spec.ControlPlaneEndpoint.Host, ".")
	splitted[0] = certs.APICert.String()
	apiServerDomain := strings.Join(splitted, ".")

	return apiServerDomain, nil
}

func newEtcdServerDomain(cr capz.AzureCluster) (string, error) {
	splitted := strings.Split(cr.Spec.ControlPlaneEndpoint.Host, ".")
	splitted[0] = certs.EtcdCert.String()
	etcdServerDomain := strings.Join(splitted, ".")

	return etcdServerDomain, nil
}

func newKubeletDomain(cr capz.AzureCluster) (string, error) {
	splitted := strings.Split(cr.Spec.ControlPlaneEndpoint.Host, ".")
	splitted[0] = certs.WorkerCert.String()
	kubeletDomain := strings.Join(splitted, ".")

	return kubeletDomain, nil
}
