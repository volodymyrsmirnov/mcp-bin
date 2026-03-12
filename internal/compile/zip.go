package compile

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CreateZipArchive writes a zip archive to w containing the given files
// and embedded paths (both files and directories). The baseDir is used to
// compute relative paths for absolute file entries in the zip.
func CreateZipArchive(w io.Writer, files map[string][]byte, paths []string, baseDir string) error {
	zw := zip.NewWriter(w)

	// Add explicit files
	for name, data := range files {
		f, err := zw.Create(name)
		if err != nil {
			return fmt.Errorf("creating zip entry %s: %w", name, err)
		}
		if _, err := f.Write(data); err != nil {
			return fmt.Errorf("writing zip entry %s: %w", name, err)
		}
	}

	// Add embedded paths (files and directories)
	for _, p := range paths {
		info, err := os.Lstat(p)
		if err != nil {
			return fmt.Errorf("stat %s: %w", p, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not allowed: %s", p)
		}
		// Preserve the original path structure under dirs/.
		// For absolute paths, compute relative path from baseDir.
		zipRelPath := p
		if filepath.IsAbs(p) && baseDir != "" {
			rel, relErr := filepath.Rel(baseDir, p)
			if relErr != nil || strings.HasPrefix(rel, "..") {
				return fmt.Errorf("path %s is outside base directory %s", p, baseDir)
			}
			zipRelPath = rel
		} else if filepath.IsAbs(p) {
			zipRelPath = filepath.Base(p)
		}
		if info.IsDir() {
			if err := addDirToZip(zw, p, filepath.Join("dirs", zipRelPath)); err != nil {
				return fmt.Errorf("adding directory %s: %w", p, err)
			}
		} else {
			if err := addFileToZip(zw, p, filepath.Join("dirs", zipRelPath)); err != nil {
				return fmt.Errorf("adding file %s: %w", p, err)
			}
		}
	}

	if err := zw.Close(); err != nil {
		return fmt.Errorf("closing zip writer: %w", err)
	}

	return nil
}

func addFileToZip(w *zip.Writer, srcPath, zipPath string) (err error) {
	info, err := os.Lstat(srcPath)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlinks are not allowed: %s", srcPath)
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filepath.ToSlash(zipPath)
	header.Method = zip.Deflate

	writer, err := w.CreateHeader(header)
	if err != nil {
		return err
	}

	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(writer, f)
	return err
}

func addDirToZip(w *zip.Writer, srcDir, zipPrefix string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not allowed: %s", path)
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		zipPath := filepath.Join(zipPrefix, relPath)
		// Normalize to forward slashes for zip
		zipPath = filepath.ToSlash(zipPath)

		if info.IsDir() {
			_, err := w.Create(zipPath + "/")
			return err
		}

		return addFileEntryToZip(w, path, zipPath, info)
	})
}

func addFileEntryToZip(w *zip.Writer, path, zipPath string, info os.FileInfo) (err error) {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = zipPath
	header.Method = zip.Deflate

	writer, err := w.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, f)
	return err
}
