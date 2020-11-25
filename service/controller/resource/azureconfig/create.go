package azureconfig

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v4/pkg/controller/context/reconciliationcanceledcontext"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	dockerVolumeSizeGB  = 50
	kubeletVolumeSizeGB = 100
	// DNS domain for dns searches in pods.
	kubeletClusterDomain = "cluster.local"
	kubeDNSIPLastOctet   = 10

	ProviderAzure = "azure"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	azureCluster, err := key.ToAzureCluster(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding required cluster api types")

	var cluster capiv1alpha3.Cluster
	{
		nsName := types.NamespacedName{
			Name:      key.ClusterName(&azureCluster),
			Namespace: azureCluster.Namespace,
		}
		err = r.ctrlClient.Get(ctx, nsName, &cluster)
		if errors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("referenced Cluster CR (%q) not found", nsName.String()))
			r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
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
				client.MatchingLabels{capiv1alpha3.ClusterLabelName: key.ClusterName(&cluster)},
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
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no control plane AzureMachines found for cluster %q", key.ClusterName(&cluster)))
			r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "found required cluster api types")
	r.logger.LogCtx(ctx, "level", "debug", "message", "building azureconfig from cluster api crs")

	var mappedAzureConfig providerv1alpha1.AzureConfig
	{
		mappedAzureConfig, err = r.buildAzureConfig(ctx, cluster, azureCluster, masterMachines, workerMachines)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "built azureconfig from cluster api crs")
	r.logger.LogCtx(ctx, "level", "debug", "message", "finding existing azureconfig")

	var presentAzureConfig providerv1alpha1.AzureConfig
	{
		nsName := types.NamespacedName{
			Name:      key.ClusterName(&cluster),
			Namespace: metav1.NamespaceDefault,
		}
		err = r.ctrlClient.Get(ctx, nsName, &presentAzureConfig)
		if errors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not found existing azureconfig")
			r.logger.LogCtx(ctx, "level", "debug", "message", "creating azureconfig")
			err = r.ctrlClient.Create(ctx, &mappedAzureConfig)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "created azureconfig")
			presentAzureConfig = mappedAzureConfig
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding if existing azureconfig needs update")
	{
		// Ensure that present network allocations are kept as-is.
		mappedAzureConfig.Spec.Azure.VirtualNetwork = presentAzureConfig.Spec.Azure.VirtualNetwork

		// Were there any changes that requires CR update?
		changed := false
		if !azureConfigsEqual(mappedAzureConfig, presentAzureConfig) {
			// Copy Spec section as-is. This should always match desired state.
			presentAzureConfig.Spec = mappedAzureConfig.Spec
			changed = true
		}

		// Copy mapped labels if missing or changed, but don't touch labels
		// that we don't manage.
		for k, v := range mappedAzureConfig.Labels {
			old, exists := presentAzureConfig.Labels[k]
			if old != v || !exists {
				presentAzureConfig.Labels[k] = v
				changed = true
			}
		}

		if changed {
			r.logger.LogCtx(ctx, "level", "debug", "message", "existing azureconfig needs update")

			err = r.ctrlClient.Update(ctx, &presentAzureConfig)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "existing azureconfig updated")
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "no update for existing azureconfig needed")
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding if existing azureconfig needs status update")
	{
		nsName := types.NamespacedName{
			Name:      key.ClusterName(&azureCluster),
			Namespace: metav1.NamespaceDefault,
		}
		err = r.ctrlClient.Get(ctx, nsName, &presentAzureConfig)
		if err != nil {
			return microerror.Mask(err)
		}

		updateStatus := false
		if len(presentAzureConfig.Status.Cluster.Conditions) == 0 {
			c := providerv1alpha1.StatusClusterCondition{
				Status: "True",
				Type:   "Creating",
			}
			presentAzureConfig.Status.Cluster.Conditions = append(presentAzureConfig.Status.Cluster.Conditions, c)
			r.logger.LogCtx(ctx, "level", "debug", "message", "cluster condition status needs update")
			updateStatus = true
		}

		if len(presentAzureConfig.Status.Cluster.Versions) == 0 {
			v := providerv1alpha1.StatusClusterVersion{
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Semver:             key.OperatorVersion(&presentAzureConfig),
			}
			presentAzureConfig.Status.Cluster.Versions = append(presentAzureConfig.Status.Cluster.Versions, v)
			r.logger.LogCtx(ctx, "level", "debug", "message", "cluster version status needs update")
			updateStatus = true
		}

		if updateStatus {
			err = r.ctrlClient.Status().Update(ctx, &presentAzureConfig)
			if err != nil {
				return microerror.Mask(err)
			}
			r.logger.LogCtx(ctx, "level", "debug", "message", "status updated")
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "no status update for existing azureconfig needed")
		}
	}

	return nil
}

