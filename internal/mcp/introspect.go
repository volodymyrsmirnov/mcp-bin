package mcp

import (
	"encoding/json"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/volodymyrsmirnov/mcp-bin/internal/config"
)

// ToolSchema is a serializable representation of an MCP tool for the manifest.
type ToolSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// Manifest holds pre-introspected tool schemas for all servers.
type Manifest struct {
	Servers map[string][]ToolSchema `json:"servers"`
}

// ToolsToSchemas converts mcp.Tool list to ToolSchema list.
func ToolsToSchemas(tools []mcplib.Tool) ([]ToolSchema, error) {
	schemas := make([]ToolSchema, len(tools))
	for i, t := range tools {
		inputBytes, err := json.Marshal(t.InputSchema)
		if err != nil {
			return nil, err
		}
		schemas[i] = ToolSchema{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: inputBytes,
		}
	}
	return schemas, nil
}

// FilterSchemas filters tool schemas based on allow/deny tool configuration.
func FilterSchemas(schemas []ToolSchema, allowTools, denyTools []string) []ToolSchema {
	if len(allowTools) == 0 && len(denyTools) == 0 {
		return schemas
	}
	var filtered []ToolSchema
	for _, s := range schemas {
		if config.MatchToolFilter(s.Name, allowTools, denyTools) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}
