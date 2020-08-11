package cloudconfig

import (
	"context"
	"sync"

	"github.com/giantswarm/certs/v3/pkg/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v8/pkg/template"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v2/pkg/controller/context/resourcecanceledcontext"
	"golang.org/x/sync/errgroup"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/service/controller/cloudconfig"
	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	machinePool, err := r.getOwnerMachinePool(ctx, cr.ObjectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cluster, err := util.GetClusterFromMetadata(ctx, r.ctrlClient, cr.ObjectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	azureCluster, err := r.getAzureClusterFromCluster(ctx, cluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	release, err := r.getReleaseFromMetadata(ctx, azureCluster.ObjectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var masterCertFiles []certs.File
	var workerCertFiles []certs.File
	{
		g := &errgroup.Group{}
		m := sync.Mutex{}

		g.Go(func() error {
			tls, err := r.certsSearcher.SearchTLS(ctx, key.ClusterID(&cr), certs.APICert)
			if err != nil {
				return microerror.Mask(err)
			}
			m.Lock()
			masterCertFiles = append(masterCertFiles, certs.NewFilesAPI(tls)...)
			m.Unlock()

			return nil
		})

		g.Go(func() error {
			tls, err := r.certsSearcher.SearchTLS(ctx, key.ClusterID(&cr), certs.CalicoEtcdClientCert)
			if err != nil {
				return microerror.Mask(err)
			}
			m.Lock()
			masterCertFiles = append(masterCertFiles, certs.NewFilesCalicoEtcdClient(tls)...)
			workerCertFiles = append(workerCertFiles, certs.NewFilesCalicoEtcdClient(tls)...)
			m.Unlock()

			return nil
		})

		g.Go(func() error {
			tls, err := r.certsSearcher.SearchTLS(ctx, key.ClusterID(&cr), certs.EtcdCert)
			if err != nil {
				return microerror.Mask(err)
			}
			m.Lock()
			masterCertFiles = append(masterCertFiles, certs.NewFilesEtcd(tls)...)
			m.Unlock()

			return nil
		})

		g.Go(func() error {
			tls, err := r.certsSearcher.SearchTLS(ctx, key.ClusterID(&cr), certs.ServiceAccountCert)
			if err != nil {
				return microerror.Mask(err)
			}
			m.Lock()
			masterCertFiles = append(masterCertFiles, certs.NewFilesServiceAccount(tls)...)
			m.Unlock()

			return nil
		})

		g.Go(func() error {
			tls, err := r.certsSearcher.SearchTLS(ctx, key.ClusterID(&cr), certs.WorkerCert)
			if err != nil {
				return microerror.Mask(err)
			}
			m.Lock()
			workerCertFiles = append(workerCertFiles, certs.NewFilesWorker(tls)...)
			m.Unlock()

			return nil
		})

		err := g.Wait()
		if certs.IsTimeout(err) {
			return "", microerror.Maskf(timeoutError, "waited too long for certificates")
		} else if err != nil {
			return "", microerror.Mask(err)
		}
	}

	storageAccountName := key.StorageAccountName(&cr)
	containerName := key.BlobContainerName()
	certificateEncryptionSecretName := key.CertificateEncryptionSecretName(&cr)

	encrypter, err := r.toEncrypterObject(ctx, certificateEncryptionSecretName)
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "encryptionkey resource is not ready")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
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
			AzureMachinePool: &cr,
			MachinePool:      machinePool,
			Cluster:          cluster,
			AzureCluster:     azureCluster,
			Images:           images,
			MasterCertFiles:  masterCertFiles,
			WorkerCertFiles:  workerCertFiles,
		}
	}

	output := []ContainerObjectState{}
	{
		b, err := cc.CloudConfig.NewMasterTemplate(ctx, ignitionTemplateData, encrypter)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		k := key.BlobName(&cr, prefixMaster)
		containerObjectState := ContainerObjectState{
			Body:               b,
			ContainerName:      containerName,
			Key:                k,
			StorageAccountName: storageAccountName,
		}

		output = append(output, containerObjectState)
	}

	{
		b, err := cc.CloudConfig.NewWorkerTemplate(ctx, ignitionTemplateData, encrypter)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		k := key.BlobName(&cr, prefixWorker)
		containerObjectState := ContainerObjectState{
			Body:               b,
			ContainerName:      containerName,
			Key:                k,
			StorageAccountName: storageAccountName,
		}

		output = append(output, containerObjectState)
	}

	return output, nil
}

// getOwnerMachinePool returns the MachinePool object owning the current resource.
func (r *Resource) getOwnerMachinePool(ctx context.Context, obj metav1.ObjectMeta) (*v1alpha3.MachinePool, error) {
	for _, ref := range obj.OwnerReferences {
		if ref.Kind == "MachinePool" && ref.APIVersion == v1alpha3.GroupVersion.String() {
			return r.getMachinePoolByName(ctx, obj.Namespace, ref.Name)
		}
	}

	return nil, nil
}

// getMachinePoolByName finds and return a MachinePool object using the specified params.
func (r *Resource) getMachinePoolByName(ctx context.Context, namespace, name string) (*v1alpha3.MachinePool, error) {
	machinePool := &v1alpha3.MachinePool{}
	objectKey := ctrlclient.ObjectKey{Name: name, Namespace: namespace}
	if err := r.ctrlClient.Get(ctx, objectKey, machinePool); err != nil {
		return nil, err
	}

	return machinePool, nil
}

func (r *Resource) getAzureClusterFromCluster(ctx context.Context, cluster *capiv1alpha3.Cluster) (*capzv1alpha3.AzureCluster, error) {
	azureCluster := &capzv1alpha3.AzureCluster{}
	azureClusterName := ctrlclient.ObjectKey{
		Namespace: cluster.Spec.InfrastructureRef.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}

	err := r.ctrlClient.Get(ctx, azureClusterName, azureCluster)
	if err != nil {
		return azureCluster, microerror.Mask(err)
	}

	return azureCluster, nil
}
