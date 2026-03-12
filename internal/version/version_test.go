package version

import "testing"

func TestString(t *testing.T) {
	got := String()
	// Default values when not set via ldflags
	expected := "dev (unknown) unknown"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestStringCustomValues(t *testing.T) {
	origVersion, origCommit, origDate := Version, Commit, Date
	defer func() {
		Version, Commit, Date = origVersion, origCommit, origDate
	}()

	Version = "1.2.3"
	Commit = "abc123"
	Date = "2025-01-01"

	got := String()
	expected := "1.2.3 (abc123) 2025-01-01"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
