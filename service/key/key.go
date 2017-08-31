package key

import (
	"github.com/giantswarm/azuretpr"
)

func ClusterCustomer(customObject azuretpr.CustomObject) string {
	return customObject.Spec.Cluster.Customer.ID
}

func ClusterID(customObject azuretpr.CustomObject) string {
	return customObject.Spec.Cluster.Cluster.ID
}

func Location(customObject azuretpr.CustomObject) string {
	return customObject.Spec.Azure.Location
}
