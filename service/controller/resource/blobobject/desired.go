package blobobject

import (
	"context"
	"sync"

	"github.com/giantswarm/certs/v3/pkg/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v8/pkg/template"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v2/pkg/controller/context/resourcecanceledcontext"
	"golang.org/x/sync/errgroup"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/cloudconfig"
	"github.com/giantswarm/azure-operator/v5/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
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

	var cluster capiv1alpha3.Cluster
	{
		cluster = capiv1alpha3.Cluster{}
		err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: cr.Namespace, Name: cr.Name}, &cluster)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// Inject in the azureConfig the SREs' public keys.
	{
		cr.Spec.Cluster.Kubernetes.SSH.UserList = key.ToClusterKubernetesSSHUser(r.sshUserList)
	}

	var ignitionTemplateData cloudconfig.IgnitionTemplateData
	{
		versions, err := k8scloudconfig.ExtractComponentVersions(cc.Release.Release.Spec.Components)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		defaultVersions := key.DefaultVersions()
		versions.KubernetesAPIHealthz = defaultVersions.KubernetesAPIHealthz
		versions.KubernetesNetworkSetupDocker = defaultVersions.KubernetesNetworkSetupDocker
		images := k8scloudconfig.BuildImages(r.registryDomain, versions)

		ignitionTemplateData = cloudconfig.IgnitionTemplateData{
			Cluster:         &cluster,
			CustomObject:    cr,
			Images:          images,
			MasterCertFiles: masterCertFiles,
			Versions:        versions,
			WorkerCertFiles: workerCertFiles,
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

	return output, nil
}
