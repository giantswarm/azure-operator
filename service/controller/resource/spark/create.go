package spark

import (
	"bytes"
	"context"
	"crypto/sha512"
	"fmt"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v7/pkg/template"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/service/controller/cloudconfig"
	"github.com/giantswarm/azure-operator/v4/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	ignitionBlobKey = "ignitionBlob"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	var sparkCR corev1alpha1.Spark
	{
		err = r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: cr.Namespace, Name: cr.Name}, &sparkCR)
		if errors.IsNotFound(err) {
			// TODO: Cancel reconciliation until Spark CR is there.
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	var ignitionBlob []byte
	var dataHash string
	{
		ignitionBlob, err = r.createIgnitionBlob(ctx, &cr, &sparkCR)
		if IsRequirementsNotMet(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "ignition blob rendering requirements not met")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		h := sha512.New()
		dataHash = fmt.Sprintf("%x", h.Sum(ignitionBlob))
	}

	var dataSecret *corev1.Secret
	{
		if sparkCR.Status.DataSecretName != "" {
			var s corev1.Secret
			err = r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: cr.Namespace, Name: sparkCR.Status.DataSecretName}, &s)
			if errors.IsNotFound(err) {
				// This is ok. We'll create it then.
			} else if err != nil {
				return microerror.Mask(err)
			} else {
				dataSecret = &s
			}
		}

		// Create Secret for ignition data.
		if dataSecret == nil {
			dataSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName(sparkCR.Name),
					Namespace: sparkCR.Namespace,
				},
				Data: map[string][]byte{
					ignitionBlobKey: ignitionBlob,
				},
			}

			err = r.ctrlClient.Create(ctx, dataSecret)
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	{
		var sparkStatusUpdateNeeded bool

		if !sparkCR.Status.Ready {
			sparkCR.Status.Ready = true
			sparkStatusUpdateNeeded = true
		}

		if sparkCR.Status.DataSecretName != dataSecret.Name {
			sparkCR.Status.DataSecretName = dataSecret.Name
			sparkStatusUpdateNeeded = true
		}

		if sparkCR.Status.Verification.Hash != dataHash {
			sparkCR.Status.Verification.Hash = dataHash
			sparkCR.Status.Verification.Algorithm = "sha512"
			sparkStatusUpdateNeeded = true
		}

		if sparkStatusUpdateNeeded {
			err = r.ctrlClient.Status().Update(ctx, &sparkCR)
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	{
		if bytes.Compare(ignitionBlob, dataSecret.Data[ignitionBlobKey]) != 0 {
			dataSecret.Data[ignitionBlobKey] = ignitionBlob

			err = r.ctrlClient.Update(ctx, dataSecret)
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	return nil
}

func (r *Resource) createIgnitionBlob(ctx context.Context, cr *expcapzv1alpha3.AzureMachinePool, sparkCR *corev1alpha1.Spark) ([]byte, error) {
	cluster, err := r.getOwnerCluster(ctx, cr.ObjectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	azureCluster, err := r.getAzureCluster(ctx, cluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	machinePool, err := r.getOwnerMachinePool(ctx, cr.ObjectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	release, err := r.getRelease(ctx, machinePool.ObjectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var clusterCerts certs.Cluster
	{
		clusterCerts, err = r.certsSearcher.SearchCluster(key.ClusterID(cluster))
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var cloudConfig cloudconfig.Interface
	{
		// TODO: Construct cloudconfig instance.
		//
		// It probably requires that some configuration settings are wired from
		// configmap to resource. Some can be hopefully hardcoded here. Some
		// should be pulled from the outside configmap later (such as OIDC).
	}

	var encrypter encrypter.Interface
	{
		certificateEncryptionSecretName := key.CertificateEncryptionSecretName(cluster)

		encrypter, err = r.toEncrypterObject(ctx, certificateEncryptionSecretName)
		if errors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "encryptionkey resource is not ready")
			return nil, microerror.Mask(requirementsNotMetError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var ignitionTemplateData cloudconfig.IgnitionTemplateData
	{
		versions, err := k8scloudconfig.ExtractComponentVersions(release.Spec.Components)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		defaultVersions := key.DefaultVersions()
		versions.KubernetesAPIHealthz = defaultVersions.KubernetesAPIHealthz
		versions.KubernetesNetworkSetupDocker = defaultVersions.KubernetesNetworkSetupDocker
		images := k8scloudconfig.BuildImages(r.registryDomain, versions)

		ignitionTemplateData = cloudconfig.IgnitionTemplateData{
			CustomObject: toAzureConfig(cluster, azureCluster, machinePool, cr),
			ClusterCerts: clusterCerts,
			Images:       images,
			Versions:     versions,
		}
	}

	b, err := cloudConfig.NewWorkerTemplate(ctx, ignitionTemplateData, encrypter)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return []byte(b), nil
}

// getAzureCluster finds and returns an AzureCluster object using the specified params.
func (r *Resource) getAzureCluster(ctx context.Context, cluster *capiv1alpha3.Cluster) (*capzv1alpha3.AzureCluster, error) {
	if cluster.Spec.InfrastructureRef == nil {
		return nil, microerror.Maskf(executionFailedError, "Cluster.Spec.InfrasturctureRef == nil")
	}

	azureCluster := &capzv1alpha3.AzureCluster{}
	objectKey := client.ObjectKey{Name: cluster.Spec.InfrastructureRef.Name, Namespace: cluster.Spec.InfrastructureRef.Namespace}
	if err := r.ctrlClient.Get(ctx, objectKey, cluster); err != nil {
		return nil, err
	}

	r.logger = r.logger.With("cluster", cluster.Name)

	return azureCluster, nil
}

// getClusterByName finds and return a Cluster object using the specified params.
func (r *Resource) getClusterByName(ctx context.Context, namespace, name string) (*capiv1alpha3.Cluster, error) {
	cluster := &capiv1alpha3.Cluster{}
	objectKey := client.ObjectKey{Name: name, Namespace: namespace}
	if err := r.ctrlClient.Get(ctx, objectKey, cluster); err != nil {
		return nil, err
	}

	r.logger = r.logger.With("cluster", cluster.Name)

	return cluster, nil
}

// getOwnerCluster returns the Cluster object owning the current resource.
func (r *Resource) getOwnerCluster(ctx context.Context, obj metav1.ObjectMeta) (*capiv1alpha3.Cluster, error) {
	for _, ref := range obj.OwnerReferences {
		if ref.Kind == "Cluster" && ref.APIVersion == capiv1alpha3.GroupVersion.String() {
			return r.getClusterByName(ctx, obj.Namespace, ref.Name)
		}
	}

	return nil, nil
}

// getMachinePoolByName finds and return a MachinePool object using the specified params.
func (r *Resource) getMachinePoolByName(ctx context.Context, namespace, name string) (*expcapiv1alpha3.MachinePool, error) {
	machinePool := &expcapiv1alpha3.MachinePool{}
	objectKey := client.ObjectKey{Name: name, Namespace: namespace}
	if err := r.ctrlClient.Get(ctx, objectKey, machinePool); err != nil {
		return nil, err
	}

	r.logger = r.logger.With("machinePool", machinePool.Name)

	return machinePool, nil
}

// getOwnerMachinePool returns the MachinePool object owning the current resource.
func (r *Resource) getOwnerMachinePool(ctx context.Context, obj metav1.ObjectMeta) (*expcapiv1alpha3.MachinePool, error) {
	for _, ref := range obj.OwnerReferences {
		if ref.Kind == "MachinePool" && ref.APIVersion == expcapiv1alpha3.GroupVersion.String() {
			return r.getMachinePoolByName(ctx, obj.Namespace, ref.Name)
		}
	}

	return nil, nil
}

func (r *Resource) getRelease(ctx context.Context, obj metav1.ObjectMeta) (*releasev1alpha1.Release, error) {
	release := &releasev1alpha1.Release{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: corev1.NamespaceAll, Name: key.ReleaseVersion(&obj)}, release)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return release, nil
}

func secretName(base string) string {
	return base + "-machine-pool-ignition"
}

// toAzureConfig performs mapping from CAPI & CAPZ types to AzureConfig which
// fulfills k8scloudconfig requirements for rendering node pool worker node
// ignition templates.
func toAzureConfig(cluster *capiv1alpha3.Cluster, azureCluster *capzv1alpha3.AzureCluster, machinePool *expcapiv1alpha3.MachinePool, azureMachinePool *expcapzv1alpha3.AzureMachinePool) providerv1alpha1.AzureConfig {
	ac := providerv1alpha1.AzureConfig{}

	// TODO: Implement <3

	return ac
}
