#!/usr/bin/env sh

shipyard_dir=".e2e-harness/workdir/.shipyard"
mkdir -p "${shipyard_dir}"
chmod -R 777 ".e2e-harness"
kind get kubeconfig --name="kind" | sudo tee "${shipyard_dir}/config" > /dev/null


shipyard_dir="/workdir/.shipyard"
sudo mkdir -p "${shipyard_dir}"
sudo chmod -R 777 "/workdir"
kind get kubeconfig --name="kind" | sudo tee "${shipyard_dir}/config" > /dev/null
