package template

import (
	"strconv"

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
	allParams := map[string]interface{}{}
	allParams["azureOperatorVersion"] = struct {
		Value interface{}
	}{
		Value: p.AzureOperatorVersion,
	}
	allParams["clusterID"] = struct {
		Value interface{}
	}{
		Value: p.ClusterID,
	}
	allParams["dataDisks"] = struct {
		Value interface{}
	}{
		Value: p.DataDisks,
	}
	allParams["nodepoolName"] = struct {
		Value interface{}
	}{
		Value: p.NodepoolName,
	}
	allParams["osImagePublisher"] = struct {
		Value interface{}
	}{
		Value: p.OSImage.Publisher,
	}
	allParams["osImageOffer"] = struct {
		Value interface{}
	}{
		Value: p.OSImage.Offer,
	}
	allParams["osImageSKU"] = struct {
		Value interface{}
	}{
		Value: p.OSImage.SKU,
	}
	allParams["osImageVersion"] = struct {
		Value interface{}
	}{
		Value: p.OSImage.Version,
	}
	allParams["minReplicas"] = struct {
		Value interface{}
	}{
		Value: p.Scaling.MinReplicas,
	}
	allParams["maxReplicas"] = struct {
		Value interface{}
	}{
		Value: p.Scaling.MaxReplicas,
	}
	allParams["currentReplicas"] = struct {
		Value interface{}
	}{
		Value: p.Scaling.CurrentReplicas,
	}
	allParams["sshPublicKey"] = struct {
		Value interface{}
	}{
		Value: p.SSHPublicKey,
	}
	allParams["subnetName"] = struct {
		Value interface{}
	}{
		Value: p.SubnetName,
	}
	allParams["vmCustomData"] = struct {
		Value interface{}
	}{
		Value: p.VMCustomData,
	}
	allParams["vmSize"] = struct {
		Value interface{}
	}{
		Value: p.VMSize,
	}
	allParams["vnetName"] = struct {
		Value interface{}
	}{
		Value: p.VnetName,
	}
	allParams["zones"] = struct {
		Value interface{}
	}{
		Value: p.Zones,
	}

	return allParams
}

var wrongTypeError = &microerror.Error{
	Kind: "wrongTypeError",
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
	minReplicas, err := strconv.ParseInt(cast(parameters["minReplicas"]), 10, 32)
	if err != nil {
		return Parameters{}, microerror.Mask(err)
	}

	maxReplicas, err := strconv.ParseInt(cast(parameters["maxReplicas"]), 10, 32)
	if err != nil {
		return Parameters{}, microerror.Mask(err)
	}

	currentReplicas, err := strconv.ParseInt(cast(parameters["currentReplicas"]), 10, 32)
	if err != nil {
		return Parameters{}, microerror.Mask(err)
	}

	disks, ok := parameters["dataDisks"].(map[string]interface{})["value"].([]v1alpha3.DataDisk)
	if !ok {
		return Parameters{}, microerror.Maskf(wrongTypeError, "zones should be []v1alpha3.DataDisk, got '%T'", parameters["dataDisks"].(map[string]interface{})["value"])
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
