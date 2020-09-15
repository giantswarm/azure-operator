package checksum

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/giantswarm/azure-operator/v4/pkg/helpers/vmss"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/templates"
)

func Test_getDeploymentTemplateChecksum(t *testing.T) {
	testCases := []struct {
		name             string
		template         map[string]interface{}
		expectedChecksum string
		errorMatcher     func(err error) bool
	}{
		{
			name:             "case 0: Successful checksum calculation",
			template:         map[string]interface{}{"bob": "5"},
			expectedChecksum: "8f3f86c9bc89affcd4eb86effad32055c6b9f575b53d44373e87cbff547b6e51",
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			properties := resources.DeploymentProperties{
				Template: tc.template,
			}
			deployment := resources.Deployment{
				Properties: &properties,
			}

			chk, err := GetDeploymentTemplateChecksum(deployment)

			switch {
			case err == nil && tc.errorMatcher == nil:
				// fall through
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("expected %#v got %#v", nil, err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("expected %#v got %#v", "error", nil)
			case !tc.errorMatcher(err):
				t.Fatalf("expected %#v got %#v", true, false)
			}

			if chk != tc.expectedChecksum {
				t.Fatal(fmt.Sprintf("Wrong checksum: expected %s got %s", tc.expectedChecksum, chk))
			}
		})
	}
}

func Test_getDeploymentParametersChecksum(t *testing.T) {
	testCases := map[string]testData{
		"case 0: Default test data":              defaultTestData(),
		"case 1: Changed Admin Username":         defaultTestData().WithadminUsername("giantswarm2"),
		"case 2: Changed SSH key":                defaultTestData().WithadminSSHKeyData("ssh-rsa AAAAB3aC1yc...k+y+ls2D0xJfqxw=="),
		"case 3: Changed OS Image Offer":         defaultTestData().WithosImageOffer("Ubuntu"),
		"case 4: Changed OS Image Publisher":     defaultTestData().WithosImagePublisher("Canonical"),
		"case 5: Changed OS Image SKU":           defaultTestData().WithosImageSKU("LTS"),
		"case 6: Changed OS Image Version":       defaultTestData().WithosImageVersion("18.04"),
		"case 7: Changed VM Size":                defaultTestData().WithvmSize("very_sml"),
		"case 8: Changed Docker Volume Size":     defaultTestData().WithdockerVolumeSizeGB(100),
		"case 9: Changed Master Blob Url":        defaultTestData().WithmasterBlobUrl("http://www.giantwarm.io"),
		"case 10: Changed Master Encryption Key": defaultTestData().WithmasterEncryptionKey("0123456789abcdef"),
		"case 11: Changed Master Initial Vector": defaultTestData().WithmasterInitialVector("fedcba9876543210"),
		"case 12: Changed Worker Blob Url":       defaultTestData().WithworkerBlobUrl("http://www.giantwarm.io"),
		"case 13: Changed Worker Encryption Key": defaultTestData().WithworkerEncryptionKey("0123456789abcdef"),
		"case 14: Changed Worker Initial Vector": defaultTestData().WithworkerInitialVector("fedcba9876543210"),
		"case 15: Changed MasterLB Backend Pool": defaultTestData().WithmasterLBBackendPoolID("/just/a/test"),
		"case 16: Changed Cluster ID":            defaultTestData().WithclusterID("abcde"),
		"case 17: Changed Master Subnet ID":      defaultTestData().WithmasterSubnetID("/and/another/one"),
		"case 18: Change VMSS MSIE enabled":      defaultTestData().WithvmssMSIEnabled(false),
		"case 19: Changed Worker Subnet ID":      defaultTestData().WithworkerSubnetID("/and/the/last/one"),
		"case 20: Added a new field":             defaultTestData().WithadditionalFields(map[string]string{"additional": "field"}),
		"case 21: Removed a field":               defaultTestData().WithremovedFields([]string{"masterSubnetID"}),
		"case 22: Changed the cloud config tmpl": defaultTestData().WithcloudConfigSmallTemplates([]string{"{}"}),
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			deployment, err := getDeployment(tc)
			if err != nil {
				t.Fatalf("Unable to construct a deployment: %v", err)
			}

			chk, err := GetDeploymentParametersChecksum(*deployment)
			if err != nil {
				t.Fatalf("Unexpected error")
			}

			if tc.checksumIs != nil && chk != *tc.checksumIs {
				t.Fatalf("Checksum calculation invalid %s", chk)
			}

			if tc.checksumIsNot != nil && chk == *tc.checksumIsNot {
				t.Fatalf("Expected checksum to change but it didn't")
			}
		})
	}
}

