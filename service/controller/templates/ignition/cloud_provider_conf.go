package ignition

const CloudProviderConf = `cloud: {{ .EnvironmentName }}
tenantId: {{ .TenantID }}
subscriptionId: {{ .SubscriptionID }}
resourceGroup: {{ .ResourceGroup }}
location: {{ .Location }}
{{- if not .UseManagedIdentityExtension }}
aadClientId: {{ .AADClientID }}
aadClientSecret: {{ .AADClientSecret }}
{{- end}}
cloudProviderBackoff: true
cloudProviderBackoffRetries: 6
cloudProviderBackoffJitter: 1
cloudProviderBackoffDuration: 6
cloudProviderBackoffExponent: 1.5
cloudProviderRateLimit: true
cloudProviderRateLimitQPS: 3
cloudProviderRateLimitBucket: 10
cloudProviderRateLimitQPSWrite: 3
cloudProviderRateLimitBucketWrite: 10
primaryScaleSetName: {{ .PrimaryScaleSetName }}
subnetName: {{ .SubnetName }}
securityGroupName: {{ .SecurityGroupName }}
vnetName: {{ .VnetName }}
vmType: vmss
routeTableName: {{ .RouteTableName }}
useManagedIdentityExtension: {{ .UseManagedIdentityExtension }}
loadBalancerSku: standard
`
