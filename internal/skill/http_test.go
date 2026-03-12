package skill

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
)

func TestGenerateHTTP(t *testing.T) {
	var buf bytes.Buffer
	GenerateHTTP(&buf, testManifest(), "my-tool", "", "https://example.com", "secret123")

	out := buf.String()

	// Front matter
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
	if !strings.Contains(out, "Base URL: `https://example.com`") {
		t.Error("expected base URL")
	}

	// Authentication
	if !strings.Contains(out, "## Authentication") {
		t.Error("expected Authentication section")
	}
	if !strings.Contains(out, "Authorization: Bearer secret123") {
		t.Error("expected bearer token in auth section")
	}

	// Endpoints
	if !strings.Contains(out, "## Endpoints") {
		t.Error("expected Endpoints section")
	}
	if !strings.Contains(out, "GET /") {
		t.Error("expected GET / endpoint")
	}
	if !strings.Contains(out, "GET /{server}/") {
		t.Error("expected GET /{server}/ endpoint")
	}
	if !strings.Contains(out, "POST /{server}/{tool}") {
		t.Error("expected POST /{server}/{tool} endpoint")
	}

	// Server sections
	if !strings.Contains(out, "## fetch") {
		t.Error("expected fetch server section")
	}
	if !strings.Contains(out, "## filesystem") {
		t.Error("expected filesystem server section")
	}

	// Tools - HTTP mode uses param names without -- prefix
	if !strings.Contains(out, "`url` string (required)") {
		t.Error("expected url parameter without -- prefix")
	}

	// Usage with curl
	if !strings.Contains(out, "## Usage") {
		t.Error("expected Usage section")
	}
	if !strings.Contains(out, "curl") {
		t.Error("expected curl examples")
	}
	if !strings.Contains(out, "https://example.com/") {
		t.Error("expected base URL in curl examples")
	}
	if !strings.Contains(out, "-H \"Authorization: Bearer secret123\"") {
		t.Error("expected auth header in curl examples")
	}
	if !strings.Contains(out, "-X POST") {
		t.Error("expected POST example")
	}
	if !strings.Contains(out, "Content-Type: application/json") {
		t.Error("expected Content-Type header in POST example")
	}
}

func TestGenerateHTTPNoToken(t *testing.T) {
	var buf bytes.Buffer
	GenerateHTTP(&buf, testManifest(), "my-tool", "", "https://example.com", "")

	out := buf.String()

	if strings.Contains(out, "## Authentication") {
		t.Error("expected no Authentication section when token is empty")
	}
	if strings.Contains(out, "Authorization: Bearer") {
		t.Error("expected no auth header in examples when token is empty")
	}
}

func TestGenerateHTTPCustomDescription(t *testing.T) {
	var buf bytes.Buffer
	GenerateHTTP(&buf, testManifest(), "my-tool", "Custom HTTP API description", "https://example.com", "")

	out := buf.String()
	if !strings.Contains(out, "description: Custom HTTP API description") {
		t.Error("expected custom description in front matter")
	}
}

func TestGenerateHTTPAutoDescription(t *testing.T) {
	tests := []struct {
		name    string
		servers map[string][]mcpclient.ToolSchema
		want    string
	}{
		{
			name:    "no servers",
			servers: map[string][]mcpclient.ToolSchema{},
			want:    "HTTP API wrapping MCP servers",
		},
		{
			name: "one server",
			servers: map[string][]mcpclient.ToolSchema{
				"fetch": {},
			},
			want: "HTTP API to work with fetch",
		},
		{
			name: "two servers",
			servers: map[string][]mcpclient.ToolSchema{
				"fetch": {},
				"time":  {},
			},
			want: "HTTP API to work with fetch, time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			manifest := &mcpclient.Manifest{Servers: tt.servers}
			GenerateHTTP(&buf, manifest, "name", "", "https://example.com", "")
			if !strings.Contains(buf.String(), "description: "+tt.want) {
				t.Errorf("expected description %q in output:\n%s", tt.want, buf.String())
			}
		})
	}
}

func TestGenerateHTTPTrailingSlash(t *testing.T) {
	var buf bytes.Buffer
	GenerateHTTP(&buf, testManifest(), "my-tool", "", "https://example.com/", "")

	out := buf.String()
	// Should not have double slash
	if !strings.Contains(out, "Base URL: `https://example.com`") {
		t.Error("expected trailing slash stripped from base URL")
	}
}

func TestGenerateHTTPEmptyManifest(t *testing.T) {
	var buf bytes.Buffer
	manifest := &mcpclient.Manifest{Servers: map[string][]mcpclient.ToolSchema{}}
	GenerateHTTP(&buf, manifest, "name", "", "https://example.com", "token")

	out := buf.String()
	if !strings.Contains(out, "---") {
		t.Error("expected front matter even with empty manifest")
	}
	if !strings.Contains(out, "# name") {
		t.Error("expected heading even with empty manifest")
	}
}

func TestGenerateHTTPPostExample(t *testing.T) {
	manifest := &mcpclient.Manifest{
		Servers: map[string][]mcpclient.ToolSchema{
			"myserver": {
				{
					Name:        "greet",
					Description: "Greets a user",
					InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string","description":"User name"}},"required":["name"]}`),
				},
			},
		},
	}

	var buf bytes.Buffer
	GenerateHTTP(&buf, manifest, "my-tool", "", "https://api.example.com", "tok123")

	out := buf.String()
	if !strings.Contains(out, `"name": "<name>"`) {
		t.Error("expected JSON body with required param placeholder")
	}
	if !strings.Contains(out, "https://api.example.com/myserver/greet") {
		t.Error("expected full URL in POST example")
	}
}
