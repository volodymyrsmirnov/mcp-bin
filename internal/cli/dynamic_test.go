package cli

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	ucli "github.com/urfave/cli/v3"
	"github.com/volodymyrsmirnov/mcp-bin/internal/config"
	mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
)

func TestParseInputSchema(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string", "description": "Name"},
			"count": {"type": "integer", "description": "Count"},
			"ratio": {"type": "number", "description": "Ratio"},
			"verbose": {"type": "boolean", "description": "Verbose"},
			"data": {"type": "object", "description": "Data object"}
		},
		"required": ["name", "count"]
	}`)

	schema := mcpclient.ParseInputSchema(raw)

	if len(schema.Properties) != 5 {
		t.Errorf("expected 5 properties, got %d", len(schema.Properties))
	}
	if len(schema.Required) != 2 {
		t.Errorf("expected 2 required, got %d", len(schema.Required))
	}

	if schema.Properties["name"].Type != "string" {
		t.Errorf("expected string, got %s", schema.Properties["name"].Type)
	}
	if schema.Properties["name"].Description != "Name" {
		t.Errorf("expected Name, got %s", schema.Properties["name"].Description)
	}
	if schema.Properties["count"].Type != "integer" {
		t.Errorf("expected integer, got %s", schema.Properties["count"].Type)
	}
	if schema.Properties["ratio"].Type != "number" {
		t.Errorf("expected number, got %s", schema.Properties["ratio"].Type)
	}
	if schema.Properties["verbose"].Type != "boolean" {
		t.Errorf("expected boolean, got %s", schema.Properties["verbose"].Type)
	}
}

func TestParseInputSchemaInvalidJSON(t *testing.T) {
	schema := mcpclient.ParseInputSchema(json.RawMessage(`not json`))
	if len(schema.Properties) != 0 {
		t.Errorf("expected empty properties for invalid JSON")
	}
}

func TestParseInputSchemaInvalidProperty(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "object",
		"properties": {
			"bad": "not an object"
		}
	}`)

	schema := mcpclient.ParseInputSchema(raw)
	// Should fallback to string type
	if schema.Properties["bad"].Type != "string" {
		t.Errorf("expected fallback to string, got %s", schema.Properties["bad"].Type)
	}
}

