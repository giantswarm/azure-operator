package masters

const (
	// Types
	Stage                        = "Stage"
	DeploymentTemplateChecksum   = "TemplateChecksum"
	DeploymentParametersChecksum = "ParametersChecksum"

	// States
	BlockAPICalls                  = "BlockAPICalls"
	CheckFlatcarMigrationNeeded    = "CheckFlatcarMigrationNeeded"
	ClusterUpgradeRequirementCheck = "ClusterUpgradeRequirementCheck"
	DeallocateLegacyInstance       = "DeallocateLegacyInstance"
	DeleteLegacyVMSS               = "DeleteLegacyVMSS"
	DeploymentUninitialized        = "DeploymentUninitialized"
	DeploymentInitialized          = "DeploymentInitialized"
	DeploymentCompleted            = "DeploymentCompleted"
	Empty                          = ""
	ManualInterventionRequired     = "ManualInterventionRequired"
	MasterInstancesUpgrading       = "MasterInstancesUpgrading"
	ProvisioningSuccessful         = "ProvisioningSuccessful"
	RestartKubeletOnWorkers        = "RestartKubeletOnWorkers"
	UnblockAPICalls                = "UnblockAPICalls"
	WaitForBackupConfirmation      = "WaitForBackupConfirmation"
	WaitForMastersToBecomeReady    = "WaitForMastersToBecomeReady"
	WaitForRestore                 = "WaitForRestore"
)
