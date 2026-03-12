package cli

import (
	"encoding/json"
	"testing"

	"github.com/volodymyrsmirnov/mcp-bin/internal/config"
	mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
)

func TestBuildAppCompiledMode(t *testing.T) {
	cfg := &config.Config{
		Servers: map[string]config.ServerConfig{
			"test": {Command: "node", Args: []string{"server.js"}},
		},
	}
	manifest := &mcpclient.Manifest{
		Servers: map[string][]mcpclient.ToolSchema{
			"test": {
				{
					Name:        "greet",
					Description: "Greet someone",
					InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`),
				},
			},
		},
	}

	app := BuildApp(cfg, manifest, true)
	if app.Name != "mcp-bin" {
		t.Errorf("expected mcp-bin, got %s", app.Name)
	}

	// Should have server commands from manifest
	found := false
	for _, cmd := range app.Commands {
		if cmd.Name == "test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'test' command in compiled mode")
	}
}

func TestBuildAppDevMode(t *testing.T) {
	app := BuildApp(nil, nil, false)
	if app.Name != "mcp-bin" {
		t.Errorf("expected mcp-bin, got %s", app.Name)
	}

	// Should have run and compile subcommands
	hasRun := false
	hasCompile := false
	for _, cmd := range app.Commands {
		if cmd.Name == "run" {
			hasRun = true
		}
		if cmd.Name == "compile" {
			hasCompile = true
		}
	}
	if !hasRun {
		t.Error("expected 'run' command in dev mode")
	}
	if !hasCompile {
		t.Error("expected 'compile' command in dev mode")
	}

	// Root should have --json but NOT --config
	hasConfig := false
	hasJSON := false
	for _, f := range app.Flags {
		for _, name := range f.Names() {
			if name == "config" {
				hasConfig = true
			}
			if name == "json" {
				hasJSON = true
			}
		}
	}
	if hasConfig {
		t.Error("--config should not be on root in dev mode")
	}
	if !hasJSON {
		t.Error("expected --json flag on root")
	}
}

func TestLoadConfigFromArgsNotFound(t *testing.T) {
	// With default os.Args (test binary), should return nil
	cfg := loadConfigFromArgs()
	if cfg != nil {
		t.Error("expected nil config when no --config arg")
	}
}
