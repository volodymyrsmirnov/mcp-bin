package config

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"

	"github.com/volodymyrsmirnov/mcp-bin/schema"
)

// validateAgainstSchema validates raw config data against the embedded JSON Schema.
// For YAML input, it unmarshals to a generic map first.
func validateAgainstSchema(data []byte, isYAML bool) error {
	sch, err := schema.ConfigSchema()
	if err != nil {
		return fmt.Errorf("compiling config schema: %w", err)
	}

	var instance any
	if isYAML {
		if err := yaml.Unmarshal(data, &instance); err != nil {
			return fmt.Errorf("parsing YAML for schema validation: %w", err)
		}
	} else {
		instance, err = jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("parsing JSON for schema validation: %w", err)
		}
	}

	if err := sch.Validate(instance); err != nil {
		return formatSchemaError(err)
	}
	return nil
}

// formatSchemaError converts a jsonschema.ValidationError into a user-friendly message.
func formatSchemaError(err error) error {
	ve, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return fmt.Errorf("config validation: %w", err)
	}

	var msgs []string
	collectErrors(ve, &msgs)
	if len(msgs) == 0 {
		return fmt.Errorf("config validation failed")
	}
	if len(msgs) == 1 {
		return fmt.Errorf("config validation: %s", msgs[0])
	}
	return fmt.Errorf("config validation errors:\n  - %s", strings.Join(msgs, "\n  - "))
}

// collectErrors recursively gathers leaf error messages from a ValidationError tree.
func collectErrors(ve *jsonschema.ValidationError, msgs *[]string) {
	if len(ve.Causes) == 0 {
		loc := "/" + strings.Join(ve.InstanceLocation, "/")
		if loc == "/" {
			loc = "(root)"
		}
		*msgs = append(*msgs, fmt.Sprintf("at %s: %s", loc, ve.ErrorKind))
		return
	}
	for _, cause := range ve.Causes {
		collectErrors(cause, msgs)
	}
}
