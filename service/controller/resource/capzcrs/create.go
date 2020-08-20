package capzcrs

import (
	"context"
	"fmt"
	"strconv"

	"github.com/giantswarm/apiextensions/pkg/annotation"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/to"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "mapping AzureConfig CR to CAPI & CAPZ CRs")

	var mappedCRs []runtime.Object
	{
		o, err := r.mapAzureConfigToAzureCluster(ctx, cr)
		if err != nil {
			return microerror.Mask(err)
		}
		mappedCRs = append(mappedCRs, o)
		azureCluster := o.(*capzv1alpha3.AzureCluster)

		infraRef := &corev1.ObjectReference{
			Kind:      "AzureCluster",
			Name:      cr.Name,
			Namespace: cr.Namespace,
		}
		o, err = r.mapAzureConfigToCluster(ctx, cr, infraRef)
		if err != nil {
			return microerror.Mask(err)
		}
		mappedCRs = append(mappedCRs, o)

		o, err = r.mapAzureConfigToAzureMachine(ctx, cr)
		if err != nil {
			return microerror.Mask(err)
		}
		mappedCRs = append(mappedCRs, o)

		if len(cr.Spec.Azure.Workers) > 0 {
			machinePoolName := "Legacy workers"
			o, err = r.mapAzureConfigToAzureMachinePool(ctx, cr, *azureCluster, machinePoolName)
			if err != nil {
				return microerror.Mask(err)
			}
			mappedCRs = append(mappedCRs, o)
			azureMachinePool := o.(*expcapzv1alpha3.AzureMachinePool)

			o, err = r.mapAzureConfigToMachinePool(ctx, cr, azureMachinePool, machinePoolName)
			if err != nil {
				return microerror.Mask(err)
			}
			mappedCRs = append(mappedCRs, o)
		}
	}

	err = r.updateCRs(ctx, mappedCRs)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "mapped AzureConfig CR to CAPI & CAPZ CRs")

	return nil
}

func (r *Resource) mapAzureConfigToCluster(ctx context.Context, cr providerv1alpha1.AzureConfig, infraRef *corev1.ObjectReference) (runtime.Object, error) {
	cluster := &capiv1alpha3.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: capiv1alpha3.GroupVersion.String(),
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				// XXX: azure-operator reconciles Cluster & MachinePool to set OwnerReferences (for now).
				label.AzureOperatorVersion:    key.OperatorVersion(&cr),
				label.ClusterOperatorVersion:  cr.Labels[label.ClusterOperatorVersion],
				label.Cluster:                 key.ClusterID(&cr),
				capiv1alpha3.ClusterLabelName: key.ClusterID(&cr),
				label.Organization:            key.OrganizationID(&cr),
				label.ReleaseVersion:          key.ReleaseVersion(&cr),
			},
		},
		Spec: capiv1alpha3.ClusterSpec{
			ClusterNetwork: &capiv1alpha3.ClusterNetwork{
				APIServerPort: toInt32P(int32(key.APISecurePort(cr))),
				Services: &capiv1alpha3.NetworkRanges{
					CIDRBlocks: []string{
						key.ClusterIPRange(cr),
					},
				},
				ServiceDomain: key.ClusterBaseDomain(cr),
			},
			ControlPlaneEndpoint: capiv1alpha3.APIEndpoint{
				Host: key.ClusterAPIEndpoint(cr),
				Port: 443,
			},
			InfrastructureRef: infraRef,
		},
	}
	return cluster, nil
}

func (r *Resource) mapAzureConfigToAzureCluster(ctx context.Context, cr providerv1alpha1.AzureConfig) (runtime.Object, error) {
	azureCluster := &capzv1alpha3.AzureCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: capzv1alpha3.GroupVersion.String(),
			Kind:       "AzureCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				label.AzureOperatorVersion:    key.OperatorVersion(&cr),
				label.Cluster:                 key.ClusterID(&cr),
				capiv1alpha3.ClusterLabelName: key.ClusterID(&cr),
				label.Organization:            key.OrganizationID(&cr),
				label.ReleaseVersion:          key.ReleaseVersion(&cr),
			},
			Annotations: map[string]string{
				annotation.ClusterDescription: cr.Annotations[annotation.ClusterDescription],
			},
		},
		Spec: capzv1alpha3.AzureClusterSpec{
			Location: r.location,
			ControlPlaneEndpoint: capiv1alpha3.APIEndpoint{
				Host: key.ClusterAPIEndpoint(cr),
				Port: 443,
			},
			NetworkSpec: capzv1alpha3.NetworkSpec{
				Vnet: capzv1alpha3.VnetSpec{
					CidrBlock:     key.VnetCIDR(cr),
					Name:          key.VnetName(cr),
					ResourceGroup: key.ResourceGroupName(cr),
				},
			},
		},
	}

	return azureCluster, nil
}

