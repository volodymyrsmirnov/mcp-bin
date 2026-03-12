package compile

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateZipArchive(t *testing.T) {
	files := map[string][]byte{
		"config.json":   []byte(`{"key": "value"}`),
		"manifest.json": []byte(`{"servers": {}}`),
	}

	data, err := CreateZipArchive(files, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's a valid zip
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}

	foundFiles := make(map[string]bool)
	for _, f := range reader.File {
		foundFiles[f.Name] = true
	}

	if !foundFiles["config.json"] {
		t.Error("missing config.json in zip")
	}
	if !foundFiles["manifest.json"] {
		t.Error("missing manifest.json in zip")
	}
}

func TestCreateZipArchiveWithDirectories(t *testing.T) {
	// Create temp directory structure
	dir := t.TempDir()
	subDir := filepath.Join(dir, "mydir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(subDir, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested", "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatal(err)
	}

	files := map[string][]byte{
		"config.json": []byte(`{}`),
	}

	data, err := CreateZipArchive(files, []string{subDir}, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}

	foundFiles := make(map[string]bool)
	for _, f := range reader.File {
		foundFiles[f.Name] = true
	}

	if !foundFiles["config.json"] {
		t.Error("missing config.json")
	}
	if !foundFiles["dirs/mydir/file1.txt"] {
		t.Error("missing dirs/mydir/file1.txt")
	}
	if !foundFiles["dirs/mydir/nested/file2.txt"] {
		t.Error("missing dirs/mydir/nested/file2.txt")
	}
}

func TestCreateZipArchiveEmptyFiles(t *testing.T) {
	data, err := CreateZipArchive(map[string][]byte{}, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}

	if len(reader.File) != 0 {
		t.Errorf("expected 0 files, got %d", len(reader.File))
	}
}

func TestCreateZipArchiveWithAbsoluteFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "single.txt")
	if err := os.WriteFile(filePath, []byte("single file content"), 0644); err != nil {
		t.Fatal(err)
	}

	files := map[string][]byte{
		"config.json": []byte(`{}`),
	}

	// Absolute path with baseDir preserves relative structure
	data, err := CreateZipArchive(files, []string{filePath}, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}

	foundFiles := make(map[string]bool)
	for _, f := range reader.File {
		foundFiles[f.Name] = true
	}

	if !foundFiles["config.json"] {
		t.Error("missing config.json")
	}
	if !foundFiles["dirs/single.txt"] {
		t.Error("missing dirs/single.txt")
	}
}

func TestCreateZipArchiveWithRelativeFile(t *testing.T) {
	// Create a relative path structure
	dir := t.TempDir()
	subDir := filepath.Join(dir, "examples")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "script.py"), []byte("print('hello')"), 0644); err != nil {
		t.Fatal(err)
	}

	files := map[string][]byte{
		"config.json": []byte(`{}`),
	}

	// Use relative path — should preserve directory structure
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Fatal(err)
		}
	}()

	data, err := CreateZipArchive(files, []string{"examples/script.py"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}

	foundFiles := make(map[string]bool)
	for _, f := range reader.File {
		foundFiles[f.Name] = true
	}

	if !foundFiles["dirs/examples/script.py"] {
		t.Errorf("expected dirs/examples/script.py, found: %v", foundFiles)
	}
}

func TestCreateZipArchiveNonexistentPath(t *testing.T) {
	_, err := CreateZipArchive(map[string][]byte{}, []string{"/nonexistent/path"}, "")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestAddDirToZipFileContent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	err := addDirToZip(w, dir, "prefix")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = w.Close()

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}

	for _, f := range reader.File {
		if f.Name == "prefix/test.txt" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("error opening file: %v", err)
			}
			content := make([]byte, 100)
			n, _ := rc.Read(content)
			defer func() { _ = rc.Close() }()
			if string(content[:n]) != "hello world" {
				t.Errorf("expected 'hello world', got %q", string(content[:n]))
			}
			return
		}
	}
	t.Error("test.txt not found in zip")
}
