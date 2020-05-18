package crmapper

import (
	"strings"

	"github.com/giantswarm/certs"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
)

func newEtcdDomain(cr capzv1alpha3.AzureCluster) (string, error) {
	splitted := strings.Split(cr.Spec.ControlPlaneEndpoint.Host, ".")
	splitted[0] = certs.EtcdCert.String()
	etcdDomain := strings.Join(splitted, ".")

	return etcdDomain, nil
}
