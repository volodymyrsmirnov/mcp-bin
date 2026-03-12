package config

import "path/filepath"

// MatchToolFilter checks whether a tool name passes the allow/deny filter.
// Returns true if the tool should be included.
//
// Rules:
//   - If neither allow nor deny is set, all tools pass.
//   - If allow is set, only tools matching at least one allow pattern pass.
//   - If deny is set, tools matching any deny pattern are excluded.
//   - If both are set, allow is applied first, then deny removes from the allowed set.
//
// Patterns support filepath.Match glob syntax (e.g., "read_*", "fs_*").
func MatchToolFilter(toolName string, allowTools, denyTools []string) bool {
	if len(allowTools) > 0 {
		allowed := false
		for _, pattern := range allowTools {
			if matched, _ := filepath.Match(pattern, toolName); matched {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	for _, pattern := range denyTools {
		if matched, _ := filepath.Match(pattern, toolName); matched {
			return false
		}
	}

	return true
}
