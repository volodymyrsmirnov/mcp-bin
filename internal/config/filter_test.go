package config

import "testing"

func TestMatchToolFilter(t *testing.T) {
	tests := []struct {
		name       string
		toolName   string
		allowTools []string
		denyTools  []string
		want       bool
	}{
		// No filters
		{"no filters", "anything", nil, nil, true},
		{"empty slices", "anything", []string{}, []string{}, true},

		// Allow only
		{"allow exact match", "read_file", []string{"read_file"}, nil, true},
		{"allow no match", "write_file", []string{"read_file"}, nil, false},
		{"allow wildcard match", "read_file", []string{"read_*"}, nil, true},
		{"allow wildcard no match", "write_file", []string{"read_*"}, nil, false},
		{"allow multiple patterns", "write_file", []string{"read_*", "write_*"}, nil, true},

		// Deny only
		{"deny exact match", "delete_file", nil, []string{"delete_file"}, false},
		{"deny no match", "read_file", nil, []string{"delete_file"}, true},
		{"deny wildcard match", "delete_all", nil, []string{"delete_*"}, false},
		{"deny wildcard no match", "read_file", nil, []string{"delete_*"}, true},

		// Both allow and deny
		{"allow then deny removes", "read_secret", []string{"read_*"}, []string{"*_secret"}, false},
		{"allow then deny keeps", "read_file", []string{"read_*"}, []string{"*_secret"}, true},
		{"deny has no effect if allow blocks first", "write_file", []string{"read_*"}, []string{"write_*"}, false},

		// Edge cases
		{"star matches all in allow", "anything", []string{"*"}, nil, true},
		{"star matches all in deny", "anything", nil, []string{"*"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchToolFilter(tt.toolName, tt.allowTools, tt.denyTools)
			if got != tt.want {
				t.Errorf("MatchToolFilter(%q, %v, %v) = %v, want %v",
					tt.toolName, tt.allowTools, tt.denyTools, got, tt.want)
			}
		})
	}
}
