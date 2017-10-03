# Running azure-operator Locally

**Note:** This should only be used for testing and development.
A production configuration using Helm will be provided later.

This guide explains how to get azure-operator running locally - on minikube, for
example.

All commands are assumed to be run from `examples/local` directory.

## Preparing Templates

All yaml files in this directory are templates. Before continuing with this
guide, all placeholders must be replaced with sensible values.

- *CLUSTER_NAME* - Cluster's name.
- *COMMON_DOMAIN* - Cluster's etcd and API common domain.
- *COMMON_DOMAIN_INGRESS* - Ingress common domain.
- *AZURE_INSTANCE_TYPE_MASTER* - Master machines instance type.
- *AZURE_LOCATION* - Azure location.
- *AZURE_CLIENT_ID* - Client ID for the Active Directory Service Principal.
- *AZURE_CLIENT_SECRET* - Client Secret for the Active Directory Service Principal.
- *AZURE_SUBSCRIPTION_ID* - Azure Subscription ID.
- *AZURE_TENANT_ID* - Azure Active Directory Tenant ID.
- *AZURE_TEMPLATE_URI_VERSION* - Deploy templates pushed to a feature branch.
- *ID_RSA_PUB* - SSH public key to be installed on nodes. Has to be formated like "ssh-rsa AAAAB3NzaC1y user@location"

This is a handy snippet that makes it painless - works in bash and zsh.

```bash
export CLUSTER_NAME="example-cluster"
export COMMON_DOMAIN="internal.company.com"
export COMMON_DOMAIN_INGRESS="company.com"
export AZURE_INSTANCE_TYPE_MASTER="Standard_A1"
export AZURE_LOCATION="westeurope"
export AZURE_CLIENT_ID="XXXXX"
export AZURE_CLIENT_SECRET="XXXXX"
export AZURE_SUBSCRIPTION_ID="XXXXX"
export AZURE_TENANT_ID="XXXXX"
export AZURE_TEMPLATE_URI_VERSION="master"
export ID_RSA_PUB="ssh-rsa AAAAB3NzaC1y user@location"

for f in *.tmpl.yaml; do
    sed \
        -e 's|${CLUSTER_NAME}|'"${CLUSTER_NAME}"'|g' \
        -e 's|${COMMON_DOMAIN}|'"${COMMON_DOMAIN}"'|g' \
        -e 's|${COMMON_DOMAIN_INGRESS}|'"${COMMON_DOMAIN_INGRESS}"'|g' \
        -e 's|${AZURE_LOCATION}|'"${AZURE_LOCATION}"'|g' \
        -e 's|${AZURE_CLIENT_ID}|'"${AZURE_CLIENT_ID}"'|g' \
        -e 's|${AZURE_CLIENT_SECRET}|'"${AZURE_CLIENT_SECRET}"'|g' \
        -e 's|${AZURE_SUBSCRIPTION_ID}|'"${AZURE_SUBSCRIPTION_ID}"'|g' \
        -e 's|${AZURE_TENANT_ID}|'"${AZURE_TENANT_ID}"'|g' \
        -e 's|${AZURE_TEMPLATE_URI_VERSION}|'"${AZURE_TEMPLATE_URI_VERSION}"'|g' \
        ./$f > ./${f%.tmpl.yaml}.yaml
done
```

- Note: `|` characters are used in `sed` substitution to avoid escaping.

## Cluster Certificates

The easiest way to create certificates is to use the local [cert-operator]
setup. See [this guide][cert-operator-local-setup] for details.

- Note: `CLUSTER_NAME` and `COMMON_DOMAIN` values must match the values used
  during this guide.

## Cluster-Local Docker Image

The operator needs a connection to the K8s API. The simplest approach is to run
the operator as a deployment and use the "in cluster" configuration.

In that case the Docker image needs to be accessible from the K8s cluster
running the operator. For Minikube run `eval $(minikube docker-env)` before
`docker build`, see [reusing the Docker daemon] for details.

[reusing the docker daemon]: https://github.com/kubernetes/minikube/blob/master/docs/reusing_the_docker_daemon.md

```bash
# Optional. Only when using Minikube.
eval $(minikube docker-env)

# From the root of the project, where the Dockerfile resides
GOOS=linux go build github.com/giantswarm/azure-operator
docker build -t quay.io/giantswarm/azure-operator:local-dev .

# Optional. Restart running operator after image update.
# Does nothing when the operator is not deployed.
#kubectl delete pod -l app=azure-operator-local
```

## Operator Startup

Create the operator config map and deployment.

```bash
kubectl apply -f ./configmap.yaml
kubectl apply -f ./deployment.yaml
```

## Creating A New Cluster

Create a new cluster ThirdPartyObject.

```bash
kubectl create -f ./cluster.yaml
```

## Access Logs

```bash
kubectl logs -l app=azure-operator-local
```

## Cleaning Up

First delete the cluster TPO.

```bash
export CLUSTER_NAME="example-cluster"

kubectl delete azurecluster ${CLUSTER_NAME}
```

Wait for the operator to delete the cluster, and then remove the operator's
deployment and configuration.

```bash
kubectl delete -f ./deployment.yaml
kubectl delete -f ./configmap.yaml
```

[cert-operator]: https://github.com/giantswarm/cert-operator
[cert-operator-local-setup]: https://github.com/giantswarm/cert-operator/tree/master/examples/local
