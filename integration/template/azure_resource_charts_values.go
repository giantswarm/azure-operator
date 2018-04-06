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
  calicoSubnetCIDR: "10.25.128.0/17"
  cidr: "10.25.0.0/16"
  masterSubnetCIDR: "10.25.0.0/24"
  workerSubnetCIDR: "10.25.1.0/24"
`
