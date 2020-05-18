package kubernetes

import (
	"github.com/giantswarm/azure-operator/v4/flag/service/cluster/kubernetes/api"
	"github.com/giantswarm/azure-operator/v4/flag/service/cluster/kubernetes/ingress"
	"github.com/giantswarm/azure-operator/v4/flag/service/cluster/kubernetes/kubectl"
	"github.com/giantswarm/azure-operator/v4/flag/service/cluster/kubernetes/kubelet"
	"github.com/giantswarm/azure-operator/v4/flag/service/cluster/kubernetes/networksetup"
	"github.com/giantswarm/azure-operator/v4/flag/service/cluster/kubernetes/ssh"
)

type Kubernetes struct {
	API               api.API
	Domain            string
	IngressController ingress.IngressController
	Kubectl           kubectl.Kubectl
	Kubelet           kubelet.Kubelet
	NetworkSetup      networksetup.NetworkSetup
	SSH               ssh.SSH
}