func TestParseToolArgs(t *testing.T) {
	schema := mcpclient.ParsedSchema{
		Properties: map[string]mcpclient.PropertyInfo{
			"name":    {Type: "string"},
			"count":   {Type: "integer"},
			"ratio":   {Type: "number"},
			"verbose": {Type: "boolean"},
			"data":    {Type: "object"},
		},
	}

	tests := []struct {
		name    string
		args    []string
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "string arg",
			args: []string{"--name", "hello"},
			want: map[string]interface{}{"name": "hello"},
		},
		{
			name: "integer arg",
			args: []string{"--count", "42"},
			want: map[string]interface{}{"count": int64(42)},
		},
		{
			name: "number arg",
			args: []string{"--ratio", "3.14"},
			want: map[string]interface{}{"ratio": 3.14},
		},
		{
			name: "boolean flag",
			args: []string{"--verbose"},
			want: map[string]interface{}{"verbose": true},
		},
		{
			name: "json object",
			args: []string{"--data", `{"key":"val"}`},
			want: map[string]interface{}{"data": map[string]interface{}{"key": "val"}},
		},
		{
			name: "equals syntax",
			args: []string{"--name=world"},
			want: map[string]interface{}{"name": "world"},
		},
		{
			name:    "unknown flag",
			args:    []string{"--unknown", "val"},
			wantErr: true,
		},
		{
			name:    "missing value",
			args:    []string{"--name"},
			wantErr: true,
		},
		{
			name:    "no prefix",
			args:    []string{"positional"},
			wantErr: true,
		},
		{
			name:    "invalid integer",
			args:    []string{"--count", "abc"},
			wantErr: true,
		},
		{
			name:    "invalid number",
			args:    []string{"--ratio", "abc"},
			wantErr: true,
		},
		{
			name: "multiple args",
			args: []string{"--name", "hello", "--count", "5", "--verbose"},
			want: map[string]interface{}{"name": "hello", "count": int64(5), "verbose": true},
		},
		{
			name: "empty args",
			args: []string{},
			want: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseToolArgs(tt.args, schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseToolArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			for k, v := range tt.want {
				got, ok := result[k]
				if !ok {
					t.Errorf("missing key %s", k)
					continue
				}
				gotJSON, _ := json.Marshal(got)
				wantJSON, _ := json.Marshal(v)
				if string(gotJSON) != string(wantJSON) {
					t.Errorf("key %s: got %v, want %v", k, got, v)
				}
			}
		})
	}
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		value   string
		typ     string
		want    interface{}
		wantErr bool
	}{
		{"hello", "string", "hello", false},
		{"42", "integer", int64(42), false},
		{"3.14", "number", 3.14, false},
		{"true", "boolean", true, false},
		{"false", "boolean", false, false},
		{"yes", "boolean", true, false},
		{"no", "boolean", false, false},
		{"1", "boolean", true, false},
		{"0", "boolean", false, false},
		{"abc", "integer", nil, true},
		{"abc", "number", nil, true},
		{"maybe", "boolean", nil, true},
		{`{"key":"val"}`, "object", map[string]interface{}{"key": "val"}, false},
		{"plaintext", "object", "plaintext", false}, // non-JSON fallback
	}

	for _, tt := range tests {
		t.Run(tt.value+"_"+tt.typ, func(t *testing.T) {
			got, err := parseValue(tt.value, tt.typ)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseValue(%q, %q) error = %v, wantErr %v", tt.value, tt.typ, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tt.want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("parseValue(%q, %q) = %v, want %v", tt.value, tt.typ, got, tt.want)
			}
		})
	}
}

func TestParseToolArgsEqualsUnknown(t *testing.T) {
	schema := mcpclient.ParsedSchema{
		Properties: map[string]mcpclient.PropertyInfo{
			"name": {Type: "string"},
		},
	}

	_, err := parseToolArgs([]string{"--unknown=value"}, schema)
	if err == nil {
		t.Error("expected error for unknown flag with equals syntax")
	}
}

