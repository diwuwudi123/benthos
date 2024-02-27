package template

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/diwuwudi123/benthos/v4/internal/bloblang/mapping"
	"github.com/diwuwudi123/benthos/v4/internal/bundle"
	"github.com/diwuwudi123/benthos/v4/internal/component/cache"
	"github.com/diwuwudi123/benthos/v4/internal/component/input"
	"github.com/diwuwudi123/benthos/v4/internal/component/metrics"
	"github.com/diwuwudi123/benthos/v4/internal/component/output"
	"github.com/diwuwudi123/benthos/v4/internal/component/processor"
	"github.com/diwuwudi123/benthos/v4/internal/component/ratelimit"
	"github.com/diwuwudi123/benthos/v4/internal/docs"
	"github.com/diwuwudi123/benthos/v4/internal/manager"
	"github.com/diwuwudi123/benthos/v4/internal/message"
)

// InitTemplates parses and registers native templates, as well as templates
// at paths provided, and returns any linting errors that occur.
func InitTemplates(templatesPaths ...string) ([]string, error) {
	var lints []string
	for _, tPath := range templatesPaths {
		tmplConf, tLints, err := ReadConfigFile(tPath)
		if err != nil {
			return nil, fmt.Errorf("template %v: %w", tPath, err)
		}
		for _, l := range tLints {
			lints = append(lints, fmt.Sprintf("template file %v: %v", tPath, l))
		}

		tmpl, err := tmplConf.compile()
		if err != nil {
			return nil, fmt.Errorf("template %v: %w", tPath, err)
		}

		if err := registerTemplate(bundle.GlobalEnvironment, tmpl); err != nil {
			return nil, fmt.Errorf("template %v: %w", tPath, err)
		}
	}
	return lints, nil
}

//------------------------------------------------------------------------------

// Compiled is a template that has been compiled from a config.
type compiled struct {
	spec           docs.ComponentSpec
	mapping        *mapping.Executor
	metricsMapping *metrics.Mapping
}

// ExpandToNode attempts to apply the template to a provided YAML node and
// returns the new expanded configuration.
func (c *compiled) ExpandToNode(node *yaml.Node) (*yaml.Node, error) {
	generic, err := c.spec.Config.Children.YAMLToMap(node, docs.ToValueConfig{})
	if err != nil {
		return nil, fmt.Errorf("invalid config for template component: %w", err)
	}

	part := message.NewPart(nil)
	part.SetStructuredMut(generic)
	msg := message.Batch{part}

	newPart, err := c.mapping.MapPart(0, msg)
	if err != nil {
		return nil, fmt.Errorf("mapping failed for template component: %w", err)
	}

	resultGeneric, err := newPart.AsStructured()
	if err != nil {
		return nil, fmt.Errorf("mapping for template component resulted in invalid config: %w", err)
	}

	var resultNode yaml.Node
	if err := resultNode.Encode(resultGeneric); err != nil {
		return nil, fmt.Errorf("mapping for template component resulted in invalid yaml: %w", err)
	}

	return &resultNode, nil
}

//------------------------------------------------------------------------------

// RegisterTemplateYAML attempts to register a new template component to the
// specified environment.
func RegisterTemplateYAML(env *bundle.Environment, template []byte) error {
	tmplConf, _, err := ReadConfigYAML(template)
	if err != nil {
		return err
	}

	tmpl, err := tmplConf.compile()
	if err != nil {
		return err
	}

	return registerTemplate(env, tmpl)
}

// RegisterTemplate attempts to add a template component to the global list of
// component types.
func registerTemplate(env *bundle.Environment, tmpl *compiled) error {
	switch tmpl.spec.Type {
	case docs.TypeCache:
		return registerCacheTemplate(tmpl, env)
	case docs.TypeInput:
		return registerInputTemplate(tmpl, env)
	case docs.TypeOutput:
		return registerOutputTemplate(tmpl, env)
	case docs.TypeProcessor:
		return registerProcessorTemplate(tmpl, env)
	case docs.TypeRateLimit:
		return registerRateLimitTemplate(tmpl, env)
	}
	return fmt.Errorf("unable to register template for component type %v", tmpl.spec.Type)
}

