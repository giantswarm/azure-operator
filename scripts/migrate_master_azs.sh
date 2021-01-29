#!/bin/bash

set -e

installation=$1
cluster=$2

read -p "Installation: $installation, Cluster: $cluster.
Press enter to continue, ctrl+c to abort"

opsctl create kubeconfig -i $installation
gsctl create kubeconfig --cluster=$cluster --certificate-organizations system:masters -e $installation

# get org name
org_namespace="$(kubectl --context=giantswarm-$installation -n org-giantswarm get cluster -l giantswarm.io/cluster=${cluster} -o yaml -A|yq -r '.items[0].metadata.namespace')"
org="$(kubectl --context=giantswarm-$installation -n org-giantswarm get cluster -l giantswarm.io/cluster=${cluster} -o yaml -A|yq -r '.items[0].metadata.labels["giantswarm.io/organization"]')"

# get azure operator version
azure_operator_version="$(kubectl --context=giantswarm-$installation -n ${org_namespace} get cluster ${cluster} -o yaml| yq -r '.metadata.labels["azure-operator.giantswarm.io/version"]')"

# find subscription ID
subscription="$(kubectl --context=giantswarm-$installation -n giantswarm get secret -l giantswarm.io/organization="${org}" -o yaml | yq -r '.items[0].data["azure.azureoperator.subscriptionid"]'| base64 -d)"

master_ip="$(kubectl --context=giantswarm-$cluster get no $cluster-master-$cluster-000000 -o yaml| yq -r '.status.addresses[] | select(.type == "InternalIP").address')"

bastion_name="$(opsctl show installation -i $installation| yq -r .Jumphosts.bastion1.Host)"

api_url="$(kubectl config view |yq -r ".clusters[] | select(.name == \"giantswarm-${cluster}\").cluster.server")"

echo "org: ${org}"
echo "org namespace: ${org_namespace}"
echo "version: ${azure_operator_version}"
echo "subscription: ${subscription}"
echo "master_ip: ${master_ip}"
echo "bastion: ${bastion_name}"
echo "API url: ${api_url}"
read -p "Press enter to continue, ctrl+c to abort"

az account set -s $subscription

# check group is visible
az group show --resource-group $cluster

# uninstall admission controller
kubectl --context=giantswarm-$installation -n giantswarm delete app azure-admission-controller-unique || true

# stop azure operator
kubectl --context=giantswarm-$installation -n giantswarm scale deploy -l app.kubernetes.io/instance=azure-operator-${azure_operator_version} --replicas=0

# update azuremachine
kubectl --context=giantswarm-$installation -n ${org_namespace} patch azuremachine ${cluster}-master-0 --patch '{"spec": { "failureDomain": "1" }}' --type merge

# update azureconfig
kubectl --context=giantswarm-$installation patch azureconfig ${cluster} --patch '{"spec": { "azure": {"availabilityZones": [1]}}}' --type merge

# install admission controller
opsctl deploy -i $installation azure-app-collection -w=false

# block connections to API on security group
rule='{     
      "access": "Deny",
      "description": "Block API access during migration to multi AZ.",
      "destinationAddressPrefix": "*",
      "destinationAddressPrefixes": [],
      "destinationApplicationSecurityGroups": null,
      "destinationPortRange": "443",
      "destinationPortRanges": [],
      "direction": "Inbound",
      "name": "blockApiAccessMultiAZMigration",
      "priority": 100,
      "protocol": "*",
      "sourceAddressPrefix": "*",
      "sourceAddressPrefixes": [],
      "sourceApplicationSecurityGroups": null,
      "sourcePortRange": "*",
      "sourcePortRanges": [],
      "type": "Microsoft.Network/networkSecurityGroups/securityRules"
}'
az network nsg update -g ${cluster} -n ${cluster}-MasterSecurityGroup  --add 'securityRules' "$rule"

# wait for API to be unreachable.
set +e
curl --connect-timeout 5 -k -s $api_url >/dev/null
while [ $? -eq 0 ]
do
    echo "Api still reachable"
    sleep 5
    curl --connect-timeout 5 -k -s $api_url >/dev/null
done
set -e

echo "API unreachable, doing backup"

# stop etcd
ssh -A ${bastion_name} "ssh -o StrictHostKeyChecking=no $master_ip 'sudo systemctl stop etcd3'"

# backup etcd
remote_tar_path="/tmp/etcd-backup-$(date +%Y%m%d%H%M%S).tar.gz"
ssh -A ${bastion_name} "ssh -o StrictHostKeyChecking=no $master_ip 'sudo tar -C /var/lib/etcd -czf $remote_tar_path .'"
ssh -A ${bastion_name} "scp -o StrictHostKeyChecking=no $master_ip:$remote_tar_path $remote_tar_path"

ssh -A ${bastion_name} "ls -lah $remote_tar_path"

# get principal ID for vmss
principal_id="$(az vmss show -g ${cluster} -n ${cluster}-master-${cluster}| jq -r .identity.principalId)"
role_assignment_id="$(az role assignment list -g ${cluster} | jq ".[] | select(.principalId == \"$principal_id\")"| jq -r .id)"

echo "Principal ID: $principal_id"
echo "Role assignment ID: $role_assignment_id"
read -p "Do you want to go on with VMSS deletion?"

# delete master VMSS
az vmss delete -g $cluster -n ${cluster}-master-${cluster}

# delete role assignment
az role assignment delete --ids $role_assignment_id

# change master resource status to "DeploymentUninitialized"
read -p "Get ready to update the 'masters' condition stage to 'DeploymentUninitialized'"

set +e
while ! opsctl update status -i $installation -p apis/provider.giantswarm.io/v1alpha1/namespaces/default/azureconfigs/${cluster}/status
do
    echo "Failed"
    sleep 3
done

# start azure operator
kubectl --context=giantswarm-$installation -n giantswarm scale deploy -l app.kubernetes.io/instance=azure-operator-${azure_operator_version} --replicas=1

# wait for master
ssh -A ${bastion_name} "ssh -o StrictHostKeyChecking=no $master_ip 'sudo systemctl stop etcd3'"
while [ $? -ne 0 ]
do
    echo "Master not up yet"
    sleep 5
    ssh -A ${bastion_name} "ssh -o StrictHostKeyChecking=no $master_ip 'sudo systemctl stop etcd3'"
done
set -e

# restore etcd
ssh -A ${bastion_name} "scp -o StrictHostKeyChecking=no $remote_tar_path $master_ip:$remote_tar_path"
ssh -A ${bastion_name} "ssh -o StrictHostKeyChecking=no $master_ip 'sudo rm -rf /var/lib/etcd/*'"
ssh -A ${bastion_name} "ssh -o StrictHostKeyChecking=no $master_ip 'sudo tar -xf $remote_tar_path -C /var/lib/etcd/'"

# reboot master
set +e
ssh -A ${bastion_name} "ssh -o StrictHostKeyChecking=no $master_ip 'sudo reboot'"
set -e

# delete security rule
az network nsg rule delete -g ${cluster} --nsg-name ${cluster}-MasterSecurityGroup -n blockApiAccessMultiAZMigration
