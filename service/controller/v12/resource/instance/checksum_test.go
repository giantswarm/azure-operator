package instance

import (
	"encoding/base64"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/azure-operator/service/controller/v12/key"
	"github.com/giantswarm/azure-operator/service/controller/v12/templates"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func Test_getDeploymentTemplateChecksum(t *testing.T) {
	testCases := []struct {
		name                string
		templateLinkPresent bool
		statusCode          int
		responseBody        string
		expectedChecksum    string
		errorMatcher        func(err error) bool
	}{
		{
			name:                "case 0: Successful checksum calculation",
			templateLinkPresent: true,
			statusCode:          http.StatusOK,
			responseBody:        `{"fake": "json string"}`,
			expectedChecksum:    "0cfe91509c17c2a9f230cd117d90e837d948639c3a2d559cf1ef6ca6ae24ec79",
		},
		{
			name:                "case 1: Missing template link",
			templateLinkPresent: false,
			expectedChecksum:    "",
			errorMatcher:        IsNilTemplateLinkError,
		},
		{
			name:                "case 2: Error downloading template from external URI",
			templateLinkPresent: true,
			expectedChecksum:    "",
			statusCode:          http.StatusInternalServerError,
			responseBody:        `{"error": "500 - Internal server error"}`,
			errorMatcher:        IsUnableToGetTemplateError,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.responseBody))
			}))
			defer ts.Close()

			var templateLink *resources.TemplateLink
			if tc.templateLinkPresent {
				templateLink = &resources.TemplateLink{
					URI: to.StringPtr(ts.URL),
				}
			}

			properties := resources.DeploymentProperties{
				TemplateLink: templateLink,
			}
			deployment := resources.Deployment{
				Properties: &properties,
			}

			chk, err := getDeploymentTemplateChecksum(deployment)

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
		"case 1: Changed Admin Username":         defaultTestData().WithAdminUsername("giantswarm2"),
		"case 2: Changed SSH key":                defaultTestData().WithAdminSSHKeyData("ssh-rsa AAAAB3aC1yc...k+y+ls2D0xJfqxw=="),
		"case 3: Changed OS Image Offer":         defaultTestData().WithOSImageOffer("Ubuntu"),
		"case 4: Changed OS Image Publisher":     defaultTestData().WithOSImagePublisher("Canonical"),
		"case 5: Changed OS Image SKU":           defaultTestData().WithOSImageSKU("LTS"),
		"case 6: Changed OS Image Version":       defaultTestData().WithOSImageVersion("18.04"),
		"case 7: Changed VM Size":                defaultTestData().WithVMSize("very_sml"),
		"case 8: Changed Docker Volume Size":     defaultTestData().WithDockerVolumeSizeGB(100),
		"case 9: Changed Master Blob Url":        defaultTestData().WithMasterBlobUrl("http://www.giantwarm.io"),
		"case 10: Changed Master Encryption Key": defaultTestData().WithMasterEncryptionKey("0123456789abcdef"),
		"case 11: Changed Master Initial Vector": defaultTestData().WithMasterInitialVector("fedcba9876543210"),
		"case 12: Changed Worker Blob Url":       defaultTestData().WithWorkerBlobUrl("http://www.giantwarm.io"),
		"case 13: Changed Worker Encryption Key": defaultTestData().WithWorkerEncryptionKey("0123456789abcdef"),
		"case 14: Changed Worker Initial Vector": defaultTestData().WithWorkerInitialVector("fedcba9876543210"),
		"case 15: Changed Api LB Backend Pool":   defaultTestData().WithApiLBBackendPoolID("/just/a/test"),
		"case 16: Changed Cluster ID":            defaultTestData().WithClusterID("abcde"),
		"case 17: Changed ETCD LB Backend Pool":  defaultTestData().WithEtcdLBBackendPoolID("/and/another/test"),
		"case 18: Changed Master Subnet ID":      defaultTestData().WithMasterSubnetID("/and/another/one"),
		"case 19: Change VMSS MSIE enabled":      defaultTestData().WithVmssMSIEnabled(false),
		"case 20: Changed Worker Subnet ID":      defaultTestData().WithWorkerSubnetID("/and/the/last/one"),
		"case 21: Added a new field":             defaultTestData().WithAdditionalFields(map[string]string{"additional": "field"}),
		"case 22: Removed a field":               defaultTestData().WithRemovedFields([]string{"masterSubnetID"}),
		"case 23: Changed the cloud config tmpl": defaultTestData().WithCloudConfigSmallTemplates([]string{"{}"}),
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			deployment, err := getDeployment(tc)
			if err != nil {
				t.Fatalf("Unable to construct a deployment: %v", err)
			}

			chk, err := getDeploymentParametersChecksum(*deployment)
			if err != nil {
				t.Fatalf("Unexpected error")
			}

			if tc.ChecksumIs != nil && chk != *tc.ChecksumIs {
				t.Fatalf("Checksum calculation invalid %s", chk)
			}

			if tc.ChecksumIsNot != nil && chk == *tc.ChecksumIsNot {
				t.Fatalf("Expected checksum to change but it didn't")
			}
		})
	}
}

