package workermigration

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Azure/go-autorest/autorest/to"
	apiextannotation "github.com/giantswarm/apiextensions/v6/pkg/annotation"
	corev1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/apiextensions/v6/pkg/label"
	apiextlabel "github.com/giantswarm/apiextensions/v6/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v8/service/controller/azureconfig/handler/workermigration/internal/azure"
	"github.com/giantswarm/azure-operator/v8/service/controller/key"
)

const (
	// Status condition types for master upgrade state checking.
	DeploymentCompleted = "DeploymentCompleted"
	Stage               = "Stage"
)

// EnsureCreated ensures that legacy workers are migrated to node pool.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// Wait for master upgrade to pass first in order to avoid conflicts with
	// Azure deployments ordering.
	masterUpgrading, err := r.isMasterUpgrading(cr)
	if err != nil {
		return microerror.Mask(err)
	}

	if masterUpgrading {
		r.logger.Debugf(ctx, "master is upgrading")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	credentialSecret := &providerv1alpha1.CredentialSecret{
		Name:      key.CredentialName(cr),
		Namespace: key.CredentialNamespace(cr),
	}
	azureAPI := r.wrapAzureAPI(r.clientFactory, credentialSecret)

	r.logger.Debugf(ctx, "ensuring that legacy workers are migrated to node pool")

	r.logger.Debugf(ctx, "ensure security group rules are updated")
	err = r.ensureSecurityGroupRulesUpdated(ctx, cr, azureAPI)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "ensured security group rules are updated")

	r.logger.Debugf(ctx, "finding legacy workers VMSS")
	var legacyVMSS azure.VMSS
	{
		legacyVMSS, err = azureAPI.GetVMSS(ctx, key.ResourceGroupName(cr), key.WorkerVMSSName(cr))
		if azure.IsNotFound(err) {
			r.logger.Debugf(ctx, "did not find legacy workers VMSS")
			r.logger.Debugf(ctx, "canceling resource")

			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		if legacyVMSS.Sku == nil {
			return microerror.Maskf(executionFailedError, "legacy VMSS Sku is nil")
		}
		if legacyVMSS.Sku.Name == nil {
			return microerror.Maskf(executionFailedError, "legacy VMSS Sku.Name is nil")
		}
		if legacyVMSS.Sku.Capacity == nil {
			return microerror.Maskf(executionFailedError, "legacy VMSS Sku.Capacity is nil")
		}
	}

	r.logger.Debugf(ctx, "found legacy workers VMSS")

	r.logger.Debugf(ctx, "ensuring AzureMachinePool CR exists for legacy workers VMSS")
	azureMachinePool, err := r.ensureAzureMachinePoolExists(ctx, cr, *legacyVMSS.Sku.Name)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "ensured AzureMachinePool CR exists for legacy workers VMSS")

	r.logger.Debugf(ctx, "ensuring Spark CR exists for legacy workers VMSS")
	spark, err := r.ensureSparkExists(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "ensured Spark CR exists for legacy workers VMSS")

	r.logger.Debugf(ctx, "ensuring MachinePool CR exists for legacy workers VMSS")
	machinePool, err := r.ensureMachinePoolExists(ctx, spark, cr, azureMachinePool, int(*legacyVMSS.Sku.Capacity))
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "ensured MachinePool CR exists for legacy workers VMSS")

	r.logger.Debugf(ctx, "finding if node pool workers are ready")
	if !machinePool.Status.InfrastructureReady || (machinePool.Status.Replicas != machinePool.Status.ReadyReplicas) {
		r.logger.Debugf(ctx, "node pool workers are not ready yet")
		r.logger.Debugf(ctx, "canceling resource")

		return nil
	}
	r.logger.Debugf(ctx, "found that node pool workers are ready")

	r.logger.Debugf(ctx, "ensuring that legacy workers have drainerconfig cr")
	err = r.ensureDrainerConfigsExists(ctx, azureAPI, cr)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "ensured that legacy workers have drainerconfig cr")

	r.logger.Debugf(ctx, "ensuring that legacy workers are drained")
	allNodesDrained, err := r.allDrainerConfigsWithDrainedState(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	if !allNodesDrained {
		r.logger.Debugf(ctx, "some old worker nodes are still draining")
		r.logger.Debugf(ctx, "deleting timed out drainerconfigs")

		// In case some draining operations timed out, delete CRs so that they
		// are recreated on next reconciliation.
		err = r.deleteTimedOutDrainerConfigs(ctx, cr)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "deleted timed out drainerconfigs")
		r.logger.Debugf(ctx, "canceling resource")

		return nil
	}
	r.logger.Debugf(ctx, "ensured that legacy workers are drained")

	r.logger.Debugf(ctx, "deleting drainerconfigs")
	err = r.deleteDrainerConfigs(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "deleted drainerconfigs")

	r.logger.Debugf(ctx, "deleting legacy workers' deployment")
	err = azureAPI.DeleteDeployment(ctx, key.ResourceGroupName(cr), key.WorkersVmssDeploymentName)
	if azure.IsNotFound(err) {
		// It's ok. We would have deleted it anyway ¯\_(ツ)_/¯
	} else if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "deleted legacy workers' deployment")

	r.logger.Debugf(ctx, "deleting legacy workers' VMSS")
	err = azureAPI.DeleteVMSS(ctx, key.ResourceGroupName(cr), *legacyVMSS.Name)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "deleted legacy workers' VMSS")

	return nil
}