// WithMetricsMapping attempts to wrap the metrics of a manager with a metrics
// mapping.
func WithMetricsMapping(nm bundle.NewManagement, m *metrics.Mapping) bundle.NewManagement {
	if t, ok := nm.(*manager.Type); ok {
		return t.WithMetricsMapping(m)
	}
	return nm
}

func registerCacheTemplate(tmpl *compiled, env *bundle.Environment) error {
	return env.CacheAdd(func(c cache.Config, nm bundle.NewManagement) (cache.V1, error) {
		newNode, err := tmpl.ExpandToNode(c.Plugin.(*yaml.Node))
		if err != nil {
			return nil, err
		}

		conf := cache.NewConfig()
		if err := newNode.Decode(&conf); err != nil {
			return nil, err
		}

		if tmpl.metricsMapping != nil {
			nm = WithMetricsMapping(nm, tmpl.metricsMapping.WithStaticVars(map[string]any{
				"label": c.Label,
			}))
		}
		return nm.NewCache(conf)
	}, tmpl.spec)
}

func registerInputTemplate(tmpl *compiled, env *bundle.Environment) error {
	return env.InputAdd(func(c input.Config, nm bundle.NewManagement) (input.Streamed, error) {
		newNode, err := tmpl.ExpandToNode(c.Plugin.(*yaml.Node))
		if err != nil {
			return nil, err
		}

		conf := input.NewConfig()
		if err := newNode.Decode(&conf); err != nil {
			return nil, err
		}

		// Template processors inserted _before_ configured processors.
		conf.Processors = append(conf.Processors, c.Processors...)

		if tmpl.metricsMapping != nil {
			nm = WithMetricsMapping(nm, tmpl.metricsMapping.WithStaticVars(map[string]any{
				"label": c.Label,
			}))
		}
		return nm.NewInput(conf)
	}, tmpl.spec)
}

func registerOutputTemplate(tmpl *compiled, env *bundle.Environment) error {
	return env.OutputAdd(func(c output.Config, nm bundle.NewManagement, pcf ...processor.PipelineConstructorFunc) (output.Streamed, error) {
		newNode, err := tmpl.ExpandToNode(c.Plugin.(*yaml.Node))
		if err != nil {
			return nil, err
		}

		conf := output.NewConfig()
		if err := newNode.Decode(&conf); err != nil {
			return nil, err
		}

		// Template processors inserted _after_ configured processors.
		conf.Processors = append(c.Processors, conf.Processors...)

		if tmpl.metricsMapping != nil {
			nm = WithMetricsMapping(nm, tmpl.metricsMapping.WithStaticVars(map[string]any{
				"label": c.Label,
			}))
		}
		return nm.NewOutput(conf, pcf...)
	}, tmpl.spec)
}

func registerProcessorTemplate(tmpl *compiled, env *bundle.Environment) error {
	return env.ProcessorAdd(func(c processor.Config, nm bundle.NewManagement) (processor.V1, error) {
		newNode, err := tmpl.ExpandToNode(c.Plugin.(*yaml.Node))
		if err != nil {
			return nil, err
		}

		conf := processor.NewConfig()
		if err := newNode.Decode(&conf); err != nil {
			return nil, err
		}

		if tmpl.metricsMapping != nil {
			nm = WithMetricsMapping(nm, tmpl.metricsMapping.WithStaticVars(map[string]any{
				"label": c.Label,
			}))
		}
		return nm.NewProcessor(conf)
	}, tmpl.spec)
}

func registerRateLimitTemplate(tmpl *compiled, env *bundle.Environment) error {
	return env.RateLimitAdd(func(c ratelimit.Config, nm bundle.NewManagement) (ratelimit.V1, error) {
		newNode, err := tmpl.ExpandToNode(c.Plugin.(*yaml.Node))
		if err != nil {
			return nil, err
		}

		conf := ratelimit.NewConfig()
		if err := newNode.Decode(&conf); err != nil {
			return nil, err
		}

		if tmpl.metricsMapping != nil {
			nm = WithMetricsMapping(nm, tmpl.metricsMapping.WithStaticVars(map[string]any{
				"label": c.Label,
			}))
		}
		return nm.NewRateLimit(conf)
	}, tmpl.spec)
}