func (r *Resource) mapAzureConfigToAzureMachine(ctx context.Context, cr providerv1alpha1.AzureConfig) (runtime.Object, error) {
	if len(cr.Spec.Azure.Masters) == 0 {
		return nil, microerror.Maskf(invalidConfigError, "no master nodes defined")
	}
	vmSize := cr.Spec.Azure.Masters[0].VMSize
	azureMachine := &capzv1alpha3.AzureMachine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: capzv1alpha3.GroupVersion.String(),
			Kind:       "AzureMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-master-0", key.ClusterID(&cr)),
			Namespace: cr.Namespace,
			Labels: map[string]string{
				label.AzureOperatorVersion:                key.OperatorVersion(&cr),
				label.Cluster:                             key.ClusterID(&cr),
				capiv1alpha3.ClusterLabelName:             key.ClusterID(&cr),
				capiv1alpha3.MachineControlPlaneLabelName: "true",
				label.Organization:                        key.OrganizationID(&cr),
				label.ReleaseVersion:                      key.ReleaseVersion(&cr),
			},
		},
		Spec: capzv1alpha3.AzureMachineSpec{
			VMSize: vmSize,
			Image: &capzv1alpha3.Image{
				Marketplace: &capzv1alpha3.AzureMarketplaceImage{
					Publisher: "kinvolk",
					Offer:     "flatcar-container-linux-free",
					SKU:       "stable",
					Version:   "2345.3.1",
				},
			},
			OSDisk: capzv1alpha3.OSDisk{
				OSType:     "Linux",
				DiskSizeGB: int32(50),
				ManagedDisk: capzv1alpha3.ManagedDisk{
					StorageAccountType: "Premium_LRS",
				},
			},
			Location:     r.location,
			SSHPublicKey: key.AdminSSHKeyData(cr),
		},
	}

	if len(key.AvailabilityZones(cr, r.location)) > 0 {
		azureMachine.Spec.FailureDomain = to.StringP(strconv.Itoa(key.AvailabilityZones(cr, r.location)[0]))
	}

	return azureMachine, nil
}

