package deployment

// Deployment defines an Azure Deployment that deploys an ARM template.
type Deployment struct {
	Name            string
	Parameters      map[string]interface{}
	ResourceGroup   string
	TemplateURI     string
	TemplateVersion string
}
