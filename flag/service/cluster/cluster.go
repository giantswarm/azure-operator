package cluster

import (
	"github.com/giantswarm/azure-operator/v6/flag/service/cluster/calico"
	"github.com/giantswarm/azure-operator/v6/flag/service/cluster/docker"
	"github.com/giantswarm/azure-operator/v6/flag/service/cluster/etcd"
	"github.com/giantswarm/azure-operator/v6/flag/service/cluster/kubernetes"
)

type Cluster struct {
	BaseDomain string
	Calico     calico.Calico
	Docker     docker.Docker
	Etcd       etcd.Etcd
	Kubernetes kubernetes.Kubernetes
}
