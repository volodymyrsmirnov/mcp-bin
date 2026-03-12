package mcp

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/volodymyrsmirnov/mcp-bin/internal/config"
)

// serverPyPath returns the absolute path to examples/server.py relative to this test file.
func serverPyPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "examples", "server.py")
}

func skipIfNoPython(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not found, skipping integration test")
	}
}

func TestIntegrationConnect(t *testing.T) {
	skipIfNoPython(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := config.ServerConfig{
		Command: "python3",
		Args:    []string{serverPyPath(t)},
	}

	client, err := Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Close()
}

func TestIntegrationListTools(t *testing.T) {
	skipIfNoPython(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := config.ServerConfig{
		Command: "python3",
		Args:    []string{serverPyPath(t)},
	}

	client, err := Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Close()

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
	}
	if !names["greet"] {
		t.Error("expected 'greet' tool")
	}
	if !names["add"] {
		t.Error("expected 'add' tool")
	}
}

func TestIntegrationCallToolGreet(t *testing.T) {
	skipIfNoPython(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := config.ServerConfig{
		Command: "python3",
		Args:    []string{serverPyPath(t)},
	}

	client, err := Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Close()

	result, err := client.CallTool(ctx, "greet", map[string]interface{}{
		"name": "World",
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatal("tool returned error")
	}
	if len(result.Content) == 0 {
		t.Fatal("empty result content")
	}
}

func TestIntegrationCallToolGreetLoud(t *testing.T) {
	skipIfNoPython(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := config.ServerConfig{
		Command: "python3",
		Args:    []string{serverPyPath(t)},
	}

	client, err := Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Close()

	result, err := client.CallTool(ctx, "greet", map[string]interface{}{
		"name": "World",
		"loud": true,
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatal("tool returned error")
	}
}

func TestIntegrationCallToolAdd(t *testing.T) {
	skipIfNoPython(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := config.ServerConfig{
		Command: "python3",
		Args:    []string{serverPyPath(t)},
	}

	client, err := Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Close()

	result, err := client.CallTool(ctx, "add", map[string]interface{}{
		"a": 5,
		"b": 3,
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatal("tool returned error")
	}
}

func TestIntegrationToolsToSchemasFromLive(t *testing.T) {
	skipIfNoPython(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := config.ServerConfig{
		Command: "python3",
		Args:    []string{serverPyPath(t)},
	}

	client, err := Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Close()

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	schemas, err := ToolsToSchemas(tools)
	if err != nil {
		t.Fatalf("ToolsToSchemas failed: %v", err)
	}

	if len(schemas) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(schemas))
	}

	for _, s := range schemas {
		if s.Name == "" {
			t.Error("schema name should not be empty")
		}
		if len(s.InputSchema) == 0 {
			t.Errorf("schema %s has empty InputSchema", s.Name)
		}
	}
}

func TestIntegrationFilterSchemasFromLive(t *testing.T) {
	skipIfNoPython(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := config.ServerConfig{
		Command: "python3",
		Args:    []string{serverPyPath(t)},
	}

	client, err := Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Close()

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	schemas, err := ToolsToSchemas(tools)
	if err != nil {
		t.Fatalf("ToolsToSchemas failed: %v", err)
	}

	// Filter to only "greet"
	filtered := FilterSchemas(schemas, []string{"greet"}, nil)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered schema, got %d", len(filtered))
	}
	if filtered[0].Name != "greet" {
		t.Errorf("expected greet, got %s", filtered[0].Name)
	}
}
