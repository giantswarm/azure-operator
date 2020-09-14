package workermigration

import (
	"context"
	"fmt"
	"strconv"

	apiextannotation "github.com/giantswarm/apiextensions/pkg/annotation"
	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/label"
	apiextlabel "github.com/giantswarm/apiextensions/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/workermigration/internal/azure"
)

// EnsureCreated ensures that built-in workers are migrated to node pool.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring that built-in workers are migrated to node pool")

	var builtinVMSS azure.VMSS
	{
		builtinVMSS, err = r.azureapi.GetVMSS(ctx, key.ResourceGroupName(cr), key.WorkerVMSSName(cr))
		if azure.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "built-in workers don't exist anymore")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		if builtinVMSS.Sku == nil {
			return microerror.Maskf(executionFailedError, "built-in VMSS Sku is nil")
		}
		if builtinVMSS.Sku.Name == nil {
			return microerror.Maskf(executionFailedError, "built-in VMSS Sku.Name is nil")
		}
		if builtinVMSS.Sku.Capacity == nil {
			return microerror.Maskf(executionFailedError, "built-in VMSS Sku.Capacity is nil")
		}
	}

	var azureMachinePool expcapzv1alpha3.AzureMachinePool
	{
		nsName := types.NamespacedName{
			Name:      cr.GetName(),
			Namespace: cr.GetNamespace(),
		}
		err = r.ctrlClient.Get(ctx, nsName, &azureMachinePool)
		if errors.IsNotFound(err) {
			azureMachinePool, err = r.createAzureMachinePool(ctx, cr, *builtinVMSS.Sku.Name)
			if err != nil {
				return microerror.Mask(err)
			}
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	var machinePool expcapiv1alpha3.MachinePool
	{
		nsName := types.NamespacedName{
			Name:      cr.GetName(),
			Namespace: cr.GetNamespace(),
		}
		err = r.ctrlClient.Get(ctx, nsName, &machinePool)
		if errors.IsNotFound(err) {
			machinePool, err = r.createMachinePool(ctx, cr, azureMachinePool, int(*builtinVMSS.Sku.Capacity))
			if err != nil {
				return microerror.Mask(err)
			}
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	var sparkCR corev1alpha1.Spark
	{
		nsName := types.NamespacedName{
			Name:      cr.GetName(),
			Namespace: cr.GetNamespace(),
		}
		err = r.ctrlClient.Get(ctx, nsName, &sparkCR)
		if errors.IsNotFound(err) {
			sparkCR, err = r.createSpark(ctx, cr)
			if err != nil {
				return microerror.Mask(err)
			}
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		if !machinePool.Status.InfrastructureReady || (machinePool.Status.Replicas != machinePool.Status.ReadyReplicas) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "node pool workers are not ready yet")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

			return nil
		}

		// TODO: Drain old workers.
	}

	{
		err = r.azureapi.DeleteVMSS(ctx, key.ResourceGroupName(cr), *builtinVMSS.Name)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "built-in workers VMSS deleted")
	}

	return nil
}

func (r *Resource) createAzureMachinePool(ctx context.Context, cr providerv1alpha1.AzureConfig, vmSize string) (expcapzv1alpha3.AzureMachinePool, error) {
	var err error
	mp := expcapzv1alpha3.AzureMachinePool{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureMachinePool",
			APIVersion: "exp.infrastructure.cluster.x-k8s.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace, // TODO: Adjust for CR namespacing.
			Labels: map[string]string{
				label.AzureOperatorVersion:    key.OperatorVersion(&cr),
				label.Cluster:                 key.ClusterID(&cr),
				capiv1alpha3.ClusterLabelName: key.ClusterName(&cr),
				label.MachinePool:             key.ClusterID(&cr),
				label.Organization:            key.OrganizationID(&cr),
				label.ReleaseVersion:          key.ReleaseVersion(&cr),
			},
		},
		Spec: expcapzv1alpha3.AzureMachinePoolSpec{
			Location: r.location,
			Template: expcapzv1alpha3.AzureMachineTemplate{
				SSHPublicKey: key.AdminSSHKeyData(cr),
				VMSize:       vmSize,
			},
		},
	}

	err = r.ctrlClient.Create(ctx, &mp)
	if err != nil {
		return expcapzv1alpha3.AzureMachinePool{}, microerror.Mask(err)
	}

	return mp, nil
}

func (r *Resource) createMachinePool(ctx context.Context, cr providerv1alpha1.AzureConfig, azureMachinePool expcapzv1alpha3.AzureMachinePool, replicas int) (expcapiv1alpha3.MachinePool, error) {
	var err error
	var infrastructureCRRef *corev1.ObjectReference
	{
		s := runtime.NewScheme()
		err := expcapzv1alpha3.AddToScheme(s)
		if err != nil {
			panic(fmt.Sprintf("expcapzv1alpha3.AddToScheme: %+v", err))
		}

		infrastructureCRRef, err = reference.GetReference(s, &azureMachinePool)
		if err != nil {
			panic(fmt.Sprintf("cannot create reference to infrastructure CR: %q", err))
		}
	}

	mp := expcapiv1alpha3.MachinePool{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MachinePool",
			APIVersion: "exp.infrastructure.cluster.x-k8s.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace, // TODO: Adjust for CR namespacing.
			Labels: map[string]string{
				apiextlabel.AzureOperatorVersion: key.OperatorVersion(&cr),
				apiextlabel.Cluster:              key.ClusterID(&cr),
				capiv1alpha3.ClusterLabelName:    key.ClusterName(&cr),
				apiextlabel.MachinePool:          key.ClusterID(&cr),
				apiextlabel.Organization:         key.OrganizationID(&cr),
				apiextlabel.ReleaseVersion:       key.ReleaseVersion(&cr),
			},
			Annotations: map[string]string{
				apiextannotation.MachinePoolName: "migrated built-in workers",
			},
		},
		Spec: expcapiv1alpha3.MachinePoolSpec{
			ClusterName:    key.ClusterName(&cr),
			Replicas:       toInt32P(int32(replicas)),
			FailureDomains: intSliceToStringSlice(key.AvailabilityZones(cr, r.location)),
			Template: capiv1alpha3.MachineTemplateSpec{
				Spec: capiv1alpha3.MachineSpec{
					ClusterName:       key.ClusterName(&cr),
					InfrastructureRef: *infrastructureCRRef,
				},
			},
		},
	}

	err = r.ctrlClient.Create(ctx, &mp)
	if err != nil {
		return expcapiv1alpha3.MachinePool{}, microerror.Mask(err)
	}

	return mp, nil
}

func (r *Resource) createSpark(ctx context.Context, cr providerv1alpha1.AzureConfig) (corev1alpha1.Spark, error) {
	spark := corev1alpha1.Spark{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Spark",
			APIVersion: "core.giantswarm.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace, // TODO: Adjust for CR namespacing.
			Labels: map[string]string{
				apiextlabel.Cluster:           key.ClusterID(&cr),
				capiv1alpha3.ClusterLabelName: key.ClusterName(&cr),
				apiextlabel.ReleaseVersion:    key.ReleaseVersion(&cr),
			},
		},
		Spec: corev1alpha1.SparkSpec{},
	}

	err := r.ctrlClient.Create(ctx, &spark)
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

func toInt32P(i int32) *int32 {
	return &i
}
