package splunk

import (
	_ "embed"

	"github.com/diwuwudi123/benthos/v4/internal/bundle"
	"github.com/diwuwudi123/benthos/v4/internal/template"
)

//go:embed template_output.yaml
var outputTemplate []byte

func init() {
	if err := template.RegisterTemplateYAML(bundle.GlobalEnvironment, outputTemplate); err != nil {
		panic(err)
	}
}
