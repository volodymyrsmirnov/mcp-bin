package embed

import (
	"archive/zip"
	"crypto/sha256"
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
}

// ExtractToCache extracts the zip contents to a content-addressed cache directory.
// The cache key is a SHA-256 hash of the embedded zip, so if the directory exists
// the contents are guaranteed correct. Safe for concurrent process launches.
func ExtractToCache(info *ZipInfo) (_ *CachePaths, err error) {
	hash, err := zipHash(info)
	if err != nil {
		return nil, err
	}

	cacheBase, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}

	parentDir := filepath.Join(cacheBase, "mcp-bin")
	cacheDir := filepath.Join(parentDir, hash)
	paths := &CachePaths{
		Root:     cacheDir,
		Config:   filepath.Join(cacheDir, "config.json"),
		Manifest: filepath.Join(cacheDir, "manifest.json"),
		DirsRoot: filepath.Join(cacheDir, "dirs"),
	}

	// Cache hit: directory exists and was atomically renamed into place,
	// so its contents are always complete.
	if _, err := os.Stat(cacheDir); err == nil {
		return paths, nil
	}

	// Cache miss: extract to a unique temp dir, then atomically rename.
	if err := os.MkdirAll(parentDir, 0700); err != nil {
		return nil, fmt.Errorf("creating cache parent: %w", err)
	}

	tmpDir, err := os.MkdirTemp(parentDir, hash+".tmp.")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir) // no-op after successful rename
	}()

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

	for _, zf := range reader.File {
		destPath := filepath.Join(tmpDir, zf.Name)

		// Prevent zip slip
		if !isSubPath(tmpDir, destPath) {
			continue
		}

		if zf.FileInfo().IsDir() {
			if mkErr := os.MkdirAll(destPath, 0700); mkErr != nil {
				return nil, fmt.Errorf("creating directory %s: %w", destPath, mkErr)
			}
			continue
		}

		if err := extractFile(zf, destPath); err != nil {
			return nil, fmt.Errorf("extracting %s: %w", zf.Name, err)
		}
	}

	// Atomically publish the cache
	if err := os.Rename(tmpDir, cacheDir); err != nil {
		// Another process may have won the race — that's fine, use their cache
		if _, statErr := os.Stat(cacheDir); statErr == nil {
			return paths, nil
		}
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

func zipHash(info *ZipInfo) (string, error) {
	f, err := os.Open(info.ExePath)
	if err != nil {
		return "", fmt.Errorf("opening binary: %w", err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	section := io.NewSectionReader(f, info.ZipStart, info.ZipSize)
	if _, err := io.Copy(h, section); err != nil {
		return "", fmt.Errorf("hashing zip: %w", err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)[:16]), nil
}

func isSubPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}
