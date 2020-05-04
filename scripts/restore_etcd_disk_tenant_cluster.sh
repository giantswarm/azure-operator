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
output="$(az vmss nic list -g "${cluster}" --vmss-name "${cluster}-master-${cluster}" 2>&1)"
check_err $? "$output"
master="$(echo "$output" | jq -r '.[0].ipConfigurations[0].privateIpAddress')"
if [ "$master" == "null" ]
then
    echo "Unexpected ip address got from Azure, aborting."
    exit 2
fi

echo "Master node IP is '$master'"

run_cmd_on_server "Stopping API server" $master "sudo mv /etc/kubernetes/manifests/k8s-api-server.yaml /root || true"

# check if kubernetes api server is down
chk="kubectl --request-timeout=10s --context=giantswarm-$cluster get no"
a=$($chk 2>&1)
status=$?
while [ $status -eq 0 ]
do
  a=$($chk 2>&1)
  status=$?
  echo "API server is still UP"
  sleep 1
done
echo "API server is down"

run_cmd_on_server "Stopping Kubelet" $master "sudo systemctl stop k8s-kubelet"
run_cmd_on_server "Stopping ETCD" $master "sudo systemctl stop etcd3"

tar_filename="etcd-backup-$cluster.tar.gz"
remote_tar_path="/tmp/$tar_filename"
local_tar_path="./$tar_filename"

set -e

echo "Copying ETCD tar archive remotely: "
opsctl scp $installation "$local_tar_path" master1:$remote_tar_path
opsctl ssh $installation master1 --cmd "scp -oStrictHostKeyChecking=no $remote_tar_path $master:$remote_tar_path"
echo "Archive copied correctly in $master:$remote_tar_path"

run_cmd_on_server "Clearing etcd directory" $master "sudo rm -rf /var/lib/etcd/*"
run_cmd_on_server "Restoring archive to ETCD directory on $master" $master "sudo tar -xf $remote_tar_path -C /var/lib/etcd/"
run_cmd_on_server "Starting API server" $master "sudo mv /root/k8s-api-server.yaml /etc/kubernetes/manifests/ || true"

set +e

echo -n "Rebooting node: "
output="$(opsctl ssh $installation master1 --cmd "ssh -oStrictHostKeyChecking=no $master \"sudo reboot\"" 2>&1)"
check_warn $? "$output"

cmd="opsctl update status -i $installation -p apis/provider.giantswarm.io/v1alpha1/namespaces/default/azureconfigs/${cluster}/status"
echo "The restore process is completed."
echo "You now have to set the 'masters' status field to 'DeleteLegacyVMSS'".
echo "Do you want to run '$cmd' now?"
read -p "Press enter to continue, ctrl+c to do it manually."

$cmd
