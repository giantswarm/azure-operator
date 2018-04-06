package template

var AzureResourceChartValues = `clusterName: ${CLUSTER_NAME}
commonDomain: ${COMMON_DOMAIN_GUEST_NO_K8S}
commonDomainResourceGroup: ${COMMON_DOMAIN_RESOURCE_GROUP}
sshUser: "test-user"
sshPublicKey: ${IDRSA_PUB}
azure:
  location: ${AZURE_LOCATION}
  vmSizeMaster: "Standard_D2s_v3"
  vmSizeWorker: "Standard_D2s_v3"
  calicoSubnetCIDR: ${AZURE_CALICO_SUBNET_CIDR}
  cidr: ${AZURE_CIDR}
  masterSubnetCIDR: ${AZURE_MASTER_SUBNET_CIDR}
  workerSubnetCIDR: ${AZURE_WORKER_SUBNET_CIDR}
`
