package schema

import (
	_ "embed"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed mcp-bin-config.schema.json
var schemaJSON string

var (
	compiledSchema *jsonschema.Schema
	compileOnce    sync.Once
	compileErr     error
)

// ConfigSchema returns the compiled JSON Schema for config validation.
// The schema is compiled once and cached.
func ConfigSchema() (*jsonschema.Schema, error) {
	compileOnce.Do(func() {
		doc, err := jsonschema.UnmarshalJSON(strings.NewReader(schemaJSON))
		if err != nil {
			compileErr = err
			return
		}
		c := jsonschema.NewCompiler()
		if err := c.AddResource("mcp-bin-config.schema.json", doc); err != nil {
			compileErr = err
			return
		}
		compiledSchema, compileErr = c.Compile("mcp-bin-config.schema.json")
	})
	return compiledSchema, compileErr
}
