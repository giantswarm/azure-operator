package masters

const (
	// Types
	Stage                        = "Stage"
	DeploymentTemplateChecksum   = "TemplateChecksum"
	DeploymentParametersChecksum = "ParametersChecksum"

	// States
	ClusterUpgradeRequirementCheck = "ClusterUpgradeRequirementCheck"
	DeploymentUninitialized        = "DeploymentUninitialized"
	DeploymentInitialized          = "DeploymentInitialized"
	DeploymentCompleted            = "DeploymentCompleted"
	Empty                          = ""
	MasterInstancesUpgrading       = "MasterInstancesUpgrading"
	ProvisioningSuccessful         = "ProvisioningSuccessful"
)
