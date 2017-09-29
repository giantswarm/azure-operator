package deployment

import "fmt"

const (
	templateURIFmt = "https://raw.githubusercontent.com/giantswarm/azure-operator/%s/service/arm_templates/%s"

	// templateVersionDefault is the default value for Config.TemplateVersion.
	templateVersionDefault = "master"

	mainTemplate = "main.json"
)

func templateURI(version, template string) string {
	return fmt.Sprintf(templateURIFmt, version, template)
}

func baseTemplateURI(version string) string {
	return templateURI(version, "")
}
