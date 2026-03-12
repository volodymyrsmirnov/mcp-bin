package compile

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateZipArchive(t *testing.T) {
	files := map[string][]byte{
		"config.json":   []byte(`{"key": "value"}`),
		"manifest.json": []byte(`{"servers": {}}`),
	}

	var buf bytes.Buffer
	if err := CreateZipArchive(&buf, files, nil, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's a valid zip
	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
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

	var buf bytes.Buffer
	if err := CreateZipArchive(&buf, files, []string{subDir}, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
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
	var buf bytes.Buffer
	if err := CreateZipArchive(&buf, map[string][]byte{}, nil, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
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
	var buf bytes.Buffer
	if err := CreateZipArchive(&buf, files, []string{filePath}, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
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

	var buf bytes.Buffer
	if err := CreateZipArchive(&buf, files, []string{"examples/script.py"}, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
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
	var buf bytes.Buffer
	err := CreateZipArchive(&buf, map[string][]byte{}, []string{"/nonexistent/path"}, "")
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

func TestCreateZipArchiveRejectsSymlinkFile(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()

	// Create a file outside the allowed directory
	secretFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("sensitive data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink inside the allowed directory pointing outside
	symlink := filepath.Join(dir, "link.txt")
	if err := os.Symlink(secretFile, symlink); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := CreateZipArchive(&buf, map[string][]byte{}, []string{symlink}, dir)
	if err == nil {
		t.Fatal("expected error for symlink file, got nil")
	}
	if !strings.Contains(err.Error(), "symlinks are not allowed") {
		t.Errorf("expected symlink error, got: %v", err)
	}
}

func TestCreateZipArchiveRejectsSymlinkInDirectory(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()

	// Create a real file and a symlink inside the directory
	if err := os.WriteFile(filepath.Join(dir, "real.txt"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
	secretFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("sensitive data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secretFile, filepath.Join(dir, "sneaky.txt")); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := CreateZipArchive(&buf, map[string][]byte{}, []string{dir}, dir)
	if err == nil {
		t.Fatal("expected error for symlink inside directory, got nil")
	}
	if !strings.Contains(err.Error(), "symlinks are not allowed") {
		t.Errorf("expected symlink error, got: %v", err)
	}
}

func TestCreateZipArchiveAbsoluteWithoutBaseDir(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "single.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	// Absolute path with empty baseDir should use filepath.Base
	if err := CreateZipArchive(&buf, map[string][]byte{}, []string{filePath}, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}

	foundFiles := make(map[string]bool)
	for _, f := range reader.File {
		foundFiles[f.Name] = true
	}

	if !foundFiles["dirs/single.txt"] {
		t.Errorf("expected dirs/single.txt, found: %v", foundFiles)
	}
}

func TestCreateZipArchivePathOutsideBaseDir(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	filePath := filepath.Join(outside, "file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := CreateZipArchive(&buf, map[string][]byte{}, []string{filePath}, dir)
	if err == nil {
		t.Error("expected error for path outside base directory")
	}
	if !strings.Contains(err.Error(), "outside base directory") {
		t.Errorf("expected 'outside base directory' error, got: %v", err)
	}
}

func TestAddFileEntryToZip(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(srcPath, []byte("hello entry"), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(srcPath)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	if err := addFileEntryToZip(w, srcPath, "prefix/test.txt", info); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = w.Close()

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}
	if len(reader.File) != 1 {
		t.Fatalf("expected 1 file, got %d", len(reader.File))
	}
	if reader.File[0].Name != "prefix/test.txt" {
		t.Errorf("expected prefix/test.txt, got %s", reader.File[0].Name)
	}
}

func TestAddFileToZip(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "data.bin")
	if err := os.WriteFile(srcPath, []byte("binary data"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	if err := addFileToZip(w, srcPath, "data.bin"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = w.Close()

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}
	if len(reader.File) != 1 {
		t.Fatalf("expected 1 file, got %d", len(reader.File))
	}
	if reader.File[0].Name != "data.bin" {
		t.Errorf("expected data.bin, got %s", reader.File[0].Name)
	}
}

func TestAddFileToZipMissingFile(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	err := addFileToZip(w, "/nonexistent/file.txt", "file.txt")
	if err == nil {
		t.Error("expected error for missing source file")
	}
}

func TestCreateZipArchiveRejectsSymlinkDirectory(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()

	// Create a directory outside and a symlink to it
	outsideSubDir := filepath.Join(outside, "data")
	if err := os.MkdirAll(outsideSubDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outsideSubDir, "secret.txt"), []byte("sensitive"), 0644); err != nil {
		t.Fatal(err)
	}

	symlink := filepath.Join(dir, "linked_dir")
	if err := os.Symlink(outsideSubDir, symlink); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := CreateZipArchive(&buf, map[string][]byte{}, []string{symlink}, dir)
	if err == nil {
		t.Fatal("expected error for symlink directory, got nil")
	}
	if !strings.Contains(err.Error(), "symlinks are not allowed") {
		t.Errorf("expected symlink error, got: %v", err)
	}
}
