package controller

import (
	"context"
	"fmt"

	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v2/pkg/controller"
	"github.com/giantswarm/operatorkit/v2/pkg/resource"
	"github.com/giantswarm/operatorkit/v2/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v2/pkg/resource/wrapper/retryresource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	capiutil "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

type MachinePoolConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
	SentryDSN string
}

func NewMachinePool(config MachinePoolConfig) (*controller.Controller, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}

	var err error

	var resources []resource.Interface
	{
		resources, err = NewMachinePoolResourceSet(config)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorkitController *controller.Controller
	{
		c := controller.Config{
			InitCtx: func(ctx context.Context, obj interface{}) (context.Context, error) {
				return ctx, nil
			},
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			// Name is used to compute finalizer names. This results in something
			// like operatorkit.giantswarm.io/azure-operator-machine-pool-controller.
			Name: project.Name() + "-machine-pool-controller",
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(expcapiv1alpha3.MachinePool)
			},
			Resources: resources,
			Selector: labels.SelectorFromSet(map[string]string{
				label.OperatorVersion: project.Version(),
			}),
			SentryDSN: config.SentryDSN,
		}

		operatorkitController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return operatorkitController, nil
}

func NewMachinePoolResourceSet(config MachinePoolConfig) ([]resource.Interface, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var err error

	var ownerReferencesResource resource.Interface
	{
		c := MachinePoolOwnerReferencesConfig{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
			Scheme:     config.K8sClient.Scheme(),
		}

		ownerReferencesResource, err = NewMachinePoolOwnerReferences(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		ownerReferencesResource,
	}

	{
		c := retryresource.WrapConfig{
			Logger: config.Logger,
		}

		resources, err = retryresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	{
		c := metricsresource.WrapConfig{}
		resources, err = metricsresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resources, nil
}

type MachinePoolOwnerReferencesConfig struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
	Scheme     *runtime.Scheme
}

type MachinePoolOwnerReferencesResource struct {
	ctrlClient client.Client
	logger     micrologger.Logger
	scheme     *runtime.Scheme
}

func NewMachinePoolOwnerReferences(config MachinePoolOwnerReferencesConfig) (*MachinePoolOwnerReferencesResource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Scheme == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Scheme must not be empty", config)
	}

	r := &MachinePoolOwnerReferencesResource{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,
		scheme:     config.Scheme,
	}

	return r, nil
}

// EnsureCreated ensures the MachinePool is owned by the Cluster it belongs to, and the AzureMachinePool is owned by the MachinePool.
func (r *MachinePoolOwnerReferencesResource) EnsureCreated(ctx context.Context, obj interface{}) error {
	machinePool, err := key.ToMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "message", fmt.Sprintf("Ensuring %s label and 'ownerReference' fields on MachinePool '%s/%s'", capiv1alpha3.ClusterLabelName, machinePool.Namespace, machinePool.Name))
	r.logger.LogCtx(ctx, "message", fmt.Sprintf("Ensuring %s label and 'ownerReference' fields on AzureMachinePool '%s/%s'", capiv1alpha3.ClusterLabelName, machinePool.Namespace, machinePool.Spec.Template.Spec.InfrastructureRef.Name))

	azureMachinePool := expcapzv1alpha3.AzureMachinePool{}
	err = r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: machinePool.Namespace, Name: machinePool.Spec.Template.Spec.InfrastructureRef.Name}, &azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	if machinePool.Labels == nil {
		machinePool.Labels = make(map[string]string)
	}
	machinePool.Labels[capiv1alpha3.ClusterLabelName] = machinePool.Spec.ClusterName

	if azureMachinePool.Labels == nil {
		azureMachinePool.Labels = make(map[string]string)
	}
	azureMachinePool.Labels[capiv1alpha3.ClusterLabelName] = machinePool.Spec.ClusterName

	cluster, err := capiutil.GetClusterByName(ctx, r.ctrlClient, machinePool.Namespace, machinePool.Spec.ClusterName)
	if err != nil {
		return microerror.Mask(err)
	}

	// Set Cluster as owner of MachinePool
	machinePool.OwnerReferences = capiutil.EnsureOwnerRef(machinePool.OwnerReferences, metav1.OwnerReference{
		APIVersion: capiv1alpha3.GroupVersion.String(),
		Kind:       "Cluster",
		Name:       cluster.Name,
		UID:        cluster.UID,
	})

	err = r.ctrlClient.Update(ctx, &machinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "message", fmt.Sprintf("Ensured %s label and 'ownerReference' fields on MachinePool '%s/%s'", capiv1alpha3.ClusterLabelName, machinePool.Namespace, machinePool.Name))

	// Set MachinePool as owner of AzureMachinePool
	err = controllerutil.SetControllerReference(&machinePool, &azureMachinePool, r.scheme)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ctrlClient.Update(ctx, &azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "message", fmt.Sprintf("Ensuring %s label and 'ownerReference' fields on AzureMachinePool '%s/%s'", capiv1alpha3.ClusterLabelName, machinePool.Namespace, machinePool.Spec.Template.Spec.InfrastructureRef.Name))

	return nil
}

// EnsureDeleted is a noop.
func (r *MachinePoolOwnerReferencesResource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}

// Name returns the resource name.
func (r *MachinePoolOwnerReferencesResource) Name() string {
	return "MachinePoolOwnerReferences"
}
