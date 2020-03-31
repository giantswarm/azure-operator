package vmsscheck

type guardJob struct {
	id            string
	resourceGroup string
	vmss          string

	onFinished func()
}

func (gj *guardJob) ID() string {
	return gj.id
}

func (gj *guardJob) Run() error {
	// TODO: Implement VMSS instance check here in following PR.
	return nil
}

func (gj *guardJob) Finished() bool {
	// TODO: If any of the VMSS instances are in Failed state, return false here.
	gj.onFinished()

	return true
}
