package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	ucli "github.com/urfave/cli/v3"
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

func TestBuildAppDevModeWithConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	configData := []byte(`{"servers":{"myserver":{"command":"echo","args":["hello"]}}}`)
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatal(err)
	}

	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"mcp-bin", "run", "--config", configPath}

	app := BuildApp(nil, nil, false)

	// The run subcommand should have a "myserver" subcommand pre-registered
	var runCmd *ucli.Command
	for _, cmd := range app.Commands {
		if cmd.Name == "run" {
			runCmd = cmd
			break
		}
	}
	if runCmd == nil {
		t.Fatal("expected 'run' command")
	}

	found := false
	for _, cmd := range runCmd.Commands {
		if cmd.Name == "myserver" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'myserver' subcommand pre-registered from config")
	}
}

func TestLoadConfigFromArgsInvalidFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(configPath, []byte(`not valid json`), 0644); err != nil {
		t.Fatal(err)
	}

	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"mcp-bin", "run", "--config", configPath}

	cfg := loadConfigFromArgs()
	if cfg != nil {
		t.Error("expected nil for invalid config file")
	}
}

func TestLoadConfigFromArgsEqualsInvalid(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(configPath, []byte(`not json`), 0644); err != nil {
		t.Fatal(err)
	}

	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"mcp-bin", "run", "--config=" + configPath}

	cfg := loadConfigFromArgs()
	if cfg != nil {
		t.Error("expected nil for invalid config via equals syntax")
	}
}

func TestLoadConfigFromArgsFlagVariants(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"servers":{}}`), 0644); err != nil {
		t.Fatal(err)
	}

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"long flag", []string{"mcp-bin", "run", "--config", configPath}, true},
		{"short flag", []string{"mcp-bin", "run", "-c", configPath}, true},
		{"long equals", []string{"mcp-bin", "run", "--config=" + configPath}, true},
		{"short equals", []string{"mcp-bin", "run", "-c=" + configPath}, true},
		{"old single dash config", []string{"mcp-bin", "run", "-config", configPath}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			cfg := loadConfigFromArgs()
			if tt.want && cfg == nil {
				t.Error("expected config to be loaded")
			}
			if !tt.want && cfg != nil {
				t.Error("expected nil config")
			}
		})
	}
}
