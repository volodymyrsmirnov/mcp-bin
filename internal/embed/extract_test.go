package embed

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractToCache(t *testing.T) {
	// Create a temp binary with an appended zip
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test-binary")

	binaryData := []byte("fake binary")
	var zipBuf bytes.Buffer
	w := zip.NewWriter(&zipBuf)
	f, _ := w.Create("config.json")
	_, _ = f.Write([]byte(`{"test": true}`))
	f2, _ := w.Create("manifest.json")
	_, _ = f2.Write([]byte(`{"servers": {}}`))
	d, _ := w.Create("dirs/mydir/file.txt")
	_, _ = d.Write([]byte("file content"))
	_ = w.Close()

	outFile, _ := os.Create(binaryPath)
	_, _ = outFile.Write(binaryData)
	zipStart := int64(len(binaryData))
	_, _ = outFile.Write(zipBuf.Bytes())
	_ = outFile.Close()

	info := &ZipInfo{
		ExePath:  binaryPath,
		ZipStart: zipStart,
		ZipSize:  int64(zipBuf.Len()),
	}

	// Override home dir for testing
	testHome := t.TempDir()
	t.Setenv("HOME", testHome)

	paths, err := ExtractToCache(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify config was extracted
	configData, err := os.ReadFile(paths.Config)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	if string(configData) != `{"test": true}` {
		t.Errorf("wrong config content: %s", string(configData))
	}

	// Verify manifest was extracted
	manifestData, err := os.ReadFile(paths.Manifest)
	if err != nil {
		t.Fatalf("reading manifest: %v", err)
	}
	if string(manifestData) != `{"servers": {}}` {
		t.Errorf("wrong manifest content: %s", string(manifestData))
	}

	// Verify directory file was extracted
	fileData, err := os.ReadFile(filepath.Join(paths.DirsRoot, "mydir", "file.txt"))
	if err != nil {
		t.Fatalf("reading dir file: %v", err)
	}
	if string(fileData) != "file content" {
		t.Errorf("wrong file content: %s", string(fileData))
	}

	// Verify completion marker exists
	if _, err := os.Stat(paths.Complete); err != nil {
		t.Fatalf("completion marker should exist: %v", err)
	}
}

func TestExtractToCacheCacheHit(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test-binary")

	binaryData := []byte("fake binary for cache test")
	var zipBuf bytes.Buffer
	w := zip.NewWriter(&zipBuf)
	f, _ := w.Create("config.json")
	_, _ = f.Write([]byte(`{"cached": true}`))
	_ = w.Close()

	outFile, _ := os.Create(binaryPath)
	_, _ = outFile.Write(binaryData)
	zipStart := int64(len(binaryData))
	_, _ = outFile.Write(zipBuf.Bytes())
	_ = outFile.Close()

	info := &ZipInfo{
		ExePath:  binaryPath,
		ZipStart: zipStart,
		ZipSize:  int64(zipBuf.Len()),
	}

	testHome := t.TempDir()
	t.Setenv("HOME", testHome)

	// First extraction
	paths1, err := ExtractToCache(info)
	if err != nil {
		t.Fatalf("first extract error: %v", err)
	}

	// Second extraction should use cache
	paths2, err := ExtractToCache(info)
	if err != nil {
		t.Fatalf("second extract error: %v", err)
	}

	if paths1.Root != paths2.Root {
		t.Errorf("cache paths should match: %s vs %s", paths1.Root, paths2.Root)
	}
}

func TestCacheKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	key1, err := cacheKey(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if key1 == "" {
		t.Error("cache key should not be empty")
	}

	// Same file should produce same key
	key2, _ := cacheKey(path)
	if key1 != key2 {
		t.Errorf("same file should produce same key: %s vs %s", key1, key2)
	}

	// Different size should produce different key
	path2 := filepath.Join(dir, "test2")
	if err := os.WriteFile(path2, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	key3, _ := cacheKey(path2)
	if key1 == key3 {
		t.Error("different size should produce different key")
	}
}

func TestCacheKeyNotFound(t *testing.T) {
	_, err := cacheKey("/nonexistent/file")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestIsSubPath(t *testing.T) {
	tests := []struct {
		parent string
		child  string
		want   bool
	}{
		{"/cache", "/cache/file.txt", true},
		{"/cache", "/cache/sub/file.txt", true},
		{"/cache", "/other/file.txt", false},
		{"/cache", "/cache/../other/file.txt", false},
	}

	for _, tt := range tests {
		got := isSubPath(tt.parent, tt.child)
		if got != tt.want {
			t.Errorf("isSubPath(%q, %q) = %v, want %v", tt.parent, tt.child, got, tt.want)
		}
	}
}

func TestExtractFile(t *testing.T) {
	// Create a zip with a file
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("test.txt")
	_, _ = f.Write([]byte("test content"))
	_ = w.Close()

	reader, _ := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))

	dir := t.TempDir()
	dest := filepath.Join(dir, "extracted", "test.txt")

	err := extractFile(reader.File[0], dest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading extracted file: %v", err)
	}
	if string(data) != "test content" {
		t.Errorf("expected 'test content', got %q", string(data))
	}
}

func TestExtractToCacheIncompleteCache(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test-binary")

	binaryData := []byte("fake binary incomplete")
	var zipBuf bytes.Buffer
	w := zip.NewWriter(&zipBuf)
	f, _ := w.Create("config.json")
	_, _ = f.Write([]byte(`{"recovered": true}`))
	m, _ := w.Create("manifest.json")
	_, _ = m.Write([]byte(`{"servers": {}}`))
	_ = w.Close()

	outFile, _ := os.Create(binaryPath)
	_, _ = outFile.Write(binaryData)
	zipStart := int64(len(binaryData))
	_, _ = outFile.Write(zipBuf.Bytes())
	_ = outFile.Close()

	info := &ZipInfo{
		ExePath:  binaryPath,
		ZipStart: zipStart,
		ZipSize:  int64(zipBuf.Len()),
	}

	testHome := t.TempDir()
	t.Setenv("HOME", testHome)

	// Simulate incomplete extraction: create cache dir with config but no .complete
	fi, _ := os.Stat(binaryPath)
	key := fmt.Sprintf("%x-%x", fi.Size(), fi.ModTime().UnixNano())
	staleDir := filepath.Join(testHome, ".cache", "mcp-bin", key)
	_ = os.MkdirAll(staleDir, 0700)
	_ = os.WriteFile(filepath.Join(staleDir, "config.json"), []byte(`{"stale": true}`), 0600)

	// Extract should clean up incomplete cache and re-extract
	paths, err := ExtractToCache(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify fresh extraction happened (not stale data)
	configData, _ := os.ReadFile(paths.Config)
	if string(configData) != `{"recovered": true}` {
		t.Errorf("expected fresh config, got: %s", string(configData))
	}

	// Verify completion marker exists
	if _, err := os.Stat(paths.Complete); err != nil {
		t.Fatalf("completion marker should exist: %v", err)
	}
}

func TestExtractToCacheWithDirectoryEntries(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test-binary")

	binaryData := []byte("fake binary dir test")
	var zipBuf bytes.Buffer
	w := zip.NewWriter(&zipBuf)
	f, _ := w.Create("config.json")
	_, _ = f.Write([]byte(`{}`))
	// Add a directory entry
	_, _ = w.Create("dirs/")
	_, _ = w.Create("dirs/subdir/")
	df, _ := w.Create("dirs/subdir/file.txt")
	_, _ = df.Write([]byte("nested"))
	_ = w.Close()

	outFile, _ := os.Create(binaryPath)
	_, _ = outFile.Write(binaryData)
	zipStart := int64(len(binaryData))
	_, _ = outFile.Write(zipBuf.Bytes())
	_ = outFile.Close()

	info := &ZipInfo{
		ExePath:  binaryPath,
		ZipStart: zipStart,
		ZipSize:  int64(zipBuf.Len()),
	}

	testHome := t.TempDir()
	t.Setenv("HOME", testHome)

	paths, err := ExtractToCache(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify nested file
	data, err := os.ReadFile(filepath.Join(paths.Root, "dirs", "subdir", "file.txt"))
	if err != nil {
		t.Fatalf("reading nested file: %v", err)
	}
	if string(data) != "nested" {
		t.Errorf("expected 'nested', got %q", string(data))
	}
}
