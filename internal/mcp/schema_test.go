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