func (r *Resource) allDrainerConfigsWithDrainedState(ctx context.Context, cr providerv1alpha1.AzureConfig) (bool, error) {
	o := client.MatchingLabels{
		capi.ClusterLabelName: key.ClusterID(&cr),
	}

	var dcList corev1alpha1.DrainerConfigList
	err := r.ctrlClient.List(ctx, &dcList, o)
	if err != nil {
		return false, microerror.Mask(err)
	}

	for _, dc := range dcList.Items {
		if !dc.Status.HasDrainedCondition() {
			return false, nil
		}
	}

	return true, nil
}

func (r *Resource) deleteTimedOutDrainerConfigs(ctx context.Context, cr providerv1alpha1.AzureConfig) error {
	o := client.MatchingLabels{
		capi.ClusterLabelName: key.ClusterID(&cr),
	}

	var dcList corev1alpha1.DrainerConfigList
	err := r.ctrlClient.List(ctx, &dcList, o)
	if err != nil {
		return microerror.Mask(err)
	}

	for i, dc := range dcList.Items {
		if dc.Status.HasTimeoutCondition() {
			err = r.ctrlClient.Delete(ctx, &dcList.Items[i])
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	return nil
}

func (r *Resource) deleteDrainerConfigs(ctx context.Context, cr providerv1alpha1.AzureConfig) error {
	err := r.ctrlClient.DeleteAllOf(ctx, &corev1alpha1.DrainerConfig{}, client.MatchingLabels{capi.ClusterLabelName: key.ClusterID(&cr)}, client.InNamespace(key.ClusterID(&cr)))
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) ensureAzureMachinePoolExists(ctx context.Context, cr providerv1alpha1.AzureConfig, vmSize string) (capzexp.AzureMachinePool, error) {
	var azureMachinePool capzexp.AzureMachinePool

	nsName := types.NamespacedName{
		Name:      cr.GetName(),
		Namespace: key.OrganizationNamespace(&cr),
	}
	err := r.ctrlClient.Get(ctx, nsName, &azureMachinePool)
	if err == nil {
		// NOTE: Success return when AzureMachinePool CR already exists.
		return azureMachinePool, nil
	} else if errors.IsNotFound(err) {
		// This is ok. CR gets created in a bit.
	} else if err != nil {
		return capzexp.AzureMachinePool{}, microerror.Mask(err)
	}

	var dockerDiskSizeGB int
	var kubeletDiskSizeGB int
	{
		if key.WorkerCount(cr) > 0 {
			dockerDiskSizeGB = cr.Spec.Azure.Workers[0].DockerVolumeSizeGB
			kubeletDiskSizeGB = cr.Spec.Azure.Workers[0].KubeletVolumeSizeGB
		}

		if dockerDiskSizeGB <= 0 {
			dockerDiskSizeGB = 100
		}

		if kubeletDiskSizeGB <= 0 {
			kubeletDiskSizeGB = 100
		}
	}

	// CR didn't exist so it's created here.
	azureMachinePool = capzexp.AzureMachinePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: key.OrganizationNamespace(&cr),
			Labels: map[string]string{
				label.AzureOperatorVersion: key.OperatorVersion(&cr),
				label.Cluster:              key.ClusterID(&cr),
				capi.ClusterLabelName:      key.ClusterName(&cr),
				label.MachinePool:          key.ClusterID(&cr),
				label.Organization:         key.OrganizationID(&cr),
				label.ReleaseVersion:       key.ReleaseVersion(&cr),
			},
		},
		Spec: capzexp.AzureMachinePoolSpec{
			Location: r.location,
			Template: capzexp.AzureMachinePoolMachineTemplate{
				DataDisks: []capz.DataDisk{
					{
						NameSuffix: "docker",
						DiskSizeGB: int32(dockerDiskSizeGB),
						Lun:        to.Int32Ptr(21),
					},
					{
						NameSuffix: "kubelet",
						DiskSizeGB: int32(kubeletDiskSizeGB),
						Lun:        to.Int32Ptr(22),
					},
				},
				VMSize: vmSize,
			},
		},
	}

	err = r.ctrlClient.Create(ctx, &azureMachinePool)
	if err != nil {
		return capzexp.AzureMachinePool{}, microerror.Mask(err)
	}

	return azureMachinePool, nil
}

