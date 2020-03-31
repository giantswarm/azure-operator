package vmsscheck

type guardJob struct {
	resourceGroup string
	vmss          string
}

func (gj *guardJob) Run() error {
	// TODO: Implement VMSS instance check here in following PR.
	return nil
}

func (gj *guardJob) Finished() bool {
	// TODO: If any of the VMSS instances are in Failed state, return false here.
	return true
}
