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

check_warn () {
  r=$?
  if [ $r -eq 0 ]
  then
    echo "OK"
  else
    echo "WARN"
    if [ "$2" != "" ]
    then
      echo "$2"
    fi
  fi
}

run_cmd_on_server () {
  desc=$1
  master=$2
  cmd=$3
  echo -n "$desc: "
  output="$(opsctl ssh $installation master1 --cmd "ssh -oStrictHostKeyChecking=no $master \"$cmd\"" 2>&1)"
  check_err $? "$output"
}

if [ "$installation" == "" ] || [ "$cluster" == "" ]
then
    echo "Usage $0 <installation name> <cluster name>"
    exit 1
fi

read -p "Installation: $installation, cluster: $cluster. Hit enter to continue, or ctrl+c to abort. "

# check for needed commands
echo "Checking for presence of mandatory commands in the system."
for required in gsctl opsctl az jq
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

echo -n "Creating kubeconfig for $cluster @ $installation: "
output="$(gsctl create kubeconfig --cluster=$cluster --certificate-organizations system:masters -e $installation --ttl 1d 2>&1)"
check_err $? "${output}"

echo -n "Connecting to azure: "
output="$(az account list -o table)"
check_err $? "${output}"

echo "Check if the correct subscription is selected."
echo "$output"

echo ""
echo "Check in the table above if the subscription your cluster is running on is the one with 'IsDefault' as True"
read -p "Hit enter to continue or ctrl+c to abort"

# try with az
echo -n "Getting master node IP address from azure CLI: "
output="$(az vmss nic list -g "${cluster}" --vmss-name "${cluster}-master" 2>&1)"
check_err $? "$output"
master="$(echo "$output" | jq -r '.[0].ipConfigurations[0].privateIpAddress')"
if [ "$master" == "null" ]
then
    echo "Unexpected ip address got from Azure, aborting."
    exit 2
fi

echo "Master node IP is '$master'"

# This step fails if the API server is down.
echo -n "Removing master node from cluster: "
output="$(kubectl --context=giantswarm-$cluster delete node -l role=master 2>&1)"
check_warn $? "$output"

run_cmd_on_server "Stopping API server" $master "sudo mv /etc/kubernetes/manifests/k8s-api-server.yaml /root || true"
run_cmd_on_server "Stopping ETCD" $master "sudo systemctl stop etcd3"

tar_filename="etcd-backup-$cluster.tar.gz"
remote_tar_path="/tmp/$tar_filename"
local_tar_path="./$tar_filename"
rm -f $local_tar_path

run_cmd_on_server "Creating tar archive from the ETCD directory on $master" $master "sudo tar -C /var/lib/etcd -czf $remote_tar_path ."

set -e

echo "Copying ETCD tar archive locally: "
opsctl ssh $installation master1 --cmd "scp $master:$remote_tar_path $remote_tar_path"
opsctl scp $installation master1:$remote_tar_path "$local_tar_path"
echo "Archive copied correctly in $local_tar_path"

set +e

cmd="opsctl update status -i $installation -p apis/provider.giantswarm.io/v1alpha1/namespaces/default/azureconfigs/${cluster}/status"
echo "The backup process is completed."
echo "You now have to set the 'masters' status field to 'DeallocateLegacyInstance'".
echo "Do you want to run '$cmd' now?"
read -p "Press enter to continue, ctrl+c to do it manually."

$cmd
