package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	ucli "github.com/urfave/cli/v3"
	"github.com/volodymyrsmirnov/mcp-bin/internal/compile"
	"github.com/volodymyrsmirnov/mcp-bin/internal/config"
	mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
)

// BuildApp creates the CLI application.
func BuildApp(cfg *config.Config, manifest *mcpclient.Manifest, compiledMode bool) *ucli.Command {
	app := &ucli.Command{
		Name:  "mcp-bin",
		Usage: "Turn MCP server tools into CLI commands",
		Flags: []ucli.Flag{
			&ucli.BoolFlag{
				Name:  "json",
				Usage: "Output raw MCP JSON response",
			},
		},
	}

	if compiledMode {
		app.Commands = buildCommandsFromManifest(cfg, manifest)
	} else {
		configFlag := &ucli.StringFlag{
			Name:     "config",
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

		app.Commands = append(app.Commands, runCmd, compileCmd)
	}

	return app
}

// loadConfigFromArgs does a best-effort parse of --config from os.Args
// to pre-register server commands before urfave/cli parses.
func loadConfigFromArgs() *config.Config {
	for i, arg := range os.Args {
		if (arg == "--config" || arg == "-config") && i+1 < len(os.Args) {
			cfg, err := config.LoadFromFile(os.Args[i+1])
			if err != nil {
				return nil
			}
			return cfg
		}
		if strings.HasPrefix(arg, "--config=") {
			cfg, err := config.LoadFromFile(strings.TrimPrefix(arg, "--config="))
			if err != nil {
				return nil
			}
			return cfg
		}
	}
	return nil
}

// RunApp runs the CLI application.
func RunApp(app *ucli.Command) error {
	return app.Run(context.Background(), os.Args)
}
