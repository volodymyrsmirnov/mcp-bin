package main

import (
	"encoding/json"
	"fmt"
	"os"

	mcpcli "github.com/volodymyrsmirnov/mcp-bin/internal/cli"
	"github.com/volodymyrsmirnov/mcp-bin/internal/config"
	mcpembed "github.com/volodymyrsmirnov/mcp-bin/internal/embed"
	mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
)

func main() {
	// Check for embedded zip (compiled mode)
	zipInfo, err := mcpembed.DetectEmbeddedZip()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error detecting embedded data: %v\n", err)
		os.Exit(1)
	}

	if zipInfo != nil {
		// Compiled mode
		paths, err := mcpembed.ExtractToCache(zipInfo)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error extracting embedded data: %v\n", err)
			os.Exit(1)
		}

		configData, err := os.ReadFile(paths.Config)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error reading embedded config: %v\n", err)
			os.Exit(1)
		}

		cfg, err := config.LoadCompiledConfig(configData)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		cfg.PatchFiles(paths.DirsRoot, cfg.ConfigDir())

		manifestData, err := os.ReadFile(paths.Manifest)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error reading manifest: %v\n", err)
			os.Exit(1)
		}

		var manifest mcpclient.Manifest
		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error parsing manifest: %v\n", err)
			os.Exit(1)
		}

		app := mcpcli.BuildApp(cfg, &manifest, true)
		if err := mcpcli.RunApp(app); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Dev mode
		app := mcpcli.BuildApp(nil, nil, false)
		if err := mcpcli.RunApp(app); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}
