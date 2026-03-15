package validate

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/volodymyrsmirnov/mcp-bin/internal/config"
)

func TestCheckEnvVars(t *testing.T) {
	tests := []struct {
		name   string
		srv    config.ServerConfig
		pass   bool
		substr string
	}{
		{
			name: "all resolved",
			srv: config.ServerConfig{
				Env: map[string]string{"TOKEN": "abc123"},
			},
			pass: true,
		},
		{
			name: "unresolved env var",
			srv: config.ServerConfig{
				Env: map[string]string{"TOKEN": "${MISSING_VAR}"},
			},
			pass:   false,
			substr: "MISSING_VAR",
		},
		{
			name: "unresolved header var",
			srv: config.ServerConfig{
				Headers: map[string]string{"Authorization": "Bearer ${API_TOKEN_MISSING}"},
			},
			pass:   false,
			substr: "API_TOKEN_MISSING",
		},
		{
			name: "no env or headers",
			srv:  config.ServerConfig{},
			pass: true,
		},
		{
			name: "mixed resolved and unresolved",
			srv: config.ServerConfig{
				Env: map[string]string{
					"GOOD": "resolved",
					"BAD":  "${NOT_SET}",
				},
			},
			pass:   false,
			substr: "NOT_SET",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			got := checkEnvVars(tt.srv, &buf)
			if got != tt.pass {
				t.Errorf("checkEnvVars() = %v, want %v\noutput: %s", got, tt.pass, buf.String())
			}
			if tt.substr != "" && !strings.Contains(buf.String(), tt.substr) {
				t.Errorf("output missing %q:\n%s", tt.substr, buf.String())
			}
		})
	}
}

func TestCheckLocalCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		pass    bool
		substr  string
	}{
		{
			name:    "existing command",
			command: "sh",
			pass:    true,
			substr:  "[PASS]",
		},
		{
			name:    "missing command",
			command: "nonexistent-command-xyz-12345",
			pass:    false,
			substr:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			srv := config.ServerConfig{Command: tt.command}
			got := checkLocalCommand(srv, &buf)
			if got != tt.pass {
				t.Errorf("checkLocalCommand() = %v, want %v\noutput: %s", got, tt.pass, buf.String())
			}
			if !strings.Contains(buf.String(), tt.substr) {
				t.Errorf("output missing %q:\n%s", tt.substr, buf.String())
			}
		})
	}
}

func TestCheckRemoteURL(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		pass   bool
		substr string
	}{
		{
			name:   "valid https",
			url:    "https://example.com/mcp",
			pass:   true,
			substr: "[PASS]",
		},
		{
			name:   "valid http",
			url:    "http://localhost:8080/mcp",
			pass:   true,
			substr: "[PASS]",
		},
		{
			name:   "bad scheme",
			url:    "ftp://example.com/mcp",
			pass:   false,
			substr: "not supported",
		},
		{
			name:   "no host",
			url:    "https://",
			pass:   false,
			substr: "no host",
		},
		{
			name:   "missing scheme",
			url:    "example.com/mcp",
			pass:   false,
			substr: "not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			srv := config.ServerConfig{URL: tt.url}
			got := checkRemoteURL(srv, &buf)
			if got != tt.pass {
				t.Errorf("checkRemoteURL() = %v, want %v\noutput: %s", got, tt.pass, buf.String())
			}
			if !strings.Contains(buf.String(), tt.substr) {
				t.Errorf("output missing %q:\n%s", tt.substr, buf.String())
			}
		})
	}
}

func TestCheckFiles(t *testing.T) {
	t.Run("existing file", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := &config.Config{Files: []string{"validate.go"}}
		checkFiles(cfg, &buf)
		if strings.Contains(buf.String(), "[WARN]") {
			t.Errorf("unexpected warning for existing file:\n%s", buf.String())
		}
	})

	t.Run("missing file", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := &config.Config{Files: []string{"/nonexistent/path/file.txt"}}
		checkFiles(cfg, &buf)
		if !strings.Contains(buf.String(), "[WARN]") {
			t.Errorf("expected warning for missing file:\n%s", buf.String())
		}
	})
}

func TestRunOutputOrdering(t *testing.T) {
	cfg := &config.Config{
		Servers: map[string]config.ServerConfig{
			"charlie": {Command: "sh"},
			"alpha":   {Command: "sh"},
			"bravo":   {Command: "sh"},
		},
	}
	var buf bytes.Buffer
	ok := Run(context.Background(), cfg, false, &buf)
	output := buf.String()

	if !ok {
		t.Fatalf("Run() returned false, want true\noutput: %s", output)
	}

	alphaIdx := strings.Index(output, `"alpha"`)
	bravoIdx := strings.Index(output, `"bravo"`)
	charlieIdx := strings.Index(output, `"charlie"`)

	if alphaIdx >= bravoIdx || bravoIdx >= charlieIdx {
		t.Errorf("servers not in alphabetical order:\n%s", output)
	}
	if !strings.Contains(output, "3 server(s) passed, 0 server(s) failed") {
		t.Errorf("unexpected summary:\n%s", output)
	}
}

func TestOutputHelpers(t *testing.T) {
	tests := []struct {
		name   string
		fn     func(io.Writer, string)
		prefix string
	}{
		{"pass", pass, "[PASS]"},
		{"fail", fail, "[FAIL]"},
		{"warn", warn, "[WARN]"},
		{"skip", skip, "[SKIP]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.fn(&buf, "test message")
			if !strings.Contains(buf.String(), tt.prefix) {
				t.Errorf("output missing %q: %s", tt.prefix, buf.String())
			}
			if !strings.Contains(buf.String(), "test message") {
				t.Errorf("output missing message: %s", buf.String())
			}
		})
	}
}

func TestCheckOAuth(t *testing.T) {
	tests := []struct {
		name   string
		srv    config.ServerConfig
		pass   bool
		substr string
	}{
		{
			name: "valid oauth on remote server",
			srv: config.ServerConfig{
				URL:   "https://example.com/mcp",
				OAuth: &config.OAuthConfig{ClientID: "my-id"},
			},
			pass:   true,
			substr: "OAuth configured",
		},
		{
			name: "oauth on local server",
			srv: config.ServerConfig{
				Command: "node",
				OAuth:   &config.OAuthConfig{ClientID: "my-id"},
			},
			pass:   false,
			substr: "only supported for remote",
		},
		{
			name: "unresolved client_id env var",
			srv: config.ServerConfig{
				URL:   "https://example.com/mcp",
				OAuth: &config.OAuthConfig{ClientID: "${MISSING_CLIENT_ID}"},
			},
			pass:   false,
			substr: "MISSING_CLIENT_ID",
		},
		{
			name: "unresolved client_secret env var",
			srv: config.ServerConfig{
				URL:   "https://example.com/mcp",
				OAuth: &config.OAuthConfig{ClientID: "id", ClientSecret: "${MISSING_SECRET}"},
			},
			pass:   false,
			substr: "MISSING_SECRET",
		},
		{
			name: "empty oauth (dynamic registration)",
			srv: config.ServerConfig{
				URL:   "https://example.com/mcp",
				OAuth: &config.OAuthConfig{},
			},
			pass:   true,
			substr: "OAuth configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			got := checkOAuth(tt.srv, &buf)
			if got != tt.pass {
				t.Errorf("checkOAuth() = %v, want %v\noutput: %s", got, tt.pass, buf.String())
			}
			if !strings.Contains(buf.String(), tt.substr) {
				t.Errorf("output missing %q:\n%s", tt.substr, buf.String())
			}
		})
	}
}
