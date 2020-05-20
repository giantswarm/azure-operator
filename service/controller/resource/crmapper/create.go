package crmapper

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strings"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	dockerVolumeSizeGB  = 50
	kubeletVolumeSizeGB = 100
	azureAPIVersion     = "provider.giantswarm.io"

	kubeDNSIPLastOctet = 10

	ProviderAzure = "azure"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	azureCluster, err := key.ToAzureCluster(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	var cluster capiv1alpha3.Cluster
	{
		nsName := types.NamespacedName{
			Name:      key.ClusterName(&azureCluster),
			Namespace: azureCluster.Namespace,
		}
		err = r.ctrlClient.Get(ctx, nsName, &cluster)
		if errors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("referenced Cluster CR (%q) not found", nsName.String()))
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	var masterMachines []capzv1alpha3.AzureMachine
	var workerMachines []capzv1alpha3.AzureMachine
	{
		azureMachineList := &capzv1alpha3.AzureMachineList{}
		{
			err := r.ctrlClient.List(
				ctx,
				azureMachineList,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabels{label.Cluster: key.ClusterID(&cluster)},
			)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		for _, m := range azureMachineList.Items {
			if key.IsControlPlaneMachine(&m) {
				masterMachines = append(masterMachines, m)
			} else {
				workerMachines = append(workerMachines, m)
			}
		}

		if len(masterMachines) < 1 {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no control plane AzureMachines found for cluster %q", key.ClusterID(&cluster)))
			r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		}

		if len(workerMachines) < 1 {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("worker AzureMachines found for cluster %q", key.ClusterID(&cluster)))
			r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		}
	}

	var mappedAzureConfig providerv1alpha1.AzureConfig
	{
		mappedAzureConfig, err = r.buildAzureConfig(ctx, cluster, azureCluster, masterMachines, workerMachines)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var presentAzureConfig providerv1alpha1.AzureConfig
	{
		nsName := types.NamespacedName{
			Name:      key.ClusterID(&cluster),
			Namespace: metav1.NamespaceDefault,
		}
		err = r.ctrlClient.Get(ctx, nsName, &presentAzureConfig)
		if errors.IsNotFound(err) {
			err = r.ctrlClient.Create(ctx, &mappedAzureConfig)
			if err != nil {
				return microerror.Mask(err)
			}

			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		// Copy object meta data such as Generation, ResourceVersion etc.
		mappedAzureConfig.ObjectMeta = presentAzureConfig.ObjectMeta

		// Were there any changes that requires CR update?
		if reflect.DeepEqual(mappedAzureConfig, presentAzureConfig) {
			return nil
		}

		err = r.ctrlClient.Update(ctx, &mappedAzureConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *Resource) buildAzureConfig(ctx context.Context, cluster capiv1alpha3.Cluster, azureCluster capzv1alpha3.AzureCluster, masters, workers []capzv1alpha3.AzureMachine) (providerv1alpha1.AzureConfig, error) {
	var err error

	azureConfig := providerv1alpha1.AzureConfig{}
	azureConfig.Labels = make(map[string]string)

	{
		azureConfig.TypeMeta.APIVersion = azureAPIVersion
		azureConfig.TypeMeta.Kind = "AzureConfig"
	}

	{
		azureConfig.ObjectMeta.Name = key.ClusterID(&cluster)
		azureConfig.ObjectMeta.Namespace = cluster.Namespace
	}

	{
		cluster, err := r.newCluster(cluster, azureCluster, workers)
		if err != nil {
			return providerv1alpha1.AzureConfig{}, microerror.Mask(err)
		}

		cluster.Kubernetes.CloudProvider = ProviderAzure

		azureConfig.Spec.Cluster = cluster
	}

	{
		azureConfig.Labels[label.ReleaseVersion] = key.ReleaseVersion(&cluster)
	}

	{
		azureConfig.Labels[label.OperatorVersion] = key.OperatorVersion(&azureCluster)

		azureConfig.Spec.Cluster.Kubernetes.Kubelet.Labels = ensureLabel(azureConfig.Spec.Cluster.Kubernetes.Kubelet.Labels, label.OperatorVersion, key.OperatorVersion(&azureCluster))
		azureConfig.Spec.Cluster.Kubernetes.Kubelet.Labels = ensureLabel(azureConfig.Spec.Cluster.Kubernetes.Kubelet.Labels, "giantswarm.io/provider", ProviderAzure)
		azureConfig.Spec.VersionBundle.Version = key.OperatorVersion(&azureCluster)

		azureConfig.Spec.Azure.AvailabilityZones, err = getAvailabilityZones(masters)
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
		// API domain are stipped, e.g.
		// api.eggs2.k8s.gollum.azure.giantswarm.io becomes
		// gollum.azure.giantswarm.io.
		hostDNSZone       = strings.Join(strings.Split(azureConfig.Spec.Cluster.Kubernetes.API.Domain, ".")[3:], ".")
		hostResourceGroup = r.viper.GetString(r.flag.Service.Azure.HostCluster.ResourceGroup)
	)

	azureConfig.Spec.Azure.DNSZones.API.Name = hostDNSZone
	azureConfig.Spec.Azure.DNSZones.API.ResourceGroup = hostResourceGroup
	azureConfig.Spec.Azure.DNSZones.Etcd.Name = hostDNSZone
	azureConfig.Spec.Azure.DNSZones.Etcd.ResourceGroup = hostResourceGroup
	azureConfig.Spec.Azure.DNSZones.Ingress.Name = hostDNSZone
	azureConfig.Spec.Azure.DNSZones.Ingress.ResourceGroup = hostResourceGroup

	{

		var masterNodes []providerv1alpha1.AzureConfigSpecAzureNode
		for _, m := range masters {
			n := providerv1alpha1.AzureConfigSpecAzureNode{
				VMSize:              m.Spec.VMSize,
				DockerVolumeSizeGB:  dockerVolumeSizeGB,
				KubeletVolumeSizeGB: kubeletVolumeSizeGB,
			}
			masterNodes = append(masterNodes, n)
		}

		var workerNodes []providerv1alpha1.AzureConfigSpecAzureNode
		for _, m := range workers {
			n := providerv1alpha1.AzureConfigSpecAzureNode{
				VMSize:              m.Spec.VMSize,
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

func (r *Resource) newCluster(cluster capiv1alpha3.Cluster, azureCluster capzv1alpha3.AzureCluster, workers []capzv1alpha3.AzureMachine) (providerv1alpha1.Cluster, error) {
	commonCluster := providerv1alpha1.Cluster{}

	{
		commonCluster.Calico.CIDR = r.viper.GetInt(r.flag.Service.Cluster.Calico.CIDR)
		commonCluster.Calico.MTU = r.viper.GetInt(r.flag.Service.Cluster.Calico.MTU)
		commonCluster.Calico.Subnet = r.viper.GetString(r.flag.Service.Cluster.Calico.Subnet)
	}

	{
		commonCluster.ID = key.ClusterID(&cluster)
	}

	{
		commonCluster.Customer.ID = key.OrganizationID(&cluster)
	}

	{
		commonCluster.Docker.Daemon.CIDR = r.viper.GetString(r.flag.Service.Cluster.Docker.Daemon.CIDR)
	}

	{
		etcdDomain, err := newEtcdDomain(azureCluster)
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}

		commonCluster.Etcd.AltNames = r.viper.GetString(r.flag.Service.Cluster.Etcd.AltNames)
		commonCluster.Etcd.Domain = etcdDomain
		commonCluster.Etcd.Port = r.viper.GetInt(r.flag.Service.Cluster.Etcd.Port)
		commonCluster.Etcd.Prefix = r.viper.GetString(r.flag.Service.Cluster.Etcd.Prefix)
	}

	{
		apiServerDomain, err := newAPIServerDomain(azureCluster)
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}
		apiServerIPRange := r.viper.GetString(r.flag.Service.Cluster.Kubernetes.API.ClusterIPRange)
		if apiServerIPRange == "" {
			return providerv1alpha1.Cluster{}, microerror.Maskf(executionFailedError, "Kubernetes IP range must not be empty")
		}

		commonCluster.Kubernetes.API.ClusterIPRange = apiServerIPRange // TODO rename (HOST_SUBNET_RANGE in k8s-vm)
		commonCluster.Kubernetes.API.Domain = apiServerDomain
		commonCluster.Kubernetes.API.SecurePort = r.viper.GetInt(r.flag.Service.Cluster.Kubernetes.API.SecurePort)
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
		commonCluster.Kubernetes.Domain = r.viper.GetString(r.flag.Service.Cluster.Kubernetes.Domain)
	}

	{
		ingressControllerDomain, err := r.newIngressControllerDomain(azureCluster)
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}
		ingressWildcardDomain, err := r.newIngressWildcardDomain(azureCluster)
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}

		commonCluster.Kubernetes.IngressController.Domain = ingressControllerDomain
		commonCluster.Kubernetes.IngressController.Docker.Image = r.viper.GetString(r.flag.Service.Cluster.Kubernetes.IngressController.Docker.Image)
		commonCluster.Kubernetes.IngressController.WildcardDomain = ingressWildcardDomain
		commonCluster.Kubernetes.IngressController.InsecurePort = r.viper.GetInt(r.flag.Service.Cluster.Kubernetes.IngressController.InsecurePort)
		commonCluster.Kubernetes.IngressController.SecurePort = r.viper.GetInt(r.flag.Service.Cluster.Kubernetes.IngressController.SecurePort)
	}

	{
		kubeletDomain, err := newKubeletDomain(azureCluster)
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}

		commonCluster.Kubernetes.Kubelet.AltNames = r.viper.GetString(r.flag.Service.Cluster.Kubernetes.Kubelet.AltNames)
		commonCluster.Kubernetes.Kubelet.Domain = kubeletDomain
		commonCluster.Kubernetes.Kubelet.Labels = r.viper.GetString(r.flag.Service.Cluster.Kubernetes.Kubelet.Labels)
		commonCluster.Kubernetes.Kubelet.Port = r.viper.GetInt(r.flag.Service.Cluster.Kubernetes.Kubelet.Port)
	}

	{
		userList, err := newSpecClusterKubernetesSSHUsers(r.viper.GetString(r.flag.Service.Cluster.Kubernetes.SSH.UserList))
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}

		commonCluster.Kubernetes.SSH.UserList = userList
	}

	{
		commonCluster.Masters = newSpecClusterMasterNodes()
		commonCluster.Workers = newSpecClusterWorkerNodes()
	}

	{
		commonCluster.Scaling.Max = len(workers)
		commonCluster.Scaling.Min = len(workers)
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

func newSpecClusterWorkerNodes() []providerv1alpha1.ClusterNode {
	var workerNodes []providerv1alpha1.ClusterNode

	for i := 0; i < 3; i++ {
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

func newKubeletDomain(cr capzv1alpha3.AzureCluster) (string, error) {
	splitted := strings.Split(cr.Spec.ControlPlaneEndpoint.Host, ".")
	splitted[0] = certs.WorkerCert.String()
	kubeletDomain := strings.Join(splitted, ".")

	return kubeletDomain, nil
}
