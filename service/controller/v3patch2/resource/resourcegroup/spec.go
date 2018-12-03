package resourcegroup

// Group defines an Azure Resource Group.
type Group struct {
	Name     string
	Location string
	Tags     map[string]string
}