func TestParseToolArgsBooleanEquals(t *testing.T) {
	schema := mcpclient.ParsedSchema{
		Properties: map[string]mcpclient.PropertyInfo{
			"verbose": {Type: "boolean"},
		},
	}

	result, err := parseToolArgs([]string{"--verbose=true"}, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["verbose"] != true {
		t.Errorf("expected true, got %v", result["verbose"])
	}
}

func TestSchemaToFlags(t *testing.T) {
	schema := mcpclient.ParsedSchema{
		Properties: map[string]mcpclient.PropertyInfo{
			"str":   {Type: "string", Description: "a string"},
			"num":   {Type: "number", Description: "a number"},
			"int":   {Type: "integer", Description: "an integer"},
			"bool":  {Type: "boolean", Description: "a boolean"},
			"obj":   {Type: "object", Description: "an object"},
			"arr":   {Type: "array", Description: "an array"},
			"other": {Type: "unknown", Description: "something"},
		},
	}

	flags := schemaToFlags(schema)
	if len(flags) != 7 {
		t.Errorf("expected 7 flags, got %d", len(flags))
	}
}

func TestBuildCommandsFromManifest(t *testing.T) {
	cfg := &config.Config{
		Servers: map[string]config.ServerConfig{
			"test-server": {Command: "node", Args: []string{"server.js"}},
		},
	}
	manifest := &mcpclient.Manifest{
		Servers: map[string][]mcpclient.ToolSchema{
			"test-server": {
				{
					Name:        "greet",
					Description: "Greet someone",
					InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`),
				},
			},
		},
	}

	commands := buildCommandsFromManifest(cfg, manifest)
	if len(commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(commands))
	}
	if commands[0].Name != "test-server" {
		t.Errorf("expected test-server, got %s", commands[0].Name)
	}
	if len(commands[0].Commands) != 1 {
		t.Fatalf("expected 1 subcommand, got %d", len(commands[0].Commands))
	}
	if commands[0].Commands[0].Name != "greet" {
		t.Errorf("expected greet, got %s", commands[0].Commands[0].Name)
	}
}

func TestBuildCommandsFromConfig(t *testing.T) {
	cfg := &config.Config{
		Servers: map[string]config.ServerConfig{
			"s1": {Command: "node"},
			"s2": {URL: "https://example.com"},
		},
	}

	commands := buildCommandsFromConfig(cfg)
	if len(commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(commands))
	}

	names := make(map[string]bool)
	for _, cmd := range commands {
		names[cmd.Name] = true
	}
	if !names["s1"] || !names["s2"] {
		t.Errorf("expected s1 and s2, got %v", names)
	}
}

func TestPrintServerHelp(t *testing.T) {
	tools := []mcpclient.ToolSchema{
		{
			Name:        "tool1",
			Description: "first tool",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`),
		},
		{
			Name:        "tool2",
			Description: "second tool",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"count":{"type":"integer"}}}`),
		},
	}

	out := captureStdout(t, func() { printServerHelp("test-server", tools) })

	// Header with tool count
	if !strings.Contains(out, "Server: test-server (2 tools)") {
		t.Errorf("missing header with tool count:\n%s", out)
	}
	// Tool names on their own lines
	if !strings.Contains(out, "  tool1\n") {
		t.Errorf("tool1 should be on its own line:\n%s", out)
	}
	// Flags indented under tool name
	if !strings.Contains(out, "    --name <string> (required)\n") {
		t.Errorf("missing indented flag for tool1:\n%s", out)
	}
	if !strings.Contains(out, "    --count <integer>\n") {
		t.Errorf("missing indented flag for tool2:\n%s", out)
	}
	// Descriptions indented
	if !strings.Contains(out, "    first tool") {
		t.Errorf("missing indented description for tool1:\n%s", out)
	}
	// Footer
	if !strings.Contains(out, "Usage:") {
		t.Errorf("missing usage line:\n%s", out)
	}
}

func TestPrintToolHelp(t *testing.T) {
	tool := mcpclient.ToolSchema{
		Name:        "greet",
		Description: "Greet someone",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string","description":"Name to greet"}},"required":["name"]}`),
	}

	out := captureStdout(t, func() { printToolHelp("test-server", tool) })

	// Tool name on its own line (no flags appended)
	if !strings.HasPrefix(out, "greet\n") {
		t.Errorf("tool name should be first line on its own:\n%s", out)
	}
	// Flag indented under tool name
	if !strings.Contains(out, "  --name <string> (required)\n") {
		t.Errorf("missing indented flag:\n%s", out)
	}
	// Description indented
	if !strings.Contains(out, "  Greet someone") {
		t.Errorf("missing indented description:\n%s", out)
	}
	// Usage line
	if !strings.Contains(out, "Usage: mcp-bin run --config <file> test-server greet") {
		t.Errorf("missing usage line:\n%s", out)
	}
}

func TestPrintToolHelpNoProperties(t *testing.T) {
	tool := mcpclient.ToolSchema{
		Name:        "simple",
		Description: "Simple tool",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}

	out := captureStdout(t, func() { printToolHelp("test-server", tool) })

	if !strings.HasPrefix(out, "simple\n") {
		t.Errorf("tool name should be first line:\n%s", out)
	}
	// No indented flags (Usage line may contain --)
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			t.Errorf("should not have flag lines:\n%s", out)
			break
		}
	}
}

