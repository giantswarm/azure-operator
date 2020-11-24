package workermigration

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/go-autorest/autorest"
	corev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/giantswarm/to"
	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	azureclient "github.com/giantswarm/azure-operator/v5/client"
	"github.com/giantswarm/azure-operator/v5/pkg/mock/mock_tenantcluster"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/masters"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/workermigration/internal/azure"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/workermigration/internal/mock_azure"
)

//go:generate mockgen -destination internal/mock_azure/api.go -source internal/azure/spec.go API

func TestMigrationWaitsForMasterUpgradeToFinish(t *testing.T) {
	masterUpgradePendingStatuses := []string{
		"ClusterUpgradeRequirementCheck",
		"DeploymentUninitialized",
		"DeploymentInitialized",
		"",
		"MasterInstancesUpgrading",
		"ProvisioningSuccessful",
		"WaitForMastersToBecomeReady",
	}

	for i, s := range masterUpgradePendingStatuses {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatal("master is upgrading but condition was not respected")
				}
			}()

			t.Log("master status condition: " + s)

			ctrlClient := newFakeClient()
			r := &Resource{
				ctrlClient:   ctrlClient,
				logger:       microloggertest.New(),
				wrapAzureAPI: func(cf *azureclient.Factory, credentials *providerv1alpha1.CredentialSecret) azure.API { return nil },
			}

			ensureCRsExist(t, ctrlClient, []string{
				"azureconfig.yaml",
			})

			o, err := loadCR("azureconfig.yaml")
			if err != nil {
				t.Fatal(err)
			}
			cr := o.(*providerv1alpha1.AzureConfig)

			err = setStatusCondition(ctrlClient, cr, masters.Name, Stage, s)
			if err != nil {
				t.Fatal(err)
			}

			err = r.EnsureCreated(context.Background(), cr)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMigrationCreatesMachinePoolCRs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctrlClient := newFakeClient()
	m := mock_azure.NewMockAPI(ctrl)
	r := &Resource{
		ctrlClient:   ctrlClient,
		logger:       microloggertest.New(),
		wrapAzureAPI: func(cf *azureclient.Factory, credentials *providerv1alpha1.CredentialSecret) azure.API { return m },
	}

	ensureCRsExist(t, ctrlClient, []string{
		"cluster.yaml",
		"azureconfig.yaml",
		"azurecluster.yaml",
	})

	o, err := loadCR("azureconfig.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cr := o.(*providerv1alpha1.AzureConfig)

	m.
		EXPECT().
		ListNetworkSecurityGroups(gomock.Any(), key.ResourceGroupName(*cr)).
		Return(nil, nil).
		Times(1)

	m.
		EXPECT().
		GetVMSS(gomock.Any(), key.ResourceGroupName(*cr), key.WorkerVMSSName(*cr)).
		Return(newBuiltinVMSS(3, key.WorkerVMSSName(*cr)), nil).
		Times(1)

	m.
		EXPECT().
		DeleteVMSS(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	err = r.EnsureCreated(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}

	// VERIFY: MachinePool is there.
	{
		opts := client.MatchingLabels{
			capiv1alpha3.ClusterLabelName: key.ClusterID(cr),
		}
		mpList := new(expcapiv1alpha3.MachinePoolList)
		err = ctrlClient.List(context.Background(), mpList, opts)
		if err != nil {
			t.Fatal(err)
		}

		if len(mpList.Items) == 0 {
			t.Fatal("expected at least one MachinePool CR to exist. got 0.")
		}
	}

	// VERIFY: AzureMachinePool is there.
	{
		opts := client.MatchingLabels{
			capiv1alpha3.ClusterLabelName: key.ClusterName(cr),
		}
		mpList := new(expcapzv1alpha3.AzureMachinePoolList)
		err = ctrlClient.List(context.Background(), mpList, opts)
		if err != nil {
			t.Fatal(err)
		}

		if len(mpList.Items) == 0 {
			t.Fatal("expected at least one AzureMachinePool CR to exist. got 0.")
		}
	}

	// VERIFY: Spark CR is there.
	{
		opts := client.MatchingLabels{
			capiv1alpha3.ClusterLabelName: key.ClusterName(cr),
		}
		sparkList := new(corev1alpha1.SparkList)
		err = ctrlClient.List(context.Background(), sparkList, opts)
		if err != nil {
			t.Fatal(err)
		}

		if len(sparkList.Items) == 0 {
			t.Fatal("expected at least one Spark CR to exist. got 0.")
		}
	}

	// gomock verifies rest of the assertions on exit.
}

func TestMigrationCreatesDrainerConfigCRs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctrlClient := newFakeClient()
	mockAzureAPI := mock_azure.NewMockAPI(ctrl)
	mockTenantClusterFactory := mock_tenantcluster.NewMockFactory(ctrl)
	r := &Resource{
		ctrlClient:          ctrlClient,
		logger:              microloggertest.New(),
		tenantClientFactory: mockTenantClusterFactory,
		wrapAzureAPI: func(cf *azureclient.Factory, credentials *providerv1alpha1.CredentialSecret) azure.API {
			return mockAzureAPI
		},
	}

	ensureCRsExist(t, ctrlClient, []string{
		"cluster.yaml",
		"azureconfig.yaml",
		"azurecluster.yaml",
		"azuremachinepool.yaml",
		"machinepool.yaml",
		"namespace.yaml",
		"spark.yaml",
	})

	o, err := loadCR("azureconfig.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cr := o.(*providerv1alpha1.AzureConfig)

	ensureNodePoolIsReady(t, ctrlClient, cr)

	mockAzureAPI.
		EXPECT().
		ListNetworkSecurityGroups(gomock.Any(), key.ResourceGroupName(*cr)).
		Return(nil, nil).
		Times(1)

	mockAzureAPI.
		EXPECT().
		GetVMSS(gomock.Any(), key.ResourceGroupName(*cr), key.WorkerVMSSName(*cr)).
		Return(newBuiltinVMSS(3, key.WorkerVMSSName(*cr)), nil).
		Times(1)

	mockAzureAPI.
		EXPECT().
		DeleteVMSS(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	mockAzureAPI.
		EXPECT().
		ListVMSSNodes(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(workersAsVMSSNodes(*cr), nil).
		Times(1)

	err = r.EnsureCreated(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}

	// VERIFY: DrainerConfig CRs are there.
	{
		opts := client.MatchingLabels{
			capiv1alpha3.ClusterLabelName: key.ClusterID(cr),
		}
		dcList := new(corev1alpha1.DrainerConfigList)
		err = ctrlClient.List(context.Background(), dcList, opts)
		if err != nil {
			t.Fatal(err)
		}

		if len(dcList.Items) != key.WorkerCount(*cr) {
			t.Fatalf("expected %d drainer config crs to exist. got %d.", key.WorkerCount(*cr), len(dcList.Items))
		}
	}

	// gomock verifies rest of the assertions on exit.
}

func TestVMSSIsNotDeletedBeforeDrainingIsDone(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctrlClient := newFakeClient()
	mockAzureAPI := mock_azure.NewMockAPI(ctrl)
	mockTenantCluster := mock_tenantcluster.NewMockFactory(ctrl)
	r := &Resource{
		ctrlClient:          ctrlClient,
		logger:              microloggertest.New(),
		tenantClientFactory: mockTenantCluster,
		wrapAzureAPI: func(cf *azureclient.Factory, credentials *providerv1alpha1.CredentialSecret) azure.API {
			return mockAzureAPI
		},
	}

	ensureCRsExist(t, ctrlClient, []string{
		"cluster.yaml",
		"azureconfig.yaml",
		"azurecluster.yaml",
		"azuremachinepool.yaml",
		"machinepool.yaml",
		"namespace.yaml",
		"spark.yaml",
		"drainerconfigs.yaml",
	})

	o, err := loadCR("azureconfig.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cr := o.(*providerv1alpha1.AzureConfig)

	ensureNodePoolIsReady(t, ctrlClient, cr)

	mockAzureAPI.
		EXPECT().
		ListNetworkSecurityGroups(gomock.Any(), key.ResourceGroupName(*cr)).
		Return(nil, nil).
		Times(1)

	mockAzureAPI.
		EXPECT().
		GetVMSS(gomock.Any(), key.ResourceGroupName(*cr), key.WorkerVMSSName(*cr)).
		Return(newBuiltinVMSS(3, key.WorkerVMSSName(*cr)), nil).
		Times(1)

	mockAzureAPI.
		EXPECT().
		DeleteVMSS(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	mockAzureAPI.
		EXPECT().
		ListVMSSNodes(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(workersAsVMSSNodes(*cr), nil).
		Times(1)

	err = r.EnsureCreated(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}

	// VERIFY: DrainerConfig CRs are there.
	{
		opts := client.MatchingLabels{
			capiv1alpha3.ClusterLabelName: key.ClusterID(cr),
		}
		dcList := new(corev1alpha1.DrainerConfigList)
		err = ctrlClient.List(context.Background(), dcList, opts)
		if err != nil {
			t.Fatal(err)
		}

		if len(dcList.Items) != key.WorkerCount(*cr) {
			t.Fatalf("expected %d drainer config crs to exist. got %d.", key.WorkerCount(*cr), len(dcList.Items))
		}
	}

	// gomock verifies rest of the assertions on exit.
}

func TestVMSSIsDeletedOnceDrainingIsDone(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctrlClient := newFakeClient()
	mockAzureAPI := mock_azure.NewMockAPI(ctrl)
	mockTenantClusterFactory := mock_tenantcluster.NewMockFactory(ctrl)
	r := &Resource{
		ctrlClient:          ctrlClient,
		logger:              microloggertest.New(),
		tenantClientFactory: mockTenantClusterFactory,
		wrapAzureAPI: func(cf *azureclient.Factory, credentials *providerv1alpha1.CredentialSecret) azure.API {
			return mockAzureAPI
		},
	}

	ensureCRsExist(t, ctrlClient, []string{
		"cluster.yaml",
		"azureconfig.yaml",
		"azurecluster.yaml",
		"azuremachinepool.yaml",
		"machinepool.yaml",
		"namespace.yaml",
		"spark.yaml",
		"drainerconfigs.yaml",
	})

	o, err := loadCR("azureconfig.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cr := o.(*providerv1alpha1.AzureConfig)

	ensureNodePoolIsReady(t, ctrlClient, cr)
	setDrainerConfigsAsDrained(t, ctrlClient, cr)

	mockAzureAPI.
		EXPECT().
		ListNetworkSecurityGroups(gomock.Any(), key.ResourceGroupName(*cr)).
		Return(nil, nil).
		Times(1)

	mockAzureAPI.
		EXPECT().
		GetVMSS(gomock.Any(), key.ResourceGroupName(*cr), key.WorkerVMSSName(*cr)).
		Return(newBuiltinVMSS(3, key.WorkerVMSSName(*cr)), nil).
		Times(1)

	mockAzureAPI.
		EXPECT().
		DeleteDeployment(gomock.Any(), key.ResourceGroupName(*cr), key.WorkersVmssDeploymentName).
		Times(1)

	mockAzureAPI.
		EXPECT().
		DeleteVMSS(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1)

	mockAzureAPI.
		EXPECT().
		ListVMSSNodes(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(workersAsVMSSNodes(*cr), nil).
		Times(1)

	err = r.EnsureCreated(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}

	// VERIFY: DrainerConfig CRs are gone.
	{
		opts := client.MatchingLabels{
			capiv1alpha3.ClusterLabelName: key.ClusterID(cr),
		}
		dcList := new(corev1alpha1.DrainerConfigList)
		err = ctrlClient.List(context.Background(), dcList, opts)
		if err != nil {
			t.Fatal(err)
		}

		if len(dcList.Items) > 0 {
			t.Fatalf("expected 0 drainer config crs to exist. got %d.", len(dcList.Items))
		}
	}

	// gomock verifies rest of the assertions on exit.
}

func TestLegacyWorkerDeploymentIsDeleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctrlClient := newFakeClient()
	mockAzureAPI := mock_azure.NewMockAPI(ctrl)
	mockTenantClusterFactory := mock_tenantcluster.NewMockFactory(ctrl)
	r := &Resource{
		ctrlClient:          ctrlClient,
		logger:              microloggertest.New(),
		tenantClientFactory: mockTenantClusterFactory,
		wrapAzureAPI: func(cf *azureclient.Factory, credentials *providerv1alpha1.CredentialSecret) azure.API {
			return mockAzureAPI
		},
	}

	ensureCRsExist(t, ctrlClient, []string{
		"cluster.yaml",
		"azureconfig.yaml",
		"azurecluster.yaml",
		"azuremachinepool.yaml",
		"machinepool.yaml",
		"namespace.yaml",
		"spark.yaml",
		"drainerconfigs.yaml",
	})

	o, err := loadCR("azureconfig.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cr := o.(*providerv1alpha1.AzureConfig)

	ensureNodePoolIsReady(t, ctrlClient, cr)
	setDrainerConfigsAsDrained(t, ctrlClient, cr)

	mockAzureAPI.
		EXPECT().
		ListNetworkSecurityGroups(gomock.Any(), key.ResourceGroupName(*cr)).
		Return(nil, nil).
		Times(1)

	mockAzureAPI.
		EXPECT().
		GetVMSS(gomock.Any(), key.ResourceGroupName(*cr), key.WorkerVMSSName(*cr)).
		Return(newBuiltinVMSS(3, key.WorkerVMSSName(*cr)), nil).
		Times(1)

	mockAzureAPI.
		EXPECT().
		DeleteDeployment(gomock.Any(), key.ResourceGroupName(*cr), key.WorkersVmssDeploymentName).
		Times(1)

	mockAzureAPI.
		EXPECT().
		DeleteVMSS(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1)

	mockAzureAPI.
		EXPECT().
		ListVMSSNodes(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(workersAsVMSSNodes(*cr), nil).
		Times(1)

	err = r.EnsureCreated(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}

	// gomock verifies rest of the assertions on exit.
}

func TestLegacyWorkerDeploymentDeletionDoesntErrorInNotFoundCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctrlClient := newFakeClient()
	mockAzureAPI := mock_azure.NewMockAPI(ctrl)
	mockTenantClusterFactory := mock_tenantcluster.NewMockFactory(ctrl)
	r := &Resource{
		ctrlClient:          ctrlClient,
		logger:              microloggertest.New(),
		tenantClientFactory: mockTenantClusterFactory,
		wrapAzureAPI: func(cf *azureclient.Factory, credentials *providerv1alpha1.CredentialSecret) azure.API {
			return mockAzureAPI
		},
	}

	ensureCRsExist(t, ctrlClient, []string{
		"cluster.yaml",
		"azureconfig.yaml",
		"azurecluster.yaml",
		"azuremachinepool.yaml",
		"machinepool.yaml",
		"namespace.yaml",
		"spark.yaml",
		"drainerconfigs.yaml",
	})

	o, err := loadCR("azureconfig.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cr := o.(*providerv1alpha1.AzureConfig)

	ensureNodePoolIsReady(t, ctrlClient, cr)
	setDrainerConfigsAsDrained(t, ctrlClient, cr)

	mockAzureAPI.
		EXPECT().
		ListNetworkSecurityGroups(gomock.Any(), key.ResourceGroupName(*cr)).
		Return(nil, nil).
		Times(1)

	mockAzureAPI.
		EXPECT().
		GetVMSS(gomock.Any(), key.ResourceGroupName(*cr), key.WorkerVMSSName(*cr)).
		Return(newBuiltinVMSS(3, key.WorkerVMSSName(*cr)), nil).
		Times(1)

	mockAzureAPI.
		EXPECT().
		DeleteVMSS(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1)

	mockAzureAPI.
		EXPECT().
		ListVMSSNodes(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(workersAsVMSSNodes(*cr), nil).
		Times(1)

	mockAzureAPI.
		EXPECT().
		DeleteDeployment(gomock.Any(), key.ResourceGroupName(*cr), key.WorkersVmssDeploymentName).
		Return(autorest.NewErrorWithResponse("deployment", "DELETE", &http.Response{StatusCode: 404}, "Deployment not found")).
		Times(1)

	err = r.EnsureCreated(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}

	// gomock verifies rest of the assertions on exit.
}

func TestFinishedMigration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := newFakeClient()
	m := mock_azure.NewMockAPI(ctrl)
	r := &Resource{
		ctrlClient:   client,
		logger:       microloggertest.New(),
		wrapAzureAPI: func(cf *azureclient.Factory, credentials *providerv1alpha1.CredentialSecret) azure.API { return m },
	}

	ensureCRsExist(t, client, []string{
		"cluster.yaml",
		"azureconfig.yaml",
		"azurecluster.yaml",
	})

	o, err := loadCR("azureconfig.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cr := o.(*providerv1alpha1.AzureConfig)

	m.
		EXPECT().
		ListNetworkSecurityGroups(gomock.Any(), key.ResourceGroupName(*cr)).
		Return(nil, nil).
		Times(0)

	m.
		EXPECT().
		GetVMSS(gomock.Any(), key.ResourceGroupName(*cr), key.WorkerVMSSName(*cr)).
		Return(nil, microerror.Mask(autorest.DetailedError{StatusCode: 404})).
		Times(1)

	m.
		EXPECT().
		DeleteVMSS(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	err = r.EnsureCreated(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}

	// gomock verifies assertions on exit.
}

func ensureCRsExist(t *testing.T, client client.Client, inputFiles []string) {
	for _, f := range inputFiles {
		o, err := loadCR(f)
		if err != nil {
			t.Fatalf("failed to load input file %s: %#v", f, err)
		}

		if o.GetObjectKind().GroupVersionKind().Kind == "DrainerConfigList" {
			lst := o.(*corev1alpha1.DrainerConfigList)
			for _, i := range lst.Items {
				err = client.Create(context.Background(), &i)
				if err != nil {
					t.Fatalf("failed to create object from input file %s: %#v", f, err)
				}
			}
			continue
		}

		if o.GetObjectKind().GroupVersionKind().Kind == "NamespaceList" {
			lst := o.(*corev1.NamespaceList)
			for _, i := range lst.Items {
				err = client.Create(context.Background(), &i)
				if err != nil {
					t.Fatalf("failed to create object from input file %s: %#v", f, err)
				}
			}
			continue
		}

		err = client.Create(context.Background(), o)
		if err != nil {
			t.Fatalf("failed to create object from input file %s: %#v", f, err)
		}
	}
}

func ensureNodePoolIsReady(t *testing.T, ctrlClient client.Client, cr *providerv1alpha1.AzureConfig) {
	t.Helper()

	var azureMachinePool expcapzv1alpha3.AzureMachinePool
	{
		o := client.ObjectKey{Namespace: key.OrganizationNamespace(cr), Name: cr.Name}
		err := ctrlClient.Get(context.Background(), o, &azureMachinePool)
		if err != nil {
			t.Fatal(err)
		}

		azureMachinePool.Status.Ready = true
		azureMachinePool.Status.Replicas = int32(key.WorkerCount(*cr))
		err = ctrlClient.Status().Update(context.Background(), &azureMachinePool)
		if err != nil {
			t.Fatal(err)
		}
	}

	var machinePool expcapiv1alpha3.MachinePool
	{
		o := client.ObjectKey{Namespace: key.OrganizationNamespace(cr), Name: cr.Name}
		err := ctrlClient.Get(context.Background(), o, &machinePool)
		if err != nil {
			t.Fatal(err)
		}

		machinePool.Status.BootstrapReady = true
		machinePool.Status.InfrastructureReady = azureMachinePool.Status.Ready
		machinePool.Status.Replicas = azureMachinePool.Status.Replicas
		machinePool.Status.ReadyReplicas = azureMachinePool.Status.Replicas
		err = ctrlClient.Status().Update(context.Background(), &machinePool)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func loadCR(fName string) (runtime.Object, error) {
	var err error
	var obj runtime.Object

	var bs []byte
	{
		bs, err = ioutil.ReadFile(filepath.Join("testdata", fName))
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// First parse kind.
	t := &metav1.TypeMeta{}
	err = yaml.Unmarshal(bs, t)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Then construct correct CR object.
	switch t.Kind {
	case "Cluster":
		obj = new(capiv1alpha3.Cluster)
	case "AzureConfig":
		obj = new(providerv1alpha1.AzureConfig)
	case "AzureCluster":
		obj = new(capzv1alpha3.AzureCluster)
	case "AzureMachine":
		obj = new(capzv1alpha3.AzureMachine)
	case "AzureMachinePool":
		obj = new(expcapzv1alpha3.AzureMachinePool)
	case "DrainerConfig":
		obj = new(corev1alpha1.DrainerConfig)
	case "DrainerConfigList":
		obj = new(corev1alpha1.DrainerConfigList)
	case "MachinePool":
		obj = new(expcapiv1alpha3.MachinePool)
	case "Namespace":
		obj = new(corev1.Namespace)
	case "NamespaceList":
		obj = new(corev1.NamespaceList)
	case "Spark":
		obj = new(corev1alpha1.Spark)
	default:
		return nil, microerror.Maskf(unknownKindError, "kind: %s", t.Kind)
	}

	// ...and unmarshal the whole object.
	err = yaml.Unmarshal(bs, obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return obj, nil
}

func newBuiltinVMSS(nodeCount int, name string) azure.VMSS {
	var vmss azure.VMSS
	{
		vmss = &compute.VirtualMachineScaleSet{
			Sku: &compute.Sku{
				Capacity: to.Int64P(int64(nodeCount)),
				Name:     &name,
			},
			Name: &name,
		}
	}
	return vmss
}

func newFakeClient() client.Client {
	scheme := runtime.NewScheme()

	err := capiv1alpha3.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	err = expcapiv1alpha3.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	err = capzv1alpha3.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	err = expcapzv1alpha3.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	err = corev1.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	err = corev1alpha1.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	err = providerv1alpha1.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	return fake.NewFakeClientWithScheme(scheme)
}

func setDrainerConfigsAsDrained(t *testing.T, ctrlClient client.Client, cr *providerv1alpha1.AzureConfig) {
	o := client.MatchingLabels{
		capiv1alpha3.ClusterLabelName: key.ClusterID(cr),
	}

	var dcList corev1alpha1.DrainerConfigList
	err := ctrlClient.List(context.Background(), &dcList, o)
	if err != nil {
		t.Fatal(err)
	}

	for _, dc := range dcList.Items {
		dc.Status.Conditions = append(dc.Status.Conditions, dc.Status.NewDrainedCondition())
		err = ctrlClient.Status().Update(context.Background(), &dc)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func setStatusCondition(ctrlClient client.Client, obj *providerv1alpha1.AzureConfig, resourceName, conditionType, status string) error {
	cr := &providerv1alpha1.AzureConfig{}
	err := ctrlClient.Get(context.Background(), client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	resourceStatus := providerv1alpha1.StatusClusterResource{
		Conditions: []providerv1alpha1.StatusClusterResourceCondition{
			{
				Status: status,
				Type:   conditionType,
			},
		},
		Name: resourceName,
	}

	var set bool
	for i, resource := range cr.Status.Cluster.Resources {
		if resource.Name != resourceName {
			continue
		}

		for _, c := range resource.Conditions {
			if c.Type == conditionType {
				continue
			}
			resourceStatus.Conditions = append(resourceStatus.Conditions, c)
		}

		cr.Status.Cluster.Resources[i] = resourceStatus
		set = true
	}

	if !set {
		cr.Status.Cluster.Resources = append(cr.Status.Cluster.Resources, resourceStatus)
	}

	err = ctrlClient.Update(context.Background(), cr)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func workersAsVMSSNodes(cr providerv1alpha1.AzureConfig) azure.VMSSNodes {
	var nodes azure.VMSSNodes
	for i := 0; i < key.WorkerCount(cr); i++ {
		nodeName := fmt.Sprintf("%s-node-%d", key.ClusterID(&cr), i)
		id := strconv.Itoa(i)
		n := compute.VirtualMachineScaleSetVM{
			InstanceID: &id,
			Name:       &nodeName,
		}
		nodes = append(nodes, n)
	}

	return nodes
}
