package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

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
	if _, err := os.Stat(serverPyPath(t)); err != nil {
		t.Skip("server.py not found, skipping integration test")
	}
}

func writeTestConfig(t *testing.T, serverPy string) string {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	cfg := map[string]interface{}{
		"servers": map[string]interface{}{
			"testserver": map[string]interface{}{
				"command": "python3",
				"args":    []string{serverPy},
			},
		},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatal(err)
	}
	return configPath
}

func TestIntegrationDevModeGreet(t *testing.T) {
	skipIfNoPython(t)
	configPath := writeTestConfig(t, serverPyPath(t))

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"mcp-bin", "run", "--config", configPath, "testserver", "greet", "--name", "Alice"}

	app := BuildApp(nil, nil, false)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := app.Run(ctx, os.Args)
	if err != nil {
		t.Fatalf("app.Run failed: %v", err)
	}
}

func TestIntegrationDevModeAdd(t *testing.T) {
	skipIfNoPython(t)
	configPath := writeTestConfig(t, serverPyPath(t))

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"mcp-bin", "run", "--config", configPath, "testserver", "add", "--a", "10", "--b", "20"}

	app := BuildApp(nil, nil, false)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := app.Run(ctx, os.Args)
	if err != nil {
		t.Fatalf("app.Run failed: %v", err)
	}
}

func TestIntegrationDevModeHelp(t *testing.T) {
	skipIfNoPython(t)
	configPath := writeTestConfig(t, serverPyPath(t))

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// No tool specified → should print server help
	os.Args = []string{"mcp-bin", "run", "--config", configPath, "testserver"}

	app := BuildApp(nil, nil, false)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := app.Run(ctx, os.Args)
	if err != nil {
		t.Fatalf("app.Run failed: %v", err)
	}
}

func TestIntegrationDevModeToolHelp(t *testing.T) {
	skipIfNoPython(t)
	configPath := writeTestConfig(t, serverPyPath(t))

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"mcp-bin", "run", "--config", configPath, "testserver", "greet", "--help"}

	app := BuildApp(nil, nil, false)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := app.Run(ctx, os.Args)
	if err != nil {
		t.Fatalf("app.Run failed: %v", err)
	}
}

func TestIntegrationDevModeUnknownTool(t *testing.T) {
	skipIfNoPython(t)
	configPath := writeTestConfig(t, serverPyPath(t))

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"mcp-bin", "run", "--config", configPath, "testserver", "nonexistent"}

	app := BuildApp(nil, nil, false)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := app.Run(ctx, os.Args)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestIntegrationDevModeMissingRequired(t *testing.T) {
	skipIfNoPython(t)
	configPath := writeTestConfig(t, serverPyPath(t))

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// greet requires --name
	os.Args = []string{"mcp-bin", "run", "--config", configPath, "testserver", "greet"}

	app := BuildApp(nil, nil, false)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := app.Run(ctx, os.Args)
	if err == nil {
		t.Fatal("expected error for missing required argument")
	}
}

func TestIntegrationDevModeJSON(t *testing.T) {
	skipIfNoPython(t)
	configPath := writeTestConfig(t, serverPyPath(t))

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"mcp-bin", "--json", "run", "--config", configPath, "testserver", "greet", "--name", "Bob"}

	app := BuildApp(nil, nil, false)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := app.Run(ctx, os.Args)
	if err != nil {
		t.Fatalf("app.Run with --json failed: %v", err)
	}
}
