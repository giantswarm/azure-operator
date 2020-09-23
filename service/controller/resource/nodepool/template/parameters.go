package template

import (
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
)

type Parameters struct {
	AzureOperatorVersion string
	ClusterID            string
	DataDisks            []v1alpha3.DataDisk
	NodepoolName         string
	OSImage              OSImage
	Scaling              Scaling
	SSHPublicKey         string
	SubnetName           string
	VMCustomData         string
	VMSize               string
	VnetName             string
	Zones                []string
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

func (p Parameters) ToDeployParams() map[string]interface{} {
	var dataDisks []interface{}
	for _, disk := range p.DataDisks {
		dataDisks = append(dataDisks, map[string]interface{}{"nameSuffix": disk.NameSuffix, "lun": disk.Lun, "diskSizeGB": disk.DiskSizeGB})
	}

	var zones []interface{}
	for _, zone := range p.Zones {
		zones = append(zones, zone)
	}

	armDeploymentParameters := map[string]interface{}{}
	armDeploymentParameters["azureOperatorVersion"] = toARMParam(p.AzureOperatorVersion)
	armDeploymentParameters["clusterID"] = toARMParam(p.ClusterID)
	armDeploymentParameters["dataDisks"] = toARMParam(dataDisks)
	armDeploymentParameters["nodepoolName"] = toARMParam(p.NodepoolName)
	armDeploymentParameters["osImagePublisher"] = toARMParam(p.OSImage.Publisher)
	armDeploymentParameters["osImageOffer"] = toARMParam(p.OSImage.Offer)
	armDeploymentParameters["osImageSKU"] = toARMParam(p.OSImage.SKU)
	armDeploymentParameters["osImageVersion"] = toARMParam(p.OSImage.Version)
	armDeploymentParameters["minReplicas"] = toARMParam(p.Scaling.MinReplicas)
	armDeploymentParameters["maxReplicas"] = toARMParam(p.Scaling.MaxReplicas)
	armDeploymentParameters["currentReplicas"] = toARMParam(p.Scaling.CurrentReplicas)
	armDeploymentParameters["sshPublicKey"] = toARMParam(p.SSHPublicKey)
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

func newParameters(parameters map[string]interface{}, cast func(param interface{}) interface{}) (Parameters, error) {
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

	return Parameters{
		AzureOperatorVersion: cast(parameters["azureOperatorVersion"]).(string),
		ClusterID:            cast(parameters["clusterID"]).(string),
		DataDisks:            dataDisks,
		NodepoolName:         cast(parameters["nodepoolName"]).(string),
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
		SSHPublicKey: cast(parameters["sshPublicKey"]).(string),
		SubnetName:   cast(parameters["subnetName"]).(string),
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
