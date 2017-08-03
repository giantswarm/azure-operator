# Running azure-operator Locally

**Note:** This should only be used for testing and development.

This guide explains how to get azure-operator running locally - on minikube, for
example. 

All commands are assumed to be run from `examples/local` directory.

## Preparing Templates

All yaml files in this directory are templates. Before continuing with this
guide, all placeholders must be replaced with sensible values.

- *CLUSTER_NAME* - Cluster's name.
- *COMMON_DOMAIN* - Cluster's etcd and API common domain.
- *COMMON_DOMAIN_INGRESS* - Ingress common domain.
- *AZURE_LOCATION* - Azure location.

This is a handy snippet that makes it painless - works in bash and zsh.

```bash
export CLUSTER_NAME="example-cluster"
export COMMON_DOMAIN="internal.company.com"
export COMMON_DOMAIN_INGRESS="company.com"
export AZURE_LOCATION="westeurope"

for f in *.tmpl.yaml; do
    sed \
        -e 's|${CLUSTER_NAME}|'"${CLUSTER_NAME}"'|g' \
        -e 's|${COMMON_DOMAIN}|'"${COMMON_DOMAIN}"'|g' \
        -e 's|${COMMON_DOMAIN_INGRESS}|'"${COMMON_DOMAIN_INGRESS}"'|g' \
        -e 's|${AZURE_LOCATION}|'"${AZURE_LOCATION}"'|g' \
        ./$f > ./${f%.tmpl.yaml}.yaml
done
```

- Note: `|` characters are used in `sed` substitution to avoid escaping.

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
