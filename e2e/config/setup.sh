#!/usr/bin/env sh

shipyard_dir="/workdir/.shipyard"
sudo mkdir -p "${shipyard_dir}"
sudo chmod -R 777 "/workdir"
kind get kubeconfig --name="kind" >"${shipyard_dir}/config"

GO111MODULE=on go get github.com/giantswarm/architect

kubectl --kubeconfig="${shipyard_dir}/config" create clusterrolebinding add-on-cluster-admin --clusterrole=cluster-admin --serviceaccount=kube-system:default
helm --kubeconfig="${shipyard_dir}/config" init --history-max 5 --wait
