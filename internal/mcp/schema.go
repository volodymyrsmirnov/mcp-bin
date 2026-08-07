package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
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
	Enum        []string                // enum constraint values
	Items       *PropertyInfo           // array: item type/schema
	Properties  map[string]PropertyInfo // object: nested properties
	Required    []string                // object: required nested fields
}

// maxParseDepth limits recursion when parsing nested schemas.
const maxParseDepth = 3

// TypeHint returns a human-readable type string with structural details.
// For simple types it returns the type name. For arrays, objects, and enums
// it includes nested structure information.
func (p PropertyInfo) TypeHint() string {
	return p.typeHint(0)
}

func (p PropertyInfo) typeHint(depth int) string {
	base := p.Type
	if base == "" {
		base = "string"
	}

	switch base {
	case "array":
		if p.Items != nil {
			base = "array[" + p.Items.typeHint(depth+1) + "]"
		}
	case "object":
		if len(p.Properties) > 0 && depth < maxParseDepth {
			base = "object{" + p.propsHint(depth) + "}"
		}
	}

	if len(p.Enum) > 0 {
		base += " (one of: " + formatEnum(p.Enum) + ")"
	}

	return base
}

func (p PropertyInfo) propsHint(depth int) string {
	keys := SortedKeys(p.Properties)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + ": " + p.Properties[k].typeHint(depth+1)
	}
	return strings.Join(parts, ", ")
}

func formatEnum(vals []string) string {
	const maxDisplay = 5
	show := vals
	suffix := ""
	if len(vals) > maxDisplay {
		show = vals[:maxDisplay]
		suffix = ", ..."
	}
	parts := make([]string, len(show))
	for i, v := range show {
		parts[i] = fmt.Sprintf("%q", v)
	}
	return strings.Join(parts, ", ") + suffix
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
		parsed.Properties[name] = parseProperty(name, propRaw, 0)
	}

	return parsed
}

// rawProperty mirrors the JSON Schema keywords parseProperty understands,
// including the anyOf/oneOf union forms used for Optional[X] (FastMCP/Pydantic)
// and genuine X|Y unions.
type rawProperty struct {
	Type        jsonStringOrArray          `json:"type"`
	Description string                     `json:"description"`
	Enum        []string                   `json:"enum"`
	Items       json.RawMessage            `json:"items"`
	Properties  map[string]json.RawMessage `json:"properties"`
	Required    []string                   `json:"required"`
	AnyOf       []json.RawMessage          `json:"anyOf"`
	OneOf       []json.RawMessage          `json:"oneOf"`
}

// isNullBranch reports whether a union branch is the JSON Schema "null" type
// marker (e.g. the {"type": "null"} half of an Optional[X] anyOf).
func isNullBranch(raw json.RawMessage) bool {
	var t struct {
		Type jsonStringOrArray `json:"type"`
	}
	if err := json.Unmarshal(raw, &t); err != nil || len(t.Type) == 0 {
		return false
	}
	for _, v := range t.Type {
		if v != "null" {
			return false
		}
	}
	return true
}

// selectUnionBranch returns the first non-null branch of an anyOf/oneOf list.
// For Optional[X] this is unambiguous (the other branch is always null). For a
// genuine union with no null branch (e.g. array|string), the first branch wins;
// this precedence is a deliberate, documented choice rather than an accident.
func selectUnionBranch(branches []json.RawMessage) json.RawMessage {
	for _, b := range branches {
		if !isNullBranch(b) {
			return b
		}
	}
	return nil
}

// mergeUnionBranch overlays description/enum declared alongside anyOf/oneOf
// (outside the union) onto the PropertyInfo resolved from the selected branch.
func mergeUnionBranch(base PropertyInfo, outer rawProperty) PropertyInfo {
	if outer.Description != "" {
		base.Description = outer.Description
	}
	if len(outer.Enum) > 0 {
		base.Enum = outer.Enum
	}
	return base
}

// parseProperty recursively parses a JSON schema property, extracting type,
// description, enum, items (for arrays), and nested properties (for objects).
func parseProperty(name string, raw json.RawMessage, depth int) PropertyInfo {
	var prop rawProperty
	if err := json.Unmarshal(raw, &prop); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: could not parse schema for property %q, defaulting to string: %v\n", name, err)
		return PropertyInfo{Type: "string"}
	}

	if len(prop.Type) == 0 {
		if branch := selectUnionBranch(prop.AnyOf); branch != nil {
			return mergeUnionBranch(parseProperty(name, branch, depth), prop)
		}
		if branch := selectUnionBranch(prop.OneOf); branch != nil {
			return mergeUnionBranch(parseProperty(name, branch, depth), prop)
		}
	}

	info := PropertyInfo{
		Type:        prop.Type.Primary(),
		Description: prop.Description,
		Enum:        prop.Enum,
	}

	if depth >= maxParseDepth {
		return info
	}

	// Parse array items
	if info.Type == "array" && len(prop.Items) > 0 {
		items := parseProperty("items", prop.Items, depth+1)
		info.Items = &items
	}

	// Parse object properties
	if info.Type == "object" && len(prop.Properties) > 0 {
		info.Properties = make(map[string]PropertyInfo, len(prop.Properties))
		for k, v := range prop.Properties {
			info.Properties[k] = parseProperty(k, v, depth+1)
		}
		info.Required = prop.Required
	}

	return info
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
