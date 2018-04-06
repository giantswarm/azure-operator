# Requirements

```
go get -u github.com/giantswarm/e2e-harness
```

## environment variables example
```
export CLUSTER_NAME="test-e2e"
export COMMON_DOMAIN_GUEST="k8s.godsmack.westeurope.azure.gigantic.io"
export COMMON_DOMAIN_GUEST_NO_K8S="godsmack.westeurope.azure.gigantic.io"
export COMMON_DOMAIN_RESOURCE_GROUP="godsmack"
export REGISTRY_PULL_SECRET="xxxxxx"

export IDRSA_PUB=$(cat ~/.ssh/id_rsa.pub)

export CIRCLE_SHA1="latest"

export AZURE_LOCATION="westeurope"
export AZURE_CLIENTID="xxxxxx"
export AZURE_CLIENTSECRET="xxxxxx"
export AZURE_SUBSCRIPTIONID="xxxxxx"
export AZURE_TENANTID="xxxxxx"
export AZURE_CALICO_SUBNET_CIDR="10.42.128.0/17"
export AZURE_CIDR="10.42.0.0/16"
export AZURE_MASTER_SUBNET_CIDR="10.42.0.0/24"
export AZURE_WORKER_SUBNET_CIDR="10.42.1.0/24"
export AZURE_TEMPLATE_URI_VERSION="master"
```

# How to run integration test

```
$ minikube start --extra-config=apiserver.Authorization.Mode=RBAC
$ e2e-harness setup --remote=false
$ e2e-harness test --test-dir=integration/test/mytest
```
