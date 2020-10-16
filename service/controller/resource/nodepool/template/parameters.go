package template

import (
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
)

type Parameters struct {
	AzureOperatorVersion        string
	ClusterID                   string
	DataDisks                   []v1alpha3.DataDisk
	EnableAcceleratedNetworking bool
	NodepoolName                string
	OSImage                     OSImage
	Scaling                     Scaling
	StorageAccountType          string
	SubnetName                  string
	VMCustomData                string
	VMSize                      string
	VnetName                    string
	Zones                       []string
}

type Scaling struct {
	MinReplicas     int32
	MaxReplicas     int32
	CurrentReplicas int32
}

type OSImage struct {
	Publisher string
	Offer     string
	SKU       string
	Version   string
}

func NewFromDeployment(deployment azureresource.Deployment) (Parameters, error) {
	parameters, ok := deployment.Properties.Parameters.(map[string]interface{})
	if !ok {
		return Parameters{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", map[string]interface{}{}, deployment.Properties.Parameters)
	}

	return newParameters(parameters, castDesired)
}

func NewFromExtendedDeployment(deployment azureresource.DeploymentExtended) (Parameters, error) {
	parameters, ok := deployment.Properties.Parameters.(map[string]interface{})
	if !ok {
		return Parameters{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", map[string]interface{}{}, deployment.Properties.Parameters)
	}

	return newParameters(parameters, castCurrent)
}

// ToDeployParams prepares the parameters to the format that ARM API understand.
// We also try to use the same types that the ARM API will return, that's why we convert to float64 or interface{} types.
func (p Parameters) ToDeployParams() map[string]interface{} {
	var dataDisks []interface{}
	for _, disk := range p.DataDisks {
		dataDisks = append(dataDisks, map[string]interface{}{"nameSuffix": disk.NameSuffix, "lun": float64(*disk.Lun), "diskSizeGB": float64(disk.DiskSizeGB)})
	}

	zones := []interface{}{}
	for _, zone := range p.Zones {
		zones = append(zones, zone)
	}

	armDeploymentParameters := map[string]interface{}{}
	armDeploymentParameters["azureOperatorVersion"] = toARMParam(p.AzureOperatorVersion)
	armDeploymentParameters["clusterID"] = toARMParam(p.ClusterID)
	armDeploymentParameters["dataDisks"] = toARMParam(dataDisks)
	armDeploymentParameters["enableAcceleratedNetworking"] = toARMParam(p.EnableAcceleratedNetworking)
	armDeploymentParameters["nodepoolName"] = toARMParam(p.NodepoolName)
	armDeploymentParameters["osImagePublisher"] = toARMParam(p.OSImage.Publisher)
	armDeploymentParameters["osImageOffer"] = toARMParam(p.OSImage.Offer)
	armDeploymentParameters["osImageSKU"] = toARMParam(p.OSImage.SKU)
	armDeploymentParameters["osImageVersion"] = toARMParam(p.OSImage.Version)
	armDeploymentParameters["minReplicas"] = toARMParam(float64(p.Scaling.MinReplicas))
	armDeploymentParameters["maxReplicas"] = toARMParam(float64(p.Scaling.MaxReplicas))
	armDeploymentParameters["currentReplicas"] = toARMParam(float64(p.Scaling.CurrentReplicas))
	armDeploymentParameters["storageAccountType"] = toARMParam(p.StorageAccountType)
	armDeploymentParameters["subnetName"] = toARMParam(p.SubnetName)
	armDeploymentParameters["vmCustomData"] = toARMParam(p.VMCustomData)
	armDeploymentParameters["vmSize"] = toARMParam(p.VMSize)
	armDeploymentParameters["vnetName"] = toARMParam(p.VnetName)
	armDeploymentParameters["zones"] = toARMParam(zones)

	return armDeploymentParameters
}

func toARMParam(v interface{}) interface{} {
	return struct {
		Value interface{}
	}{
		Value: v,
	}
}

func newParameters(parameters map[string]interface{}, cast func(param interface{}) interface{}) (Parameters, error) {
	// DataDisks is an untyped array so we need to work a little bit to get the right types.
	var dataDisks []v1alpha3.DataDisk
	disks, ok := cast(parameters["dataDisks"]).([]interface{})
	if !ok {
		return Parameters{}, microerror.Maskf(wrongTypeError, "dataDisks should be []interface, got '%T'", cast(parameters["dataDisks"]))
	}

	for _, disk := range disks {
		d, ok := disk.(map[string]interface{})
		if !ok {
			return Parameters{}, microerror.Maskf(wrongTypeError, "disk should be map[string]interface{}, got '%T'", disk)
		}
		dataDisks = append(dataDisks, v1alpha3.DataDisk{
			NameSuffix: d["nameSuffix"].(string),
			DiskSizeGB: int32(d["diskSizeGB"].(float64)),
			Lun:        to.Int32Ptr(int32(d["lun"].(float64))),
		})
	}

	// Zones is an untyped array so we need to work a little bit to get the right types.
	var zones []string
	rawZones, ok := cast(parameters["zones"]).([]interface{})
	if !ok {
		return Parameters{}, microerror.Maskf(wrongTypeError, "zones should be []interface, got '%T'", cast(parameters["zones"]))
	}

	for _, rawZone := range rawZones {
		zone, ok := rawZone.(string)
		if !ok {
			return Parameters{}, microerror.Maskf(wrongTypeError, "zone should be string, got '%T'", rawZone)
		}

		zones = append(zones, zone)
	}

	// Finally return typed parameters.
	return Parameters{
		AzureOperatorVersion:        cast(parameters["azureOperatorVersion"]).(string),
		ClusterID:                   cast(parameters["clusterID"]).(string),
		DataDisks:                   dataDisks,
		EnableAcceleratedNetworking: cast(parameters["enableAcceleratedNetworking"]).(bool),
		NodepoolName:                cast(parameters["nodepoolName"]).(string),
		OSImage: OSImage{
			Publisher: cast(parameters["osImagePublisher"]).(string),
			Offer:     cast(parameters["osImageOffer"]).(string),
			SKU:       cast(parameters["osImageSKU"]).(string),
			Version:   cast(parameters["osImageVersion"]).(string),
		},
		Scaling: Scaling{
			MinReplicas:     int32(cast(parameters["minReplicas"]).(float64)),
			MaxReplicas:     int32(cast(parameters["maxReplicas"]).(float64)),
			CurrentReplicas: int32(cast(parameters["currentReplicas"]).(float64)),
		},
		StorageAccountType: cast(parameters["storageAccountType"]).(string),
		SubnetName:         cast(parameters["subnetName"]).(string),
		// It comes empty from Azure API.
		VMCustomData: "",
		VMSize:       cast(parameters["vmSize"]).(string),
		VnetName:     cast(parameters["vnetName"]).(string),
		Zones:        zones,
	}, nil
}

func castCurrent(param interface{}) interface{} {
	return param.(map[string]interface{})["value"]
}

func castDesired(param interface{}) interface{} {
	return param.(struct{ Value interface{} }).Value
}
