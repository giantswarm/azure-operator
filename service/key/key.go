package key

import (
	"fmt"

	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/certificatetpr"
)

// ClusterCustomer returns the customer ID for this cluster.
func ClusterCustomer(customObject azuretpr.CustomObject) string {
	return customObject.Spec.Cluster.Customer.ID
}

// ClusterID returns the unique ID for this cluster.
func ClusterID(customObject azuretpr.CustomObject) string {
	return customObject.Spec.Cluster.Cluster.ID
}

// KeyVaultName returns the Azure Key Vault name for this cluster.
func KeyVaultName(customObject azuretpr.CustomObject) string {
	return customObject.Spec.Azure.KeyVault.VaultName
}

// Location returns the physical location where the Resource Group is deployed.
func Location(customObject azuretpr.CustomObject) string {
	return customObject.Spec.Azure.Location
}

// SecretName returns the name of the Key Vault secret for this certificate
// asset.
func SecretName(assetKey certificatetpr.AssetsBundleKey) string {
	return fmt.Sprintf("%s-%s", assetKey.Component, assetKey.Type)
}
