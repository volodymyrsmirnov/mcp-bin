package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestServerConfigIsLocal(t *testing.T) {
	s := ServerConfig{Command: "node"}
	if !s.IsLocal() {
		t.Error("expected IsLocal() to be true")
	}
	if s.IsRemote() {
		t.Error("expected IsRemote() to be false")
	}
}

func TestServerConfigIsRemote(t *testing.T) {
	s := ServerConfig{URL: "https://example.com"}
	if s.IsLocal() {
		t.Error("expected IsLocal() to be false")
	}
	if !s.IsRemote() {
		t.Error("expected IsRemote() to be true")
	}
}

func TestServerConfigNeither(t *testing.T) {
	s := ServerConfig{}
	if s.IsLocal() {
		t.Error("expected IsLocal() to be false")
	}
	if s.IsRemote() {
		t.Error("expected IsRemote() to be false")
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
		"files": ["/tmp/dir1"],
		"servers": {
			"s1": {"command": "node", "args": ["server.js"]},
			"s2": {"url": "https://example.com"}
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(cfg.Files))
	}
	if len(cfg.Servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(cfg.Servers))
	}
	if !cfg.Servers["s1"].IsLocal() {
		t.Error("s1 should be local")
	}
	if !cfg.Servers["s2"].IsRemote() {
		t.Error("s2 should be remote")
	}
}

func TestLoadFromFileResolvesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
		"files": ["examples/server.py", "/absolute/path"],
		"servers": {"s1": {"command": "node"}}
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Relative path should be resolved relative to config file directory
	if cfg.Files[0] != filepath.Join(dir, "examples/server.py") {
		t.Errorf("expected %s, got %s", filepath.Join(dir, "examples/server.py"), cfg.Files[0])
	}
	// Absolute path should be unchanged
	if cfg.Files[1] != "/absolute/path" {
		t.Errorf("expected /absolute/path, got %s", cfg.Files[1])
	}
	// Local server without explicit cwd gets config dir as cwd
	if cfg.Servers["s1"].Cwd != dir {
		t.Errorf("expected cwd %s, got %s", dir, cfg.Servers["s1"].Cwd)
	}
}

func TestLoadFromFileNotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/config.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadFromFileInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFromFile(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadFromFileValidationError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// Server with both command and url
	content := `{"servers": {"s1": {"command": "node", "url": "https://example.com"}}}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFromFile(path)
	if err == nil {
		t.Error("expected validation error")
	}
}

func TestLoadFromBytes(t *testing.T) {
	data := []byte(`{"files": [], "servers": {"s1": {"command": "node"}}}`)
	cfg, err := LoadFromBytes(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(cfg.Servers))
	}
}

func TestLoadFromBytesInvalidJSON(t *testing.T) {
	_, err := LoadFromBytes([]byte("invalid"))
	if err == nil {
		t.Error("expected error")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "valid local",
			config:  Config{Servers: map[string]ServerConfig{"s": {Command: "node"}}},
			wantErr: false,
		},
		{
			name:    "valid remote",
			config:  Config{Servers: map[string]ServerConfig{"s": {URL: "https://example.com"}}},
			wantErr: false,
		},
		{
			name:    "no command or url",
			config:  Config{Servers: map[string]ServerConfig{"s": {}}},
			wantErr: true,
		},
		{
			name:    "both command and url",
			config:  Config{Servers: map[string]ServerConfig{"s": {Command: "node", URL: "https://example.com"}}},
			wantErr: true,
		},
		{
			name:    "empty servers",
			config:  Config{Servers: map[string]ServerConfig{}},
			wantErr: false,
		},
		{
			name:    "valid tool filters",
			config:  Config{Servers: map[string]ServerConfig{"s": {Command: "node", AllowTools: []string{"read_*", "write_file"}, DenyTools: []string{"*_secret"}}}},
			wantErr: false,
		},
		{
			name:    "invalid allow_tools pattern",
			config:  Config{Servers: map[string]ServerConfig{"s": {Command: "node", AllowTools: []string{"[invalid"}}}},
			wantErr: true,
		},
		{
			name:    "invalid deny_tools pattern",
			config:  Config{Servers: map[string]ServerConfig{"s": {Command: "node", DenyTools: []string{"[invalid"}}}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPatchFiles(t *testing.T) {
	cfg := &Config{
		Files: []string{"/original/dir1", "/original/dir2"},
		Servers: map[string]ServerConfig{
			"local":  {Command: "node", Cwd: "/original/dir1"},
			"remote": {URL: "https://example.com"},
		},
	}

	// Without baseDir, absolute paths fall back to basename
	cfg.PatchFiles("/cache/dirs", "")

	if cfg.Files[0] != filepath.Join("/cache/dirs", "dir1") {
		t.Errorf("expected patched dir, got %s", cfg.Files[0])
	}
	if cfg.Files[1] != filepath.Join("/cache/dirs", "dir2") {
		t.Errorf("expected patched dir, got %s", cfg.Files[1])
	}

	// Explicit cwd falls back to basename without baseDir
	local := cfg.Servers["local"]
	if local.Cwd != filepath.Join("/cache/dirs", "dir1") {
		t.Errorf("expected patched cwd, got %s", local.Cwd)
	}

	// Remote server should not be patched
	remote := cfg.Servers["remote"]
	if remote.Cwd != "" {
		t.Errorf("expected empty cwd for remote, got %s", remote.Cwd)
	}
}

func TestPatchFilesWithBaseDir(t *testing.T) {
	cfg := &Config{
		Files: []string{"/project/src/server.py", "/project/lib/utils.py"},
		Servers: map[string]ServerConfig{
			"local": {Command: "python3", Cwd: "/project/src"},
		},
	}

	// With baseDir, absolute paths preserve relative structure
	cfg.PatchFiles("/cache/dirs", "/project")

	if cfg.Files[0] != filepath.Join("/cache/dirs", "src/server.py") {
		t.Errorf("expected preserved structure, got %s", cfg.Files[0])
	}
	if cfg.Files[1] != filepath.Join("/cache/dirs", "lib/utils.py") {
		t.Errorf("expected preserved structure, got %s", cfg.Files[1])
	}

	local := cfg.Servers["local"]
	if local.Cwd != filepath.Join("/cache/dirs", "src") {
		t.Errorf("expected preserved cwd structure, got %s", local.Cwd)
	}
}

func TestPatchFilesSetsDefaultCwd(t *testing.T) {
	cfg := &Config{
		Files: []string{"examples/server.py"},
		Servers: map[string]ServerConfig{
			"s1": {Command: "python3", Args: []string{"examples/server.py"}},
		},
	}

	cfg.PatchFiles("/cache/dirs", "")

	s1 := cfg.Servers["s1"]
	// Server without explicit cwd gets dirsRoot as cwd
	if s1.Cwd != "/cache/dirs" {
		t.Errorf("expected cwd /cache/dirs, got %s", s1.Cwd)
	}
	// Args are NOT patched — cwd makes them resolve correctly
	if s1.Args[0] != "examples/server.py" {
		t.Errorf("expected unchanged arg, got %s", s1.Args[0])
	}
}

func TestPatchFilesRelativePaths(t *testing.T) {
	cfg := &Config{
		Files: []string{"examples/server.py", "lib/utils"},
	}

	cfg.PatchFiles("/cache/dirs", "")

	// Relative paths are preserved under dirsRoot
	if cfg.Files[0] != filepath.Join("/cache/dirs", "examples/server.py") {
		t.Errorf("expected preserved relative path, got %s", cfg.Files[0])
	}
	if cfg.Files[1] != filepath.Join("/cache/dirs", "lib/utils") {
		t.Errorf("expected preserved relative path, got %s", cfg.Files[1])
	}
}

func TestPatchFilesNoFilesSkipsCwd(t *testing.T) {
	cfg := &Config{
		Servers: map[string]ServerConfig{
			"s1": {Command: "python3"},
		},
	}

	cfg.PatchFiles("/cache/dirs", "")

	s1 := cfg.Servers["s1"]
	if s1.Cwd != "" {
		t.Errorf("expected empty cwd when no files, got %s", s1.Cwd)
	}
}

func TestLoadCompiledConfig(t *testing.T) {
	t.Setenv("TEST_RUNTIME_VAR", "runtime-value")

	data := []byte(`{
		"files": ["/dir1"],
		"servers": {
			"s1": {
				"command": "node",
				"env": {
					"KEY1": {"value": "compile-value", "envVar": "TEST_RUNTIME_VAR"},
					"KEY2": {"value": "static-value", "envVar": ""}
				}
			},
			"s2": {
				"url": "https://example.com",
				"headers": {
					"Auth": {"value": "compile-token", "envVar": "TEST_RUNTIME_VAR"}
				}
			}
		}
	}`)

	cfg, err := LoadCompiledConfig(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Runtime env should override compile-time value
	if cfg.Servers["s1"].Env["KEY1"] != "runtime-value" {
		t.Errorf("expected runtime-value, got %s", cfg.Servers["s1"].Env["KEY1"])
	}

	// Static value should be kept
	if cfg.Servers["s1"].Env["KEY2"] != "static-value" {
		t.Errorf("expected static-value, got %s", cfg.Servers["s1"].Env["KEY2"])
	}

	// Headers should also be resolved
	if cfg.Servers["s2"].Headers["Auth"] != "runtime-value" {
		t.Errorf("expected runtime-value, got %s", cfg.Servers["s2"].Headers["Auth"])
	}
}

func TestLoadCompiledConfigFallback(t *testing.T) {
	_ = os.Unsetenv("NONEXISTENT_VAR_12345")

	data := []byte(`{
		"files": [],
		"servers": {
			"s1": {
				"command": "node",
				"env": {
					"KEY": {"value": "fallback", "envVar": "NONEXISTENT_VAR_12345"}
				}
			}
		}
	}`)

	cfg, err := LoadCompiledConfig(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Servers["s1"].Env["KEY"] != "fallback" {
		t.Errorf("expected fallback, got %s", cfg.Servers["s1"].Env["KEY"])
	}
}

func TestLoadCompiledConfigInvalidJSON(t *testing.T) {
	_, err := LoadCompiledConfig([]byte("bad json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadCompiledConfigNilMaps(t *testing.T) {
	data := []byte(`{
		"files": [],
		"servers": {
			"s1": {"command": "node"}
		}
	}`)

	cfg, err := LoadCompiledConfig(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Servers["s1"].Env != nil {
		t.Error("expected nil env")
	}
}

func TestLoadCompiledConfigPreservesToolFilters(t *testing.T) {
	data := []byte(`{
		"files": [],
		"servers": {
			"s1": {
				"command": "node",
				"allow_tools": ["read_*"],
				"deny_tools": ["read_secret"]
			}
		}
	}`)

	cfg, err := LoadCompiledConfig(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s1 := cfg.Servers["s1"]
	if len(s1.AllowTools) != 1 || s1.AllowTools[0] != "read_*" {
		t.Errorf("expected allow_tools [read_*], got %v", s1.AllowTools)
	}
	if len(s1.DenyTools) != 1 || s1.DenyTools[0] != "read_secret" {
		t.Errorf("expected deny_tools [read_secret], got %v", s1.DenyTools)
	}
}

// YAML tests

func TestLoadFromFileYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
files:
  - /tmp/dir1
servers:
  s1:
    command: node
    args:
      - server.js
  s2:
    url: https://example.com
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(cfg.Files))
	}
	if len(cfg.Servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(cfg.Servers))
	}
	if !cfg.Servers["s1"].IsLocal() {
		t.Error("s1 should be local")
	}
	if !cfg.Servers["s2"].IsRemote() {
		t.Error("s2 should be remote")
	}
}

func TestLoadFromFileYMLExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	content := `
servers:
  s1:
    command: node
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(cfg.Servers))
	}
}

func TestLoadFromFileInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(":\n  invalid: [yaml: bad"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFromFile(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadFromFileYAMLValidationError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
servers:
  s1:
    command: node
    url: https://example.com
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFromFile(path)
	if err == nil {
		t.Error("expected validation error")
	}
}

func TestLoadFromFileYAMLWithToolFilters(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
servers:
  s1:
    command: node
    allow_tools:
      - "read_*"
      - "list_*"
    deny_tools:
      - read_secret
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s1 := cfg.Servers["s1"]
	if len(s1.AllowTools) != 2 {
		t.Errorf("expected 2 allow_tools, got %d", len(s1.AllowTools))
	}
	if len(s1.DenyTools) != 1 {
		t.Errorf("expected 1 deny_tools, got %d", len(s1.DenyTools))
	}
}

func TestLoadFromFileJSONWithToolFilters(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
		"servers": {
			"s1": {
				"command": "node",
				"allow_tools": ["read_*", "list_*"],
				"deny_tools": ["read_secret"]
			}
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s1 := cfg.Servers["s1"]
	if len(s1.AllowTools) != 2 {
		t.Errorf("expected 2 allow_tools, got %d", len(s1.AllowTools))
	}
	if len(s1.DenyTools) != 1 {
		t.Errorf("expected 1 deny_tools, got %d", len(s1.DenyTools))
	}
}