func TestPrintToolHelpMultilineDescription(t *testing.T) {
	tool := mcpclient.ToolSchema{
		Name:        "fetch",
		Description: "Fetch data from API.\n\nSupports pagination\nand filtering.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"url":{"type":"string"}},"required":["url"]}`),
	}

	out := captureStdout(t, func() { printToolHelp("srv", tool) })

	// All description lines should be indented
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed == "fetch" || strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "Usage:") || strings.HasPrefix(trimmed, "Run with") {
			continue
		}
		if !strings.HasPrefix(line, "  ") {
			t.Errorf("description line not indented: %q\nfull output:\n%s", line, out)
		}
	}
}

func TestFormatConsistency(t *testing.T) {
	tool := mcpclient.ToolSchema{
		Name:        "query",
		Description: "Run a query.\n\nDetailed explanation here.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"sql":{"type":"string"},"limit":{"type":"integer"}},"required":["sql"]}`),
	}

	serverOut := captureStdout(t, func() { printServerHelp("db", []mcpclient.ToolSchema{tool}) })
	toolOut := captureStdout(t, func() { printToolHelp("db", tool) })

	// Both should contain flags one per line (not appended to tool name)
	for _, name := range []string{"--sql <string> (required)", "--limit <integer>"} {
		if !strings.Contains(serverOut, name) {
			t.Errorf("server help missing flag %q:\n%s", name, serverOut)
		}
		if !strings.Contains(toolOut, name) {
			t.Errorf("tool help missing flag %q:\n%s", name, toolOut)
		}
	}

	// Both should have description indented with \n\n before and \n\n after
	for label, out := range map[string]string{"server": serverOut, "tool": toolOut} {
		if !strings.Contains(out, "\n\n") {
			t.Errorf("%s help missing blank line separators:\n%s", label, out)
		}
		if !strings.Contains(out, "Detailed explanation here.") {
			t.Errorf("%s help missing second paragraph:\n%s", label, out)
		}
	}

	// Tool name line should not have flags (skip Usage/footer lines)
	for label, out := range map[string]string{"server": serverOut, "tool": toolOut} {
		for _, line := range strings.Split(out, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "Usage:") || strings.HasPrefix(trimmed, "Run with") {
				continue
			}
			if strings.Contains(line, "query") && strings.Contains(line, "--") {
				t.Errorf("%s help has flags on same line as tool name: %q", label, line)
			}
		}
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestBuildServerCommandFromSchemas(t *testing.T) {
	serverCfg := &config.ServerConfig{Command: "node"}
	tools := []mcpclient.ToolSchema{
		{
			Name:        "tool1",
			Description: "First tool",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"arg":{"type":"string"}}}`),
		},
		{
			Name:        "tool2",
			Description: "Second tool",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
	}

	cmd := buildServerCommandFromSchemas("myserver", serverCfg, tools)
	if cmd.Name != "myserver" {
		t.Errorf("expected myserver, got %s", cmd.Name)
	}
	if len(cmd.Commands) != 2 {
		t.Errorf("expected 2 subcommands, got %d", len(cmd.Commands))
	}
}

func TestParseToolArgsEqualsInvalidValue(t *testing.T) {
	schema := mcpclient.ParsedSchema{
		Properties: map[string]mcpclient.PropertyInfo{
			"count": {Type: "integer"},
		},
	}

	_, err := parseToolArgs([]string{"--count=abc"}, schema)
	if err == nil {
		t.Error("expected error for invalid integer with equals syntax")
	}
}

func TestParseToolArgsPassthrough(t *testing.T) {
	emptySchema := mcpclient.ParsedSchema{Properties: map[string]mcpclient.PropertyInfo{}}

	result, err := parseToolArgs([]string{"--path", ".", "--name", "hello"}, emptySchema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["path"] != "." {
		t.Errorf("expected '.', got %v", result["path"])
	}
	if result["name"] != "hello" {
		t.Errorf("expected 'hello', got %v", result["name"])
	}
}

func TestParseToolArgsPassthroughEquals(t *testing.T) {
	emptySchema := mcpclient.ParsedSchema{Properties: map[string]mcpclient.PropertyInfo{}}

	result, err := parseToolArgs([]string{"--path=/tmp"}, emptySchema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["path"] != "/tmp" {
		t.Errorf("expected '/tmp', got %v", result["path"])
	}
}

func TestParseToolArgsPassthroughJSON(t *testing.T) {
	emptySchema := mcpclient.ParsedSchema{Properties: map[string]mcpclient.PropertyInfo{}}

	result, err := parseToolArgs([]string{"--data", `{"key":"val"}`}, emptySchema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result["data"])
	}
	if m["key"] != "val" {
		t.Errorf("expected 'val', got %v", m["key"])
	}
}

func TestParseToolArgsPassthroughScalarsAsStrings(t *testing.T) {
	emptySchema := mcpclient.ParsedSchema{Properties: map[string]mcpclient.PropertyInfo{}}

	result, err := parseToolArgs([]string{"--count", "42", "--flag", "true"}, emptySchema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Numbers and booleans stay as strings in passthrough mode
	if result["count"] != "42" {
		t.Errorf("expected string '42', got %v (%T)", result["count"], result["count"])
	}
	if result["flag"] != "true" {
		t.Errorf("expected string 'true', got %v (%T)", result["flag"], result["flag"])
	}
}

func TestParseToolArgsPassthroughMissingValue(t *testing.T) {
	emptySchema := mcpclient.ParsedSchema{Properties: map[string]mcpclient.PropertyInfo{}}

	_, err := parseToolArgs([]string{"--path"}, emptySchema)
	if err == nil {
		t.Error("expected error for missing value in passthrough mode")
	}
}

func TestParseToolArgsPassthroughNoPrefix(t *testing.T) {
	emptySchema := mcpclient.ParsedSchema{Properties: map[string]mcpclient.PropertyInfo{}}

	_, err := parseToolArgs([]string{"positional"}, emptySchema)
	if err == nil {
		t.Error("expected error for positional arg in passthrough mode")
	}
}

func TestParseToolArgsPassthroughEmpty(t *testing.T) {
	emptySchema := mcpclient.ParsedSchema{Properties: map[string]mcpclient.PropertyInfo{}}

	result, err := parseToolArgs([]string{}, emptySchema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestAutoParseValue(t *testing.T) {
	tests := []struct {
		input string
		isMap bool
		isArr bool
		isStr bool
	}{
		{`{"key":"val"}`, true, false, false},
		{`[1,2,3]`, false, true, false},
		{"hello", false, false, true},
		{"42", false, false, true},
		{"true", false, false, true},
	}
	for _, tt := range tests {
		result := autoParseValue(tt.input)
		switch {
		case tt.isMap:
			if _, ok := result.(map[string]interface{}); !ok {
				t.Errorf("autoParseValue(%q) expected map, got %T", tt.input, result)
			}
		case tt.isArr:
			if _, ok := result.([]interface{}); !ok {
				t.Errorf("autoParseValue(%q) expected slice, got %T", tt.input, result)
			}
		case tt.isStr:
			if _, ok := result.(string); !ok {
				t.Errorf("autoParseValue(%q) expected string, got %T", tt.input, result)
			}
		}
	}
}

func TestBuildToolCommandEmptySchema(t *testing.T) {
	serverCfg := &config.ServerConfig{Command: "node"}
	tool := mcpclient.ToolSchema{
		Name:        "empty_tool",
		Description: "Tool with empty schema",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}

	cmd := buildToolCommand("test-server", serverCfg, tool)
	if !cmd.SkipFlagParsing {
		t.Error("expected SkipFlagParsing to be true for empty schema")
	}
}

func TestBuildToolCommandWithSchema(t *testing.T) {
	serverCfg := &config.ServerConfig{Command: "node"}
	tool := mcpclient.ToolSchema{
		Name:        "typed_tool",
		Description: "Tool with schema",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`),
	}

	cmd := buildToolCommand("test-server", serverCfg, tool)
	if cmd.SkipFlagParsing {
		t.Error("expected SkipFlagParsing to be false for non-empty schema")
	}
}

