package annotation

const (
	IsMasterUpgrading        = "azure-machine-pool.giantswarm.io/is-master-upgrading"
	StateMachineCurrentState = "azure-machine-pool.giantswarm.io/state-machine-current-state"

	// UpgradingToNodePools is set to True during the first cluster upgrade to node pools release.
	UpgradingToNodePools = "release.giantswarm.io/upgrading-to-node-pools"
)
