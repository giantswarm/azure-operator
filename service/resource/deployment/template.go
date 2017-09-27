package deployment

import "fmt"

const (
	templateURIVersionDefault = "master"
	templateURIFmt            = "https://raw.githubusercontent.com/giantswarm/azure-operator/%s/service/arm_templates/%s"

	mainTemplate = "main.json"
)

func templateURI(uriVersion, template string) string {
	return fmt.Sprintf(templateURIFmt, uriVersion, template)
}

func baseTemplateURI(uriVersion string) string {
	return templateURI(uriVersion, "")
}
