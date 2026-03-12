package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	ucli "github.com/urfave/cli/v3"
	"github.com/volodymyrsmirnov/mcp-bin/internal/config"
	mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
	"github.com/volodymyrsmirnov/mcp-bin/internal/output"
)

// buildCommandsFromManifest creates CLI commands from a pre-built manifest (compiled mode).
func buildCommandsFromManifest(cfg *config.Config, manifest *mcpclient.Manifest) []*ucli.Command {
	var commands []*ucli.Command
	for serverName, tools := range manifest.Servers {
		serverCfg := cfg.Servers[serverName]
		cmd := buildServerCommandFromSchemas(serverName, &serverCfg, tools)
		commands = append(commands, cmd)
	}
	return commands
}

// buildCommandsFromConfig creates CLI commands for dev mode.
// Tool subcommands are discovered dynamically on invocation.
func buildCommandsFromConfig(cfg *config.Config) []*ucli.Command {
	var commands []*ucli.Command
	for serverName := range cfg.Servers {
		srvCfg := cfg.Servers[serverName]
		name := serverName
		cmd := &ucli.Command{
			Name:            name,
			Usage:           fmt.Sprintf("Commands for %s server", name),
			SkipFlagParsing: true,
			Action: func(ctx context.Context, cmd *ucli.Command) error {
				return handleServerCommand(ctx, cmd, name, &srvCfg)
			},
		}
		commands = append(commands, cmd)
	}
	return commands
}

// handleServerCommand handles all invocations of a server command in dev mode.
// It introspects the server, builds a urfave/cli command tree from the discovered
// schemas, and runs it. This produces identical help output to compiled mode.
func handleServerCommand(ctx context.Context, cmd *ucli.Command, serverName string, serverCfg *config.ServerConfig) error {
	args := cmd.Args().Slice()

	// Connect and introspect
	client, err := mcpclient.Connect(ctx, *serverCfg)
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", serverName, err)
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		client.Close()
		return fmt.Errorf("listing tools from %s: %w", serverName, err)
	}

	schemas, err := mcpclient.ToolsToSchemas(tools)
	if err != nil {
		client.Close()
		return err
	}

	schemas = mcpclient.FilterSchemas(schemas, serverCfg.AllowTools, serverCfg.DenyTools)
	client.Close()

	// Build urfave/cli command tree (same as compiled mode) and run it.
	// This delegates help rendering to urfave/cli for identical output.
	srvCmd := buildServerCommandFromSchemas(serverName, serverCfg, schemas)

	wrapper := &ucli.Command{
		Name: cmd.Root().Name,
		Flags: []ucli.Flag{
			&ucli.BoolFlag{
				Name:  "json",
				Usage: "Output raw MCP JSON response",
			},
		},
		Commands: []*ucli.Command{srvCmd},
		// Return errors to the caller instead of calling os.Exit.
		ExitErrHandler: func(_ context.Context, _ *ucli.Command, _ error) {},
	}

	// Forward --json flag if set on the outer app
	wrapperArgs := []string{wrapper.Name}
	if cmd.Root().Bool("json") {
		wrapperArgs = append(wrapperArgs, "--json")
	}
	wrapperArgs = append(wrapperArgs, serverName)
	wrapperArgs = append(wrapperArgs, args...)

	// Use a detached context to prevent urfave/cli from inheriting the outer
	// command's name chain (which would cause NAME duplication in help output).
	// Preserve deadline and cancellation from the parent context.
	var detached context.Context
	var cancel context.CancelFunc
	if deadline, ok := ctx.Deadline(); ok {
		detached, cancel = context.WithDeadline(context.Background(), deadline)
	} else {
		detached, cancel = context.WithCancel(context.Background())
	}
	go func() {
		<-ctx.Done()
		cancel()
	}()
	defer cancel()

	return wrapper.Run(detached, wrapperArgs)
}

