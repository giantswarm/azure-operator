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
      Update:
        Enabled: ${GUEST_UPDATE_ENABLED}
    Name: ci-azure-operator
    Provider:
      Azure:
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
                  version: ${AZURE_TEMPLATE_URI_VERSION}
      Registry:
        PullSecret:
          DockerConfigJSON: "{\"auths\":{\"quay.io\":{\"auth\":\"${REGISTRY_PULL_SECRET}\"}}}"
`
