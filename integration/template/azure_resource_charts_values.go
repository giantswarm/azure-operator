package template

var AzureResourceChartValues = `clusterName: ${CLUSTER_NAME}
commonDomain: ${COMMON_DOMAIN_GUEST}
commonDomainResourceGroup: ${COMMON_DOMAIN_RESOURCE_GROUP}
sshUser: "test-user"
sshPublicKey: ${IDRSA_PUB}
azure:
  location: ${AZURE_LOCATION}
  vmSizeMaster: "Standard_A1"
  vmSizeWorker: "Standard_A1"
`
