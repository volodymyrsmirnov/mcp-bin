package compile

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/volodymyrsmirnov/mcp-bin/internal/config"
	mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
)

// Compile creates a self-contained binary by:
// 1. Introspecting all MCP servers for tool schemas
// 2. Creating a zip with config, manifest, and directories
// 3. Appending the zip to a copy of the current binary
func Compile(ctx context.Context, cfg *config.Config, outputPath string) (err error) {
	fmt.Println("Introspecting MCP servers...")
	manifest, err := mcpclient.IntrospectAll(ctx, cfg)
	if err != nil {
		return err
	}
	for name, schemas := range manifest.Servers {
		fmt.Printf("  Found %d tools in %s\n", len(schemas), name)
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

	// Copy current binary and append zip
	fmt.Println("Building binary...")
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}

	binaryFile, err := os.Open(exePath)
	if err != nil {
		return fmt.Errorf("reading current binary: %w", err)
	}
	defer func() {
		if cerr := binaryFile.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// Guard against overwriting the source binary
	absExe, _ := filepath.Abs(exePath)
	absOut, _ := filepath.Abs(outputPath)
	if absExe == absOut {
		return fmt.Errorf("output path %q is the same as the source binary; use a different --output", outputPath)
	}

	outFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer func() {
		if cerr := outFile.Close(); cerr != nil && err == nil {
			err = cerr
		}
		// Remove partially written output on error
		if err != nil {
			_ = os.Remove(outputPath)
		}
	}()

	if _, err := io.Copy(outFile, binaryFile); err != nil {
		return fmt.Errorf("writing binary: %w", err)
	}

	// Stream zip archive directly to output file
	zipFiles := map[string][]byte{
		"config.json":   configData,
		"manifest.json": manifestData,
	}
	if err := CreateZipArchive(outFile, zipFiles, cfg.Files, cfg.ConfigDir()); err != nil {
		return fmt.Errorf("creating zip archive: %w", err)
	}

	fmt.Printf("Compiled binary written to: %s\n", outputPath)
	return nil
}
