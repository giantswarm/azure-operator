#!/usr/bin/env sh

mkdir -p /workdir/.shipyard
cp "$(kind get kubeconfig-path --name="kind")" "/workdir/.shipyard/config"