func TestBuildToolCommandDefaultDescription(t *testing.T) {
	serverCfg := &config.ServerConfig{Command: "node"}
	tool := mcpclient.ToolSchema{
		Name:        "my_tool",
		Description: "",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"x":{"type":"string"}}}`),
	}

	cmd := buildToolCommand("test-server", serverCfg, tool)
	if cmd.Description != "Call the my_tool tool" {
		t.Errorf("expected default description, got %q", cmd.Description)
	}
}

func TestBuildToolCommandFlags(t *testing.T) {
	serverCfg := &config.ServerConfig{Command: "node"}
	tool := mcpclient.ToolSchema{
		Name:        "tool",
		Description: "desc",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"count":{"type":"integer"}},"required":["name"]}`),
	}

	cmd := buildToolCommand("srv", serverCfg, tool)
	if len(cmd.Flags) != 2 {
		t.Fatalf("expected 2 flags, got %d", len(cmd.Flags))
	}
}

func TestCollectArgs(t *testing.T) {
	schema := mcpclient.ParsedSchema{
		Properties: map[string]mcpclient.PropertyInfo{
			"name":    {Type: "string"},
			"count":   {Type: "integer"},
			"ratio":   {Type: "number"},
			"verbose": {Type: "boolean"},
			"data":    {Type: "object"},
		},
	}

	flags := schemaToFlags(schema)

	var collected map[string]interface{}
	app := &ucli.Command{
		Flags: flags,
		Action: func(ctx context.Context, cmd *ucli.Command) error {
			collected = collectArgs(cmd, schema)
			return nil
		},
	}

	err := app.Run(context.Background(), []string{"test",
		"--name", "hello",
		"--count", "42",
		"--ratio", "3.14",
		"--verbose",
		"--data", `{"key":"val"}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if collected["name"] != "hello" {
		t.Errorf("expected 'hello', got %v", collected["name"])
	}
	if collected["count"] != int64(42) {
		t.Errorf("expected 42, got %v", collected["count"])
	}
	if collected["ratio"] != 3.14 {
		t.Errorf("expected 3.14, got %v", collected["ratio"])
	}
	if collected["verbose"] != true {
		t.Errorf("expected true, got %v", collected["verbose"])
	}
	if m, ok := collected["data"].(map[string]interface{}); !ok || m["key"] != "val" {
		t.Errorf("expected {key:val}, got %v", collected["data"])
	}
}

func TestCollectArgsUnsetFlags(t *testing.T) {
	schema := mcpclient.ParsedSchema{
		Properties: map[string]mcpclient.PropertyInfo{
			"name":    {Type: "string"},
			"verbose": {Type: "boolean"},
		},
	}

	flags := schemaToFlags(schema)

	var collected map[string]interface{}
	app := &ucli.Command{
		Flags: flags,
		Action: func(ctx context.Context, cmd *ucli.Command) error {
			collected = collectArgs(cmd, schema)
			return nil
		},
	}

	// Run without setting any flags
	err := app.Run(context.Background(), []string{"test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(collected) != 0 {
		t.Errorf("expected empty args when no flags set, got %v", collected)
	}
}

func TestCollectArgsObjectFallback(t *testing.T) {
	schema := mcpclient.ParsedSchema{
		Properties: map[string]mcpclient.PropertyInfo{
			"data": {Type: "object"},
		},
	}

	flags := schemaToFlags(schema)

	var collected map[string]interface{}
	app := &ucli.Command{
		Flags: flags,
		Action: func(ctx context.Context, cmd *ucli.Command) error {
			collected = collectArgs(cmd, schema)
			return nil
		},
	}

	// Non-JSON string for object type falls back to raw string
	err := app.Run(context.Background(), []string{"test", "--data", "plain text"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if collected["data"] != "plain text" {
		t.Errorf("expected 'plain text', got %v", collected["data"])
	}
}
