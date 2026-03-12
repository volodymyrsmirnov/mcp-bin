package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

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

// IntrospectAll connects to all servers in parallel, introspects their tools,
// applies tool filters, and returns a populated Manifest.
func IntrospectAll(ctx context.Context, cfg *config.Config) (*Manifest, error) {
	type serverResult struct {
		name    string
		schemas []ToolSchema
		err     error
	}

	results := make(chan serverResult, len(cfg.Servers))
	var wg sync.WaitGroup

	for name, srv := range cfg.Servers {
		wg.Add(1)
		go func(name string, srv config.ServerConfig) {
			defer wg.Done()
			client, err := Connect(ctx, srv)
			if err != nil {
				results <- serverResult{name: name, err: fmt.Errorf("connecting to server %s: %w", name, err)}
				return
			}
			defer client.Close()

			tools, err := client.ListTools(ctx)
			if err != nil {
				results <- serverResult{name: name, err: fmt.Errorf("listing tools from %s: %w", name, err)}
				return
			}

			schemas, err := ToolsToSchemas(tools)
			if err != nil {
				results <- serverResult{name: name, err: fmt.Errorf("converting tools from %s: %w", name, err)}
				return
			}

			schemas = FilterSchemas(schemas, srv.AllowTools, srv.DenyTools)
			results <- serverResult{name: name, schemas: schemas}
		}(name, srv)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	manifest := &Manifest{
		Servers: make(map[string][]ToolSchema),
	}
	for res := range results {
		if res.err != nil {
			return nil, res.err
		}
		manifest.Servers[res.name] = res.schemas
	}

	return manifest, nil
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
