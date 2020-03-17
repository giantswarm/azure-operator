#!/usr/bin/env sh

shipyard_dir=".e2e-harness/workdir/.shipyard"
mkdir -p "${shipyard_dir}"
chmod -R 777 ".e2e-harness"
cp "$(kind get kubeconfig-path --name="kind")" "${shipyard_dir}/config"
