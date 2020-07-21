package spark

import (
	"bytes"
	"context"
	"crypto/sha512"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure/auth"
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

	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/service/controller/cloudconfig"
	"github.com/giantswarm/azure-operator/v4/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	dockerVolumeSizeGB  = 50
	ignitionBlobKey     = "ignitionBlob"
	kubeletVolumeSizeGB = 100
	kubeDNSIPLastOctet  = 10
	ProviderAzure       = "azure"
)

// EnsureCreated is checking if corresponding Spark CRD exists. In that case it renders
// k8scloudconfig for a node pool based on existing CAPI & CAPZ CRs.
// The rendered config is saved in a Secret, referenced by the Spark CRD.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	var sparkCR corev1alpha1.Spark
	{
		err = r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: azureMachinePool.Namespace, Name: azureMachinePool.Name}, &sparkCR)
		if errors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "bootstrap CR not found")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	var ignitionBlob []byte
	var dataHash string
	{
		ignitionBlob, err = r.createIgnitionBlob(ctx, &azureMachinePool)
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
			err = r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: azureMachinePool.Namespace, Name: sparkCR.Status.DataSecretName}, &s)
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
					Namespace: azureMachinePool.Namespace,
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

func (r *Resource) createIgnitionBlob(ctx context.Context, azureMachinePool *expcapzv1alpha3.AzureMachinePool) ([]byte, error) {
	cluster, err := r.getOwnerCluster(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	azureCluster, err := r.getAzureCluster(ctx, cluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	machinePool, err := r.getOwnerMachinePool(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	release, err := r.getRelease(ctx, machinePool.ObjectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	credentialSecret, err := r.getCredentialSecret(ctx, azureMachinePool)
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
			CertsSearcher:          r.certsSearcher,
			Logger:                 r.logger,
			Ignition:               r.ignition,
			OIDC:                   r.oidc,
			RandomkeysSearcher:     r.randomKeysSearcher,
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

		mappedAzureConfig, err := r.buildAzureConfig(ctx, cluster, azureCluster, machinePool, azureMachinePool)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		ignitionTemplateData = cloudconfig.IgnitionTemplateData{
			CustomObject: mappedAzureConfig,
			ClusterCerts: clusterCerts,
			Images:       images,
			Versions:     versions,
		}
	}

	b, err := cloudConfig.NewWorkerTemplate(ctx, ignitionTemplateData, encrypterObject)
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

func getAvailabilityZones(machinePool expcapiv1alpha3.MachinePool) ([]int, error) {
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

func (r *Resource) buildAzureConfig(ctx context.Context, cluster *capiv1alpha3.Cluster, azureCluster *capzv1alpha3.AzureCluster, machinePool *expcapiv1alpha3.MachinePool, azureMachinePool *expcapzv1alpha3.AzureMachinePool) (providerv1alpha1.AzureConfig, error) {
	var err error

	azureConfig := providerv1alpha1.AzureConfig{}
	azureConfig.Labels = make(map[string]string)

	{
		azureConfig.ObjectMeta.Name = key.ClusterID(cluster)
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
		azureConfig.Labels[label.Cluster] = key.ClusterID(cluster)
		azureConfig.Labels[label.XCluster] = key.ClusterID(cluster)
		azureConfig.Labels[label.Organization] = key.OrganizationID(cluster)
		azureConfig.Labels[label.ReleaseVersion] = key.ReleaseVersion(cluster)
	}

	{
		azureConfig.Labels[label.OperatorVersion] = key.OperatorVersion(azureCluster)

		azureConfig.Spec.Cluster.Kubernetes.Kubelet.Labels = ensureLabel(azureConfig.Spec.Cluster.Kubernetes.Kubelet.Labels, label.OperatorVersion, key.OperatorVersion(azureCluster))
		azureConfig.Spec.Cluster.Kubernetes.Kubelet.Labels = ensureLabel(azureConfig.Spec.Cluster.Kubernetes.Kubelet.Labels, "giantswarm.io/provider", ProviderAzure)

		azureConfig.Spec.Azure.AvailabilityZones, err = getAvailabilityZones(*machinePool)
		if err != nil {
			return providerv1alpha1.AzureConfig{}, microerror.Mask(err)
		}
	}

	/*
			XXX: Let's see if we can live without this. -tuommaki
		{
			clusterStatus, err := r.newClusterStatus(ctx, request, azureConfig.Labels[label.OperatorVersion])
			if err != nil {
				return nil, microerror.Mask(err)
			}

			azureConfig.Status.Cluster = clusterStatus
		}
	*/

	var (
		// hostDNSZone is created by stripping 3 first components from
		// API domain are e.g.
		// api.eggs2.k8s.gollum.azure.giantswarm.io becomes
		// gollum.azure.giantswarm.io.
		hostDNSZone       = strings.Join(strings.Split(azureConfig.Spec.Cluster.Kubernetes.API.Domain, ".")[3:], ".")
		hostResourceGroup = r.azure.HostCluster.ResourceGroup
	)

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
		for range newSpecClusterWorkerNodes(int(*machinePool.Spec.Replicas)) {
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
		credentialSecret, err := r.getCredentialSecret(ctx, cluster)
		if err != nil {
			return providerv1alpha1.AzureConfig{}, microerror.Mask(err)
		}

		azureConfig.Spec.Azure.CredentialSecret = *credentialSecret
	}

	return azureConfig, nil
}

func (r *Resource) newCluster(cluster *capiv1alpha3.Cluster, azureCluster *capzv1alpha3.AzureCluster, machinePool *expcapiv1alpha3.MachinePool) (providerv1alpha1.Cluster, error) {
	commonCluster := providerv1alpha1.Cluster{}

	{
		commonCluster.Calico.CIDR = r.calico.CIDRSize
		commonCluster.Calico.MTU = r.calico.MTU
		commonCluster.Calico.Subnet = r.calico.Subnet
	}

	{
		commonCluster.ID = key.ClusterID(cluster)
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
		apiServerIPRange := r.clusterIPRange
		if apiServerIPRange == "" {
			return providerv1alpha1.Cluster{}, microerror.Maskf(executionFailedError, "Kubernetes IP range must not be empty")
		}

		commonCluster.Kubernetes.API.ClusterIPRange = apiServerIPRange // TODO rename (HOST_SUBNET_RANGE in k8s-vm)
		commonCluster.Kubernetes.API.Domain = apiServerDomain
		commonCluster.Kubernetes.API.SecurePort = r.apiServerSecurePort
	}

	{
		_, ipNet, err := net.ParseCIDR(commonCluster.Kubernetes.API.ClusterIPRange)
		ip := ipNet.IP
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}
		ip[3] = kubeDNSIPLastOctet

		commonCluster.Kubernetes.DNS.IP = ip.String()
	}

	{
		kubeletDomain, err := newKubeletDomain(*azureCluster)
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}

		commonCluster.Kubernetes.Kubelet.Domain = kubeletDomain
		commonCluster.Kubernetes.Kubelet.Labels = r.kubeletLabels
	}

	{
		userList, err := newSpecClusterKubernetesSSHUsers(r.sshUserList)
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}

		commonCluster.Kubernetes.SSH.UserList = userList
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

func ensureLabel(labels string, key string, value string) string {
	if key == "" {
		return labels
	}
	if value == "" {
		return labels
	}

	var split []string
	if labels != "" {
		split = strings.Split(labels, ",")
	}

	var found bool
	for i, l := range split {
		if !strings.HasPrefix(l, key+"=") {
			continue
		}

		found = true
		split[i] = key + "=" + value
	}

	if !found {
		split = append(split, key+"="+value)
	}

	joined := strings.Join(split, ",")

	return joined
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

func newSpecClusterKubernetesSSHUsers(userList string) ([]providerv1alpha1.ClusterKubernetesSSHUser, error) {
	var sshUsers []providerv1alpha1.ClusterKubernetesSSHUser

	for _, user := range strings.Split(userList, ",") {
		if user == "" {
			continue
		}

		trimmed := strings.TrimSpace(user)
		split := strings.Split(trimmed, ":")

		if len(split) != 2 {
			return nil, microerror.Maskf(executionFailedError, "SSH user format must be <name>:<public key>")
		}

		u := providerv1alpha1.ClusterKubernetesSSHUser{
			Name:      split[0],
			PublicKey: split[1],
		}

		sshUsers = append(sshUsers, u)
	}

	return sshUsers, nil
}

func newAPIServerDomain(cr capzv1alpha3.AzureCluster) (string, error) {
	splitted := strings.Split(cr.Spec.ControlPlaneEndpoint.Host, ".")
	splitted[0] = certs.APICert.String()
	apiServerDomain := strings.Join(splitted, ".")

	return apiServerDomain, nil
}

func newEtcdServerDomain(cr capzv1alpha3.AzureCluster) (string, error) {
	splitted := strings.Split(cr.Spec.ControlPlaneEndpoint.Host, ".")
	splitted[0] = certs.EtcdCert.String()
	etcdServerDomain := strings.Join(splitted, ".")

	return etcdServerDomain, nil
}

func newKubeletDomain(cr capzv1alpha3.AzureCluster) (string, error) {
	splitted := strings.Split(cr.Spec.ControlPlaneEndpoint.Host, ".")
	splitted[0] = certs.WorkerCert.String()
	kubeletDomain := strings.Join(splitted, ".")

	return kubeletDomain, nil
}
