# CLAUDE.md - mcp-bin

## Project Overview

Go CLI utility that introspects MCP (Model Context Protocol) servers, converts their tools into CLI commands, and can compile itself into a self-contained distributable binary via zip-append.

## Architecture

Two-mode binary:
- **Dev mode**: `./mcp-bin run --config config.json <server> <tool> [--flags]` — live introspection
- **Compiled mode**: `./compiled-binary <server> <tool> [--flags]` — pre-introspected, self-contained

Both modes support `validate` for config/environment diagnostics and `skill` for generating markdown skill documents.

### Package Structure

```
cmd/mcp-bin/          — entrypoint, mode detection (compiled vs dev)
internal/
  config/             — JSON/YAML config parsing, ${VAR} env var resolution, tool filtering
  mcp/                — MCP client wrapper (stdio, SSE, Streamable HTTP), introspection, schema types
  cli/                — urfave/cli v3 app, dynamic tool-to-command conversion, arg parsing
  compile/            — zip-append compilation (no Go toolchain needed)
  embed/              — embedded zip detection (EOCD signature) + extraction to cache
  validate/           — config validation, env/command/URL diagnostics
  skill/              — markdown skill document generation for LLM consumption
  output/             — text vs --json output formatting
examples/             — example configs and MCP servers
schema/               — JSON schema for config file validation
```

### Key Types

- `config.Config` / `config.ServerConfig` — configuration (local command or remote URL)
- `config.CompiledConfig` — config format for embedded binaries with env var metadata
- `mcp.Client` — wraps MCP client connection (stdio/HTTP/SSE)
- `mcp.ToolSchema` / `mcp.Manifest` — serializable tool schemas for compiled mode
- `mcp.ParsedSchema` / `mcp.PropertyInfo` — parsed JSON schema for flag generation
- `embed.ZipInfo` / `embed.CachePaths` — embedded zip location and extracted paths

### Data Flow

**Dev mode**: config file → `config.LoadFromFile` → `mcp.Connect` → `mcp.ListTools` → `mcp.ToolsToSchemas` → `cli.parseToolArgs` → `mcp.CallTool` → `output.FormatResult`

**Compile**: config → `mcp.IntrospectAll` (parallel connect+ListTools per server) → `compile.CreateZipArchive` → append zip to binary copy

**Validate**: config → `validate.Run` → check env vars, commands/URLs, files; optionally `mcp.Connect` + `ListTools` with `--connect`

**Skill**: config → `mcp.IntrospectAll` → `skill.Generate` → markdown to stdout

**Compiled mode**: `embed.DetectEmbeddedZip` → `embed.ExtractToCache` → `config.LoadCompiledConfig` → `cli.BuildApp` with manifest → urfave/cli flag parsing → `mcp.Connect` → `mcp.CallTool`

## Build & Test

```bash
make build          # build binary
make test           # run all tests
make fmt            # gofmt -s -w .
make lint           # go vet + golangci-lint
make vet            # go vet only
make vulncheck      # govulncheck ./...
make clean          # remove built binaries
```

Or directly:
```bash
go build -o mcp-bin ./cmd/mcp-bin/
go test ./...
go test -cover ./...
go test -race ./...
golangci-lint run ./...
govulncheck ./...
```

## Dependencies

- `github.com/mark3labs/mcp-go` v0.45.0 — MCP client (stdio, SSE, Streamable HTTP transports)
- `github.com/urfave/cli/v3` v3.7.0 — CLI framework
- `gopkg.in/yaml.v3` v3.0.1 — YAML config parsing
- `golangci-lint` v2 — linter (install via `brew install golangci-lint` or [official docs](https://golangci-lint.run/docs/welcome/install/))
- `govulncheck` — vulnerability scanner (install via `go install golang.org/x/vuln/cmd/govulncheck@latest`)

## Code Style

### Imports

Three groups separated by blank lines: stdlib, external, internal.

```go
import (
    "context"
    "fmt"

    ucli "github.com/urfave/cli/v3"
    "github.com/volodymyrsmirnov/mcp-bin/internal/config"
    mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
)
```

Package aliases used: `ucli` for urfave/cli, `mcpclient` for internal/mcp, `mcplib` for mark3labs/mcp-go/mcp.

### Error Handling

- Wrap with context: `fmt.Errorf("connecting to %s: %w", name, err)`
- Errors to stderr: `fmt.Fprintf(os.Stderr, ...)`
- Tool results to stdout (or writer parameter)
- Deferred cleanup preserves first error:
  ```go
  defer func() {
      if cerr := f.Close(); cerr != nil && err == nil {
          err = cerr
      }
  }()
  ```
- Suppressed write errors use `_, _ =` prefix: `_, _ = fmt.Fprintln(w, s)`

### Naming

- Short receiver names: `c *Client`, `cfg *Config`
- Variables: `cfg`, `srv`, `ctx`, `cmd`, `err` — short, conventional
- Functions: verb-first (`LoadFromFile`, `ParseInputSchema`, `FilterSchemas`)
- Test functions: `Test<Function><Scenario>` (e.g., `TestLoadFromFileNotFound`)

### Testing

- Table-driven tests with `t.Run()` for multiple scenarios
- Temp dirs via `t.TempDir()`, env overrides via `t.Setenv()`
- No test framework — stdlib `testing` only
- Test files colocated with source: `foo.go` / `foo_test.go`

## Workflow Rules

- **After every code change**, run `make fmt` then `make lint` and fix any issues before considering the change complete.
- **Before finalizing dependency changes** (adding/updating modules in `go.mod`), run `make vulncheck` and resolve any reported vulnerabilities.

## Conventions

- All business logic in `internal/` packages
- Config uses `${VAR}` syntax for env var substitution (resolved at load time; in compiled mode, overridable at runtime)
- Compiled binaries use zip-append — no Go toolchain required to distribute
- Cache at `~/.cache/mcp-bin/<sha256[:16]>/` with 0700 permissions
- Dev mode uses `SkipFlagParsing` + manual `parseToolArgs` for dynamic tools
- Compiled mode uses urfave/cli's built-in flag parsing with pre-registered commands
- Tool filtering uses `filepath.Match` glob patterns via `allow_tools` / `deny_tools`
- Remote transport: tries Streamable HTTP first, falls back to SSE only on `ErrLegacySSEServer`
