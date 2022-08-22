package spark

import (
	"bytes"
	"context"
	"crypto/sha512"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	corev1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs/v4/pkg/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v14/pkg/template"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/reconciliationcanceledcontext"
	releasev1alpha1 "github.com/giantswarm/release-operator/v3/api/v1alpha1"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	capiutil "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/service/controller/cloudconfig"
	"github.com/giantswarm/azure-operator/v5/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
	"github.com/giantswarm/azure-operator/v5/service/network"
)

const (
	dockerVolumeSizeGB  = 50
	kubeletVolumeSizeGB = 100
	// DNS domain for dns searches in pods.
	kubeletClusterDomain = "cluster.local"
	kubeDNSIPLastOctet   = 10
	ProviderAzure        = "azure"
)

// EnsureCreated is checking if corresponding Spark CRD exists. In that case it renders
// k8scloudconfig for a node pool based on existing CAPI & CAPZ CRs.
// The rendered config is saved in a Secret, referenced by the Spark CRD.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	machinePool, err := r.getOwnerMachinePool(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	if machinePool == nil {
		r.logger.Debugf(ctx, "AzureMachinePool Owner Reference hasn't been set yet, so we don't know parent MachinePool")
		r.logger.Debugf(ctx, "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
		return nil
	}

	if !machinePool.GetDeletionTimestamp().IsZero() {
		r.logger.Debugf(ctx, "MachinePool is being deleted, skipping rendering cloud config")
		return nil
	}

	cluster, err := capiutil.GetClusterByName(ctx, r.ctrlClient, machinePool.Namespace, machinePool.Spec.ClusterName)
	if err != nil {
		return microerror.Mask(err)
	}

	if !cluster.GetDeletionTimestamp().IsZero() {
		r.logger.Debugf(ctx, "Cluster is being deleted, skipping rendering cloud config")
		return nil
	}

	azureCluster, err := r.getAzureCluster(ctx, cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	if !azureCluster.GetDeletionTimestamp().IsZero() {
		r.logger.Debugf(ctx, "AzureCluster is being deleted, skipping rendering cloud config")
		return nil
	}

	var sparkCR corev1alpha1.Spark
	{
		err = r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: azureMachinePool.Namespace, Name: azureMachinePool.Name}, &sparkCR)
		if errors.IsNotFound(err) {
			r.logger.Debugf(ctx, "bootstrap CR not found")
			r.logger.Debugf(ctx, "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	var ignitionBlob []byte
	var dataHash string
	{
		r.logger.Debugf(ctx, "trying to render ignition cloud config for this node pool")

		ignitionBlob, err = r.createIgnitionBlob(ctx, cluster, azureCluster, machinePool, &azureMachinePool)
		if IsRequirementsNotMet(err) {
			r.logger.Debugf(ctx, "ignition blob rendering requirements not met")
			r.logger.Debugf(ctx, "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		} else if certs.IsTimeout(err) {
			r.logger.Debugf(ctx, "waited too long for certificates")
			r.logger.Debugf(ctx, "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		h := sha512.New()
		dataHash = fmt.Sprintf("%x", h.Sum(ignitionBlob))

		r.logger.Debugf(ctx, "rendered ignition cloud config for this node pool")
	}

	var dataSecret *corev1.Secret
	{
		if sparkCR.Status.DataSecretName != "" {
			r.logger.Debugf(ctx, "Spark CR status already references a Secret %#q", sparkCR.Status.DataSecretName)
			var s corev1.Secret
			err = r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: azureMachinePool.Namespace, Name: sparkCR.Status.DataSecretName}, &s)
			if errors.IsNotFound(err) {
				// This is ok. We'll create it then.
				r.logger.Debugf(ctx, "Secret %#q was not found, it will be created", sparkCR.Status.DataSecretName)
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
					Namespace: azureMachinePool.Namespace,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         corev1alpha1.SchemeGroupVersion.String(),
							BlockOwnerDeletion: to.BoolPtr(true),
							Kind:               "Spark",
							Name:               sparkCR.Name,
							UID:                sparkCR.UID,
						},
					},
				},
				Data: map[string][]byte{
					key.CloudConfigSecretKey: ignitionBlob,
				},
			}

			err = r.ctrlClient.Create(ctx, dataSecret)
			if err != nil {
				return microerror.Mask(err)
			}
			r.logger.Debugf(ctx, "Secret %#q created with ignition cloud config", secretName(sparkCR.Name))
		}
	}

	{
		if !bytes.Equal(ignitionBlob, dataSecret.Data[key.CloudConfigSecretKey]) {
			r.logger.Debugf(ctx, "Ignition cloud config in Secret %#q is out of date, updating it", dataSecret.Name)

			dataSecret.Data[key.CloudConfigSecretKey] = ignitionBlob

			err = r.ctrlClient.Update(ctx, dataSecret)
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
			r.logger.Debugf(ctx, "Updating Spark CR %#q", sparkCR.Name)
			err = r.ctrlClient.Status().Update(ctx, &sparkCR)
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	return nil
}

func (r *Resource) createIgnitionBlob(ctx context.Context, cluster *capi.Cluster, azureCluster *capz.AzureCluster, machinePool *capiexp.MachinePool, azureMachinePool *capzexp.AzureMachinePool) ([]byte, error) {
	release, err := r.getRelease(ctx, machinePool.ObjectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	credentialSecret, err := r.clientFactory.GetCredentialSecret(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var masterCertFiles []certs.File
	var workerCertFiles []certs.File
	{
		g := &errgroup.Group{}
		m := sync.Mutex{}

		g.Go(func() error {
			tls, err := r.certsSearcher.SearchTLS(ctx, key.ClusterName(cluster), certs.APICert)
			if err != nil {
				return microerror.Mask(err)
			}
			m.Lock()
			masterCertFiles = append(masterCertFiles, certs.NewFilesAPI(tls)...)
			m.Unlock()

			return nil
		})

		g.Go(func() error {
			tls, err := r.certsSearcher.SearchTLS(ctx, key.ClusterName(cluster), certs.CalicoEtcdClientCert)
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
			tls, err := r.certsSearcher.SearchTLS(ctx, key.ClusterName(cluster), certs.EtcdCert)
			if err != nil {
				return microerror.Mask(err)
			}
			m.Lock()
			masterCertFiles = append(masterCertFiles, certs.NewFilesEtcd(tls)...)
			m.Unlock()

			return nil
		})

		g.Go(func() error {
			tls, err := r.certsSearcher.SearchTLS(ctx, key.ClusterName(cluster), certs.ServiceAccountCert)
			if err != nil {
				return microerror.Mask(err)
			}
			m.Lock()
			masterCertFiles = append(masterCertFiles, certs.NewFilesServiceAccount(tls)...)
			m.Unlock()

			return nil
		})

		g.Go(func() error {
			tls, err := r.certsSearcher.SearchTLS(ctx, key.ClusterName(cluster), certs.WorkerCert)
			if err != nil {
				return microerror.Mask(err)
			}
			m.Lock()
			workerCertFiles = append(workerCertFiles, certs.NewFilesWorker(tls)...)
			m.Unlock()

			return nil
		})

		err := g.Wait()
		if err != nil {
			return nil, microerror.Mask(err)
		}

		// Sort slices so we have a deterministic output.
		sort.SliceStable(masterCertFiles, func(i, j int) bool {
			return masterCertFiles[i].AbsolutePath < masterCertFiles[j].AbsolutePath
		})

		sort.SliceStable(workerCertFiles, func(i, j int) bool {
			return workerCertFiles[i].AbsolutePath < workerCertFiles[j].AbsolutePath
		})
	}

	var organizationAzureClientCredentialsConfig auth.ClientCredentialsConfig
	var subscriptionID string
	{
		organizationAzureClientCredentialsConfig, subscriptionID, _, err = r.credentialProvider.GetOrganizationAzureCredentials(ctx, credentialSecret.Namespace, credentialSecret.Name)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var cloudConfig cloudconfig.Interface
	{
		// It probably requires that some configuration settings are wired from
		// configmap to resource. Some can be hopefully hardcoded here. Some
		// should be pulled from the outside configmap later (such as OIDC).
		c := cloudconfig.Config{
			Azure:                  r.azure,
			AzureClientCredentials: organizationAzureClientCredentialsConfig,
			CtrlClient:             r.ctrlClient,
			DockerhubToken:         r.dockerhubToken,
			Logger:                 r.logger,
			Ignition:               r.ignition,
			OIDC:                   r.oidc,
			RegistryMirrors:        r.registryMirrors,
			SSOPublicKey:           r.ssoPublicKey,
			SubscriptionID:         subscriptionID,
		}
		cloudConfig, err = cloudconfig.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var encrypterObject encrypter.Interface
	{
		certificateEncryptionSecretName := key.CertificateEncryptionSecretName(cluster)

		encrypterObject, err = r.toEncrypterObject(ctx, certificateEncryptionSecretName)
		if errors.IsNotFound(microerror.Cause(err)) {
			r.logger.Debugf(ctx, "encryptionkey resource is not ready")
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

		// We already have AzureConfig saved in the API, since we create those whenever we create Cluster and AzureCluster.
		// But we need to create an AzureConfig on the fly here because different node pools will use different cloudconfigs,
		// and the existing AzureConfig wouldn't contain the right values (vmsize, # of replicas, etc) for this specific node pool that we are creating.
		mappedAzureConfig, err := r.buildAzureConfig(cluster, azureCluster, machinePool, azureMachinePool, credentialSecret)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		ignitionTemplateData = cloudconfig.IgnitionTemplateData{
			AzureMachinePool: azureMachinePool,
			CustomObject:     mappedAzureConfig,
			Images:           images,
			MachinePool:      machinePool,
			MasterCertFiles:  masterCertFiles,
			Versions:         versions,
			WorkerCertFiles:  workerCertFiles,
		}
	}

	b, err := cloudConfig.NewWorkerTemplate(ctx, ignitionTemplateData, encrypterObject)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return []byte(b), nil
}

// getAzureCluster finds and returns an AzureCluster object using the specified params.
func (r *Resource) getAzureCluster(ctx context.Context, cluster *capi.Cluster) (*capz.AzureCluster, error) {
	if cluster.Spec.InfrastructureRef == nil {
		return nil, microerror.Maskf(executionFailedError, "Cluster.Spec.InfrasturctureRef == nil")
	}

	azureCluster := &capz.AzureCluster{}
	objectKey := client.ObjectKey{Name: cluster.Spec.InfrastructureRef.Name, Namespace: cluster.Namespace}
	if err := r.ctrlClient.Get(ctx, objectKey, azureCluster); err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger = r.logger.With("azureCluster", azureCluster.Name)

	return azureCluster, nil
}

// getMachinePoolByName finds and return a MachinePool object using the specified params.
func (r *Resource) getMachinePoolByName(ctx context.Context, namespace, name string) (*capiexp.MachinePool, error) {
	machinePool := &capiexp.MachinePool{}
	objectKey := client.ObjectKey{Name: name, Namespace: namespace}
	if err := r.ctrlClient.Get(ctx, objectKey, machinePool); err != nil {
		return nil, err
	}

	r.logger = r.logger.With("machinePool", machinePool.Name)

	return machinePool, nil
}

// getOwnerMachinePool returns the MachinePool object owning the current resource.
func (r *Resource) getOwnerMachinePool(ctx context.Context, obj metav1.ObjectMeta) (*capiexp.MachinePool, error) {
	for _, ref := range obj.OwnerReferences {
		if ref.Kind == "MachinePool" && ref.APIVersion == capiexp.GroupVersion.String() {
			return r.getMachinePoolByName(ctx, obj.Namespace, ref.Name)
		}
	}

	return nil, nil
}

func (r *Resource) getRelease(ctx context.Context, obj metav1.ObjectMeta) (*releasev1alpha1.Release, error) {
	rName := key.ReleaseVersion(&obj)
	if !strings.HasPrefix(rName, "v") {
		rName = "v" + rName
	}

	release := &releasev1alpha1.Release{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: corev1.NamespaceAll, Name: rName}, release)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return release, nil
}

func getAvailabilityZones(machinePool capiexp.MachinePool) ([]int, error) {
	var availabilityZones []int
	for _, fd := range machinePool.Spec.FailureDomains {
		intFd, err := strconv.Atoi(fd)
		if err != nil {
			return availabilityZones, microerror.Mask(err)
		}
		availabilityZones = append(availabilityZones, intFd)
	}

	return availabilityZones, nil
}

func (r *Resource) buildAzureConfig(cluster *capi.Cluster, azureCluster *capz.AzureCluster, machinePool *capiexp.MachinePool, azureMachinePool *capzexp.AzureMachinePool, credentialSecret *providerv1alpha1.CredentialSecret) (providerv1alpha1.AzureConfig, error) {
	var err error

	azureConfig := providerv1alpha1.AzureConfig{}
	azureConfig.Labels = make(map[string]string)

	{
		azureConfig.ObjectMeta.Name = key.ClusterName(cluster)
		azureConfig.ObjectMeta.Namespace = cluster.Namespace
	}

	{
		cluster, err := r.newCluster(cluster, azureCluster, machinePool)
		if err != nil {
			return providerv1alpha1.AzureConfig{}, microerror.Mask(err)
		}

		cluster.Kubernetes.CloudProvider = ProviderAzure

		azureConfig.Spec.Cluster = cluster
	}

	{
		azureConfig.Labels[label.Cluster] = key.ClusterName(cluster)
		azureConfig.Labels[capi.ClusterLabelName] = key.ClusterName(cluster)
		azureConfig.Labels[label.Organization] = key.OrganizationID(cluster)
		azureConfig.Labels[label.ReleaseVersion] = key.ReleaseVersion(cluster)
		azureConfig.Labels[label.OperatorVersion] = key.OperatorVersion(azureCluster)
	}

	{
		azureConfig.Spec.Azure.AvailabilityZones, err = getAvailabilityZones(*machinePool)
		if err != nil {
			return providerv1alpha1.AzureConfig{}, microerror.Mask(err)
		}
	}

	var (
		// hostDNSZone is created by stripping 3 first components from
		// API domain are e.g.
		// api.eggs2.k8s.gollum.azure.giantswarm.io becomes
		// gollum.azure.giantswarm.io.
		hostDNSZone       = strings.Join(strings.Split(azureConfig.Spec.Cluster.Kubernetes.API.Domain, ".")[3:], ".")
		hostResourceGroup = r.azure.HostCluster.ResourceGroup
	)

	{
		if len(azureCluster.Spec.NetworkSpec.Vnet.CIDRBlocks) < 1 {
			return providerv1alpha1.AzureConfig{}, microerror.Mask(vnetCidrNotSetError)
		}

		azureConfig.Spec.Azure.VirtualNetwork.CIDR = azureCluster.Spec.NetworkSpec.Vnet.CIDRBlocks[0]
	}

	azureConfig.Spec.Azure.DNSZones.API.Name = hostDNSZone
	azureConfig.Spec.Azure.DNSZones.API.ResourceGroup = hostResourceGroup
	azureConfig.Spec.Azure.DNSZones.Etcd.Name = hostDNSZone
	azureConfig.Spec.Azure.DNSZones.Etcd.ResourceGroup = hostResourceGroup
	azureConfig.Spec.Azure.DNSZones.Ingress.Name = hostDNSZone
	azureConfig.Spec.Azure.DNSZones.Ingress.ResourceGroup = hostResourceGroup

	{

		masterNodes := []providerv1alpha1.AzureConfigSpecAzureNode{
			{
				VMSize:              "Standard_D4_v2",
				DockerVolumeSizeGB:  dockerVolumeSizeGB,
				KubeletVolumeSizeGB: kubeletVolumeSizeGB,
			},
		}

		var workerNodes []providerv1alpha1.AzureConfigSpecAzureNode
		for i := int32(0); i < *machinePool.Spec.Replicas; i++ {
			n := providerv1alpha1.AzureConfigSpecAzureNode{
				VMSize:              azureMachinePool.Spec.Template.VMSize,
				DockerVolumeSizeGB:  dockerVolumeSizeGB,
				KubeletVolumeSizeGB: kubeletVolumeSizeGB,
			}
			workerNodes = append(workerNodes, n)
		}

		azureConfig.Spec.Azure.Masters = masterNodes
		azureConfig.Spec.Azure.Workers = workerNodes
	}

	{
		azureConfig.Spec.Azure.CredentialSecret = *credentialSecret
	}

	return azureConfig, nil
}

func (r *Resource) newCluster(cluster *capi.Cluster, azureCluster *capz.AzureCluster, machinePool *capiexp.MachinePool) (providerv1alpha1.Cluster, error) {
	commonCluster := providerv1alpha1.Cluster{}

	{
		if len(azureCluster.Spec.NetworkSpec.Vnet.CIDRBlocks) < 1 {
			return providerv1alpha1.Cluster{}, microerror.Mask(vnetCidrNotSetError)
		}

		_, networkCIDR, err := net.ParseCIDR(azureCluster.Spec.NetworkSpec.Vnet.CIDRBlocks[0])
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}

		azureNetwork, err := network.Compute(*networkCIDR)
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}

		commonCluster.Calico.CIDR = r.calicoCIDRSize
		commonCluster.Calico.MTU = r.calicoMTU
		commonCluster.Calico.Subnet = azureNetwork.Calico.String()
	}

	{
		commonCluster.ID = key.ClusterName(cluster)
	}

	{
		commonCluster.Customer.ID = key.OrganizationID(cluster)
	}

	{
		etcdServerDomain, err := newEtcdServerDomain(*azureCluster)
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}

		commonCluster.Etcd.Domain = etcdServerDomain
		commonCluster.Etcd.Prefix = r.etcdPrefix
	}

	{
		apiServerDomain, err := newAPIServerDomain(*azureCluster)
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}

		commonCluster.Kubernetes.API.ClusterIPRange = r.clusterIPRange // TODO rename (HOST_SUBNET_RANGE in k8s-vm)
		commonCluster.Kubernetes.API.Domain = apiServerDomain
		commonCluster.Kubernetes.API.SecurePort = r.apiServerSecurePort
	}

	{
		_, ipNet, err := net.ParseCIDR(commonCluster.Kubernetes.API.ClusterIPRange)
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}
		ip := ipNet.IP
		ip[3] = kubeDNSIPLastOctet

		commonCluster.Kubernetes.DNS.IP = ip.String()
	}

	{
		kubeletDomain, err := newKubeletDomain(*azureCluster)
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}

		kubeletLabels, err := key.KubeletLabelsNodePool(machinePool)
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}

		commonCluster.Kubernetes.Kubelet.Domain = kubeletDomain
		commonCluster.Kubernetes.Kubelet.Labels = kubeletLabels
	}

	{
		commonCluster.Kubernetes.Domain = kubeletClusterDomain
	}

	{
		commonCluster.Kubernetes.SSH.UserList = key.ToClusterKubernetesSSHUser(r.sshUserList)
	}

	{
		commonCluster.Masters = newSpecClusterMasterNodes()
		commonCluster.Workers = newSpecClusterWorkerNodes(int(*machinePool.Spec.Replicas))
	}

	{
		commonCluster.Scaling.Max = int(*machinePool.Spec.Replicas)
		commonCluster.Scaling.Min = int(*machinePool.Spec.Replicas)
	}

	return commonCluster, nil

}