func (r *Resource) ensureDrainerConfigsExists(ctx context.Context, azureAPI azure.API, cr providerv1alpha1.AzureConfig) error {
	nodes, err := azureAPI.ListVMSSNodes(ctx, key.ResourceGroupName(cr), key.WorkerVMSSName(cr))
	if err != nil {
		return microerror.Mask(err)
	}

	for _, n := range nodes {
		name := nodeName(key.ClusterID(&cr), *n.InstanceID)

		r.logger.Debugf(ctx, "creating drainer config for tenant cluster node %q", name)

		c := &corev1alpha1.DrainerConfig{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					label.Cluster:         key.ClusterID(&cr),
					capi.ClusterLabelName: key.ClusterID(&cr),
				},
				Name:      name,
				Namespace: key.ClusterID(&cr),
			},
			Spec: corev1alpha1.DrainerConfigSpec{
				Guest: corev1alpha1.DrainerConfigSpecGuest{
					Cluster: corev1alpha1.DrainerConfigSpecGuestCluster{
						API: corev1alpha1.DrainerConfigSpecGuestClusterAPI{
							Endpoint: key.ClusterAPIEndpoint(cr),
						},
						ID: key.ClusterID(&cr),
					},
					Node: corev1alpha1.DrainerConfigSpecGuestNode{
						Name: name,
					},
				},
				VersionBundle: corev1alpha1.DrainerConfigSpecVersionBundle{
					Version: "0.2.0",
				},
			},
		}

		err := r.ctrlClient.Create(ctx, c)
		if errors.IsAlreadyExists(err) {
			r.logger.Debugf(ctx, "did not create drainer config for tenant cluster node %q", name)
			r.logger.Debugf(ctx, "drainer config for tenant cluster node does already exist")
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.Debugf(ctx, "created drainer config for tenant cluster node %q", name)
		}
	}

	return nil
}

