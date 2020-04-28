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

read -p "Installation: $installation, cluster: $cluster. Hit enter to continue, or ctrl+c to abort. "

# check for needed commands
echo "Checking for presence of mandatory commands in the system."
for required in gsctl opsctl
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

echo -n "Creating kubeconfig for $installation: "
opsctl create kubeconfig -i $installation >/dev/null
check_err $?
echo -n "Creating kubeconfig for $cluster @ $installation: "
output="$(gsctl create kubeconfig --cluster=$cluster --certificate-organizations system:masters -e $installation --ttl 1d 2>&1)"
check_err $? "${output}"

echo -n "Getting master node name from kubernetes API: "
# try with kubectl
master=""
output="$(kubectl --context=giantswarm-$cluster get no -l role=master 2>&1)"
if [ $? -eq 0 ]
then
  master="$(echo "${output}"|grep -v NAME|awk '{print $1}')"
  if [ $? -ne 0 ] || [ "$master" == "" ]
  then
    echo "WARN"
    echo "Failed getting master node name from kubernetes API, trying with azure CLI client."
  else
    echo "OK"
  fi
else
  echo "WARN"
  echo "${output}"
fi

if [ "$master" == "" ]
then
  echo -n "Checking if 'az' is available: "
  if [ -x "$(command -v az)" ]
  then
    echo "OK"
    echo -n "Connecting to azure: "
    output="$(az account list -o table)"
    check_err $? "${output}"

    echo "Check if the correct subscription is selected."
    echo "$output"

    echo ""
    echo "Check in the table above if the subscription your cluster is running on is the one with 'IsDefault' as True"
    read -p "Hit enter to continue or ctrl+c to abort"
  else
    echo "ERR"
    echo "The az command was not found in your system, can't get master node hostname."
    exit 1
  fi

  # try with az
  echo -n "Getting master node name from azure CLI: "
  output="$(az vmss list-instances -g "${cluster}" --name "${cluster}-master" 2>&1)"
  check_err $? "$output"
  master="$(echo "$output" | jq -r '.[0].osProfile.computerName')"
  if [ "$master" == "null" ]
  then
      echo "Unexpected server name got from Azure, aborting."
      exit 2
  fi
fi

echo "Master node name is '$master'"

echo -n "Stopping API server: "
output="$(opsctl ssh $installation $master --cmd "sudo mv /etc/kubernetes/manifests/k8s-api-server.yaml /root || true" 2>&1)"
check_err $? "$output"
echo -n "Stopping ETCD: "
output="$(opsctl ssh $installation $master --cmd "sudo systemctl stop etcd3" 2>&1)"
check_err $? "$output"

tar_filename="etcd-backup-$cluster.tar.gz"
remote_tar_path="/tmp/$tar_filename"
local_tar_path="./$tar_filename"

echo -n "Creating tar archive from the ETCD directory on $master: "
rm -f $local_tar_path
output="$(opsctl ssh $installation $master --cmd "sudo tar -C /var/lib/etcd -czf $remote_tar_path ." 2>&1)"
check_err $? "$output"

set -e

echo "Copying ETCD tar archive locally: "
opsctl scp $installation $master:$remote_tar_path "./$tar_filename"
echo "Archive copied correctly in $local_tar_path"

cmd="opsctl update status -i $installation -p apis/provider.giantswarm.io/v1alpha1/namespaces/default/azureconfigs/${cluster}/status"
echo "The restore process is completed."
echo "You now have to set the 'masters' status field to 'DeallocateLegacyInstance'".
echo "Do you want to run '$cmd' now?"
read -p "Press enter to continue, ctrl+c to do it manually."

$cmd
