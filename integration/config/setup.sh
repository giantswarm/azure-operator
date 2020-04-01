#!/usr/bin/env sh

shipyard_dir=".e2e-harness/workdir/.shipyard"
mkdir -p "${shipyard_dir}"
chmod -R 777 ".e2e-harness"
kind get kubeconfig --name="kind" | sudo tee "${shipyard_dir}/config" > /dev/null


shipyard_dir="/workdir/.shipyard"
sudo mkdir -p "${shipyard_dir}"
sudo chmod -R 777 "/workdir"
kind get kubeconfig --name="kind" > "${shipyard_dir}/config"

curl -L https://get.helm.sh/helm-v2.16.1-linux-amd64.tar.gz > ./helm.tar.gz
tar xzvf helm.tar.gz
chmod u+x linux-amd64/helm
sudo mv linux-amd64/helm /usr/local/bin/

kubectl --kubeconfig="${shipyard_dir}/config" create clusterrolebinding add-on-cluster-admin --clusterrole=cluster-admin --serviceaccount=kube-system:default
helm --kubeconfig="${shipyard_dir}/config" init --history-max 5 --wait
