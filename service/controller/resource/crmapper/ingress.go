package crmapper

import (
	"strings"

	"github.com/giantswarm/microerror"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
)

const (
	ingressControllerDomainPrefix = "ingress"
	ingressWildcardDomainPrefix   = "*"
)

func (r *Resource) newIngressDomain(cr capzv1alpha3.AzureCluster, domainPrefix string) (string, error) {
	split := strings.Split(cr.Spec.ControlPlaneEndpoint.Host, ".")
	if len(split) < 3 {
		return "", microerror.Maskf(invalidDomainError, "can't get ingress domain for '%s'", cr.Spec.ControlPlaneEndpoint.Host)
	}

	split[0] = domainPrefix

	// Replace the API base domain with the Ingress Controller base domain.
	ingressSplit := strings.Split(r.viper.GetString(r.flag.Service.Cluster.Kubernetes.IngressController.BaseDomain), ".")
	replacePos := len(split) - len(ingressSplit)
	for _, v := range ingressSplit {
		split[replacePos] = v
		replacePos++
	}

	return strings.Join(split, "."), nil
}

func (r *Resource) newIngressControllerDomain(cr capzv1alpha3.AzureCluster) (string, error) {
	ingressControllerDomain, err := r.newIngressDomain(cr, ingressControllerDomainPrefix)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return ingressControllerDomain, nil
}

func (r *Resource) newIngressWildcardDomain(cr capzv1alpha3.AzureCluster) (string, error) {
	ingressControllerDomain, err := r.newIngressDomain(cr, ingressWildcardDomainPrefix)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return ingressControllerDomain, nil
}
