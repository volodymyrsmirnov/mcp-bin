package mcp

import (
	"encoding/json"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func TestToolsToSchemas(t *testing.T) {
	tools := []mcplib.Tool{
		{
			Name:        "greet",
			Description: "Greet someone",
			InputSchema: mcplib.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Name to greet",
					},
				},
				Required: []string{"name"},
			},
		},
		{
			Name:        "add",
			Description: "Add numbers",
			InputSchema: mcplib.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"a": map[string]any{"type": "number"},
					"b": map[string]any{"type": "number"},
				},
			},
		},
	}

	schemas, err := ToolsToSchemas(tools)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(schemas) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(schemas))
	}

	if schemas[0].Name != "greet" {
		t.Errorf("expected greet, got %s", schemas[0].Name)
	}
	if schemas[0].Description != "Greet someone" {
		t.Errorf("expected 'Greet someone', got %s", schemas[0].Description)
	}

	// Verify InputSchema is valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(schemas[0].InputSchema, &parsed); err != nil {
		t.Errorf("InputSchema is not valid JSON: %v", err)
	}
	if parsed["type"] != "object" {
		t.Errorf("expected type=object, got %v", parsed["type"])
	}
}

func TestToolsToSchemasEmpty(t *testing.T) {
	schemas, err := ToolsToSchemas([]mcplib.Tool{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(schemas) != 0 {
		t.Errorf("expected 0 schemas, got %d", len(schemas))
	}
}

func TestManifestSerialization(t *testing.T) {
	manifest := Manifest{
		Servers: map[string][]ToolSchema{
			"server1": {
				{Name: "tool1", Description: "desc1", InputSchema: json.RawMessage(`{}`)},
			},
		},
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed Manifest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parsed.Servers["server1"]) != 1 {
		t.Errorf("expected 1 tool, got %d", len(parsed.Servers["server1"]))
	}
	if parsed.Servers["server1"][0].Name != "tool1" {
		t.Errorf("expected tool1, got %s", parsed.Servers["server1"][0].Name)
	}
}
