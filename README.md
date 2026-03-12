# mcp-bin

Turn MCP server tools into CLI commands. Compile into a single self-contained binary.

## Quick Start

```bash
# Build
go build -o mcp-bin ./cmd/mcp-bin/

# Create a config file
cat > config.json << 'EOF'
{
  "servers": {
    "my-server": {
      "command": "node",
      "args": ["./server.js"],
      "env": {
        "API_KEY": "${API_KEY}"
      }
    }
  }
}
EOF

# List available tools
./mcp-bin run --config config.json my-server --help

# Call a tool
./mcp-bin run --config config.json my-server my-tool --arg1 value1

# Get raw JSON output
./mcp-bin --json run --config config.json my-server my-tool --arg1 value1

# Compile into a self-contained binary
./mcp-bin compile --config config.json --output my-tools

# Use the compiled binary (no config needed)
./my-tools my-server my-tool --arg1 value1
```

## Config Format

```json
{
  "files": ["/path/to/dir1", "/path/to/dir2"],
  "servers": {
    "local-server": {
      "command": "node",
      "args": ["./server.js"],
      "env": {
        "API_KEY": "${API_KEY}"
      },
      "cwd": "/path/to/directory"
    },
    "remote-server": {
      "url": "https://mcp.example.com",
      "headers": {
        "Authorization": "Bearer ${TOKEN}"
      }
    }
  }
}
```

### Fields

- **`files`** — File and directory paths to embed in the compiled binary (extracted to a cache location at runtime)
- **`servers`** — Named MCP server definitions
  - **Local servers**: `command` + `args` + optional `env` and `cwd`
  - **Remote servers**: `url` + optional `headers`
- **`${VAR}`** — Environment variable substitution. Resolved at compile time, overridable at runtime.

## CLI Usage

### Dev Mode

```bash
./mcp-bin run --config config.json <server> --help        # list tools
./mcp-bin run --config config.json <server> <tool> --help  # show tool help
./mcp-bin run --config config.json <server> <tool> [args]  # call tool
./mcp-bin compile --config config.json [--output FILE]     # compile binary
./mcp-bin skill --config config.json [--name NAME] [--description DESC]  # generate skill doc
```

### Compiled Mode (self-contained)

```bash
./my-tools <server> --help
./my-tools <server> <tool> --help
./my-tools <server> <tool> [args]
./my-tools skill [--name NAME] [--description DESC]       # generate skill doc
```

### Global Flags

- `--json` — Output raw MCP JSON response instead of formatted text
- `--help`, `-h` — Context-sensitive help at every level

### Tool Arguments

MCP tool input schemas are converted to CLI flags:
- `string` → `--flag value`
- `number` → `--flag 3.14`
- `integer` → `--flag 42`
- `boolean` → `--flag`
- `object`/`array` → `--flag '{"key": "value"}'`

## Compile Command

```bash
./mcp-bin compile --config config.json --output ./my-tools
```

Creates a self-contained binary for the current OS/architecture:
1. Introspects all configured MCP servers to collect tool schemas
2. Creates a zip archive with config, tool manifest, and embedded directories
3. Appends the zip to a copy of the current binary

The compiled binary:
- Requires no `--config` flag
- Has no `compile` command
- Extracts embedded data to `~/.cache/mcp-bin/` on first run
- Resolves `${VAR}` from environment at runtime, falling back to compile-time values

## Skill Command

```bash
./mcp-bin skill --config config.json
./mcp-bin skill --config config.json --name my-tool --description "Custom description"
```

Generates a markdown skill document to stdout — designed for LLM consumption. Includes YAML front matter, server/tool listing with flags, and usage examples. Available in both dev and compiled modes.

## Transport Support

- **stdio** — Local servers (spawns process, communicates via stdin/stdout)
- **Streamable HTTP** — Remote servers (newer MCP transport)
- **SSE** — Remote servers (legacy transport, automatic fallback)

For remote servers, Streamable HTTP is attempted first with automatic fallback to SSE.

## Environment Variables

Config values using `${VAR}` syntax are handled as:

- **Dev mode**: Resolved from current environment at runtime
- **Compiled binary**: Resolved at compile time, but overridable via `os.Getenv()` at runtime

## License

MIT
