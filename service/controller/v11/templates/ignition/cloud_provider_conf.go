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
