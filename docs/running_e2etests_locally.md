# Running integration tests locally

Build the application, which means compiling the Go code, building the docker image and the helm chart package.

```bash
CGO_ENABLED=0 go build -a -v -ldflags "-w -linkmode 'auto' -extldflags '-static' -X 'main.gitCommit=`git rev-parse HEAD`'" \
  && docker build -t "quay.io/giantswarm/azure-operator:`architect project version`" . \
  && architect helm template --dir ./helm/azure-operator \
  && helm package helm/azure-operator \
  && git checkout HEAD -- ./helm/azure-operator/Chart.yaml ./helm/azure-operator/values.yaml
```

Prepare the environment using kind

```bash
(kind delete cluster || true) && kind create cluster && kind load docker-image quay.io/giantswarm/azure-operator:`architect project version` \
  && kind get kubeconfig > kubeconfig.yaml && sudo ln -fs "${PWD}/kubeconfig.yaml" /workdir/.shipyard/config \
  && helm init --wait && k create clusterrolebinding add-on-cluster-admin --clusterrole=cluster-admin --serviceaccount=kube-system:default
```


You can then run the tests passing the helm chart package that we just generated
```bash
LATEST_OPERATOR_RELEASE="$(architect project version)" OPERATOR_HELM_TARBALL_PATH="${PWD}/azure-operator-`architect project version`.tgz" AZURE_SUBSCRIPTIONID="${AZURE_SUBSCRIPTIONID}" AZURE_TENANTID="${AZURE_TENANTID}" AZURE_CLIENTID="${AZURE_CLIENTID}" AZURE_CLIENTSECRET="${AZURE_CLIENTSECRET}" CIRCLE_SHA1="`git rev-parse HEAD`" TEST_DIR="integration/test/multiaz" AZURE_AZS=1 go test -v -tags=k8srequired ./integration/test/multiaz
```
