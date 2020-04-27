#!/bin/bash

set -e

installation=$1
cluster=$2

if [ "$installation" == "" ] || [ "$cluster" == "" ]
then
    echo "Usage $0 <installation name> <cluster name>"
    exit 1
fi

read -p "Installation: $installation, cluster: $cluster. Hit enter to continue, or ctrl+c to abort. "

echo -n "Creating kubeconfig for $installation: "
opsctl create kubeconfig -i $installation >/dev/null
echo "OK"
echo -n "Creating kubeconfig for $cluster @ $installation: "
gsctl create kubeconfig --cluster=$cluster --certificate-organizations system:masters -e $installation >/dev/null
echo "OK"

echo -n "Getting master node name: "
master="`kubectl --context=giantswarm-$cluster get no -l role=master|grep -v NAME|awk '{print $1}'`"

# TODO check there is only one line in the above result

if [ "$master" == "" ]
then
    echo "ERR"
    exit 2
fi

echo "OK"

echo -n "Stopping API server: "
opsctl ssh $installation $master --cmd "sudo mv /etc/kubernetes/manifests/k8s-api-server.yaml /root || true" 2>&1 >/dev/null
echo "OK"

echo -n "Waiting 10 seconds to ensure the api server goes down: "
sleep 10
echo "OK"

echo -n "Stopping Kubelet: "
opsctl ssh $installation $master --cmd "sudo systemctl stop k8s-kubelet" 2>&1 >/dev/null
echo "OK"
echo -n "Stopping ETCD: "
opsctl ssh $installation $master --cmd "sudo systemctl stop etcd3" 2>&1 >/dev/null
echo "OK"

tar_filename="etcd-backup-$cluster.tar.gz"
remote_tar_path="/tmp/$tar_filename"
local_tar_path="./$tar_filename"

echo "Copying ETCD tar archive remotely: "
opsctl scp $installation "./$tar_filename" $master:$remote_tar_path 
echo "Archive copied correctly in $master:$remote_tar_path"

echo -n "Clearing etcd directory: "
opsctl ssh $installation $master --cmd "sudo rm -rf /var/lib/etcd/*" 2>&1 >/dev/null
echo "OK"

echo -n "Restoring archive to ETCD directory on $master: "
rm -f $local_tar_path
opsctl ssh $installation $master --cmd "sudo tar -xf $remote_tar_path -C /var/lib/etcd/" 2>&1 >/dev/null
echo "OK"

echo -n "Starting API server: "
opsctl ssh $installation $master --cmd "sudo mv /root/k8s-api-server.yaml /etc/kubernetes/manifests/ || true" 2>&1 >/dev/null
echo "OK"

echo -n "Rebooting node "
opsctl ssh $installation $master --cmd "sudo reboot" 2>&1 >/dev/null
echo "OK"

