# Running azure-operator Locally

**Note:** This should only be used for testing and development.
A production configuration will be provided later.

This guide explains how to get azure-operator running locally - on minikube, for
example.

All commands are assumed to be run from `examples` directory.

## Cluster Certificates

The easiest way to create certificates is to use the local [cert-operator]
setup. See [this guide][cert-operator-local-setup] for details. The operator and
certificates to be used during azure-operator setup can be installed with:

```bash
git clone https://github.com/giantswarm/cert-operator ./cert-operator

helm \
  install -n cert-operator-lab ./cert-operator/examples/cert-operator-lab-chart/ \
  --set clusterName=my-cluster \
  --set commonDomain=my-common-domain \
  --set imageTag=latest \
  --wait

# here the certificate TPR is created, wait until `kubectl get certificate` returns
# `No resources found.` before running the next command

helm install -n cert-resource-lab ./cert-operator/examples/cert-resource-lab-chart/ \
  --set clusterName=my-cluster \
  --set commonDomain=my-common-domain
  --set commaonDomainResourceGroup=resource-group-name
```

- Note: `clusterName` and `commonDomain` chart values must match the values used
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
docker build -t quay.io/giantswarm/azure-operator:local-lab .

# Optional. Restart running operator after image update.
# Does nothing when the operator is not deployed.
#kubectl delete pod -l app=azure-operator-local
```

## Deploying the lab charts

The lab consist of two Helm charts, `azure-operator-lab-chart`, which sets up azure-operator,
and `azure-resource-lab-chart`, which defines the cluster to be created.

With a working Helm installation they can be created from the `examples` dir with:

```bash
$ helm install -n azure-operator-lab ./azure-operator-lab-chart/ --wait
$ helm install -n azure-resource-lab ./azure-resource-lab-chart/ --wait
```

`azure-operator-lab-chart` accepts the following configuration parameters:
* `azure.clientId` - Azure client ID.
* `azure.clientSecret` - Azure client secret.
* `azure.subscriptionId` - Azure subscription ID.
* `azure.tenantId` - Azure tenant ID.
* `imageTag` - Tag of the azure-operator image to be used, by default `local-lab` to use a locally created image

For instance, to pass your default ssh public key to the install command, along with Azure credentials from the environment, you could do:

```bash
$ helm install -n azure-operator-lab --set azure.clientId=${AZURE_CLIENT_ID} \
                                   --set azure.clientSecret=${AZURE_CLIENT_SECRET} \
                                   --set azure.subscriptionId=${AZURE_SUBSCRIPTION_ID} \
                                   --set azure.tenantId=${AZURE_TENANT_ID} \
                                   --set imageTag=local-lab \
                                   ./azure-operator-lab-chart/ --wait
```

`azure-resource-lab-chart` accepts the following configuration parameters:
* `clusterName` - Cluster's name.
* `commonDomain` - Cluster's etcd and API common domain.
* `encryptionKey` - Used for encrypting etcd storage. See https://github.com/giantswarm/k8scloudconfig/blob/30de6e46e3cd97ad7a6da9005fa00f9dc785b179/v_0_1_0/master_template.go#L2038
* `sshUser` - SSH user created via cloudconfig.
* `sshPublicKey` - SSH user created via cloudconfig.
* `azure.location` - Azure region to launch cluster in.
* `azure.coreOSVersion` - version of CoreOS to use for VMs.
* `azure.vmSizeMaster` - master VM size.
* `azure.vmSizeWorker` - worker VM size.

For instance, to create a SSH user with your current user and default public key.

```bash
$ helm install -n azure-resource-lab --set sshUser="$(whoami)" \
                                  --set sshPublicKey="$(cat ~/.ssh/id_rsa.pub)" \
                                   ./azure-resource-lab-chart/ --wait
```

## Access Logs

```bash
kubectl logs -l app=azure-operator-local
```

## Cleaning Up

First delete the cluster TPO.

```bash
$ helm delete azure-resource-lab --purge
```

Wait for the operator to delete the cluster, you should see a message like
this in the logs.

```{"caller":"github.com/giantswarm/azure-operator/service/resource/resourcegroup/resource.go:244","cluster":"test-cluster","debug":"deleted the resource group in the Azure API","resource":"resourcegroup","time":"17-10-23 08:23:00.396"}
```

Then remove the operator's deployment and configuration.

```bash
$ helm delete azure-operator-lab --purge
```

[cert-operator]: https://github.com/giantswarm/cert-operator
[cert-operator-local-setup]: https://github.com/giantswarm/cert-operator/tree/master/examples/local
