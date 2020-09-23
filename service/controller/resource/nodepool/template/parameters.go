package template

import (
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
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

func newParameters(parameters map[string]interface{}, cast func(param interface{}) interface{}) (Parameters, error) {
	dataDisks, ok := cast(parameters["dataDisks"]).([]v1alpha3.DataDisk)
	if !ok {
		return Parameters{}, microerror.Maskf(wrongTypeError, "dataDisks should be []v1alpha3.DataDisk, got '%T'", cast(parameters["dataDisks"]))
	}

	zones, ok := cast(parameters["zones"]).([]string)
	if !ok {
		return Parameters{}, microerror.Maskf(wrongTypeError, "zones should be []string, got '%T'", cast(parameters["zones"]))
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