type testData struct {
	adminUsername             string
	adminSSHKeyData           string
	osImageOffer              string
	osImagePublisher          string
	osImageSKU                string
	osImageVersion            string
	vmSize                    string
	dockerVolumeSizeGB        int
	masterBlobUrl             string
	masterEncryptionKey       string
	masterInitialVector       string
	masterInstanceRole        string
	workerBlobUrl             string
	workerEncryptionKey       string
	workerInitialVector       string
	workerInstanceRole        string
	masterLBBackendPoolID     string
	clusterID                 string
	masterSubnetID            string
	vmssMSIEnabled            bool
	workerSubnetID            string
	additionalFields          map[string]string
	removedFields             []string
	cloudConfigSmallTemplates []string

	checksumIs    *string
	checksumIsNot *string
}

func defaultTestData() testData {
	return testData{
		adminUsername:             "giantswarm",
		adminSSHKeyData:           "ssh-rsa AAAAB3NzaC1yc...k+y+ls2D0xJfqxw==",
		osImageOffer:              "CoreOS",
		osImagePublisher:          "CoreOS",
		osImageSKU:                "Stable",
		osImageVersion:            "2191.5.0",
		vmSize:                    "Standard_D4s_v3",
		dockerVolumeSizeGB:        50,
		masterBlobUrl:             "https://gssatjb62.blob.core.windows.net/ignition/2.8.0-v4.7.0-worker?se=2020-05-18T13%3A60%3A03Z&sig=9tXJCWxsZb6MxBQZDDbVykB3VMs0CxxoIDHJtpKs10g%3D&sp=r&spr=https&sr=b&sv=2018-03-28",
		masterEncryptionKey:       "00112233445566778899aabbccddeeff00112233445566778899aabbccddee",
		masterInitialVector:       "0011223344556677889900aabbccddee",
		masterInstanceRole:        "master",
		workerBlobUrl:             "https://gssatjb62.blob.core.windows.net/ignition/2.8.0-v4.7.0-worker?se=2020-05-18T13%3A61%3A03Z&sig=9tXJCWxsZb6MxBQZDDbVykB3VMs0CxxoIDHJtpKs10g%3D&sp=r&spr=https&sr=b&sv=2018-03-28",
		workerEncryptionKey:       "eeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100",
		workerInitialVector:       "eeddccbbaa0099887766554433221100",
		workerInstanceRole:        "worker",
		masterLBBackendPoolID:     "/subscriptions/746379f9-ad35-1d92-1829-cba8579d71e6/resourceGroups/tjb62/providers/Microsoft.Network/loadBalancers/tjb62-API-PublicLoadBalancer/backendAddressPools/tjb62-API-PublicLoadBalancer-BackendPool",
		clusterID:                 "tjb62",
		masterSubnetID:            "/subscriptions/746379f9-ad35-1d92-1829-cba8579d71e6/resourceGroups/tjb62/providers/Microsoft.Network/virtualNetworks/tjb62-VirtualNetwork/subnets/tjb62-VirtualNetwork-MasterSubnet",
		vmssMSIEnabled:            true,
		workerSubnetID:            "/subscriptions/746379f9-ad35-1d92-1829-cba8579d71e6/resourceGroups/tjb62/providers/Microsoft.Network/virtualNetworks/tjb62-VirtualNetwork/subnets/tjb62-VirtualNetwork-WorkerSubnet",
		additionalFields:          nil,
		removedFields:             nil,
		cloudConfigSmallTemplates: key.CloudConfigSmallTemplates(),

		checksumIs:    to.StringPtr("ffa47332b6cac79fd0976cb867be0af03d01ffaead0ca7a467404938d7e653e9"),
		checksumIsNot: nil,
	}
}

