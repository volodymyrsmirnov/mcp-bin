package compile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/volodymyrsmirnov/mcp-bin/internal/config"
	"github.com/volodymyrsmirnov/mcp-bin/internal/embed"
	mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
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

func TestIntegrationCompile(t *testing.T) {
	skipIfNoPython(t)

	pyPath := serverPyPath(t)
	dir := t.TempDir()

	// Write config
	configPath := filepath.Join(dir, "config.json")
	cfgData := map[string]interface{}{
		"servers": map[string]interface{}{
			"testserver": map[string]interface{}{
				"command": "python3",
				"args":    []string{pyPath},
			},
		},
	}
	data, err := json.Marshal(cfgData)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	outputPath := filepath.Join(dir, "compiled-binary")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Compile
	if err := Compile(ctx, cfg, outputPath); err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	// Verify output exists and is executable
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("output binary not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output binary is empty")
	}

	// Detect the embedded zip
	zipInfo, err := embed.DetectEmbeddedZipFromPath(outputPath)
	if err != nil {
		t.Fatalf("detecting embedded zip: %v", err)
	}
	if zipInfo == nil {
		t.Fatal("expected embedded zip in compiled binary")
	}

	// Extract and verify contents
	testHome := t.TempDir()
	t.Setenv("HOME", testHome)

	paths, err := embed.ExtractToCache(zipInfo)
	if err != nil {
		t.Fatalf("extracting cache: %v", err)
	}

	// Verify config.json exists
	configData, err := os.ReadFile(paths.Config)
	if err != nil {
		t.Fatalf("reading extracted config: %v", err)
	}
	if len(configData) == 0 {
		t.Fatal("extracted config is empty")
	}

	// Verify manifest.json contains tools
	manifestData, err := os.ReadFile(paths.Manifest)
	if err != nil {
		t.Fatalf("reading extracted manifest: %v", err)
	}

	var manifest mcpclient.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parsing manifest: %v", err)
	}

	tools, ok := manifest.Servers["testserver"]
	if !ok {
		t.Fatal("expected 'testserver' in manifest")
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
	}
	if !names["greet"] {
		t.Error("expected 'greet' tool in manifest")
	}
	if !names["add"] {
		t.Error("expected 'add' tool in manifest")
	}
}

func TestIntegrationCompileWithEmbeddedFiles(t *testing.T) {
	skipIfNoPython(t)

	pyPath := serverPyPath(t)
	dir := t.TempDir()

	// Create a file to embed
	embeddedDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(embeddedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(embeddedDir, "readme.txt"), []byte("embedded content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Write config with files
	configPath := filepath.Join(dir, "config.json")
	cfgData := map[string]interface{}{
		"files": []string{"data"},
		"servers": map[string]interface{}{
			"testserver": map[string]interface{}{
				"command": "python3",
				"args":    []string{pyPath},
			},
		},
	}
	data, err := json.Marshal(cfgData)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	outputPath := filepath.Join(dir, "compiled-with-files")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := Compile(ctx, cfg, outputPath); err != nil {
		t.Fatalf("Compile with files failed: %v", err)
	}

	// Detect and extract
	zipInfo, err := embed.DetectEmbeddedZipFromPath(outputPath)
	if err != nil {
		t.Fatalf("detecting embedded zip: %v", err)
	}
	if zipInfo == nil {
		t.Fatal("expected embedded zip")
	}

	testHome := t.TempDir()
	t.Setenv("HOME", testHome)

	paths, err := embed.ExtractToCache(zipInfo)
	if err != nil {
		t.Fatalf("extracting cache: %v", err)
	}

	// Verify embedded file was included
	embeddedFile := filepath.Join(paths.DirsRoot, "data", "readme.txt")
	content, err := os.ReadFile(embeddedFile)
	if err != nil {
		t.Fatalf("reading embedded file: %v", err)
	}
	if string(content) != "embedded content" {
		t.Errorf("expected 'embedded content', got %q", string(content))
	}
}

func TestIntegrationCompileWithToolFilter(t *testing.T) {
	skipIfNoPython(t)

	pyPath := serverPyPath(t)
	dir := t.TempDir()

	// Config that only allows the "greet" tool
	configPath := filepath.Join(dir, "config.json")
	cfgData := map[string]interface{}{
		"servers": map[string]interface{}{
			"testserver": map[string]interface{}{
				"command":     "python3",
				"args":        []string{pyPath},
				"allow_tools": []string{"greet"},
			},
		},
	}
	data, err := json.Marshal(cfgData)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	outputPath := filepath.Join(dir, "compiled-filtered")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := Compile(ctx, cfg, outputPath); err != nil {
		t.Fatalf("Compile with filter failed: %v", err)
	}

	// Extract and verify only greet is in manifest
	zipInfo, err := embed.DetectEmbeddedZipFromPath(outputPath)
	if err != nil {
		t.Fatalf("detecting embedded zip: %v", err)
	}

	testHome := t.TempDir()
	t.Setenv("HOME", testHome)

	paths, err := embed.ExtractToCache(zipInfo)
	if err != nil {
		t.Fatalf("extracting cache: %v", err)
	}

	manifestData, err := os.ReadFile(paths.Manifest)
	if err != nil {
		t.Fatalf("reading manifest: %v", err)
	}

	var manifest mcpclient.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parsing manifest: %v", err)
	}

	tools := manifest.Servers["testserver"]
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool (filtered), got %d", len(tools))
	}
	if tools[0].Name != "greet" {
		t.Errorf("expected greet, got %s", tools[0].Name)
	}
}