type testData struct {
	AdminUsername             string
	AdminSSHKeyData           string
	OSImageOffer              string
	OSImagePublisher          string
	OSImageSKU                string
	OSImageVersion            string
	VMSize                    string
	DockerVolumeSizeGB        int
	MasterBlobUrl             string
	MasterEncryptionKey       string
	MasterInitialVector       string
	MasterInstanceRole        string
	WorkerBlobUrl             string
	WorkerEncryptionKey       string
	WorkerInitialVector       string
	WorkerInstanceRole        string
	ApiLBBackendPoolID        string
	ClusterID                 string
	EtcdLBBackendPoolID       string
	MasterSubnetID            string
	VmssMSIEnabled            bool
	WorkerSubnetID            string
	AdditionalFields          map[string]string
	RemovedFields             []string
	CloudConfigSmallTemplates []string

	ChecksumIs    *string
	ChecksumIsNot *string
}

func defaultTestData() testData {
	return testData{
		AdminUsername:             "giantswarm",
		AdminSSHKeyData:           "ssh-rsa AAAAB3NzaC1yc...k+y+ls2D0xJfqxw==",
		OSImageOffer:              "CoreOS",
		OSImagePublisher:          "CoreOS",
		OSImageSKU:                "Stable",
		OSImageVersion:            "2191.5.0",
		VMSize:                    "Standard_D4s_v3",
		DockerVolumeSizeGB:        50,
		MasterBlobUrl:             "https://gssatjb62.blob.core.windows.net/ignition/2.8.0-v4.7.0-worker?se=2020-05-18T13%3A60%3A03Z&sig=9tXJCWxsZb6MxBQZDDbVykB3VMs0CxxoIDHJtpKs10g%3D&sp=r&spr=https&sr=b&sv=2018-03-28",
		MasterEncryptionKey:       "00112233445566778899aabbccddeeff00112233445566778899aabbccddee",
		MasterInitialVector:       "0011223344556677889900aabbccddee",
		MasterInstanceRole:        "master",
		WorkerBlobUrl:             "https://gssatjb62.blob.core.windows.net/ignition/2.8.0-v4.7.0-worker?se=2020-05-18T13%3A61%3A03Z&sig=9tXJCWxsZb6MxBQZDDbVykB3VMs0CxxoIDHJtpKs10g%3D&sp=r&spr=https&sr=b&sv=2018-03-28",
		WorkerEncryptionKey:       "eeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100",
		WorkerInitialVector:       "eeddccbbaa0099887766554433221100",
		WorkerInstanceRole:        "worker",
		ApiLBBackendPoolID:        "/subscriptions/746379f9-ad35-1d92-1829-cba8579d71e6/resourceGroups/tjb62/providers/Microsoft.Network/loadBalancers/tjb62-API-PublicLoadBalancer/backendAddressPools/tjb62-API-PublicLoadBalancer-BackendPool",
		ClusterID:                 "tjb62",
		EtcdLBBackendPoolID:       "/subscriptions/746379f9-ad35-1d92-1829-cba8579d71e6/resourceGroups/tjb62/providers/Microsoft.Network/loadBalancers/tjb62-ETCD-PrivateLoadBalancer/backendAddressPools/tjb62-ETCD-PrivateLoadBalancer-BackendPool", // string
		MasterSubnetID:            "/subscriptions/746379f9-ad35-1d92-1829-cba8579d71e6/resourceGroups/tjb62/providers/Microsoft.Network/virtualNetworks/tjb62-VirtualNetwork/subnets/tjb62-VirtualNetwork-MasterSubnet",
		VmssMSIEnabled:            true,
		WorkerSubnetID:            "/subscriptions/746379f9-ad35-1d92-1829-cba8579d71e6/resourceGroups/tjb62/providers/Microsoft.Network/virtualNetworks/tjb62-VirtualNetwork/subnets/tjb62-VirtualNetwork-WorkerSubnet",
		AdditionalFields:          nil,
		RemovedFields:             nil,
		CloudConfigSmallTemplates: key.CloudConfigSmallTemplates(),

		ChecksumIs:    to.StringPtr("5bd677fda75a9855203689725977c4d3118b3a0f8204674266bab7cf1ee2881b"),
		ChecksumIsNot: nil,
	}
}

