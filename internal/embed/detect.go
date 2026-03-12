package embed

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	zipEndSignature = 0x06054b50
	zipEndMinSize   = 22
	zipEndMaxSize   = 65557 // 22 + max comment size (65535)
)

// ZipInfo holds the location of an appended zip in the binary.
type ZipInfo struct {
	ExePath  string
	ZipStart int64
	ZipSize  int64
}

// DetectEmbeddedZip checks if the current binary has a zip archive appended.
// Returns ZipInfo if found, nil otherwise.
func DetectEmbeddedZip() (_ *ZipInfo, err error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("getting executable path: %w", err)
	}

	f, err := os.Open(exePath)
	if err != nil {
		return nil, fmt.Errorf("opening executable: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := fi.Size()
	if fileSize < zipEndMinSize {
		return nil, nil
	}

	// Search for zip end-of-central-directory record from the end of file
	searchSize := int64(zipEndMaxSize)
	if searchSize > fileSize {
		searchSize = fileSize
	}

	buf := make([]byte, searchSize)
	_, err = f.ReadAt(buf, fileSize-searchSize)
	if err != nil && err != io.EOF {
		return nil, err
	}

	// Search backwards for the EOCD signature
	eocdOffset := int64(-1)
	for i := len(buf) - zipEndMinSize; i >= 0; i-- {
		if binary.LittleEndian.Uint32(buf[i:]) == zipEndSignature {
			eocdOffset = fileSize - searchSize + int64(i)
			break
		}
	}

	if eocdOffset < 0 {
		return nil, nil // no zip found
	}

	// Read the EOCD to find the start of the zip central directory
	eocdBuf := buf[eocdOffset-(fileSize-searchSize):]
	if len(eocdBuf) < zipEndMinSize {
		return nil, nil
	}

	centralDirOffset := int64(binary.LittleEndian.Uint32(eocdBuf[16:20]))
	centralDirSize := int64(binary.LittleEndian.Uint32(eocdBuf[12:16]))

	commentLen := int64(binary.LittleEndian.Uint16(eocdBuf[20:22]))
	totalEOCDSize := int64(zipEndMinSize) + commentLen
	zipEndPos := eocdOffset + totalEOCDSize

	// The zip starts where central dir offset says, adjusted for the binary prefix
	zipStart := eocdOffset - centralDirSize - centralDirOffset

	if zipStart < 0 {
		return nil, nil
	}

	zipSize := zipEndPos - zipStart

	return &ZipInfo{
		ExePath:  exePath,
		ZipStart: zipStart,
		ZipSize:  zipSize,
	}, nil
}
