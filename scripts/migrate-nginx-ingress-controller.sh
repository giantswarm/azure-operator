#!/bin/bash

installation=$1
cluster=$2

check_err () {
  r=$?
  if [ $r -eq 0 ]
  then
    echo "OK"
  else
    echo "ERR"
    if [ "$2" != "" ]
    then
      echo "$2"
    fi
    exit 1
  fi
}

if [ "$installation" == "" ] || [ "$cluster" == "" ]
then
    echo "Usage $0 <installation name> <cluster name>"
    exit 1
fi

# check for needed commands
echo "Checking for presence of mandatory commands in the system."
for required in gsctl kubectl opsctl jq dig
do
  echo -n "Looking for command $required: "
  if [ ! -x "$(command -v $required)" ]
  then
    echo "ERR"
    echo "The required command $required was not found in your system. Aborting."
    exit 1
  fi

  echo "OK"
done

read -p "Installation: $installation, cluster: $cluster. Hit enter to continue, or ctrl+c to abort. "

echo -n "Creating kubeconfig for $installation: "
output="$(opsctl create kubeconfig -i $installation 2>&1)"
check_err $? "${output}"

echo -n "Creating kubeconfig for $cluster @ $installation: "
output="$(gsctl create kubeconfig --cluster=$cluster --certificate-organizations system:masters -e $installation --ttl 1d 2>&1)"
check_err $? "${output}"

# 1) Deploy the new nginx ingress app (1.6.10) with the following custom user config:

echo "Installing the nginx ingress controller app"

appname="nginx-ingress-controller-app"
appversion="1.6.10-a0c46e931fd99bd75eb7f3677b9c5ccd74e09598"

read -r -d '' APP << EOM
apiVersion: v1
data:
  values: |
    controller:
      name: ${appname}
      configmap:
        name: ${appname}
      role:
        name: ${appname}
      service:
        enabled: false
    ingressController:
      legacy: false
    provider: azure
kind: ConfigMap
metadata:
  name: ${appname}-user-values
  namespace: ${cluster}
---
apiVersion: application.giantswarm.io/v1alpha1
kind: App
metadata:
  annotations:
    chart-operator.giantswarm.io/force-helm-upgrade: "false"
  labels:
    app: ${appname}
    app-operator.giantswarm.io/version: 1.0.0
    giantswarm.io/cluster: ${cluster}
    giantswarm.io/organization: giantswarm
    giantswarm.io/service-type: managed
  name: ${appname}
  namespace: ${cluster}
spec:
  catalog: default-test
  config:
    configMap:
      name: ingress-controller-values
      namespace: ${cluster}
    secret:
      name: ""
      namespace: ""
  kubeConfig:
    context:
      name: ${cluster}
    inCluster: false
    secret:
      name: ${cluster}-kubeconfig
      namespace: ${cluster}
  name: ${appname}
  namespace: kube-system
  userConfig:
    configMap:
      name: "${appname}-user-values"
      namespace: "${cluster}"
    secret:
      name: ""
      namespace: ""
  version: "${appversion}"
EOM

echo "$APP" | kubectl --context=giantswarm-${installation} apply -f -

echo "Waiting for ${appname} chart to be deployed"

status=""
while [ "$status" != "DEPLOYED" ]
do
  if [ "$status" != "" ]
  then
    echo "Expecting the state to be DEPLOYED but it is $status"
  fi
  status=$(kubectl --context=giantswarm-${cluster} --namespace=giantswarm get chart ${appname} -o json |jq -r '.status.release.status')
  sleep 5
done

echo "Chart is deployed"
echo ""

# 2) Wait for the new load balancer to come up, communicate the public IP address to the customer

echo "Waiting for LoadBalancer to have a public IP"

ip=""
while [ "$ip" == "" ] || [ "$ip" == "null" ]
do
  ip="$(kubectl --context=giantswarm-${cluster} --namespace=kube-system get svc nginx-ingress-controller-app -o json| jq -r '.status.loadBalancer.ingress[0].ip')"
  sleep 5
done

read -p "IP is $ip. Ping the customer to inform him about the public IP change. Click enter to continue or ctrl+c to abort."

# 3) kubectl --context=giantswarm-ami5q --namespace=kube-system edit svc ingress-loadbalancer
echo -n "Removing external dns annotation from ingress-loadbalancer svc: "
output="$(kubectl --context=giantswarm-${cluster} --namespace=kube-system annotate svc ingress-loadbalancer 'external-dns.alpha.kubernetes.io/hostname'- 2>&1)"
check_err $? "$output"

# 4) wait until the external-dns updates the value of the ingress. ... DNS record.

hostname="$(kubectl --context=giantswarm-godsmack get azureconfig ${cluster} -o json |jq -r '.spec.cluster.kubernetes.ingressController.domain')"

if [ "$hostname" == "null" ]
then
  echo "Unable to get ingress hostname, aborting"
  exit 1
fi

echo "Hostname is $hostname"

echo "Checking DNS resolution"

res=""
while [ "$res" != "$ip" ]
do
  if [ "$res" != "" ]
  then
    echo "Expected $hostname to resolve to $ip but it resolves to $res"
  fi
  res="$(dig +short $hostname @8.8.8.8)"
  sleep 5
done

echo "$hostname resolves as $res as expected"
echo ""

# 5) upgrade the cluster

echo "You can now upgrade the cluster"
