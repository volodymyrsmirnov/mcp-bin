package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/volodymyrsmirnov/mcp-bin/internal/config"
)

// Client wraps an MCP client connection.
type Client struct {
	mcpClient *client.Client
}

// Connect creates and initializes an MCP client for the given server config.
func Connect(ctx context.Context, cfg config.ServerConfig) (*Client, error) {
	if cfg.IsLocal() {
		return connectStdio(ctx, cfg)
	}
	return connectRemote(ctx, cfg)
}

func connectStdio(ctx context.Context, cfg config.ServerConfig) (*Client, error) {
	// Use a short-lived init context for Start + Initialize only.
	// The transport outlives this context — we must not cancel the connection context.
	initCtx, initCancel := withDefaultTimeout(ctx, 3*time.Minute)
	defer initCancel()

	var env []string
	if cfg.Env != nil {
		env = make([]string, 0, len(cfg.Env))
		for k, v := range cfg.Env {
			env = append(env, k+"="+v)
		}
	}

	var opts []transport.StdioOption
	opts = append(opts, transport.WithCommandLogger(&nopLogger{}))
	if cfg.Cwd != "" {
		cwd := cfg.Cwd
		opts = append(opts, transport.WithCommandFunc(
			func(ctx context.Context, command string, cmdEnv []string, args []string) (*exec.Cmd, error) {
				cmd := exec.CommandContext(ctx, command, args...)
				cmd.Dir = cwd
				cmd.Env = append(os.Environ(), cmdEnv...)
				return cmd, nil
			},
		))
	}

	t := transport.NewStdioWithOptions(cfg.Command, env, cfg.Args, opts...)

	c := client.NewClient(t)
	if err := c.Start(initCtx); err != nil {
		return nil, stderrError(fmt.Errorf("starting stdio client: %w", err), t.Stderr())
	}

	if err := initialize(initCtx, c); err != nil {
		connErr := stderrError(err, t.Stderr())
		_ = c.Close()
		return nil, connErr
	}

	return &Client{mcpClient: c}, nil
}

// stderrError appends captured stderr output to a connection error.
// Reads at most 4KB with a 2-second timeout to avoid hanging on
// servers that keep stderr open or produce unbounded output.
func stderrError(err error, stderr io.Reader) error {
	if stderr == nil {
		return err
	}
	ch := make(chan []byte, 1)
	go func() {
		out, _ := io.ReadAll(io.LimitReader(stderr, 4096))
		ch <- out
	}()
	select {
	case out := <-ch:
		if len(out) > 0 {
			return fmt.Errorf("%w\nserver stderr:\n%s", err, strings.TrimSpace(string(out)))
		}
	case <-time.After(2 * time.Second):
	}
	return err
}

func connectRemote(ctx context.Context, cfg config.ServerConfig) (*Client, error) {
	// Use a short-lived init context for Start + Initialize only.
	initCtx, initCancel := withDefaultTimeout(ctx, 3*time.Minute)
	defer initCancel()

	// Try Streamable HTTP first
	c, err := tryStreamableHTTP(initCtx, cfg)
	if err == nil {
		return c, nil
	}

	// If the server indicated it's a legacy SSE server, try SSE
	if errors.Is(err, transport.ErrLegacySSEServer) {
		return trySSE(initCtx, cfg)
	}

	// For other errors, return directly — don't mask real connection problems
	return nil, err
}

func tryStreamableHTTP(ctx context.Context, cfg config.ServerConfig) (*Client, error) {
	var opts []transport.StreamableHTTPCOption
	if cfg.Headers != nil {
		opts = append(opts, transport.WithHTTPHeaders(cfg.Headers))
	}

	t, err := transport.NewStreamableHTTP(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating streamable HTTP transport: %w", err)
	}

	c := client.NewClient(t)
	if err := c.Start(ctx); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("starting streamable HTTP client: %w", err)
	}

	if err := initialize(ctx, c); err != nil {
		_ = c.Close()
		return nil, err
	}

	return &Client{mcpClient: c}, nil
}

func trySSE(ctx context.Context, cfg config.ServerConfig) (*Client, error) {
	var opts []transport.ClientOption
	if cfg.Headers != nil {
		opts = append(opts, transport.WithHeaders(cfg.Headers))
	}

	t, err := transport.NewSSE(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating SSE transport: %w", err)
	}

	c := client.NewClient(t)
	if err := c.Start(ctx); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("starting SSE client: %w", err)
	}

	if err := initialize(ctx, c); err != nil {
		_ = c.Close()
		return nil, err
	}

	return &Client{mcpClient: c}, nil
}

func initialize(ctx context.Context, c *client.Client) error {
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "mcp-bin",
		Version: "1.0.0",
	}
	initReq.Params.Capabilities = mcp.ClientCapabilities{}

	_, err := c.Initialize(ctx, initReq)
	if err != nil {
		return fmt.Errorf("initializing client: %w", err)
	}
	return nil
}

// ListTools returns the tools available on the connected server.
func (c *Client) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	listCtx, cancel := withDefaultTimeout(ctx, 3*time.Minute)
	defer cancel()

	result, err := c.mcpClient.ListTools(listCtx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, err
	}
	return result.Tools, nil
}

// CallTool invokes a tool with the given arguments.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	return c.mcpClient.CallTool(ctx, req)
}

// Close closes the client connection.
func (c *Client) Close() {
	if c.mcpClient != nil {
		_ = c.mcpClient.Close()
	}
}

// withDefaultTimeout returns a context with the given timeout if the parent
// context has no deadline. If the parent already has a deadline, it is returned as-is.
func withDefaultTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

// nopLogger suppresses log output from the mcp-go stdio transport.
// The transport's readResponses goroutine logs spurious "Error reading from
// stdout: file already closed" messages when the connection is closed normally.
type nopLogger struct{}

func (*nopLogger) Infof(string, ...any)  {}
func (*nopLogger) Errorf(string, ...any) {}