func (r *Resource) buildAzureConfig(ctx context.Context, cluster capiv1alpha3.Cluster, azureCluster capzv1alpha3.AzureCluster, masters, workers []capzv1alpha3.AzureMachine) (providerv1alpha1.AzureConfig, error) {
	var err error

	azureConfig := providerv1alpha1.AzureConfig{}
	azureConfig.Labels = make(map[string]string)
	azureConfig.Annotations = make(map[string]string)

	{
		azureConfig.ObjectMeta.Name = key.ClusterName(&cluster)
		azureConfig.ObjectMeta.Namespace = metav1.NamespaceDefault
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
		azureConfig.Annotations[annotation.ClusterDescription] = cluster.Annotations[annotation.ClusterDescription]
	}

	{
		azureConfig.Labels[label.Cluster] = key.ClusterName(&cluster)
		azureConfig.Labels[capiv1alpha3.ClusterLabelName] = key.ClusterName(&cluster)
		azureConfig.Labels[label.Organization] = key.OrganizationID(&cluster)
		azureConfig.Labels[label.ReleaseVersion] = key.ReleaseVersion(&cluster)
		azureConfig.Labels[label.OperatorVersion] = key.OperatorVersion(&azureCluster)
	}

	{
		azureConfig.Spec.Azure.AvailabilityZones, err = getAvailabilityZones(masters, workers)
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
		hostResourceGroup = r.managementClusterResourceGroup
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
		credentialSecret, err := r.getCredentialSecret(ctx, cluster.ObjectMeta)
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
		commonCluster.Calico.CIDR = r.calico.CIDRSize
		commonCluster.Calico.MTU = r.calico.MTU
		commonCluster.Calico.Subnet = r.calico.Subnet
	}

	{
		commonCluster.ID = key.ClusterName(&cluster)
	}

	{
		commonCluster.Customer.ID = key.OrganizationID(&cluster)
	}

	{
		etcdServerDomain, err := newEtcdServerDomain(azureCluster)
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}

		commonCluster.Etcd.Domain = etcdServerDomain
		commonCluster.Etcd.Prefix = r.etcdPrefix
		commonCluster.Etcd.Port = 2379
	}

	{
		apiServerDomain, err := newAPIServerDomain(azureCluster)
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
		kubeletDomain, err := newKubeletDomain(azureCluster)
		if err != nil {
			return providerv1alpha1.Cluster{}, microerror.Mask(err)
		}

		// We ingore the error here. It happens because AzureCluster or AzureConfig don't know about MachinePoolID.
		kubeletLabels, _ := key.KubeletLabelsNodePool(&azureCluster)

		commonCluster.Kubernetes.Kubelet.Domain = kubeletDomain
		commonCluster.Kubernetes.Kubelet.Labels = kubeletLabels
	}

	{
		commonCluster.Kubernetes.Domain = kubeletClusterDomain
	}

	{
		// The AzureConfig field containing the list of SSH keys is not used any more,
		// but is a mandatory field so we set it to an empty slice.
		commonCluster.Kubernetes.SSH.UserList = []providerv1alpha1.ClusterKubernetesSSHUser{}
	}

	{
		commonCluster.Masters = newSpecClusterMasterNodes()
		commonCluster.Workers = newSpecClusterWorkerNodes(len(workers))
	}

	{
		commonCluster.Scaling.Max = len(workers)
		commonCluster.Scaling.Min = len(workers)
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
