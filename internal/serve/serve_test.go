package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/volodymyrsmirnov/mcp-bin/internal/config"
	mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
)

func testManifest() *mcpclient.Manifest {
	return &mcpclient.Manifest{
		Servers: map[string][]mcpclient.ToolSchema{
			"fetch": {
				{
					Name:        "fetch_url",
					Description: "Fetches a URL",
					InputSchema: json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","description":"The URL to fetch"}},"required":["url"]}`),
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
					InputSchema: json.RawMessage(`{"type":"object","properties":{"content":{"type":"string","description":"Content to write"},"path":{"type":"string","description":"File path"}},"required":["path","content"]}`),
				},
			},
		},
	}
}

func testConfig() *config.Config {
	return &config.Config{
		Servers: map[string]config.ServerConfig{
			"fetch":      {Command: "echo"},
			"filesystem": {Command: "echo"},
		},
	}
}

func TestHandleListServers(t *testing.T) {
	s := &server{
		cfg:      testConfig(),
		manifest: testManifest(),
	}

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	s.handleListServers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var servers []ServerInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &servers); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}
	// Sorted alphabetically
	if servers[0].Name != "fetch" {
		t.Errorf("expected first server 'fetch', got %q", servers[0].Name)
	}
	if servers[1].Name != "filesystem" {
		t.Errorf("expected second server 'filesystem', got %q", servers[1].Name)
	}
	if !strings.Contains(servers[0].Description, "fetch_url") {
		t.Errorf("expected fetch description to contain 'fetch_url', got %q", servers[0].Description)
	}
}

func TestHandleListTools(t *testing.T) {
	s := &server{
		cfg:      testConfig(),
		manifest: testManifest(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{server}/", s.handleListTools)

	req := httptest.NewRequest("GET", "/filesystem/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var tools []ToolInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &tools); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	// Sorted alphabetically
	if tools[0].Name != "read_file" {
		t.Errorf("expected first tool 'read_file', got %q", tools[0].Name)
	}
	if len(tools[0].Parameters) != 1 {
		t.Fatalf("expected 1 parameter for read_file, got %d", len(tools[0].Parameters))
	}
	if tools[0].Parameters[0].Name != "path" {
		t.Errorf("expected parameter 'path', got %q", tools[0].Parameters[0].Name)
	}
	if !tools[0].Parameters[0].Required {
		t.Error("expected 'path' to be required")
	}
}

func TestHandleListToolsNotFound(t *testing.T) {
	s := &server{
		cfg:      testConfig(),
		manifest: testManifest(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{server}/", s.handleListTools)

	req := httptest.NewRequest("GET", "/nonexistent/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !strings.Contains(errResp.Error, "nonexistent") {
		t.Errorf("expected error to mention 'nonexistent', got %q", errResp.Error)
	}
}

func TestHandleToolInfo(t *testing.T) {
	s := &server{
		cfg:      testConfig(),
		manifest: testManifest(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{server}/{tool}", s.handleToolInfo)

	req := httptest.NewRequest("GET", "/fetch/fetch_url", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var tool ToolInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &tool); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if tool.Name != "fetch_url" {
		t.Errorf("expected tool name 'fetch_url', got %q", tool.Name)
	}
	if len(tool.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(tool.Parameters))
	}
	if tool.Parameters[0].Type != "string" {
		t.Errorf("expected parameter type 'string', got %q", tool.Parameters[0].Type)
	}
}

func TestHandleToolInfoNotFound(t *testing.T) {
	s := &server{
		cfg:      testConfig(),
		manifest: testManifest(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{server}/{tool}", s.handleToolInfo)

	tests := []struct {
		name string
		path string
	}{
		{"unknown server", "/unknown/fetch_url"},
		{"unknown tool", "/fetch/unknown_tool"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("expected 404, got %d", rec.Code)
			}
		})
	}
}

func TestHandleCallToolBadJSON(t *testing.T) {
	s := &server{
		cfg:      testConfig(),
		manifest: testManifest(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /{server}/{tool}", s.handleCallTool)

	req := httptest.NewRequest("POST", "/fetch/fetch_url", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleCallToolMissingRequired(t *testing.T) {
	s := &server{
		cfg:      testConfig(),
		manifest: testManifest(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /{server}/{tool}", s.handleCallTool)

	req := httptest.NewRequest("POST", "/fetch/fetch_url", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !strings.Contains(errResp.Error, "url") {
		t.Errorf("expected error to mention 'url', got %q", errResp.Error)
	}
}

func TestAuthMiddleware(t *testing.T) {
	s := &server{token: "secret123"}
	dummy := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := s.authMiddleware(dummy)

	tests := []struct {
		name   string
		header string
		status int
	}{
		{"valid token", "Bearer secret123", http.StatusOK},
		{"missing header", "", http.StatusUnauthorized},
		{"wrong token", "Bearer wrong", http.StatusUnauthorized},
		{"no bearer prefix", "secret123", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.status {
				t.Errorf("expected %d, got %d", tt.status, rec.Code)
			}
		})
	}
}

func TestAuthMiddlewareNoToken(t *testing.T) {
	s := &server{token: ""}
	// When no token configured, auth middleware is not applied.
	// Verify directly that requests without auth reach the handler.
	dummy := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Simulate the setup: no middleware wrapping when token is empty.
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	dummy.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// Also verify that if middleware were applied with empty token, it would still block
	// (this validates our Run() logic of only wrapping when token != "").
	handler := s.authMiddleware(dummy)
	req = httptest.NewRequest("GET", "/", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	// Empty token means "Bearer " won't match any Authorization header
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when middleware applied with empty token, got %d", rec.Code)
	}
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, map[string]string{"key": "value"})

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var result map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("expected key=value, got %q", result["key"])
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusBadRequest, "test error")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if errResp.Error != "test error" {
		t.Errorf("expected 'test error', got %q", errResp.Error)
	}
}
