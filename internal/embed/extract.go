package embed

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CachePaths holds the paths to extracted files in the cache directory.
type CachePaths struct {
	Root     string
	Config   string
	Manifest string
	DirsRoot string
	Complete string
}

// ExtractToCache extracts the zip contents to a cache directory.
// If the cache already exists, it returns the existing paths.
func ExtractToCache(info *ZipInfo) (_ *CachePaths, err error) {
	key, err := cacheKey(info.ExePath)
	if err != nil {
		return nil, err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cacheDir := filepath.Join(homeDir, ".cache", "mcp-bin", key)
	paths := &CachePaths{
		Root:     cacheDir,
		Config:   filepath.Join(cacheDir, "config.json"),
		Manifest: filepath.Join(cacheDir, "manifest.json"),
		DirsRoot: filepath.Join(cacheDir, "dirs"),
		Complete: filepath.Join(cacheDir, ".complete"),
	}

	// Check if already extracted
	if _, err := os.Stat(paths.Complete); err == nil {
		return paths, nil
	}

	// Extract to a temporary directory, then atomically rename into place.
	// This prevents partial extractions from being visible to other processes.
	tmpDir := cacheDir + ".tmp"

	// Clean up any previous incomplete extraction
	_ = os.RemoveAll(tmpDir)
	_ = os.RemoveAll(cacheDir)

	// Open the binary and create zip reader
	f, err := os.Open(info.ExePath)
	if err != nil {
		return nil, fmt.Errorf("opening binary: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	section := io.NewSectionReader(f, info.ZipStart, info.ZipSize)
	reader, err := zip.NewReader(section, info.ZipSize)
	if err != nil {
		return nil, fmt.Errorf("reading zip: %w", err)
	}

	// Extract to temp dir
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}

	for _, zf := range reader.File {
		destPath := filepath.Join(tmpDir, zf.Name)

		// Prevent zip slip
		if !isSubPath(tmpDir, destPath) {
			continue
		}

		if zf.FileInfo().IsDir() {
			if mkErr := os.MkdirAll(destPath, 0700); mkErr != nil {
				_ = os.RemoveAll(tmpDir)
				return nil, fmt.Errorf("creating directory %s: %w", destPath, mkErr)
			}
			continue
		}

		if err := extractFile(zf, destPath); err != nil {
			_ = os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("extracting %s: %w", zf.Name, err)
		}
	}

	// Write completion marker inside temp dir before rename
	completeTmp := filepath.Join(tmpDir, ".complete")
	if err := os.WriteFile(completeTmp, nil, 0600); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("writing completion marker: %w", err)
	}

	// Atomically publish the cache
	if err := os.Rename(tmpDir, cacheDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("publishing cache: %w", err)
	}

	return paths, nil
}

func extractFile(f *zip.File, dest string) (err error) {
	if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() {
		if cerr := rc.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(out, rc)
	return err
}

func cacheKey(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x-%x", fi.Size(), fi.ModTime().UnixNano()), nil
}

func isSubPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}