func (r *Resource) ensureMachinePoolExists(ctx context.Context, spark corev1alpha1.Spark, cr providerv1alpha1.AzureConfig, azureMachinePool capzexp.AzureMachinePool, replicas int) (capiexp.MachinePool, error) {
	var machinePool capiexp.MachinePool

	nsName := types.NamespacedName{
		Name:      cr.GetName(),
		Namespace: key.OrganizationNamespace(&cr),
	}
	err := r.ctrlClient.Get(ctx, nsName, &machinePool)
	if err == nil {
		// NOTE: Success return when MachinePool CR already exists.
		return machinePool, nil
	} else if errors.IsNotFound(err) {
		// This is ok. CR gets created in a bit.
	} else if err != nil {
		return capiexp.MachinePool{}, microerror.Mask(err)
	}

	s := runtime.NewScheme()
	{
		err = corev1alpha1.AddToScheme(s)
		if err != nil {
			panic(fmt.Sprintf("corev1alpha1.AddToScheme: %+v", err))
		}

		err = capzexp.AddToScheme(s)
		if err != nil {
			panic(fmt.Sprintf("capzexp.AddToScheme: %+v", err))
		}
	}

	var bootstrapCRRef *corev1.ObjectReference
	{
		bootstrapCRRef, err = reference.GetReference(s, &spark)
		if err != nil {
			panic(fmt.Sprintf("cannot create reference to bootstrap CR: %q", err))
		}
	}

	var infrastructureCRRef *corev1.ObjectReference
	{
		infrastructureCRRef, err = reference.GetReference(s, &azureMachinePool)
		if err != nil {
			panic(fmt.Sprintf("cannot create reference to infrastructure CR: %q", err))
		}
	}

	mp := capiexp.MachinePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: key.OrganizationNamespace(&cr),
			Labels: map[string]string{
				apiextlabel.AzureOperatorVersion: key.OperatorVersion(&cr),
				apiextlabel.Cluster:              key.ClusterID(&cr),
				capi.ClusterLabelName:            key.ClusterID(&cr),
				apiextlabel.MachinePool:          key.ClusterID(&cr),
				apiextlabel.Organization:         key.OrganizationID(&cr),
				apiextlabel.ReleaseVersion:       key.ReleaseVersion(&cr),
			},
			Annotations: map[string]string{
				apiextannotation.MachinePoolName: "migrated legacy workers",
				apiextannotation.NodePoolMinSize: fmt.Sprintf("%d", replicas),
				apiextannotation.NodePoolMaxSize: fmt.Sprintf("%d", replicas),
			},
		},
		Spec: capiexp.MachinePoolSpec{
			ClusterName:    key.ClusterID(&cr),
			Replicas:       to.Int32Ptr(int32(replicas)),
			FailureDomains: intSliceToStringSlice(key.AvailabilityZones(cr, r.location)),
			Template: capi.MachineTemplateSpec{
				Spec: capi.MachineSpec{
					Bootstrap: capi.Bootstrap{
						ConfigRef: bootstrapCRRef,
					},
					ClusterName:       key.ClusterID(&cr),
					InfrastructureRef: *infrastructureCRRef,
				},
			},
		},
	}

	err = r.ctrlClient.Create(ctx, &mp)
	if err != nil {
		return capiexp.MachinePool{}, microerror.Mask(err)
	}

	return mp, nil
}

func (r *Resource) ensureSparkExists(ctx context.Context, cr providerv1alpha1.AzureConfig) (corev1alpha1.Spark, error) {
	var spark corev1alpha1.Spark

	nsName := types.NamespacedName{
		Name:      cr.GetName(),
		Namespace: key.OrganizationNamespace(&cr),
	}
	err := r.ctrlClient.Get(ctx, nsName, &spark)
	if err == nil {
		// NOTE: Success return when Spark CR already exists.
		return spark, nil
	} else if errors.IsNotFound(err) {
		// This is ok. CR gets created in a bit.
	} else if err != nil {
		return corev1alpha1.Spark{}, microerror.Mask(err)
	}

	spark = corev1alpha1.Spark{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: key.OrganizationNamespace(&cr),
			Labels: map[string]string{
				apiextlabel.Cluster:        key.ClusterID(&cr),
				capi.ClusterLabelName:      key.ClusterName(&cr),
				apiextlabel.ReleaseVersion: key.ReleaseVersion(&cr),
			},
		},
		Spec: corev1alpha1.SparkSpec{},
	}

	err = r.ctrlClient.Create(ctx, &spark)
	if err != nil {
		return corev1alpha1.Spark{}, microerror.Mask(err)
	}

	return spark, nil
}

func intSliceToStringSlice(xs []int) []string {
	ys := make([]string, 0, len(xs))
	for _, x := range xs {
		ys = append(ys, strconv.Itoa(x))
	}
	return ys
}

func (r *Resource) isMasterUpgrading(obj providerv1alpha1.AzureConfig) (bool, error) {
	cr := &providerv1alpha1.AzureConfig{}
	err := r.ctrlClient.Get(context.Background(), client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, cr)
	if err != nil {
		return false, microerror.Mask(err)
	}

	var status string
	{
		for _, r := range cr.Status.Cluster.Resources {
			if r.Name != "masters" {
				continue
			}

			for _, c := range r.Conditions {
				if c.Type == Stage {
					status = c.Status
				}
			}
		}
	}

	return (status != DeploymentCompleted), nil
}

func nodeName(clusterID, instanceID string) string {
	i, err := strconv.ParseUint(instanceID, 10, 64)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s-worker-%s-%06s", clusterID, clusterID, strconv.FormatUint(i, 36))
}
