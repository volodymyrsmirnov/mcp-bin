package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/volodymyrsmirnov/mcp-bin/internal/config"
	mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
)

// server holds state for the HTTP API server.
type server struct {
	cfg      *config.Config
	manifest *mcpclient.Manifest
	token    string
}

// Run starts the HTTP API server and blocks until ctx is cancelled.
func Run(ctx context.Context, cfg *config.Config, manifest *mcpclient.Manifest, listenAddr, token string) error {
	s := &server{cfg: cfg, manifest: manifest, token: token}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleListServers)
	mux.HandleFunc("GET /{server}/", s.handleListTools)
	mux.HandleFunc("GET /{server}/{tool}", s.handleToolInfo)
	mux.HandleFunc("POST /{server}/{tool}", s.handleCallTool)

	var handler http.Handler = mux
	if token != "" {
		handler = s.authMiddleware(handler)
	}

	srv := &http.Server{
		Addr:    listenAddr,
		Handler: handler,
	}

	errCh := make(chan error, 1)
	go func() {
		_, _ = fmt.Fprintf(os.Stderr, "Listening on %s\n", listenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func (s *server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || auth != "Bearer "+s.token {
			writeError(w, http.StatusUnauthorized, "unauthorized: missing or invalid bearer token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) handleListServers(w http.ResponseWriter, _ *http.Request) {
	names := sortedServerNames(s.manifest)
	servers := make([]ServerInfo, 0, len(names))
	for _, name := range names {
		tools := s.manifest.Servers[name]
		desc := serverDescription(tools)
		servers = append(servers, ServerInfo{Name: name, Description: desc})
	}
	writeJSON(w, http.StatusOK, servers)
}

func (s *server) handleListTools(w http.ResponseWriter, r *http.Request) {
	serverName := r.PathValue("server")
	tools, ok := s.manifest.Servers[serverName]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("server not found: %s", serverName))
		return
	}
	writeJSON(w, http.StatusOK, buildToolInfoList(tools))
}

func (s *server) handleToolInfo(w http.ResponseWriter, r *http.Request) {
	serverName := r.PathValue("server")
	toolName := r.PathValue("tool")

	tools, ok := s.manifest.Servers[serverName]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("server not found: %s", serverName))
		return
	}

	for _, tool := range tools {
		if tool.Name == toolName {
			writeJSON(w, http.StatusOK, buildToolInfo(tool))
			return
		}
	}

	writeError(w, http.StatusNotFound, fmt.Sprintf("tool not found: %s in server %s", toolName, serverName))
}

func (s *server) handleCallTool(w http.ResponseWriter, r *http.Request) {
	serverName := r.PathValue("server")
	toolName := r.PathValue("tool")

	tools, ok := s.manifest.Servers[serverName]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("server not found: %s", serverName))
		return
	}

	var found *mcpclient.ToolSchema
	for i := range tools {
		if tools[i].Name == toolName {
			found = &tools[i]
			break
		}
	}
	if found == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("tool not found: %s in server %s", toolName, serverName))
		return
	}

	var args map[string]interface{}
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON body: %v", err))
			return
		}
	}
	if args == nil {
		args = make(map[string]interface{})
	}

	// Validate required parameters
	schema := mcpclient.ParseInputSchema(found.InputSchema)
	for _, req := range schema.Required {
		if _, ok := args[req]; !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("missing required parameter: %s", req))
			return
		}
	}

	serverCfg := s.cfg.Servers[serverName]
	client, err := mcpclient.Connect(r.Context(), serverCfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("connecting to %s: %v", serverName, err))
		return
	}
	defer client.Close()

	result, err := client.CallTool(r.Context(), toolName, args)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("calling tool %s: %v", toolName, err))
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func buildToolInfoList(tools []mcpclient.ToolSchema) []ToolInfo {
	infos := make([]ToolInfo, 0, len(tools))
	sorted := make([]mcpclient.ToolSchema, len(tools))
	copy(sorted, tools)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	for _, tool := range sorted {
		infos = append(infos, buildToolInfo(tool))
	}
	return infos
}

func buildToolInfo(tool mcpclient.ToolSchema) ToolInfo {
	schema := mcpclient.ParseInputSchema(tool.InputSchema)
	requiredSet := make(map[string]bool)
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	params := make([]ParamInfo, 0, len(schema.Properties))
	names := mcpclient.SortedKeys(schema.Properties)
	for _, name := range names {
		prop := schema.Properties[name]
		typ := prop.Type
		if typ == "" {
			typ = "string"
		}
		params = append(params, ParamInfo{
			Name:        name,
			Type:        typ,
			Required:    requiredSet[name],
			Description: prop.Description,
		})
	}

	return ToolInfo{
		Name:        tool.Name,
		Description: tool.Description,
		Parameters:  params,
	}
}

func serverDescription(tools []mcpclient.ToolSchema) string {
	if len(tools) == 0 {
		return ""
	}
	sorted := make([]mcpclient.ToolSchema, len(tools))
	copy(sorted, tools)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	names := make([]string, len(sorted))
	for i, t := range sorted {
		names[i] = t.Name
	}
	return "Provides tools: " + strings.Join(names, ", ")
}

func sortedServerNames(manifest *mcpclient.Manifest) []string {
	names := make([]string, 0, len(manifest.Servers))
	for name := range manifest.Servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}
