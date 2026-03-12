package mcp

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestStderrErrorNilReader(t *testing.T) {
	origErr := errors.New("connection failed")
	got := stderrError(origErr, nil)
	if got != origErr {
		t.Errorf("expected original error, got %v", got)
	}
}

func TestStderrErrorEmptyOutput(t *testing.T) {
	origErr := errors.New("connection failed")
	got := stderrError(origErr, strings.NewReader(""))
	if got != origErr {
		t.Errorf("expected original error when stderr is empty, got %v", got)
	}
}

func TestStderrErrorWithOutput(t *testing.T) {
	origErr := errors.New("connection failed")
	got := stderrError(origErr, strings.NewReader("server crashed\n"))
	if !errors.Is(got, origErr) {
		t.Errorf("wrapped error should contain original: %v", got)
	}
	if !strings.Contains(got.Error(), "server crashed") {
		t.Errorf("expected stderr content in error, got: %v", got)
	}
}

func TestStderrErrorLargeOutput(t *testing.T) {
	origErr := errors.New("connection failed")
	// Create output larger than 4KB limit
	large := strings.Repeat("x", 8192)
	got := stderrError(origErr, strings.NewReader(large))
	if !errors.Is(got, origErr) {
		t.Errorf("wrapped error should contain original: %v", got)
	}
	// The output is limited to 4096 bytes
	if !strings.Contains(got.Error(), "server stderr:") {
		t.Errorf("expected stderr label in error, got: %v", got)
	}
}

func TestStderrErrorTimeout(t *testing.T) {
	origErr := errors.New("connection failed")
	// A reader that blocks forever
	r, _ := io.Pipe()
	defer func() { _ = r.Close() }()

	start := time.Now()
	got := stderrError(origErr, r)
	elapsed := time.Since(start)

	// Should timeout within ~2s (the function's timeout)
	if elapsed > 5*time.Second {
		t.Errorf("stderrError took too long: %v", elapsed)
	}
	if got != origErr {
		t.Errorf("expected original error on timeout, got %v", got)
	}
}

func TestStderrErrorTrimsWhitespace(t *testing.T) {
	origErr := errors.New("failed")
	got := stderrError(origErr, strings.NewReader("  some error  \n\n"))
	if !strings.Contains(got.Error(), "some error") {
		t.Errorf("expected trimmed stderr, got: %v", got)
	}
	if strings.Contains(got.Error(), "  some error  \n\n") {
		t.Errorf("stderr should be trimmed, got: %v", got)
	}
}

func TestCloseNilClient(t *testing.T) {
	c := &Client{mcpClient: nil}
	// Should not panic
	c.Close()
}
