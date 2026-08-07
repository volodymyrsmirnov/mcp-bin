package mcp

import (
	"encoding/json"
	"testing"
)

func TestParseInputSchemaArrayType(t *testing.T) {
	// Schema where "type" is an array like ["string", "null"] (nullable field).
	raw := json.RawMessage(`{
		"type": "object",
		"properties": {
			"access_token": {
				"type": ["string", "null"],
				"description": "OAuth access token"
			},
			"name": {
				"type": "string",
				"description": "Account name"
			}
		},
		"required": ["name"]
	}`)

	parsed := ParseInputSchema(raw)

	if len(parsed.Properties) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(parsed.Properties))
	}

	at := parsed.Properties["access_token"]
	if at.Type != "string" {
		t.Errorf("access_token type: got %q, want %q", at.Type, "string")
	}
	if at.Description != "OAuth access token" {
		t.Errorf("access_token description: got %q, want %q", at.Description, "OAuth access token")
	}

	name := parsed.Properties["name"]
	if name.Type != "string" {
		t.Errorf("name type: got %q, want %q", name.Type, "string")
	}
}

func TestParseInputSchemaNullOnlyType(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "object",
		"properties": {
			"field": {
				"type": ["null"],
				"description": "null-only field"
			}
		}
	}`)

	parsed := ParseInputSchema(raw)
	f := parsed.Properties["field"]
	if f.Type != "null" {
		t.Errorf("field type: got %q, want %q", f.Type, "null")
	}
}

func TestParseInputSchemaAnyOfOptionalScalar(t *testing.T) {
	// anyOf[integer, null] is what FastMCP/Pydantic emits for Optional[int].
	raw := json.RawMessage(`{
		"type": "object",
		"properties": {
			"count": {
				"anyOf": [{"type": "integer"}, {"type": "null"}],
				"default": null,
				"title": "Count"
			}
		}
	}`)

	parsed := ParseInputSchema(raw)
	got := parsed.Properties["count"]
	if got.Type != "integer" {
		t.Errorf("count type: got %q, want %q", got.Type, "integer")
	}
}

func TestParseInputSchemaAnyOfOptionalBoolean(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "object",
		"properties": {
			"verbose": {
				"anyOf": [{"type": "boolean"}, {"type": "null"}],
				"default": null
			}
		}
	}`)

	parsed := ParseInputSchema(raw)
	got := parsed.Properties["verbose"]
	if got.Type != "boolean" {
		t.Errorf("verbose type: got %q, want %q", got.Type, "boolean")
	}
}

func TestParseInputSchemaAnyOfOptionalArray(t *testing.T) {
	// An array type reachable only through anyOf must resolve with its
	// nested items intact, not just the bare "array" type name.
	raw := json.RawMessage(`{
		"type": "object",
		"properties": {
			"tags": {
				"anyOf": [
					{"items": {"type": "string"}, "type": "array"},
					{"type": "null"}
				],
				"default": null,
				"title": "Tags"
			}
		}
	}`)

	parsed := ParseInputSchema(raw)
	got := parsed.Properties["tags"]
	if got.Type != "array" {
		t.Errorf("tags type: got %q, want %q", got.Type, "array")
	}
	if got.Items == nil {
		t.Fatal("tags.Items is nil")
	}
	if got.Items.Type != "string" {
		t.Errorf("tags.Items.Type: got %q, want %q", got.Items.Type, "string")
	}
}

func TestParseInputSchemaAnyOfOptionalObject(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "object",
		"properties": {
			"config": {
				"anyOf": [
					{
						"type": "object",
						"properties": {
							"host": {"type": "string"}
						}
					},
					{"type": "null"}
				],
				"default": null
			}
		}
	}`)

	parsed := ParseInputSchema(raw)
	got := parsed.Properties["config"]
	if got.Type != "object" {
		t.Errorf("config type: got %q, want %q", got.Type, "object")
	}
	if len(got.Properties) != 1 || got.Properties["host"].Type != "string" {
		t.Errorf("config.Properties: got %v, want host: string", got.Properties)
	}
}

func TestParseInputSchemaOneOfOptional(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "object",
		"properties": {
			"flag": {
				"oneOf": [{"type": "boolean"}, {"type": "null"}]
			}
		}
	}`)

	parsed := ParseInputSchema(raw)
	got := parsed.Properties["flag"]
	if got.Type != "boolean" {
		t.Errorf("flag type: got %q, want %q", got.Type, "boolean")
	}
}

