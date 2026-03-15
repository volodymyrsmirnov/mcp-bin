package validate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"sync"

	"github.com/volodymyrsmirnov/mcp-bin/internal/config"
	mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
)

// Run validates the configuration and optionally tests server connectivity.
// Returns true if all checks pass. Servers are validated in parallel.
func Run(ctx context.Context, cfg *config.Config, connect bool, w io.Writer) bool {
	if len(cfg.Files) > 0 {
		checkFiles(cfg, w)
	}

	names := config.SortedServerNames(cfg)

	type serverResult struct {
		name   string
		ok     bool
		output string
	}

	results := make(chan serverResult, len(cfg.Servers))
	var wg sync.WaitGroup

	for _, name := range names {
		wg.Add(1)
		go func(name string, srv config.ServerConfig) {
			defer wg.Done()
			var buf bytes.Buffer
			ok := checkServer(ctx, name, srv, connect, &buf)
			results <- serverResult{name: name, ok: ok, output: buf.String()}
		}(name, cfg.Servers[name])
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	resultMap := make(map[string]serverResult, len(cfg.Servers))
	for res := range results {
		resultMap[res.name] = res
	}

	passed, failed := 0, 0
	for _, name := range names {
		res := resultMap[name]
		_, _ = fmt.Fprint(w, res.output)
		if res.ok {
			passed++
		} else {
			failed++
		}
	}

	_, _ = fmt.Fprintf(w, "\nValidation complete: %d server(s) passed, %d server(s) failed.\n", passed, failed)
	return failed == 0
}

func checkServer(ctx context.Context, name string, srv config.ServerConfig, connect bool, w io.Writer) bool {
	if srv.IsLocal() {
		_, _ = fmt.Fprintf(w, "Validating server %q (local: %s)...\n", name, srv.Command)
	} else if srv.IsRemote() {
		_, _ = fmt.Fprintf(w, "Validating server %q (remote: %s)...\n", name, srv.URL)
	} else {
		_, _ = fmt.Fprintf(w, "Validating server %q...\n", name)
		fail(w, "server has neither command nor url configured")
		return false
	}

	ok := true

	if srv.IsLocal() {
		if !checkLocalCommand(srv, w) {
			ok = false
		}
	}
	if srv.IsRemote() {
		if !checkRemoteURL(srv, w) {
			ok = false
		}
	}
	if !checkEnvVars(srv, w) {
		ok = false
	}
	if srv.OAuth != nil {
		if !checkOAuth(srv, w) {
			ok = false
		}
	}

	if connect {
		if !checkConnectivity(ctx, name, srv, w) {
			ok = false
		}
	} else {
		skip(w, "connection test (use --connect)")
	}

	return ok
}

func checkEnvVars(srv config.ServerConfig, w io.Writer) bool {
	ok := true
	for k, v := range srv.Env {
		if matches := config.EnvVarPattern.FindAllStringSubmatch(v, -1); len(matches) > 0 {
			for _, m := range matches {
				fail(w, fmt.Sprintf("env var %s not set (in env.%s)", m[1], k))
			}
			ok = false
		}
	}
	for k, v := range srv.Headers {
		if matches := config.EnvVarPattern.FindAllStringSubmatch(v, -1); len(matches) > 0 {
			for _, m := range matches {
				fail(w, fmt.Sprintf("env var %s not set (in headers.%s)", m[1], k))
			}
			ok = false
		}
	}
	if ok {
		pass(w, "environment variables resolved")
	}
	return ok
}

func checkLocalCommand(srv config.ServerConfig, w io.Writer) bool {
	path, err := exec.LookPath(srv.Command)
	if err != nil {
		fail(w, fmt.Sprintf("command %q not found in PATH", srv.Command))
		return false
	}
	pass(w, fmt.Sprintf("command %q found at %s", srv.Command, path))
	return true
}

func checkRemoteURL(srv config.ServerConfig, w io.Writer) bool {
	u, err := url.Parse(srv.URL)
	if err != nil {
		fail(w, fmt.Sprintf("invalid URL: %v", err))
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		fail(w, fmt.Sprintf("URL scheme %q not supported (use http or https)", u.Scheme))
		return false
	}
	if u.Host == "" {
		fail(w, "URL has no host")
		return false
	}
	pass(w, "URL format valid")
	return true
}

func checkOAuth(srv config.ServerConfig, w io.Writer) bool {
	if srv.IsLocal() {
		fail(w, "OAuth is only supported for remote servers (url)")
		return false
	}
	o := srv.OAuth
	if o.ClientID != "" {
		if matches := config.EnvVarPattern.FindAllStringSubmatch(o.ClientID, -1); len(matches) > 0 {
			for _, m := range matches {
				fail(w, fmt.Sprintf("env var %s not set (in oauth.client_id)", m[1]))
			}
			return false
		}
	}
	if o.ClientSecret != "" {
		if matches := config.EnvVarPattern.FindAllStringSubmatch(o.ClientSecret, -1); len(matches) > 0 {
			for _, m := range matches {
				fail(w, fmt.Sprintf("env var %s not set (in oauth.client_secret)", m[1]))
			}
			return false
		}
	}
	pass(w, "OAuth configured")
	return true
}

func checkFiles(cfg *config.Config, w io.Writer) {
	for _, f := range cfg.Files {
		if _, err := os.Stat(f); err != nil {
			warn(w, fmt.Sprintf("file %q: %v", f, err))
		}
	}
}

func checkConnectivity(ctx context.Context, name string, srv config.ServerConfig, w io.Writer) bool {
	c, err := mcpclient.Connect(ctx, srv)
	if err != nil {
		fail(w, fmt.Sprintf("connection failed: %v", err))
		return false
	}
	defer c.Close()

	tools, err := c.ListTools(ctx)
	if err != nil {
		fail(w, fmt.Sprintf("listing tools failed: %v", err))
		return false
	}

	total := len(tools)
	schemas, err := mcpclient.ToolsToSchemas(tools)
	if err != nil {
		fail(w, fmt.Sprintf("converting tool schemas: %v", err))
		return false
	}

	filtered := mcpclient.FilterSchemas(schemas, srv.AllowTools, srv.DenyTools)
	pass(w, fmt.Sprintf("connected, %d tool(s) available", total))

	if len(filtered) < total {
		warn(w, fmt.Sprintf("tool filters reduce %d to %d tool(s)", total, len(filtered)))
	}

	return true
}

func pass(w io.Writer, msg string) {
	_, _ = fmt.Fprintf(w, "  [PASS] %s\n", msg)
}

func fail(w io.Writer, msg string) {
	_, _ = fmt.Fprintf(w, "  [FAIL] %s\n", msg)
}

func warn(w io.Writer, msg string) {
	_, _ = fmt.Fprintf(w, "  [WARN] %s\n", msg)
}

func skip(w io.Writer, msg string) {
	_, _ = fmt.Fprintf(w, "  [SKIP] %s\n", msg)
}
