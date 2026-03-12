package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// PropertyInfo holds parsed info about a JSON schema property.
type PropertyInfo struct {
	Type        string
	Description string
}

// ParsedSchema holds parsed JSON schema information for a tool.
type ParsedSchema struct {
	Properties map[string]PropertyInfo
	Required   []string
}

// ParseInputSchema parses a JSON Schema (from a tool's InputSchema field) into a ParsedSchema.
func ParseInputSchema(raw json.RawMessage) ParsedSchema {
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: could not parse tool input schema: %v\n", err)
		return ParsedSchema{}
	}

	parsed := ParsedSchema{
		Properties: make(map[string]PropertyInfo),
		Required:   schema.Required,
	}

	for name, propRaw := range schema.Properties {
		var prop struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(propRaw, &prop); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: could not parse schema for property %q, defaulting to string: %v\n", name, err)
			parsed.Properties[name] = PropertyInfo{Type: "string"}
			continue
		}
		parsed.Properties[name] = PropertyInfo{
			Type:        prop.Type,
			Description: prop.Description,
		}
	}

	return parsed
}

// SortedKeys returns the keys of a PropertyInfo map in sorted order.
func SortedKeys(m map[string]PropertyInfo) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
