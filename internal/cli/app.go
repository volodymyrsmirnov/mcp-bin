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
	"github.com/volodymyrsmirnov/mcp-bin/internal/oauth"
	"github.com/volodymyrsmirnov/mcp-bin/internal/skill"
	"github.com/volodymyrsmirnov/mcp-bin/internal/validate"
	"github.com/volodymyrsmirnov/mcp-bin/internal/version"
)

// reservedCommands are names that cannot be used as server names because they
// conflict with built-in CLI subcommands.
var reservedCommands = map[string]bool{
	"run": true, "compile": true, "validate": true,
	"skill": true, "oauth": true, "help": true, "version": true,
}

// skillFlags returns the shared flag definitions for the skill subcommand.
func skillFlags(includeConfig bool) []ucli.Flag {
	flags := []ucli.Flag{
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
			Name:  "version",
			Usage: "Skill version (defaults to application version)",
		},
	}
	if includeConfig {
		flags = append([]ucli.Flag{
			&ucli.StringFlag{
				Name:     "config",
				Aliases:  []string{"c"},
				Usage:    "Path to config file (JSON or YAML)",
				Required: true,
			},
		}, flags...)
	}
	return flags
}

// oauthCmd builds the oauth command tree for managing OAuth2 authentication.
func oauthCmd(loadConfig func(cmd *ucli.Command) (*config.Config, error)) *ucli.Command {
	return &ucli.Command{
		Name:  "oauth",
		Usage: "Manage OAuth2 authentication for remote servers",
		Commands: []*ucli.Command{
			{
				Name:  "login",
				Usage: "Authenticate with a remote server via OAuth2",
				Flags: []ucli.Flag{
					&ucli.IntFlag{
						Name:  "port",
						Usage: "Local callback server port (0 = auto)",
						Value: 0,
					},
					&ucli.BoolFlag{
						Name:  "no-browser",
						Usage: "Don't open browser; paste redirect URL manually",
					},
				},
				Action: func(ctx context.Context, cmd *ucli.Command) error {
					cfg, err := loadConfig(cmd)
					if err != nil {
						return err
					}
					serverName := cmd.Args().First()
					if serverName == "" {
						return fmt.Errorf("server name required: mcp-bin oauth login <server>")
					}
					srv, ok := cfg.Servers[serverName]
					if !ok {
						return fmt.Errorf("server %q not found in config", serverName)
					}
					if !srv.IsRemote() {
						return fmt.Errorf("server %q is not a remote server (no url configured)", serverName)
					}
					store := oauth.NewKeychainStore(oauth.SystemKeyring(), srv.URL)
					opts := oauth.FlowOptions{
						Port:      int(cmd.Int("port")),
						NoBrowser: cmd.Bool("no-browser"),
					}
					return oauth.Login(ctx, os.Stderr, srv.URL, srv.OAuth, store, opts)
				},
			},
			{
				Name:  "logout",
				Usage: "Remove stored OAuth2 tokens for a server",
				Action: func(ctx context.Context, cmd *ucli.Command) error {
					cfg, err := loadConfig(cmd)
					if err != nil {
						return err
					}
					serverName := cmd.Args().First()
					if serverName == "" {
						return fmt.Errorf("server name required: mcp-bin oauth logout <server>")
					}
					srv, ok := cfg.Servers[serverName]
					if !ok {
						return fmt.Errorf("server %q not found in config", serverName)
					}
					if !srv.IsRemote() {
						return fmt.Errorf("server %q is not a remote server (no url configured)", serverName)
					}
					store := oauth.NewKeychainStore(oauth.SystemKeyring(), srv.URL)
					return oauth.Logout(os.Stdout, store)
				},
			},
			{
				Name:  "check",
				Usage: "Check OAuth2 token status for a server",
				Action: func(ctx context.Context, cmd *ucli.Command) error {
					cfg, err := loadConfig(cmd)
					if err != nil {
						return err
					}
					serverName := cmd.Args().First()
					if serverName == "" {
						return fmt.Errorf("server name required: mcp-bin oauth check <server>")
					}
					srv, ok := cfg.Servers[serverName]
					if !ok {
						return fmt.Errorf("server %q not found in config", serverName)
					}
					if !srv.IsRemote() {
						return fmt.Errorf("server %q is not a remote server (no url configured)", serverName)
					}
					store := oauth.NewKeychainStore(oauth.SystemKeyring(), srv.URL)
					return oauth.Check(ctx, os.Stdout, srv.URL, store)
				},
			},
		},
	}
}

// warnReservedNames logs a warning for server names that conflict with built-in commands.
func warnReservedNames(servers map[string]config.ServerConfig) {
	for name := range servers {
		if reservedCommands[name] {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: server name %q conflicts with a built-in command and will be shadowed\n", name)
		}
	}
}

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
		warnReservedNames(cfg.Servers)

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
			Flags: skillFlags(false),
			Action: func(ctx context.Context, cmd *ucli.Command) error {
				binaryName := cmd.Root().Name
				skill.Generate(os.Stdout, manifest, binaryName, cmd.String("name"), cmd.String("description"), cmd.String("version"))
				return nil
			},
		}
		oauthCommand := oauthCmd(func(_ *ucli.Command) (*config.Config, error) {
			return cfg, nil
		})

		app.Commands = buildCommandsFromManifest(cfg, manifest)
		app.Commands = append(app.Commands, validateCmd, skillCmd, oauthCommand)
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

		// Pre-register server commands only if the first subcommand is "run"
		if isRunSubcommand() {
			if devCfg := loadConfigFromArgs(); devCfg != nil {
				warnReservedNames(devCfg.Servers)
				serverCmds := buildCommandsFromConfig(devCfg)
				runCmd.Commands = append(runCmd.Commands, serverCmds...)
			}
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
			Flags: skillFlags(true),
			Action: func(ctx context.Context, cmd *ucli.Command) error {
				loadedCfg, err := config.LoadFromFile(cmd.String("config"))
				if err != nil {
					return fmt.Errorf("loading config: %w", err)
				}
				manifest, err := mcpclient.IntrospectAll(ctx, loadedCfg)
				if err != nil {
					return fmt.Errorf("introspecting servers: %w", err)
				}
				binaryName := cmd.Root().Name
				skill.Generate(os.Stdout, manifest, binaryName, cmd.String("name"), cmd.String("description"), cmd.String("version"))
				return nil
			},
		}

		oauthCommand := oauthCmd(func(cmd *ucli.Command) (*config.Config, error) {
			// In dev mode, --config is on the parent oauth command
			configPath := cmd.Root().Command("oauth").String("config")
			if configPath == "" {
				return nil, fmt.Errorf("--config flag is required")
			}
			return config.LoadFromFile(configPath)
		})
		oauthCommand.Flags = []ucli.Flag{
			&ucli.StringFlag{
				Name:     "config",
				Aliases:  []string{"c"},
				Usage:    "Path to config file (JSON or YAML)",
				Required: true,
			},
		}

		app.Commands = append(app.Commands, runCmd, compileCmd, validateCmd, skillCmd, oauthCommand)
	}

	return app
}

// isRunSubcommand checks if the first positional argument to the binary is "run".
func isRunSubcommand() bool {
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return arg == "run"
	}
	return false
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
