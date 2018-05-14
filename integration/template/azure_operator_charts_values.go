package template

// AzureOperatorChartValues values required by aws-operator-chart, the environment
// variables will be expanded before writing the contents to a file.
var AzureOperatorChartValues = `Installation:
  V1:
    Guest:
      Kubernetes:
        API:
          Auth:
            Provider:
              OIDC:
                ClientID: ""
                IssueURL: ""
                UsernameClaim: ""
                GroupsClaim: ""
    Name: ci-azure-operator
    Provider:
      Azure:
        Cloud: AZUREPUBLICCLOUD
        HostCluster:
          CIDR: "10.0.0.0/16"
          ResourceGroup: "godsmack"
          VirtualNetwork: "godsmack"
        Location: ${AZURE_LOCATION}
    Secret:
      AzureOperator:
        SecretYaml: |
          service:
            azure:
              clientid: ${AZURE_CLIENTID}
              clientsecret: ${AZURE_CLIENTSECRET}
              subscriptionid: ${AZURE_SUBSCRIPTIONID}
              tenantid: ${AZURE_TENANTID}
              template:
                uri:
                  version: ${CIRCLE_SHA1}
      Registry:
        PullSecret:
          DockerConfigJSON: "{\"auths\":{\"quay.io\":{\"auth\":\"${REGISTRY_PULL_SECRET}\"}}}"
`
