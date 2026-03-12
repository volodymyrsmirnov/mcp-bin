package embed

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestExtractToCache(t *testing.T) {
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

func TestZipHash(t *testing.T) {
	dir := t.TempDir()

	// Create first binary with zip
	path1 := filepath.Join(dir, "bin1")
	var zip1 bytes.Buffer
	w1 := zip.NewWriter(&zip1)
	f1, _ := w1.Create("config.json")
	_, _ = f1.Write([]byte(`{"v": 1}`))
	_ = w1.Close()

	out1, _ := os.Create(path1)
	_, _ = out1.Write([]byte("binary"))
	zipStart1 := int64(len("binary"))
	_, _ = out1.Write(zip1.Bytes())
	_ = out1.Close()

	info1 := &ZipInfo{ExePath: path1, ZipStart: zipStart1, ZipSize: int64(zip1.Len())}

	hash1, err := zipHash(info1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash1 == "" {
		t.Error("hash should not be empty")
	}

	// Same content should produce same hash
	hash1b, _ := zipHash(info1)
	if hash1 != hash1b {
		t.Errorf("same zip should produce same hash: %s vs %s", hash1, hash1b)
	}

	// Different zip content should produce different hash
	path2 := filepath.Join(dir, "bin2")
	var zip2 bytes.Buffer
	w2 := zip.NewWriter(&zip2)
	f2, _ := w2.Create("config.json")
	_, _ = f2.Write([]byte(`{"v": 2}`))
	_ = w2.Close()

	out2, _ := os.Create(path2)
	_, _ = out2.Write([]byte("binary"))
	zipStart2 := int64(len("binary"))
	_, _ = out2.Write(zip2.Bytes())
	_ = out2.Close()

	info2 := &ZipInfo{ExePath: path2, ZipStart: zipStart2, ZipSize: int64(zip2.Len())}

	hash2, _ := zipHash(info2)
	if hash1 == hash2 {
		t.Error("different zip content should produce different hash")
	}
}

func TestZipHashNotFound(t *testing.T) {
	info := &ZipInfo{ExePath: "/nonexistent/file", ZipStart: 0, ZipSize: 10}
	_, err := zipHash(info)
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

func TestExtractToCacheWithDirectoryEntries(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test-binary")

	binaryData := []byte("fake binary dir test")
	var zipBuf bytes.Buffer
	w := zip.NewWriter(&zipBuf)
	f, _ := w.Create("config.json")
	_, _ = f.Write([]byte(`{}`))
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

	data, err := os.ReadFile(filepath.Join(paths.Root, "dirs", "subdir", "file.txt"))
	if err != nil {
		t.Fatalf("reading nested file: %v", err)
	}
	if string(data) != "nested" {
		t.Errorf("expected 'nested', got %q", string(data))
	}
}

func TestExtractToCacheZipSlipPrevention(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test-binary")

	binaryData := []byte("fake binary zip slip")
	var zipBuf bytes.Buffer
	w := zip.NewWriter(&zipBuf)
	f, _ := w.Create("config.json")
	_, _ = f.Write([]byte(`{}`))
	// Create a path-traversal entry
	bad, _ := w.Create("../../../etc/evil.txt")
	_, _ = bad.Write([]byte("evil"))
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

	// Should succeed but skip the evil entry
	paths, err := ExtractToCache(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The evil file should not exist outside the cache
	if _, statErr := os.Stat(filepath.Join(paths.Root, "..", "..", "..", "etc", "evil.txt")); statErr == nil {
		t.Error("zip slip: evil file was extracted outside cache")
	}
}

func TestExtractToCacheInvalidZip(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test-binary")

	binaryData := []byte("fake binary")
	// Not a valid zip
	invalidZip := []byte("this is not a zip archive at all")

	outFile, _ := os.Create(binaryPath)
	_, _ = outFile.Write(binaryData)
	zipStart := int64(len(binaryData))
	_, _ = outFile.Write(invalidZip)
	_ = outFile.Close()

	info := &ZipInfo{
		ExePath:  binaryPath,
		ZipStart: zipStart,
		ZipSize:  int64(len(invalidZip)),
	}

	testHome := t.TempDir()
	t.Setenv("HOME", testHome)

	_, err := ExtractToCache(info)
	if err == nil {
		t.Error("expected error for invalid zip data")
	}
}

func TestExtractToCacheMissingBinary(t *testing.T) {
	info := &ZipInfo{
		ExePath:  "/nonexistent/binary",
		ZipStart: 0,
		ZipSize:  100,
	}

	_, err := ExtractToCache(info)
	if err == nil {
		t.Error("expected error for missing binary")
	}
}

func TestExtractToCacheConcurrent(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test-binary")

	binaryData := []byte("fake binary concurrent")
	var zipBuf bytes.Buffer
	w := zip.NewWriter(&zipBuf)
	f, _ := w.Create("config.json")
	_, _ = f.Write([]byte(`{"concurrent": true}`))
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

	const n = 10
	var wg sync.WaitGroup
	errs := make([]error, n)
	results := make([]*CachePaths, n)

	wg.Add(n)
	for i := range n {
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = ExtractToCache(info)
		}(i)
	}
	wg.Wait()

	for i := range n {
		if errs[i] != nil {
			t.Fatalf("goroutine %d failed: %v", i, errs[i])
		}
		if results[i].Root != results[0].Root {
			t.Errorf("goroutine %d got different root: %s vs %s", i, results[i].Root, results[0].Root)
		}
	}

	// Verify the cache content is valid
	configData, err := os.ReadFile(results[0].Config)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	if string(configData) != `{"concurrent": true}` {
		t.Errorf("wrong config: %s", string(configData))
	}
}
