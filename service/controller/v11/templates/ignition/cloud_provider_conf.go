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
primaryScaleSetName: {{ .PrimaryScaleSetName }}
subnetName: {{ .SubnetName }}
securityGroupName: {{ .SecurityGroupName }}
vnetName: {{ .VnetName }}
vmType: vmss
routeTableName: {{ .RouteTableName }}
useManagedIdentityExtension: {{ .UseManagedIdentityExtension }}
loadBalancerSku: standard
`
