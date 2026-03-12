package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	ucli "github.com/urfave/cli/v3"
	"github.com/volodymyrsmirnov/mcp-bin/internal/compile"
	"github.com/volodymyrsmirnov/mcp-bin/internal/config"
	mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
	"github.com/volodymyrsmirnov/mcp-bin/internal/serve"
	"github.com/volodymyrsmirnov/mcp-bin/internal/skill"
	"github.com/volodymyrsmirnov/mcp-bin/internal/validate"
	"github.com/volodymyrsmirnov/mcp-bin/internal/version"
)

// BuildApp creates the CLI application.
func BuildApp(cfg *config.Config, manifest *mcpclient.Manifest, compiledMode bool) *ucli.Command {
	appName := "mcp-bin"
	if compiledMode && len(os.Args) > 0 {
		appName = filepath.Base(os.Args[0])
	}

	app := &ucli.Command{
		Name:    appName,
		Usage:   "Turn MCP server tools into CLI commands",
		Version: version.String(),
		Flags: []ucli.Flag{
			&ucli.BoolFlag{
				Name:  "json",
				Usage: "Output raw MCP JSON response",
			},
		},
	}

	connectFlag := &ucli.BoolFlag{
		Name:  "connect",
		Usage: "Test live server connectivity",
	}

	if compiledMode {
		validateCmd := &ucli.Command{
			Name:  "validate",
			Usage: "Validate configuration and server connectivity",
			Flags: []ucli.Flag{connectFlag},
			Action: func(ctx context.Context, cmd *ucli.Command) error {
				if !validate.Run(ctx, cfg, cmd.Bool("connect"), os.Stdout) {
					return fmt.Errorf("validation failed")
				}
				return nil
			},
		}
		skillCmd := &ucli.Command{
			Name:  "skill",
			Usage: "Generate a markdown skill document",
			Flags: []ucli.Flag{
				&ucli.StringFlag{
					Name:  "name",
					Usage: "Skill name",
					Value: "mcp-bin",
				},
				&ucli.StringFlag{
					Name:  "description",
					Usage: "Skill description (auto-generated if omitted)",
				},
				&ucli.StringFlag{
					Name:  "mode",
					Usage: "Output mode: cli or http",
					Value: "cli",
				},
				&ucli.StringFlag{
					Name:  "base-url",
					Usage: "Base URL for HTTP mode (e.g., https://example.com)",
				},
				&ucli.StringFlag{
					Name:  "token",
					Usage: "Bearer token for HTTP mode examples",
				},
			},
			Action: func(ctx context.Context, cmd *ucli.Command) error {
				mode := cmd.String("mode")
				switch mode {
				case "cli":
					binaryName := cmd.Root().Name
					skill.Generate(os.Stdout, manifest, binaryName, cmd.String("name"), cmd.String("description"))
				case "http":
					baseURL := cmd.String("base-url")
					if baseURL == "" {
						baseURL = "http://localhost:8819"
					}
					skill.GenerateHTTP(os.Stdout, manifest, cmd.String("name"), cmd.String("description"), baseURL, cmd.String("token"))
				default:
					return fmt.Errorf("unknown mode: %s (use cli or http)", mode)
				}
				return nil
			},
		}
		serveCmd := &ucli.Command{
			Name:  "serve",
			Usage: "Start REST API server",
			Flags: []ucli.Flag{
				&ucli.StringFlag{
					Name:  "listen",
					Usage: "Listen address (host:port)",
					Value: "localhost:8819",
				},
				&ucli.StringFlag{
					Name:  "token",
					Usage: "Bearer token for authentication",
				},
			},
			Action: func(ctx context.Context, cmd *ucli.Command) error {
				return serve.Run(ctx, cfg, manifest, cmd.String("listen"), cmd.String("token"))
			},
		}
		app.Commands = buildCommandsFromManifest(cfg, manifest)
		app.Commands = append(app.Commands, validateCmd, skillCmd, serveCmd)
	} else {
		configFlag := &ucli.StringFlag{
			Name:     "config",
			Aliases:  []string{"c"},
			Usage:    "Path to config file (JSON or YAML)",
			Required: true,
		}

		runCmd := &ucli.Command{
			Name:  "run",
			Usage: "Run MCP server tools",
			Flags: []ucli.Flag{configFlag},
		}

		// Pre-register server commands if we can parse config from args
		if devCfg := loadConfigFromArgs(); devCfg != nil {
			serverCmds := buildCommandsFromConfig(devCfg)
			runCmd.Commands = append(runCmd.Commands, serverCmds...)
		}

		compileCmd := &ucli.Command{
			Name:  "compile",
			Usage: "Compile into a self-contained binary",
			Flags: []ucli.Flag{
				&ucli.StringFlag{
					Name:     "config",
					Aliases:  []string{"c"},
					Usage:    "Path to config file (JSON or YAML)",
					Required: true,
				},
				&ucli.StringFlag{
					Name:  "output",
					Usage: "Output binary path",
					Value: "mcp-bin-compiled",
				},
			},
			Action: func(ctx context.Context, cmd *ucli.Command) error {
				configPath := cmd.String("config")
				loadedCfg, err := config.LoadFromFile(configPath)
				if err != nil {
					return fmt.Errorf("loading config: %w", err)
				}
				return compile.Compile(ctx, loadedCfg, cmd.String("output"))
			},
		}

		validateCmd := &ucli.Command{
			Name:  "validate",
			Usage: "Validate configuration and server connectivity",
			Flags: []ucli.Flag{
				&ucli.StringFlag{
					Name:     "config",
					Aliases:  []string{"c"},
					Usage:    "Path to config file (JSON or YAML)",
					Required: true,
				},
				connectFlag,
			},
			Action: func(ctx context.Context, cmd *ucli.Command) error {
				loadedCfg, err := config.LoadFromFile(cmd.String("config"))
				if err != nil {
					return fmt.Errorf("loading config: %w", err)
				}
				if !validate.Run(ctx, loadedCfg, cmd.Bool("connect"), os.Stdout) {
					return fmt.Errorf("validation failed")
				}
				return nil
			},
		}

		skillCmd := &ucli.Command{
			Name:  "skill",
			Usage: "Generate a markdown skill document",
			Flags: []ucli.Flag{
				&ucli.StringFlag{
					Name:     "config",
					Aliases:  []string{"c"},
					Usage:    "Path to config file (JSON or YAML)",
					Required: true,
				},
				&ucli.StringFlag{
					Name:  "name",
					Usage: "Skill name",
					Value: "mcp-bin",
				},
				&ucli.StringFlag{
					Name:  "description",
					Usage: "Skill description (auto-generated if omitted)",
				},
				&ucli.StringFlag{
					Name:  "mode",
					Usage: "Output mode: cli or http",
					Value: "cli",
				},
				&ucli.StringFlag{
					Name:  "base-url",
					Usage: "Base URL for HTTP mode (e.g., https://example.com)",
				},
				&ucli.StringFlag{
					Name:  "token",
					Usage: "Bearer token for HTTP mode examples",
				},
			},
			Action: func(ctx context.Context, cmd *ucli.Command) error {
				loadedCfg, err := config.LoadFromFile(cmd.String("config"))
				if err != nil {
					return fmt.Errorf("loading config: %w", err)
				}
				manifest, err := mcpclient.IntrospectAll(ctx, loadedCfg)
				if err != nil {
					return fmt.Errorf("introspecting servers: %w", err)
				}
				mode := cmd.String("mode")
				switch mode {
				case "cli":
					binaryName := cmd.Root().Name
					skill.Generate(os.Stdout, manifest, binaryName, cmd.String("name"), cmd.String("description"))
				case "http":
					baseURL := cmd.String("base-url")
					if baseURL == "" {
						baseURL = "http://localhost:8819"
					}
					skill.GenerateHTTP(os.Stdout, manifest, cmd.String("name"), cmd.String("description"), baseURL, cmd.String("token"))
				default:
					return fmt.Errorf("unknown mode: %s (use cli or http)", mode)
				}
				return nil
			},
		}

		serveCmd := &ucli.Command{
			Name:  "serve",
			Usage: "Start REST API server",
			Flags: []ucli.Flag{
				&ucli.StringFlag{
					Name:     "config",
					Aliases:  []string{"c"},
					Usage:    "Path to config file (JSON or YAML)",
					Required: true,
				},
				&ucli.StringFlag{
					Name:  "listen",
					Usage: "Listen address (host:port)",
					Value: "localhost:8819",
				},
				&ucli.StringFlag{
					Name:  "token",
					Usage: "Bearer token for authentication",
				},
			},
			Action: func(ctx context.Context, cmd *ucli.Command) error {
				loadedCfg, err := config.LoadFromFile(cmd.String("config"))
				if err != nil {
					return fmt.Errorf("loading config: %w", err)
				}
				manifest, err := mcpclient.IntrospectAll(ctx, loadedCfg)
				if err != nil {
					return fmt.Errorf("introspecting servers: %w", err)
				}
				return serve.Run(ctx, loadedCfg, manifest, cmd.String("listen"), cmd.String("token"))
			},
		}

		app.Commands = append(app.Commands, runCmd, compileCmd, validateCmd, skillCmd, serveCmd)
	}

	return app
}

// loadConfigFromArgs does a best-effort parse of --config / -c from os.Args
// to pre-register server commands before urfave/cli parses.
func loadConfigFromArgs() *config.Config {
	for i, arg := range os.Args {
		if (arg == "--config" || arg == "-c") && i+1 < len(os.Args) {
			cfg, err := config.LoadFromFile(os.Args[i+1])
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Warning: could not pre-load config: %v\n", err)
				return nil
			}
			return cfg
		}
		if strings.HasPrefix(arg, "--config=") || strings.HasPrefix(arg, "-c=") {
			value := arg[strings.Index(arg, "=")+1:]
			cfg, err := config.LoadFromFile(value)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Warning: could not pre-load config: %v\n", err)
				return nil
			}
			return cfg
		}
	}
	return nil
}

// RunApp runs the CLI application with graceful shutdown on SIGINT/SIGTERM.
func RunApp(app *ucli.Command) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return app.Run(ctx, os.Args)
}
