package capzcrs

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/to"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	azopannotation "github.com/giantswarm/azure-operator/v5/pkg/annotation"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

type crmapping struct {
	obj runtime.Object

	needUpdateFunc func(orig, desired runtime.Object) (bool, error)
	mergeFunc      func(orig, desired runtime.Object) (runtime.Object, error)
}

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "mapping AzureConfig CR to CAPI & CAPZ CRs")

	{
		objKey := client.ObjectKey{
			Name:      cr.Name,
			Namespace: key.OrganizationNamespace(&cr),
		}
		cluster := new(capiv1alpha3.Cluster)
		err = r.ctrlClient.Get(ctx, objKey, cluster)
		if errors.IsNotFound(err) {
			// all good
		} else if err != nil {
			return microerror.Mask(err)
		} else if !cluster.GetDeletionTimestamp().IsZero() {
			r.logger.LogCtx(ctx, "level", "debug", "message", "Cluster is being deleted, skipping mapping AzureConfig CR to CAPI & CAPZ CRs")
			return nil
		}
	}

	var mappedCRs []crmapping
	{
		o, err := r.mapAzureConfigToAzureCluster(ctx, cr)
		if err != nil {
			return microerror.Mask(err)
		}
		mappedCRs = append(mappedCRs, crmapping{
			obj:            o,
			needUpdateFunc: detectAzureClusterUpdate,
			mergeFunc:      mergeAzureCluster,
		})

		infraRef := &corev1.ObjectReference{
			Kind:      "AzureCluster",
			Name:      cr.Name,
			Namespace: key.OrganizationNamespace(&cr),
		}
		o, err = r.mapAzureConfigToCluster(ctx, cr, infraRef)
		if err != nil {
			return microerror.Mask(err)
		}
		mappedCRs = append(mappedCRs, crmapping{
			obj:            o,
			needUpdateFunc: nil,
			mergeFunc:      nil,
		})

		o, err = r.mapAzureConfigToAzureMachine(ctx, cr)
		if err != nil {
			return microerror.Mask(err)
		}
		mappedCRs = append(mappedCRs, crmapping{
			obj:            o,
			needUpdateFunc: nil,
			mergeFunc:      nil,
		})
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
			Namespace: key.OrganizationNamespace(&cr),
			Annotations: map[string]string{
				annotation.ClusterDescription:       cr.Annotations[annotation.ClusterDescription],
				azopannotation.UpgradingToNodePools: "True",
			},
			Labels: map[string]string{
				// XXX: azure-operator reconciles Cluster & MachinePool to set OwnerReferences (for now).
				label.AzureOperatorVersion:    key.OperatorVersion(&cr),
				label.ClusterOperatorVersion:  cr.Labels[label.ClusterOperatorVersion],
				label.Cluster:                 cr.Name,
				capiv1alpha3.ClusterLabelName: cr.Name,
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
				ServiceDomain: cr.Spec.Cluster.Kubernetes.Domain,
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
			Namespace: key.OrganizationNamespace(&cr),
			Labels: map[string]string{
				label.AzureOperatorVersion:    key.OperatorVersion(&cr),
				label.Cluster:                 cr.Name,
				capiv1alpha3.ClusterLabelName: cr.Name,
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
			ResourceGroup: key.ClusterID(&cr),
			NetworkSpec: capzv1alpha3.NetworkSpec{
				Vnet: capzv1alpha3.VnetSpec{
					CIDRBlocks:    []string{key.VnetCIDR(cr)},
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
			Name:      key.AzureMachineName(&cr),
			Namespace: key.OrganizationNamespace(&cr),
			Labels: map[string]string{
				label.AzureOperatorVersion:                key.OperatorVersion(&cr),
				label.Cluster:                             cr.Name,
				capiv1alpha3.ClusterLabelName:             cr.Name,
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
			Location: r.location,
			// We use ignition for SSH keys deployment.
			SSHPublicKey: "",
		},
	}

	if len(key.AvailabilityZones(cr, r.location)) > 0 {
		azureMachine.Spec.FailureDomain = to.StringP(strconv.Itoa(key.AvailabilityZones(cr, r.location)[0]))
	}

	return azureMachine, nil
}

func (r *Resource) updateCRs(ctx context.Context, crmappings []crmapping) error {
	for _, m := range crmappings {
		// Construct new instance via reflection to ensure clean zero value object.
		var readCR runtime.Object
		{
			// Get the underlying type of desired runtime.Object.
			t := reflect.TypeOf(m.obj)

			// We know that underlying type is a pointer so let's dereference
			// it before cloning so that we get the actual object instance
			// instead of just an instance of a pointer.
			e := t.Elem()

			// Construct a new instance of that type and receive
			// `reflect.Value` object containing pointer to that instance.
			v := reflect.New(e)

			// Finally, extract the encapsulated `interface{}` from the
			// `reflect.Value` and cast it to instance of `runtime.Object`
			// interface.
			readCR = v.Interface().(runtime.Object)
		}

		// Acquire accessors for ObjectMeta and TypeMeta fields of CR.
		desiredMeta, err := meta.Accessor(m.obj)
		if err != nil {
			return microerror.Mask(err)
		}
		desiredType, err := meta.TypeAccessor(m.obj)
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
			err = r.ctrlClient.Create(ctx, m.obj)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created %s %s", desiredType.GetKind(), nsName.String()))
			continue
		} else if err != nil {
			return microerror.Mask(err)
		}

		// If needUpdateFunc or mergeFunc are nil, it means that given CR is
		// not intended to be updated. Only created when it doesn't already
		// exist.
		if m.needUpdateFunc == nil || m.mergeFunc == nil {
			return nil
		}

		updateNeeded, err := m.needUpdateFunc(readCR, m.obj)
		if err != nil {
			return microerror.Mask(err)
		}

		if updateNeeded {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found that %s %s needs updating", desiredType.GetKind(), nsName.String()))

			merged, err := m.mergeFunc(readCR, m.obj)
			if err != nil {
				return microerror.Mask(err)
			}

			readMeta, err := meta.Accessor(readCR)
			if err != nil {
				return microerror.Mask(err)
			}
			mergedMeta, err := meta.Accessor(merged)
			if err != nil {
				return microerror.Mask(err)
			}

			// Copy read CR's resource version to update object for optimistic
			// locking.
			mergedMeta.SetResourceVersion(readMeta.GetResourceVersion())

			err = r.ctrlClient.Update(ctx, merged)
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

func detectAzureClusterUpdate(orig, desired runtime.Object) (bool, error) {
	o := orig.(*capzv1alpha3.AzureCluster)
	d := desired.(*capzv1alpha3.AzureCluster)

	return len(o.Spec.NetworkSpec.Vnet.CIDRBlocks) != len(d.Spec.NetworkSpec.Vnet.CIDRBlocks) ||
		o.Spec.NetworkSpec.Vnet.CIDRBlocks[0] != d.Spec.NetworkSpec.Vnet.CIDRBlocks[0] ||
		o.Spec.NetworkSpec.Vnet.Name != d.Spec.NetworkSpec.Vnet.Name ||
		o.Spec.NetworkSpec.Vnet.ResourceGroup != d.Spec.NetworkSpec.Vnet.ResourceGroup ||
		o.Spec.ResourceGroup != d.Spec.ResourceGroup ||
		o.Spec.Location != d.Spec.Location ||
		o.Spec.ControlPlaneEndpoint != d.Spec.ControlPlaneEndpoint, nil
}

func mergeAzureCluster(orig, desired runtime.Object) (runtime.Object, error) {
	o := orig.(*capzv1alpha3.AzureCluster)
	d := desired.(*capzv1alpha3.AzureCluster)

	// Only copy specific parts of desired opbject
	o.Spec.NetworkSpec.Vnet.CIDRBlocks = d.Spec.NetworkSpec.Vnet.CIDRBlocks
	o.Spec.NetworkSpec.Vnet.Name = d.Spec.NetworkSpec.Vnet.Name
	o.Spec.NetworkSpec.Vnet.ResourceGroup = d.Spec.NetworkSpec.Vnet.ResourceGroup
	o.Spec.ResourceGroup = d.Spec.ResourceGroup
	o.Spec.Location = d.Spec.Location
	o.Spec.ControlPlaneEndpoint = d.Spec.ControlPlaneEndpoint

	return o, nil
}

func toInt32P(v int32) *int32 {
	return &v
}