func newSpecClusterMasterNodes() []providerv1alpha1.ClusterNode {
	// Return one master node with empty ID. I don't expect it to be used
	// anywhere.
	masterNodes := make([]providerv1alpha1.ClusterNode, 1)
	masterNodes[0].ID = "master-0"
	return masterNodes
}

func newSpecClusterWorkerNodes(numWorkers int) []providerv1alpha1.ClusterNode {
	var workerNodes []providerv1alpha1.ClusterNode

	for i := 0; i < numWorkers; i++ {
		n := providerv1alpha1.ClusterNode{
			ID: fmt.Sprintf("node-%d", i),
		}

		workerNodes = append(workerNodes, n)
	}

	return workerNodes
}

func newAPIServerDomain(cr capz.AzureCluster) (string, error) {
	splitted := strings.Split(cr.Spec.ControlPlaneEndpoint.Host, ".")
	splitted[0] = certs.APICert.String()
	apiServerDomain := strings.Join(splitted, ".")

	return apiServerDomain, nil
}

func newEtcdServerDomain(cr capz.AzureCluster) (string, error) {
	splitted := strings.Split(cr.Spec.ControlPlaneEndpoint.Host, ".")
	splitted[0] = certs.EtcdCert.String()
	etcdServerDomain := strings.Join(splitted, ".")

	return etcdServerDomain, nil
}

func newKubeletDomain(cr capz.AzureCluster) (string, error) {
	splitted := strings.Split(cr.Spec.ControlPlaneEndpoint.Host, ".")
	splitted[0] = certs.WorkerCert.String()
	kubeletDomain := strings.Join(splitted, ".")

	return kubeletDomain, nil
}
