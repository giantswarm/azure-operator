package template

var AzureResourceChartValues = `clusterName: ${CLUSTER_NAME}
commonDomain: ${COMMON_DOMAIN_GUEST_NO_K8S}
commonDomainResourceGroup: ${COMMON_DOMAIN_RESOURCE_GROUP}
azure:
  location: ${AZURE_LOCATION}
  vmSizeMaster: "Standard_DS1_v2"
  vmSizeWorker: "Standard_DS1_v2"
  calicoSubnetCIDR: ${AZURE_CALICO_SUBNET_CIDR}
  cidr: ${AZURE_CIDR}
  masterSubnetCIDR: ${AZURE_MASTER_SUBNET_CIDR}
  workerSubnetCIDR: ${AZURE_WORKER_SUBNET_CIDR}
`
