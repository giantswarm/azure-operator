package template

var AzureConfigE2EChartValues = `
azure:
  calicoSubnetCIDR: ${AZURE_CALICO_SUBNET_CIDR}
  cidr: ${AZURE_CIDR}
  location: ${AZURE_LOCATION}
  masterSubnetCIDR: ${AZURE_MASTER_SUBNET_CIDR}
  vmSizeMaster: "Standard_D2s_v3"
  vmSizeWorker: "Standard_D2s_v3"
  workerSubnetCIDR: ${AZURE_WORKER_SUBNET_CIDR}
  vpnSubnetCIDR: ${AZURE_VPN_SUBNET_CIDR}
clusterName: ${CLUSTER_NAME}
commonDomain: ${COMMON_DOMAIN}
commonDomainResourceGroup: ${COMMON_DOMAIN_RESOURCE_GROUP}
versionBundleVersion: ${VERSION_BUNDLE_VERSION}
`
