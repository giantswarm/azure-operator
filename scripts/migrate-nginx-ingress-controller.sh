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

# 1) Ensure the new load balancer is up, communicate the public IP address to the customer

echo "Waiting for LoadBalancer to have a public IP"

ip=""
while [ "$ip" == "" ] || [ "$ip" == "null" ]
do
  ip="$(kubectl --context=giantswarm-${cluster} --namespace=kube-system get svc nginx-ingress-controller -o json| jq -r '.status.loadBalancer.ingress[0].ip')"
  sleep 5
done

read -p "IP is $ip. Ping the customer to inform him about the public IP change. Click enter to continue or ctrl+c to abort."

# 2) Switch DNS records to the new load balancer IP, to allow old LoadBalancer Service from being deleted
echo -n "Removing external dns annotation from ingress-loadbalancer svc: "
output="$(kubectl --context=giantswarm-${cluster} --namespace=kube-system annotate svc ingress-loadbalancer 'external-dns.alpha.kubernetes.io/hostname'- 2>&1)"
check_err $? "$output"

# 3) wait until the external-dns updates the value of the ingress. ... DNS record.

hostname="$(kubectl --context=giantswarm-${installation} get azureconfig ${cluster} -o json |jq -r '.spec.cluster.kubernetes.ingressController.domain')"

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
