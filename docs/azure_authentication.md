# Azure Operator authentication with Azure
Both the Azure operator and Kubernetes need to send authenticated requests to the Azure API. That means we have different processes talking to the Azure API in different ways

- Azure operator running on the control plane worker nodes
- `credentiald` running on the control plane worker nodes
- Kubernetes cloud provider integration running on the control plane master nodes
- Kubernetes cloud provider integration running on the control plane worker nodes
- Kubernetes cloud provider integration running on tenant clusters master nodes
- Kubernetes cloud provider integration running on tenant clusters worker nodes
			
## Azure Operator
This process runs on the worker nodes on the control planes. [When deployed](https://github.com/giantswarm/installations/blob/ad518ba95584685bf37cd9c3cca403115153dc14/ghost/draughtsman-secret-values.yaml#L20-L27), it receives a set of [parameters](https://github.com/giantswarm/azure-operator/blob/d0185b1de2cf6975f8e795a38bb55dfc875ecdeb/main.go#L125-L126) 
such as the `Azure.ClientID`, the `Azure.ClientSecret`, or `Azure.MSI.Enabled` to select whether to enabled [Managed Service Identity](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview).

## Credentiald server
It's using the same credentials than Azure Operator is using.

## Control plane clusters
## Cloud Provider k8s integration on masters
[A new client id is created](https://github.com/giantswarm/giantnetes-terraform/blob/master/docs/installation-guide-azure.md#create-service-principal) when deploying a new control plane cluster, and it's assigned the [Azure's built-in Contributor role](https://docs.microsoft.com/es-es/azure/role-based-access-control/built-in-roles#contributor). 
The credentials are passed in the deployment step as Terraform [variables](https://github.com/giantswarm/giantnetes-terraform/blob/0e5f27fa08b15461097029d50f3ca132845a81d8/platforms/azure/giantnetes/main.tf#L73). 

[The cloud provider integration](https://github.com/giantswarm/giantnetes-terraform/blob/0e5f27fa08b15461097029d50f3ca132845a81d8/templates/master.yaml.tmpl#L257-L265) is configured using a template, that is rendered [using those credentials](https://github.com/giantswarm/giantnetes-terraform/blob/4c8814897a3f571223066ec07641fabd9cc6a78d/templates/files/config/azure.yaml).

They are later stored in [our installations repository](https://github.com/giantswarm/installations/blob/ad518ba95584685bf37cd9c3cca403115153dc14/godsmack/terraform/bootstrap.sh#L27-L34).

## Cloud Provider k8s integration on workers
Same as master nodes.

## Tenant clusters
### Cloud Provider k8s integration on masters
The Azure operator creates the tenant clusters. It decides whether to enabled [Managed Service Identity](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview) on the clusters, 
based on a parameter pased to the operator. 

[When Managed Service Identity is enabled](https://github.com/giantswarm/azure-operator/blob/9bbd4581983fbdbf780586b8861bd9986cb2ef4c/service/controller/v11/resource/instance/template/vmss.json#L275-L288), the VMSS will configure their instances to use the Managed Identity that was generated for them.
This is the identity that the Kubernetes cloud provider integration will use.
This identity is assigned the [Azure's built-in role called `Contributor`](https://docs.microsoft.com/es-es/azure/role-based-access-control/built-in-roles#contributor).

[When Managed Service Identity is disabled](https://github.com/giantswarm/azure-operator/blob/d0185b1de2cf6975f8e795a38bb55dfc875ecdeb/service/controller/v11/templates/ignition/cloud_provider_conf.go#L9-L10), the Kubernetes cloud provider integration will use the same credentials than the Azure operator is using.

### Cloud Provider k8s integration on workers
This works exactly the same than for master nodes. 
Workers will use a different Identity than masters, since they are created using different VMSS.