// buildServerCommandFromSchemas builds a server command with pre-known tool schemas (compiled mode).
func buildServerCommandFromSchemas(serverName string, serverCfg *config.ServerConfig, tools []mcpclient.ToolSchema) *ucli.Command {
	cmd := &ucli.Command{
		Name:  serverName,
		Usage: fmt.Sprintf("Commands for %s server", serverName),
	}

	for _, tool := range tools {
		t := tool
		toolCmd := buildToolCommand(serverName, serverCfg, t)
		cmd.Commands = append(cmd.Commands, toolCmd)
	}

	return cmd
}

func buildToolCommand(serverName string, serverCfg *config.ServerConfig, tool mcpclient.ToolSchema) *ucli.Command {
	schema := mcpclient.ParseInputSchema(tool.InputSchema)
	flags := schemaToFlags(schema)
	passthrough := len(schema.Properties) == 0

	description := strings.TrimSpace(tool.Description)
	if description == "" {
		description = fmt.Sprintf("Call the %s tool", tool.Name)
	}
	usage, _ := splitFirst(description)

	return &ucli.Command{
		Name:            tool.Name,
		Usage:           usage,
		Description:     description,
		Flags:           flags,
		SkipFlagParsing: passthrough,
		Action: func(ctx context.Context, cmd *ucli.Command) error {
			var args map[string]interface{}
			if passthrough {
				var err error
				args, err = parseToolArgs(cmd.Args().Slice(), schema)
				if err != nil {
					return err
				}
			} else {
				args = collectArgs(cmd, schema)
			}

			for _, r := range schema.Required {
				if _, ok := args[r]; !ok {
					return fmt.Errorf("missing required argument: --%s", r)
				}
			}

			client, err := mcpclient.Connect(ctx, *serverCfg)
			if err != nil {
				return fmt.Errorf("connecting to %s: %w", serverName, err)
			}
			defer client.Close()

			result, err := client.CallTool(ctx, tool.Name, args)
			if err != nil {
				return fmt.Errorf("calling tool %s: %w", tool.Name, err)
			}

			jsonMode := cmd.Root().Bool("json")
			return output.FormatResult(result, jsonMode)
		},
	}
}

func schemaToFlags(schema mcpclient.ParsedSchema) []ucli.Flag {
	requiredSet := make(map[string]bool)
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	var flags []ucli.Flag
	names := mcpclient.SortedKeys(schema.Properties)
	for _, name := range names {
		prop := schema.Properties[name]
		usage := flagUsage(prop, requiredSet[name])
		switch prop.Type {
		case "string":
			flags = append(flags, &ucli.StringFlag{
				Name:     name,
				Usage:    usage,
				Required: requiredSet[name],
			})
		case "number":
			flags = append(flags, &ucli.FloatFlag{
				Name:     name,
				Usage:    usage,
				Required: requiredSet[name],
			})
		case "integer":
			flags = append(flags, &ucli.Int64Flag{
				Name:     name,
				Usage:    usage,
				Required: requiredSet[name],
			})
		case "boolean":
			flags = append(flags, &ucli.BoolFlag{
				Name:  name,
				Usage: usage,
			})
		default:
			hint := prop.TypeHint()
			jsonSuffix := " (JSON: " + hint + ")"
			flags = append(flags, &ucli.StringFlag{
				Name:     name,
				Usage:    usage + jsonSuffix,
				Required: requiredSet[name],
			})
		}
	}
	return flags
}

func flagUsage(prop mcpclient.PropertyInfo, required bool) string {
	desc := prop.Description
	if desc == "" {
		desc = fmt.Sprintf("%s value", prop.TypeHint())
	}
	if required {
		desc += " (required)"
	}
	return desc
}

