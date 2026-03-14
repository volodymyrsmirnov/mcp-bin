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

// maxExtractSize limits the total bytes extracted from the embedded zip
// to prevent a corrupted or malicious zip from filling the disk.
const maxExtractSize = 1 << 30 // 1 GB

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
	// Open the binary once for both hashing and extraction
	f, err := os.Open(info.ExePath)
	if err != nil {
		return nil, fmt.Errorf("opening binary: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// Compute content-addressed hash
	h := sha256.New()
	section := io.NewSectionReader(f, info.ZipStart, info.ZipSize)
	if _, err := io.Copy(h, section); err != nil {
		return nil, fmt.Errorf("hashing zip: %w", err)
	}
	hash := fmt.Sprintf("%x", h.Sum(nil)[:16])

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

	// Re-create section reader (previous one was consumed by hash)
	section = io.NewSectionReader(f, info.ZipStart, info.ZipSize)
	reader, err := zip.NewReader(section, info.ZipSize)
	if err != nil {
		return nil, fmt.Errorf("reading zip: %w", err)
	}

	var totalExtracted int64
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

		n, extractErr := extractFile(zf, destPath)
		if extractErr != nil {
			return nil, fmt.Errorf("extracting %s: %w", zf.Name, extractErr)
		}
		totalExtracted += n
		if totalExtracted > maxExtractSize {
			return nil, fmt.Errorf("extraction exceeded %d byte limit", maxExtractSize)
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

func extractFile(f *zip.File, dest string) (written int64, err error) {
	if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return 0, err
	}

	rc, err := f.Open()
	if err != nil {
		return 0, err
	}
	defer func() {
		if cerr := rc.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return 0, err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	written, err = io.Copy(out, io.LimitReader(rc, maxExtractSize))
	return written, err
}

func isSubPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}