func TestParseInputSchemaAnyOfNoNullBranch(t *testing.T) {
	// Genuine unions with no null branch have no unambiguous answer; the
	// documented precedence is "first branch wins".
	tests := []struct {
		name     string
		raw      json.RawMessage
		wantType string
	}{
		{
			name: "array or string",
			raw: json.RawMessage(`{
				"type": "object",
				"properties": {
					"filter": {
						"anyOf": [
							{"type": "array", "items": {"type": "string"}},
							{"type": "string"}
						]
					}
				}
			}`),
			wantType: "array",
		},
		{
			name: "object or string",
			raw: json.RawMessage(`{
				"type": "object",
				"properties": {
					"range": {
						"anyOf": [
							{"type": "object", "properties": {"since": {"type": "string"}}},
							{"type": "string"}
						]
					}
				}
			}`),
			wantType: "object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := ParseInputSchema(tt.raw)
			var got PropertyInfo
			for _, v := range parsed.Properties {
				got = v
			}
			if got.Type != tt.wantType {
				t.Errorf("type: got %q, want %q", got.Type, tt.wantType)
			}
		})
	}
}

func TestParseInputSchemaAnyOfDescriptionOverlay(t *testing.T) {
	// Description declared alongside anyOf (outside the union) takes
	// precedence over anything declared inside the selected branch.
	raw := json.RawMessage(`{
		"type": "object",
		"properties": {
			"count": {
				"anyOf": [{"type": "integer"}, {"type": "null"}],
				"description": "Outer description"
			}
		}
	}`)

	parsed := ParseInputSchema(raw)
	got := parsed.Properties["count"]
	if got.Description != "Outer description" {
		t.Errorf("description: got %q, want %q", got.Description, "Outer description")
	}
}

func TestParseInputSchemaUnhandledRefAndAllOf(t *testing.T) {
	// $ref and allOf are not resolved; document the fallback rather than
	// crash or silently mistype the property.
	raw := json.RawMessage(`{
		"type": "object",
		"properties": {
			"via_ref": {"$ref": "#/$defs/Thing"},
			"via_allof": {"allOf": [{"type": "string"}]}
		}
	}`)

	parsed := ParseInputSchema(raw)
	if got := parsed.Properties["via_ref"].Type; got != "string" {
		t.Errorf("via_ref type: got %q, want %q (fallback)", got, "string")
	}
	if got := parsed.Properties["via_allof"].Type; got != "string" {
		t.Errorf("via_allof type: got %q, want %q (fallback)", got, "string")
	}
}

func TestParseInputSchemaArrayItems(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "object",
		"properties": {
			"tags": {
				"type": "array",
				"items": {"type": "string"},
				"description": "List of tags"
			},
			"matrix": {
				"type": "array",
				"items": {
					"type": "array",
					"items": {"type": "number"}
				}
			}
		}
	}`)

	parsed := ParseInputSchema(raw)

	tags := parsed.Properties["tags"]
	if tags.Type != "array" {
		t.Errorf("tags type: got %q, want %q", tags.Type, "array")
	}
	if tags.Items == nil {
		t.Fatal("tags.Items is nil")
	}
	if tags.Items.Type != "string" {
		t.Errorf("tags.Items.Type: got %q, want %q", tags.Items.Type, "string")
	}

	matrix := parsed.Properties["matrix"]
	if matrix.Items == nil || matrix.Items.Items == nil {
		t.Fatal("matrix nested items are nil")
	}
	if matrix.Items.Items.Type != "number" {
		t.Errorf("matrix.Items.Items.Type: got %q, want %q", matrix.Items.Items.Type, "number")
	}
}

func TestParseInputSchemaObjectProperties(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "object",
		"properties": {
			"config": {
				"type": "object",
				"properties": {
					"host": {"type": "string", "description": "Hostname"},
					"port": {"type": "integer"}
				},
				"required": ["host"]
			}
		}
	}`)

	parsed := ParseInputSchema(raw)
	cfg := parsed.Properties["config"]
	if cfg.Type != "object" {
		t.Errorf("config type: got %q, want %q", cfg.Type, "object")
	}
	if len(cfg.Properties) != 2 {
		t.Fatalf("expected 2 nested properties, got %d", len(cfg.Properties))
	}
	if cfg.Properties["host"].Type != "string" {
		t.Errorf("host type: got %q, want %q", cfg.Properties["host"].Type, "string")
	}
	if cfg.Properties["port"].Type != "integer" {
		t.Errorf("port type: got %q, want %q", cfg.Properties["port"].Type, "integer")
	}
	if len(cfg.Required) != 1 || cfg.Required[0] != "host" {
		t.Errorf("config required: got %v, want [host]", cfg.Required)
	}
}

