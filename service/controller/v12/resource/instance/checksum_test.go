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
	"testing"
)

func Test_getDeploymentTemplateChecksum(t *testing.T) {
	testCases := []struct {
		Name                string
		TemplateLinkPresent bool
		StatusCode          int
		ResponseBody        string
		ExpectedChecksum    string
		ErrorMatcher        func(err error) bool
	}{
		{
			Name:                "Case 1: Successful checksum calculation",
			TemplateLinkPresent: true,
			StatusCode:          http.StatusOK,
			ResponseBody:        `{"fake": "json string"}`,
			ExpectedChecksum:    "0cfe91509c17c2a9f230cd117d90e837d948639c3a2d559cf1ef6ca6ae24ec79",
		},
		{
			Name:                "Case 2: Missing template link",
			TemplateLinkPresent: false,
			ExpectedChecksum:    "",
			ErrorMatcher:        IsNilTemplateLinkError,
		},
		{
			Name:                "Case 3: Error downloading template from external URI",
			TemplateLinkPresent: true,
			ExpectedChecksum:    "",
			StatusCode:          http.StatusInternalServerError,
			ResponseBody:        `{"error": "500 - Internal server error"}`,
			ErrorMatcher:        IsUnableToGetTemplateError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.StatusCode)
				w.Write([]byte(tc.ResponseBody))
			}))
			defer ts.Close()

			var templateLink *resources.TemplateLink
			if tc.TemplateLinkPresent {
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

			if chk != tc.ExpectedChecksum {
				t.Fatal(fmt.Sprintf("Wrong checksum: expected %s got %s", tc.ExpectedChecksum, chk))
			}

			switch {
			case err == nil && tc.ErrorMatcher == nil:
				// fall through
			case err != nil && tc.ErrorMatcher == nil:
				t.Fatalf("expected %#v got %#v", nil, err)
			case err == nil && tc.ErrorMatcher != nil:
				t.Fatalf("expected %#v got %#v", "error", nil)
			case !tc.ErrorMatcher(err):
				t.Fatalf("expected %#v got %#v", true, false)
			}
		})
	}
}

func Test_getDeploymentParametersChecksum(t *testing.T) {
	deployment, err := getDeployment()
	if err != nil {
		t.Fatalf("Unable to construct a deployment: %v", err)
	}

	chk, err := getDeploymentParametersChecksum(*deployment)

	if err != nil {
		t.Fatalf("Unexpected error")
	}

	if chk != "dd10f2e2f6877ba305a2af480687f06402ac0cb9835516662835dd0884337a44" {
		t.Fatalf("Checksum calculation invalid")
	}
}

func getDeployment() (*resources.Deployment, error) {
	nodes := []node{
		{
			AdminUsername:   "giantswarm",
			AdminSSHKeyData: "ssh-rsa AAAAB3NzaC1yc...k+y+ls2D0xJfqxw==",
			OSImage: nodeOSImage{
				Offer:     "CoreOS",
				Publisher: "CoreOS",
				SKU:       "Stable",
				Version:   "2191.5.0",
			},
			VMSize:             "Standard_D4s_v3",
			DockerVolumeSizeGB: 50,
		},
	}

	c := SmallCloudconfigConfig{
		BlobURL:       "https://gssatjb62.blob.core.windows.net/ignition/2.8.0-v4.7.0-worker?se=2020-05-18T13%3A60%3A03Z&sig=9tXJCWxsZb6MxBQZDDbVykB3VMs0CxxoIDHJtpKs10g%3D&sp=r&spr=https&sr=b&sv=2018-03-28",
		EncryptionKey: "00112233445566778899aabbccddeeff00112233445566778899aabbccddee",
		InitialVector: "0011223344556677889900aabbccddee",
		InstanceRole:  "master",
	}
	masterCloudConfig, err := templates.Render(key.CloudConfigSmallTemplates(), c)
	if err != nil {
		return nil, err
	}
	encodedMasterCloudConfig := base64.StdEncoding.EncodeToString([]byte(masterCloudConfig))

	c = SmallCloudconfigConfig{
		BlobURL:       "https://gssatjb62.blob.core.windows.net/ignition/2.8.0-v4.7.0-worker?se=2020-05-18T13%3A61%3A03Z&sig=9tXJCWxsZb6MxBQZDDbVykB3VMs0CxxoIDHJtpKs10g%3D&sp=r&spr=https&sr=b&sv=2018-03-28",
		EncryptionKey: "eeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100",
		InitialVector: "eeddccbbaa0099887766554433221100",
		InstanceRole:  "worker",
	}
	workerCloudConfig, err := templates.Render(key.CloudConfigSmallTemplates(), c)
	if err != nil {
		return nil, err
	}
	encodedWorkerCloudConfig := base64.StdEncoding.EncodeToString([]byte(workerCloudConfig))

	parameters := map[string]interface{}{
		"apiLBBackendPoolID":    "/subscriptions/746379f9-ad35-1d92-1829-cba8579d71e6/resourceGroups/tjb62/providers/Microsoft.Network/loadBalancers/tjb62-API-PublicLoadBalancer/backendAddressPools/tjb62-API-PublicLoadBalancer-BackendPool",
		"clusterID":             "tjb62",
		"etcdLBBackendPoolID":   "/subscriptions/746379f9-ad35-1d92-1829-cba8579d71e6/resourceGroups/tjb62/providers/Microsoft.Network/loadBalancers/tjb62-ETCD-PrivateLoadBalancer/backendAddressPools/tjb62-ETCD-PrivateLoadBalancer-BackendPool", // string
		"masterCloudConfigData": struct{ Value interface{} }{Value: encodedMasterCloudConfig},
		"masterNodes":           nodes,
		"masterSubnetID":        "/subscriptions/746379f9-ad35-1d92-1829-cba8579d71e6/resourceGroups/tjb62/providers/Microsoft.Network/virtualNetworks/tjb62-VirtualNetwork/subnets/tjb62-VirtualNetwork-MasterSubnet",
		"vmssMSIEnabled":        true,
		"workerCloudConfigData": struct{ Value interface{} }{Value: encodedWorkerCloudConfig},
		"workerNodes":           nodes,
		"workerSubnetID":        "/subscriptions/746379f9-ad35-1d92-1829-cba8579d71e6/resourceGroups/tjb62/providers/Microsoft.Network/virtualNetworks/tjb62-VirtualNetwork/subnets/tjb62-VirtualNetwork-WorkerSubnet",
	}

	properties := resources.DeploymentProperties{
		Parameters: parameters,
	}
	return &resources.Deployment{Properties: &properties}, nil
}