func (r *Resource) mapAzureConfigToMachinePool(_ context.Context, azureConfig providerv1alpha1.AzureConfig, azureMachinePool *expcapzv1alpha3.AzureMachinePool, machinePoolName string) (runtime.Object, error) {
	nodeCount := int32(len(azureConfig.Spec.Azure.Workers))

	var infrastructureCRRef *corev1.ObjectReference
	{
		s := runtime.NewScheme()
		err := expcapzv1alpha3.AddToScheme(s)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		infrastructureCRRef, err = reference.GetReference(s, azureMachinePool)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	machinePool := &expcapiv1alpha3.MachinePool{
		TypeMeta: metav1.TypeMeta{
			APIVersion: expcapiv1alpha3.GroupVersion.String(),
			Kind:       "MachinePool",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      azureMachinePool.Name,
			Namespace: azureMachinePool.GetNamespace(),
			Labels: map[string]string{
				label.AzureOperatorVersion: key.OperatorVersion(&azureConfig),
				label.Cluster:              key.ClusterID(&azureConfig),
				// label.ClusterOperatorVersion: // TODO: do we need this?
				label.MachinePool:    azureMachinePool.Name,
				label.Organization:   key.OrganizationID(&azureConfig),
				label.ReleaseVersion: key.ReleaseVersion(&azureConfig),
			},
			Annotations: map[string]string{
				annotation.MachinePoolName: machinePoolName, // TODO: do we annotate both MachinePool and AzureMachinePool?
			},
		},
		Spec: expcapiv1alpha3.MachinePoolSpec{
			ClusterName: key.ClusterID(&azureConfig),
			Replicas:    &nodeCount,
			Template: capiv1alpha3.MachineTemplateSpec{
				Spec: capiv1alpha3.MachineSpec{
					ClusterName:       key.ClusterID(&azureConfig),
					InfrastructureRef: *infrastructureCRRef,
				},
			},
		},
	}

	availabilityZones := key.AvailabilityZones(azureConfig, r.location)
	if len(availabilityZones) > 0 {
		var azStrings []string
		for _, az := range availabilityZones {
			azStrings = append(azStrings, strconv.Itoa(az))
		}
		machinePool.Spec.FailureDomains = azStrings
	}

	return machinePool, nil
}

func (r *Resource) mapAzureConfigToAzureMachinePool(_ context.Context, azureConfig providerv1alpha1.AzureConfig, azureCluster capzv1alpha3.AzureCluster, machinePoolName string) (runtime.Object, error) {
	vmSize := azureConfig.Spec.Azure.Workers[0].VMSize

	// The first node pool (migrated legacy workers) get the same ID as the cluster
	machinePoolID := azureCluster.Name

	azureMachinePool := &expcapzv1alpha3.AzureMachinePool{
		TypeMeta: metav1.TypeMeta{
			APIVersion: expcapzv1alpha3.GroupVersion.String(),
			Kind:       "AzureMachinePool",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      machinePoolID,
			Namespace: azureCluster.GetNamespace(),
			Labels: map[string]string{
				label.AzureOperatorVersion: key.OperatorVersion(&azureConfig),
				label.Cluster:              key.ClusterID(&azureConfig),
				label.MachinePool:          machinePoolID,
				label.Organization:         key.OrganizationID(&azureConfig),
				label.ReleaseVersion:       key.ReleaseVersion(&azureConfig),
			},
			Annotations: map[string]string{
				annotation.MachinePoolName: machinePoolName,
			},
		},
		Spec: expcapzv1alpha3.AzureMachinePoolSpec{
			Location: azureCluster.Spec.Location,
			Template: expcapzv1alpha3.AzureMachineTemplate{
				VMSize: vmSize,
			},
		},
	}

	return azureMachinePool, nil
}

func (r *Resource) updateCRs(ctx context.Context, desiredCRs []runtime.Object) error {
	for _, desired := range desiredCRs {
		// Construct new instance by creating deep copy of desired object.
		readCR := desired.DeepCopyObject()

		// Acquire accessors for ObjectMeta and TypeMeta fields of CR.
		desiredMeta, err := meta.Accessor(desired)
		if err != nil {
			return microerror.Mask(err)
		}
		desiredType, err := meta.TypeAccessor(desired)
		if err != nil {
			return microerror.Mask(err)
		}

		nsName := types.NamespacedName{
			Name:      desiredMeta.GetName(),
			Namespace: desiredMeta.GetNamespace(),
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("reading present %s %s", desiredType.GetKind(), nsName.String()))

		err = r.ctrlClient.Get(ctx, nsName, readCR)
		if errors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("%s %s did not exist. creating", desiredType.GetKind(), nsName.String()))

			// It's ok. Let's create it.
			err = r.ctrlClient.Create(ctx, desired)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created %s %s", desiredType.GetKind(), nsName.String()))
			continue
		} else if err != nil {
			return microerror.Mask(err)
		}

		readMeta, err := meta.Accessor(readCR)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding if %s %s needs updating", desiredType.GetKind(), nsName.String()))

		var updateNeeded bool
		if !cmp.Equal(desiredMeta.GetLabels(), readMeta.GetLabels()) {
			labels := readMeta.GetLabels()
			for k, v := range desiredMeta.GetLabels() {
				labels[k] = v
			}
			// Set merged labels on desired object as that's the one we write
			// in update.
			desiredMeta.SetLabels(labels)
			updateNeeded = true
		}

		if !cmp.Equal(desiredMeta.GetAnnotations(), readMeta.GetAnnotations()) {
			annotations := readMeta.GetAnnotations()
			for k, v := range desiredMeta.GetAnnotations() {
				annotations[k] = v
			}
			// Set merged labels on desired object as that's the one we write
			// in update.
			desiredMeta.SetAnnotations(annotations)
			updateNeeded = true
		}

		if !cmp.Equal(desired, readCR, cmpopts.IgnoreTypes(&metav1.ObjectMeta{}), cmpopts.IgnoreTypes(&metav1.TypeMeta{})) {
			updateNeeded = true
		}

		if updateNeeded {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found that %s %s needs updating", desiredType.GetKind(), nsName.String()))

			// Copy read CR's resource version to update object for optimistic
			// locking.
			desiredMeta.SetResourceVersion(readMeta.GetResourceVersion())

			err = r.ctrlClient.Update(ctx, desired)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated %s %s", desiredType.GetKind(), nsName.String()))
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no need to %s %s", desiredType.GetKind(), nsName.String()))
		}
	}

	return nil
}

func toInt32P(v int32) *int32 {
	return &v
}