func collectArgs(cmd *ucli.Command, schema mcpclient.ParsedSchema) map[string]interface{} {
	args := make(map[string]interface{})
	for name, prop := range schema.Properties {
		switch prop.Type {
		case "string":
			if cmd.IsSet(name) {
				args[name] = cmd.String(name)
			}
		case "number":
			if cmd.IsSet(name) {
				args[name] = cmd.Float(name)
			}
		case "integer":
			if cmd.IsSet(name) {
				args[name] = cmd.Int64(name)
			}
		case "boolean":
			if cmd.IsSet(name) {
				args[name] = cmd.Bool(name)
			}
		default:
			if cmd.IsSet(name) {
				v := cmd.String(name)
				var jsonVal interface{}
				if err := json.Unmarshal([]byte(v), &jsonVal); err == nil {
					args[name] = jsonVal
				} else {
					args[name] = v
				}
			}
		}
	}
	return args
}

// parseToolArgs parses --flag value pairs from raw args for dev mode.
// When the schema has no properties (e.g. server returned empty inputSchema),
// it operates in passthrough mode: all --flag value pairs are accepted and
// values are auto-parsed (JSON objects/arrays decoded, everything else as string).
func parseToolArgs(args []string, schema mcpclient.ParsedSchema) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	passthrough := len(schema.Properties) == 0

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			return nil, fmt.Errorf("unexpected argument: %s (expected --flag)", arg)
		}

		flagName := strings.TrimPrefix(arg, "--")

		// Handle --flag=value syntax
		if idx := strings.Index(flagName, "="); idx >= 0 {
			value := flagName[idx+1:]
			flagName = flagName[:idx]
			if passthrough {
				result[flagName] = autoParseValue(value)
				continue
			}
			prop, ok := schema.Properties[flagName]
			if !ok {
				return nil, fmt.Errorf("unknown flag: --%s", flagName)
			}
			parsed, err := parseValue(value, prop.Type)
			if err != nil {
				return nil, fmt.Errorf("flag --%s: %w", flagName, err)
			}
			result[flagName] = parsed
			continue
		}

		if passthrough {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("flag --%s requires a value", flagName)
			}
			i++
			result[flagName] = autoParseValue(args[i])
			continue
		}

		prop, ok := schema.Properties[flagName]
		if !ok {
			return nil, fmt.Errorf("unknown flag: --%s", flagName)
		}

		// Boolean flags don't need a value
		if prop.Type == "boolean" {
			result[flagName] = true
			continue
		}

		// Need a value
		if i+1 >= len(args) {
			return nil, fmt.Errorf("flag --%s requires a value", flagName)
		}
		i++
		parsed, err := parseValue(args[i], prop.Type)
		if err != nil {
			return nil, fmt.Errorf("flag --%s: %w", flagName, err)
		}
		result[flagName] = parsed
	}

	return result, nil
}

// autoParseValue tries to decode a value as JSON (for objects/arrays),
// otherwise returns it as a string.
func autoParseValue(s string) interface{} {
	var jsonVal interface{}
	if err := json.Unmarshal([]byte(s), &jsonVal); err == nil {
		// Only keep structured types (objects, arrays); scalars stay as strings
		// so that "42" is sent as string "42", not number 42.
		switch jsonVal.(type) {
		case map[string]interface{}, []interface{}:
			return jsonVal
		}
	}
	return s
}

func parseValue(value, typ string) (interface{}, error) {
	switch typ {
	case "string":
		return value, nil
	case "number":
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, fmt.Errorf("expected number, got %q", value)
		}
		return f, nil
	case "integer":
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("expected integer, got %q", value)
		}
		return n, nil
	case "boolean":
		switch strings.ToLower(value) {
		case "true", "1", "yes":
			return true, nil
		case "false", "0", "no":
			return false, nil
		default:
			return nil, fmt.Errorf("expected boolean, got %q", value)
		}
	default:
		// Try JSON
		var jsonVal interface{}
		if err := json.Unmarshal([]byte(value), &jsonVal); err == nil {
			return jsonVal, nil
		}
		return value, nil
	}
}

// splitFirst splits text into the first paragraph and the rest.
func splitFirst(text string) (first, rest string) {
	if idx := strings.Index(text, "\n\n"); idx >= 0 {
		return strings.TrimSpace(text[:idx]), strings.TrimSpace(text[idx+2:])
	}
	return text, ""
}