func (td testData) WithAdminUsername(data string) testData {
	td.AdminUsername = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithAdminSSHKeyData(data string) testData {
	td.AdminSSHKeyData = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithOSImageOffer(data string) testData {
	td.OSImageOffer = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithOSImagePublisher(data string) testData {
	td.OSImagePublisher = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithOSImageSKU(data string) testData {
	td.OSImageSKU = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithOSImageVersion(data string) testData {
	td.OSImageVersion = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithVMSize(data string) testData {
	td.VMSize = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithDockerVolumeSizeGB(data int) testData {
	td.DockerVolumeSizeGB = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithMasterBlobUrl(data string) testData {
	td.MasterBlobUrl = data
	// checksum isn't expected to change

	return td
}

func (td testData) WithMasterEncryptionKey(data string) testData {
	td.MasterEncryptionKey = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithMasterInitialVector(data string) testData {
	td.MasterInitialVector = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithWorkerBlobUrl(data string) testData {
	td.WorkerBlobUrl = data
	// checksum isn't expected to change

	return td
}

func (td testData) WithWorkerEncryptionKey(data string) testData {
	td.WorkerEncryptionKey = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithWorkerInitialVector(data string) testData {
	td.WorkerInitialVector = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithApiLBBackendPoolID(data string) testData {
	td.ApiLBBackendPoolID = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithClusterID(data string) testData {
	td.ClusterID = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithEtcdLBBackendPoolID(data string) testData {
	td.EtcdLBBackendPoolID = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithMasterSubnetID(data string) testData {
	td.MasterSubnetID = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithVmssMSIEnabled(data bool) testData {
	td.VmssMSIEnabled = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithWorkerSubnetID(data string) testData {
	td.WorkerSubnetID = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithAdditionalFields(data map[string]string) testData {
	td.AdditionalFields = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithRemovedFields(data []string) testData {
	td.RemovedFields = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func (td testData) WithCloudConfigSmallTemplates(data []string) testData {
	td.CloudConfigSmallTemplates = data
	td.ChecksumIsNot = td.ChecksumIs
	td.ChecksumIs = nil

	return td
}

func getDeployment(data testData) (*resources.Deployment, error) {
	nodes := []node{
		{
			AdminUsername:   data.AdminUsername,
			AdminSSHKeyData: data.AdminSSHKeyData,
			OSImage: nodeOSImage{
				Offer:     data.OSImageOffer,
				Publisher: data.OSImagePublisher,
				SKU:       data.OSImageSKU,
				Version:   data.OSImageVersion,
			},
			VMSize:             data.VMSize,
			DockerVolumeSizeGB: data.DockerVolumeSizeGB,
		},
	}

	_ = struct {
	}{}

	c := SmallCloudconfigConfig{
		BlobURL:       data.MasterBlobUrl,
		EncryptionKey: data.MasterEncryptionKey,
		InitialVector: data.MasterInitialVector,
		InstanceRole:  data.MasterInstanceRole,
	}
	masterCloudConfig, err := templates.Render(data.CloudConfigSmallTemplates, c)
	if err != nil {
		return nil, err
	}
	encodedMasterCloudConfig := base64.StdEncoding.EncodeToString([]byte(masterCloudConfig))

	c = SmallCloudconfigConfig{
		BlobURL:       data.WorkerBlobUrl,
		EncryptionKey: data.WorkerEncryptionKey,
		InitialVector: data.WorkerInitialVector,
		InstanceRole:  data.WorkerInstanceRole,
	}
	workerCloudConfig, err := templates.Render(data.CloudConfigSmallTemplates, c)
	if err != nil {
		return nil, err
	}
	encodedWorkerCloudConfig := base64.StdEncoding.EncodeToString([]byte(workerCloudConfig))

	parameters := map[string]interface{}{
		"apiLBBackendPoolID":    data.ApiLBBackendPoolID,
		"clusterID":             data.ClusterID,
		"etcdLBBackendPoolID":   data.EtcdLBBackendPoolID,
		"masterCloudConfigData": struct{ Value interface{} }{Value: encodedMasterCloudConfig},
		"masterNodes":           nodes,
		"masterSubnetID":        data.MasterSubnetID,
		"vmssMSIEnabled":        data.VmssMSIEnabled,
		"workerCloudConfigData": struct{ Value interface{} }{Value: encodedWorkerCloudConfig},
		"workerNodes":           nodes,
		"workerSubnetID":        data.WorkerSubnetID,
	}

	if data.AdditionalFields != nil {
		for k, v := range data.AdditionalFields {
			parameters[k] = v
		}
	}

	if data.RemovedFields != nil {
		for _, v := range data.RemovedFields {
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