func (td testData) WithadminUsername(data string) testData {
	td.adminUsername = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithadminSSHKeyData(data string) testData {
	td.adminSSHKeyData = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithosImageOffer(data string) testData {
	td.osImageOffer = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithosImagePublisher(data string) testData {
	td.osImagePublisher = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithosImageSKU(data string) testData {
	td.osImageSKU = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithosImageVersion(data string) testData {
	td.osImageVersion = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithvmSize(data string) testData {
	td.vmSize = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithdockerVolumeSizeGB(data int) testData {
	td.dockerVolumeSizeGB = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithmasterBlobUrl(data string) testData {
	td.masterBlobUrl = data
	// checksum isn't expected to change

	return td
}

func (td testData) WithmasterEncryptionKey(data string) testData {
	td.masterEncryptionKey = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithmasterInitialVector(data string) testData {
	td.masterInitialVector = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithworkerBlobUrl(data string) testData {
	td.workerBlobUrl = data
	// checksum isn't expected to change

	return td
}

func (td testData) WithworkerEncryptionKey(data string) testData {
	td.workerEncryptionKey = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithworkerInitialVector(data string) testData {
	td.workerInitialVector = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithmasterLBBackendPoolID(data string) testData {
	td.masterLBBackendPoolID = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithclusterID(data string) testData {
	td.clusterID = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithmasterSubnetID(data string) testData {
	td.masterSubnetID = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithvmssMSIEnabled(data bool) testData {
	td.vmssMSIEnabled = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithworkerSubnetID(data string) testData {
	td.workerSubnetID = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithadditionalFields(data map[string]string) testData {
	td.additionalFields = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithremovedFields(data []string) testData {
	td.removedFields = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func (td testData) WithcloudConfigSmallTemplates(data []string) testData {
	td.cloudConfigSmallTemplates = data
	td.checksumIsNot = td.checksumIs
	td.checksumIs = nil

	return td
}

func getDeployment(data testData) (*resources.Deployment, error) {
	nodes := []vmss.Node{
		{
			AdminUsername:   data.adminUsername,
			AdminSSHKeyData: data.adminSSHKeyData,
			OSImage: vmss.NodeOSImage{
				Offer:     data.osImageOffer,
				Publisher: data.osImagePublisher,
				SKU:       data.osImageSKU,
				Version:   data.osImageVersion,
			},
			VMSize:             data.vmSize,
			DockerVolumeSizeGB: data.dockerVolumeSizeGB,
		},
	}

	_ = struct {
	}{}

	c := vmss.SmallCloudconfigConfig{
		BlobURL:       data.masterBlobUrl,
		EncryptionKey: data.masterEncryptionKey,
		InitialVector: data.masterInitialVector,
		InstanceRole:  data.masterInstanceRole,
	}
	masterCloudConfig, err := templates.Render(data.cloudConfigSmallTemplates, c)
	if err != nil {
		return nil, err
	}
	encodedMasterCloudConfig := base64.StdEncoding.EncodeToString([]byte(masterCloudConfig))

	c = vmss.SmallCloudconfigConfig{
		BlobURL:       data.workerBlobUrl,
		EncryptionKey: data.workerEncryptionKey,
		InitialVector: data.workerInitialVector,
		InstanceRole:  data.workerInstanceRole,
	}
	workerCloudConfig, err := templates.Render(data.cloudConfigSmallTemplates, c)
	if err != nil {
		return nil, err
	}
	encodedWorkerCloudConfig := base64.StdEncoding.EncodeToString([]byte(workerCloudConfig))

	parameters := map[string]interface{}{
		"masterLBBackendPoolID": data.masterLBBackendPoolID,
		"clusterID":             data.clusterID,
		"masterCloudConfigData": struct{ Value interface{} }{Value: encodedMasterCloudConfig},
		"masterNodes":           nodes,
		"masterSubnetID":        data.masterSubnetID,
		"vmssMSIEnabled":        data.vmssMSIEnabled,
		"workerCloudConfigData": struct{ Value interface{} }{Value: encodedWorkerCloudConfig},
		"workerNodes":           nodes,
		"workerSubnetID":        data.workerSubnetID,
	}

	if data.additionalFields != nil {
		for k, v := range data.additionalFields {
			parameters[k] = v
		}
	}

	if data.removedFields != nil {
		for _, v := range data.removedFields {
			_, ok := parameters[v]
			if !ok {
				panic(fmt.Sprintf("Field '%s' was not found for removal", v))
			}

			delete(parameters, v)
		}
	}

	properties := resources.DeploymentProperties{
		Parameters: parameters,
	}
	return &resources.Deployment{Properties: &properties}, nil
}
