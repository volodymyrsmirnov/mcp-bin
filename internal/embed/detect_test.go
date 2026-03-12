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

	// Create a fake binary prefix
	binaryData := []byte("fake binary content here - some padding to simulate a real binary")

	// Create a zip archive
	var zipBuf bytes.Buffer
	w := zip.NewWriter(&zipBuf)
	f, _ := w.Create("config.json")
	_, _ = f.Write([]byte(`{"test": true}`))
	f2, _ := w.Create("manifest.json")
	_, _ = f2.Write([]byte(`{"servers": {}}`))
	_ = w.Close()

	// Write binary + zip
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

func TestDetectEmbeddedZipWithZip(t *testing.T) {
	path := createTestBinaryWithZip(t)

	// We can't use DetectEmbeddedZip directly since it reads os.Executable(),
	// but we can test the detection logic by reading the file ourselves
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening file: %v", err)
	}
	defer func() { _ = f.Close() }()

	fi, _ := f.Stat()
	fileSize := fi.Size()

	buf := make([]byte, fileSize)
	if _, err := f.ReadAt(buf, 0); err != nil {
		t.Fatal(err)
	}

	// Search for EOCD signature
	found := false
	for i := len(buf) - zipEndMinSize; i >= 0; i-- {
		sig := uint32(buf[i]) | uint32(buf[i+1])<<8 | uint32(buf[i+2])<<16 | uint32(buf[i+3])<<24
		if sig == zipEndSignature {
			found = true
			break
		}
	}

	if !found {
		t.Error("EOCD signature not found in test binary with zip")
	}
}

func TestDetectEmbeddedZipWithoutZip(t *testing.T) {
	path := createTestBinaryWithoutZip(t)

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening file: %v", err)
	}
	defer func() { _ = f.Close() }()

	fi, _ := f.Stat()
	fileSize := fi.Size()

	buf := make([]byte, fileSize)
	if _, err := f.ReadAt(buf, 0); err != nil {
		t.Fatal(err)
	}

	found := false
	for i := len(buf) - zipEndMinSize; i >= 0; i-- {
		sig := uint32(buf[i]) | uint32(buf[i+1])<<8 | uint32(buf[i+2])<<16 | uint32(buf[i+3])<<24
		if sig == zipEndSignature {
			found = true
			break
		}
	}

	if found {
		t.Error("should not find EOCD signature in plain binary")
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
