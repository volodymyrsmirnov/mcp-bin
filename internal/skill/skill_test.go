package skill

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
)

func testManifest() *mcpclient.Manifest {
	return &mcpclient.Manifest{
		Servers: map[string][]mcpclient.ToolSchema{
			"fetch": {
				{
					Name:        "fetch",
					Description: "Fetches a URL from the internet",
					InputSchema: json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","description":"URL to fetch"},"raw":{"type":"boolean","description":"Return raw content"}},"required":["url"]}`),
				},
			},
			"filesystem": {
				{
					Name:        "read_file",
					Description: "Read complete contents of a file",
					InputSchema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"File path"}},"required":["path"]}`),
				},
				{
					Name:        "write_file",
					Description: "Write content to a file",
					InputSchema: json.RawMessage(`{"type":"object","properties":{"content":{"type":"string","description":"Content to write"},"path":{"type":"string","description":"File path"}},"required":["content","path"]}`),
				},
			},
		},
	}
}

func TestGenerate(t *testing.T) {
	var buf bytes.Buffer
	Generate(&buf, testManifest(), "mcp-bin", "my-tool", "")

	out := buf.String()

	// Front matter
	if !strings.Contains(out, "---\n") {
		t.Error("expected YAML front matter delimiters")
	}
	if !strings.Contains(out, "name: my-tool") {
		t.Error("expected skill name in front matter")
	}
	if !strings.Contains(out, "version:") {
		t.Error("expected version in front matter")
	}

	// Header
	if !strings.Contains(out, "# my-tool") {
		t.Error("expected H1 heading")
	}
	if !strings.Contains(out, "mcp-bin <server> --help") {
		t.Error("expected server-level --help hint with binary name")
	}
	if !strings.Contains(out, "mcp-bin <server> <tool> --help") {
		t.Error("expected tool-level --help hint with binary name")
	}

	// Server sections
	if !strings.Contains(out, "## fetch") {
		t.Error("expected fetch server section")
	}
	if !strings.Contains(out, "## filesystem") {
		t.Error("expected filesystem server section")
	}

	// Tools
	if !strings.Contains(out, "`fetch`") {
		t.Error("expected fetch tool")
	}
	if !strings.Contains(out, "`read_file`") {
		t.Error("expected read_file tool")
	}
	if !strings.Contains(out, "`write_file`") {
		t.Error("expected write_file tool")
	}

	// Flags
	if !strings.Contains(out, "`--url` string (required)") {
		t.Error("expected --url flag with required")
	}
	if !strings.Contains(out, "`--raw` boolean") {
		t.Error("expected --raw flag")
	}

	// Usage section
	if !strings.Contains(out, "## Usage") {
		t.Error("expected Usage section")
	}
	if !strings.Contains(out, "mcp-bin fetch --help") {
		t.Error("expected server --help example")
	}
	if !strings.Contains(out, "mcp-bin fetch fetch --help") {
		t.Error("expected tool --help example")
	}
	if !strings.Contains(out, "mcp-bin fetch fetch --url <url>") {
		t.Error("expected fetch usage example")
	}
}

func TestGenerateCustomDescription(t *testing.T) {
	var buf bytes.Buffer
	Generate(&buf, testManifest(), "my-binary", "my-tool", "Custom description here")

	out := buf.String()
	if !strings.Contains(out, "description: Custom description here") {
		t.Error("expected custom description in front matter")
	}
}

func TestGenerateAutoDescription(t *testing.T) {
	tests := []struct {
		name    string
		servers map[string][]mcpclient.ToolSchema
		want    string
	}{
		{
			name:    "no servers",
			servers: map[string][]mcpclient.ToolSchema{},
			want:    "CLI tool wrapping MCP servers",
		},
		{
			name: "one server",
			servers: map[string][]mcpclient.ToolSchema{
				"fetch": {},
			},
			want: "CLI tool to work with fetch",
		},
		{
			name: "two servers",
			servers: map[string][]mcpclient.ToolSchema{
				"fetch": {},
				"time":  {},
			},
			want: "CLI tool to work with fetch, time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			manifest := &mcpclient.Manifest{Servers: tt.servers}
			Generate(&buf, manifest, "bin", "name", "")
			if !strings.Contains(buf.String(), "description: "+tt.want) {
				t.Errorf("expected description %q in output:\n%s", tt.want, buf.String())
			}
		})
	}
}

func TestGenerateSortOrder(t *testing.T) {
	manifest := &mcpclient.Manifest{
		Servers: map[string][]mcpclient.ToolSchema{
			"zebra": {
				{Name: "z_tool", Description: "Z tool", InputSchema: json.RawMessage(`{"type":"object"}`)},
				{Name: "a_tool", Description: "A tool", InputSchema: json.RawMessage(`{"type":"object"}`)},
			},
			"alpha": {
				{Name: "tool1", Description: "Tool 1", InputSchema: json.RawMessage(`{"type":"object"}`)},
			},
		},
	}

	var buf bytes.Buffer
	Generate(&buf, manifest, "bin", "name", "desc")
	out := buf.String()

	// alpha should come before zebra
	alphaIdx := strings.Index(out, "## alpha")
	zebraIdx := strings.Index(out, "## zebra")
	if alphaIdx >= zebraIdx {
		t.Error("expected alpha before zebra")
	}

	// a_tool should come before z_tool within zebra
	aToolIdx := strings.Index(out, "`a_tool`")
	zToolIdx := strings.Index(out, "`z_tool`")
	if aToolIdx >= zToolIdx {
		t.Error("expected a_tool before z_tool")
	}
}

func TestGenerateEmptyManifest(t *testing.T) {
	var buf bytes.Buffer
	manifest := &mcpclient.Manifest{Servers: map[string][]mcpclient.ToolSchema{}}
	Generate(&buf, manifest, "bin", "name", "")

	out := buf.String()
	if !strings.Contains(out, "---") {
		t.Error("expected front matter even with empty manifest")
	}
	if !strings.Contains(out, "# name") {
		t.Error("expected heading even with empty manifest")
	}
}

func TestGenerateKebabCaseName(t *testing.T) {
	var buf bytes.Buffer
	Generate(&buf, testManifest(), "oracle", "Peak Ventures Oracle", "A tool for peak ventures")

	out := buf.String()

	// Front matter name should be kebab-case
	if !strings.Contains(out, "name: peak-ventures-oracle") {
		t.Error("expected kebab-case name in front matter")
	}

	// Header should keep the original human-readable name
	if !strings.Contains(out, "# Peak Ventures Oracle") {
		t.Error("expected original name in heading")
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Peak Ventures Oracle", "peak-ventures-oracle"},
		{"my-tool", "my-tool"},
		{"My Tool", "my-tool"},
		{"  spaces  everywhere  ", "spaces-everywhere"},
		{"MixedCASE", "mixedcase"},
		{"already-kebab-case", "already-kebab-case"},
		{"with--multiple---dashes", "with-multiple-dashes"},
		{"special!@#chars", "special-chars"},
	}

	for _, tt := range tests {
		got := toKebabCase(tt.input)
		if got != tt.want {
			t.Errorf("toKebabCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFirstLine(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"first\nsecond", "first"},
		{"  padded  ", "padded"},
		{"", ""},
		{"line1\n\nline3", "line1"},
	}

	for _, tt := range tests {
		got := firstLine(tt.input)
		if got != tt.want {
			t.Errorf("firstLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
