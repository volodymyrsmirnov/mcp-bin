package embed

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func createTestBinaryWithZip(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test-binary")

	binaryData := []byte("fake binary content here - some padding to simulate a real binary")

	var zipBuf bytes.Buffer
	w := zip.NewWriter(&zipBuf)
	f, _ := w.Create("config.json")
	_, _ = f.Write([]byte(`{"test": true}`))
	f2, _ := w.Create("manifest.json")
	_, _ = f2.Write([]byte(`{"servers": {}}`))
	_ = w.Close()

	outFile, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating test file: %v", err)
	}
	_, _ = outFile.Write(binaryData)
	_, _ = outFile.Write(zipBuf.Bytes())
	_ = outFile.Close()

	return path
}

func createTestBinaryWithoutZip(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test-binary")

	if err := os.WriteFile(path, []byte("just a plain binary with no zip"), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDetectEmbeddedZipFound(t *testing.T) {
	path := createTestBinaryWithZip(t)

	info, err := DetectEmbeddedZipFromPath(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected ZipInfo, got nil")
	}
	if info.ExePath != path {
		t.Errorf("expected ExePath=%s, got %s", path, info.ExePath)
	}
	if info.ZipStart <= 0 {
		t.Errorf("expected positive ZipStart, got %d", info.ZipStart)
	}
	if info.ZipSize <= 0 {
		t.Errorf("expected positive ZipSize, got %d", info.ZipSize)
	}
}

func TestDetectEmbeddedZipNotFound(t *testing.T) {
	path := createTestBinaryWithoutZip(t)

	info, err := DetectEmbeddedZipFromPath(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Errorf("expected nil ZipInfo for plain binary, got %+v", info)
	}
}

func TestDetectEmbeddedZipMissingFile(t *testing.T) {
	_, err := DetectEmbeddedZipFromPath("/nonexistent/file")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestDetectEmbeddedZipTinyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tiny")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectEmbeddedZipFromPath(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Errorf("expected nil for tiny file, got %+v", info)
	}
}

func TestDetectEmbeddedZipExtractable(t *testing.T) {
	// Verify that detected zip info can actually be used to read the zip
	path := createTestBinaryWithZip(t)

	info, err := DetectEmbeddedZipFromPath(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected ZipInfo")
	}

	// Use the ZipInfo to extract
	testHome := t.TempDir()
	t.Setenv("HOME", testHome)

	paths, err := ExtractToCache(info)
	if err != nil {
		t.Fatalf("extracting zip: %v", err)
	}

	configData, err := os.ReadFile(paths.Config)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	if string(configData) != `{"test": true}` {
		t.Errorf("wrong config content: %s", string(configData))
	}
}

func TestDetectEmbeddedZipEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty")
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectEmbeddedZipFromPath(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Errorf("expected nil for empty file, got %+v", info)
	}
}

func TestZipInfoFields(t *testing.T) {
	info := &ZipInfo{
		ExePath:  "/path/to/binary",
		ZipStart: 1000,
		ZipSize:  500,
	}

	if info.ExePath != "/path/to/binary" {
		t.Errorf("wrong ExePath: %s", info.ExePath)
	}
	if info.ZipStart != 1000 {
		t.Errorf("wrong ZipStart: %d", info.ZipStart)
	}
	if info.ZipSize != 500 {
		t.Errorf("wrong ZipSize: %d", info.ZipSize)
	}
}
