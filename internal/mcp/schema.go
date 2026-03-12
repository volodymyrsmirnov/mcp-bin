package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// jsonStringOrArray handles JSON Schema "type" which can be a string ("string")
// or an array (["string", "null"]).
type jsonStringOrArray []string

func (t *jsonStringOrArray) UnmarshalJSON(data []byte) error {
	// Try string first.
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*t = []string{single}
		return nil
	}
	// Try array of strings.
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	*t = arr
	return nil
}

// Primary returns the first non-"null" type, or "string" if empty.
func (t jsonStringOrArray) Primary() string {
	for _, v := range t {
		if v != "null" {
			return v
		}
	}
	if len(t) > 0 {
		return t[0]
	}
	return "string"
}

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
			Type        jsonStringOrArray `json:"type"`
			Description string            `json:"description"`
		}
		if err := json.Unmarshal(propRaw, &prop); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: could not parse schema for property %q, defaulting to string: %v\n", name, err)
			parsed.Properties[name] = PropertyInfo{Type: "string"}
			continue
		}
		parsed.Properties[name] = PropertyInfo{
			Type:        prop.Type.Primary(),
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