func TestParseInputSchemaEnum(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "object",
		"properties": {
			"mode": {
				"type": "string",
				"enum": ["fast", "slow", "auto"]
			}
		}
	}`)

	parsed := ParseInputSchema(raw)
	mode := parsed.Properties["mode"]
	if len(mode.Enum) != 3 {
		t.Fatalf("expected 3 enum values, got %d", len(mode.Enum))
	}
	if mode.Enum[0] != "fast" || mode.Enum[1] != "slow" || mode.Enum[2] != "auto" {
		t.Errorf("unexpected enum values: %v", mode.Enum)
	}
}

func TestTypeHint(t *testing.T) {
	tests := []struct {
		name string
		prop PropertyInfo
		want string
	}{
		{
			name: "simple string",
			prop: PropertyInfo{Type: "string"},
			want: "string",
		},
		{
			name: "empty type defaults to string",
			prop: PropertyInfo{},
			want: "string",
		},
		{
			name: "string with enum",
			prop: PropertyInfo{Type: "string", Enum: []string{"fast", "slow"}},
			want: `string (one of: "fast", "slow")`,
		},
		{
			name: "enum with many values",
			prop: PropertyInfo{Type: "string", Enum: []string{"a", "b", "c", "d", "e", "f"}},
			want: `string (one of: "a", "b", "c", "d", "e", ...)`,
		},
		{
			name: "array without items",
			prop: PropertyInfo{Type: "array"},
			want: "array",
		},
		{
			name: "array of strings",
			prop: PropertyInfo{Type: "array", Items: &PropertyInfo{Type: "string"}},
			want: "array[string]",
		},
		{
			name: "array of objects",
			prop: PropertyInfo{
				Type: "array",
				Items: &PropertyInfo{
					Type: "object",
					Properties: map[string]PropertyInfo{
						"name": {Type: "string"},
						"age":  {Type: "integer"},
					},
				},
			},
			want: "array[object{age: integer, name: string}]",
		},
		{
			name: "object without properties",
			prop: PropertyInfo{Type: "object"},
			want: "object",
		},
		{
			name: "object with properties",
			prop: PropertyInfo{
				Type: "object",
				Properties: map[string]PropertyInfo{
					"host": {Type: "string"},
					"port": {Type: "integer"},
				},
			},
			want: "object{host: string, port: integer}",
		},
		{
			name: "integer",
			prop: PropertyInfo{Type: "integer"},
			want: "integer",
		},
		{
			name: "boolean",
			prop: PropertyInfo{Type: "boolean"},
			want: "boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.prop.TypeHint()
			if got != tt.want {
				t.Errorf("TypeHint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestJsonStringOrArrayPrimary(t *testing.T) {
	tests := []struct {
		name string
		val  jsonStringOrArray
		want string
	}{
		{"single string", jsonStringOrArray{"string"}, "string"},
		{"nullable string", jsonStringOrArray{"string", "null"}, "string"},
		{"null first", jsonStringOrArray{"null", "integer"}, "integer"},
		{"null only", jsonStringOrArray{"null"}, "null"},
		{"empty", jsonStringOrArray{}, "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.val.Primary()
			if got != tt.want {
				t.Errorf("Primary() = %q, want %q", got, tt.want)
			}
		})
	}
}
