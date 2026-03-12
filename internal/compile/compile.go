package compile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/volodymyrsmirnov/mcp-bin/internal/config"
	mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
)

// Compile creates a self-contained binary by:
// 1. Introspecting all MCP servers for tool schemas
// 2. Creating a zip with config, manifest, and directories
// 3. Appending the zip to a copy of the current binary
func Compile(ctx context.Context, cfg *config.Config, outputPath string) (err error) {
	fmt.Println("Introspecting MCP servers...")

	// Build manifest by introspecting all servers in parallel
	type serverResult struct {
		name    string
		schemas []mcpclient.ToolSchema
		err     error
	}

	results := make(chan serverResult, len(cfg.Servers))
	var wg sync.WaitGroup

	for name, srv := range cfg.Servers {
		wg.Add(1)
		go func(name string, srv config.ServerConfig) {
			defer wg.Done()
			fmt.Printf("  Connecting to %s...\n", name)
			client, err := mcpclient.Connect(ctx, srv)
			if err != nil {
				results <- serverResult{name: name, err: fmt.Errorf("connecting to server %s: %w", name, err)}
				return
			}
			defer client.Close()

			tools, err := client.ListTools(ctx)
			if err != nil {
				results <- serverResult{name: name, err: fmt.Errorf("listing tools from %s: %w", name, err)}
				return
			}

			schemas, err := mcpclient.ToolsToSchemas(tools)
			if err != nil {
				results <- serverResult{name: name, err: fmt.Errorf("converting tools from %s: %w", name, err)}
				return
			}

			schemas = mcpclient.FilterSchemas(schemas, srv.AllowTools, srv.DenyTools)
			fmt.Printf("  Found %d tools in %s\n", len(schemas), name)
			results <- serverResult{name: name, schemas: schemas}
		}(name, srv)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	manifest := &mcpclient.Manifest{
		Servers: make(map[string][]mcpclient.ToolSchema),
	}
	for res := range results {
		if res.err != nil {
			return res.err
		}
		manifest.Servers[res.name] = res.schemas
	}

	// Build compiled config with env var metadata
	compiledCfg := config.BuildCompiledConfig(cfg)
	configData, err := json.MarshalIndent(compiledCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}

	// Create zip archive
	fmt.Println("Creating archive...")
	zipFiles := map[string][]byte{
		"config.json":   configData,
		"manifest.json": manifestData,
	}

	zipData, err := CreateZipArchive(zipFiles, cfg.Files, cfg.ConfigDir())
	if err != nil {
		return fmt.Errorf("creating zip archive: %w", err)
	}

	// Copy current binary and append zip
	fmt.Println("Building binary...")
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}

	binaryData, err := os.ReadFile(exePath)
	if err != nil {
		return fmt.Errorf("reading current binary: %w", err)
	}

	outFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer func() {
		if cerr := outFile.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if _, err := outFile.Write(binaryData); err != nil {
		return fmt.Errorf("writing binary: %w", err)
	}

	if _, err := outFile.Write(zipData); err != nil {
		return fmt.Errorf("appending zip: %w", err)
	}

	fmt.Printf("Compiled binary written to: %s\n", outputPath)
	return nil
}
