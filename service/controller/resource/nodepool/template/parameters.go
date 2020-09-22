package template

import (
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/giantswarm/microerror"
)

type Parameters struct {
	AzureOperatorVersion string
	ClusterID            string
	DataDisks            []Disk
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

type Disk struct {
	nameSuffix string
	diskSizeGB float64
	lun        float64
}

func NewDisk(nameSuffix string, diskSizeGB, lun int32) Disk {
	return Disk{
		nameSuffix: nameSuffix,
		diskSizeGB: float64(diskSizeGB),
		lun:        float64(lun),
	}
}

func (p Parameters) ToDeployParams() map[string]interface{} {
	armDeploymentParameters := map[string]interface{}{}
	armDeploymentParameters["azureOperatorVersion"] = toARMParam(p.AzureOperatorVersion)
	armDeploymentParameters["clusterID"] = toARMParam(p.ClusterID)
	armDeploymentParameters["dataDisks"] = toARMParam(p.DataDisks)
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
	armDeploymentParameters["zones"] = toARMParam(p.Zones)

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

func newParameters(parameters map[string]interface{}, cast func(param interface{}) string) (Parameters, error) {
	minReplicas, ok := parameters["minReplicas"].(map[string]interface{})["value"].(float64)
	if !ok {
		return Parameters{}, microerror.Maskf(wrongTypeError, "minReplicas should be float64, got '%T'", parameters["minReplicas"].(map[string]interface{})["value"])
	}

	maxReplicas, ok := parameters["maxReplicas"].(map[string]interface{})["value"].(float64)
	if !ok {
		return Parameters{}, microerror.Maskf(wrongTypeError, "maxReplicas should be float64, got '%T'", parameters["maxReplicas"].(map[string]interface{})["value"])
	}

	currentReplicas, ok := parameters["currentReplicas"].(map[string]interface{})["value"].(float64)
	if !ok {
		return Parameters{}, microerror.Maskf(wrongTypeError, "currentReplicas should be float64, got '%T'", parameters["currentReplicas"].(map[string]interface{})["value"])
	}

	disks, ok := parameters["dataDisks"].(map[string]interface{})["value"].([]Disk)
	if !ok {
		return Parameters{}, microerror.Maskf(wrongTypeError, "dataDisks should be []Disk, got '%T'", parameters["dataDisks"].(map[string]interface{})["value"])
	}

	zones, ok := parameters["zones"].(map[string]interface{})["value"].([]string)
	if !ok {
		return Parameters{}, microerror.Maskf(wrongTypeError, "zones should be []string, got '%T'", parameters["zones"].(map[string]interface{})["value"])
	}

	return Parameters{
		AzureOperatorVersion: cast(parameters["azureOperatorVersion"]),
		ClusterID:            cast(parameters["clusterID"]),
		DataDisks:            disks,
		NodepoolName:         cast(parameters["nodepoolName"]),
		OSImage: OSImage{
			Publisher: cast(parameters["osImagePublisher"]),
			Offer:     cast(parameters["osImageOffer"]),
			SKU:       cast(parameters["osImageSKU"]),
			Version:   cast(parameters["osImageVersion"]),
		},
		Scaling: Scaling{
			MinReplicas:     int32(minReplicas),
			MaxReplicas:     int32(maxReplicas),
			CurrentReplicas: int32(currentReplicas),
		},
		SSHPublicKey: cast(parameters["sshPublicKey"]),
		SubnetName:   cast(parameters["subnetName"]),
		VMCustomData: cast(parameters["vmCustomData"]),
		VMSize:       cast(parameters["vmSize"]),
		VnetName:     cast(parameters["vnetName"]),
		Zones:        zones,
	}, nil
}

func castCurrent(param interface{}) string {
	return param.(map[string]interface{})["value"].(string)
}

func castDesired(param interface{}) string {
	return param.(struct{ Value interface{} }).Value.(string)
}
