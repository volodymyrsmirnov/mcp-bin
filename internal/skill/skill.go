package skill

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode"

	mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
	"github.com/volodymyrsmirnov/mcp-bin/internal/version"
)

// sortedManifestServerNames returns the server names from a manifest in sorted order.
func sortedManifestServerNames(manifest *mcpclient.Manifest) []string {
	names := make([]string, 0, len(manifest.Servers))
	for name := range manifest.Servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

var nonAlphanumRe = regexp.MustCompile(`[^a-z0-9]+`)

// Generate writes a markdown skill document to w.
// If skillVersion is empty, the application version is used.
func Generate(w io.Writer, manifest *mcpclient.Manifest, binaryName, skillName, description, skillVersion string) {
	serverNames := sortedManifestServerNames(manifest)

	if description == "" {
		description = autoDescription(serverNames)
	}

	if skillVersion == "" {
		skillVersion = version.Version
	}

	// YAML front matter
	_, _ = fmt.Fprintln(w, "---")
	_, _ = fmt.Fprintf(w, "name: %s\n", toKebabCase(skillName))
	_, _ = fmt.Fprintf(w, "description: %s\n", description)
	_, _ = fmt.Fprintln(w, "metadata:")
	_, _ = fmt.Fprintf(w, "  version: %s\n", skillVersion)
	_, _ = fmt.Fprintln(w, "---")
	_, _ = fmt.Fprintln(w)

	// Header
	_, _ = fmt.Fprintf(w, "# %s\n\n", skillName)
	_, _ = fmt.Fprintf(w, "Use `%s <server> --help` to list tools for a server.\n", binaryName)
	_, _ = fmt.Fprintf(w, "Use `%s <server> <tool> --help` for detailed flag descriptions.\n\n", binaryName)

	// Server sections
	for _, serverName := range serverNames {
		tools := manifest.Servers[serverName]
		writeServerSection(w, serverName, manifest.Descriptions[serverName], tools)
	}

	// Usage examples
	writeUsageExamples(w, binaryName, serverNames, manifest)
}

func writeServerSection(w io.Writer, serverName, description string, tools []mcpclient.ToolSchema) {
	_, _ = fmt.Fprintf(w, "## %s\n\n", serverName)
	if description != "" {
		_, _ = fmt.Fprintf(w, "%s\n\n", description)
	}

	for _, tool := range sortTools(tools) {
		desc := firstLine(tool.Description)
		_, _ = fmt.Fprintf(w, "- `%s` - %s\n", tool.Name, desc)
	}
	_, _ = fmt.Fprintln(w)
}

func writeUsageExamples(w io.Writer, binaryName string, serverNames []string, manifest *mcpclient.Manifest) {
	_, _ = fmt.Fprintln(w, "## Usage")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "```")

	// Help examples
	if len(serverNames) > 0 {
		first := serverNames[0]
		_, _ = fmt.Fprintf(w, "%s %s --help                   # list tools\n", binaryName, first)
		tools := manifest.Servers[first]
		if len(tools) > 0 {
			sorted := sortTools(tools)
			_, _ = fmt.Fprintf(w, "%s %s %s --help       # tool help\n", binaryName, first, sorted[0].Name)
		}
		_, _ = fmt.Fprintln(w)
	}

	// Tool invocation examples
	for _, serverName := range serverNames {
		tools := manifest.Servers[serverName]
		if len(tools) == 0 {
			continue
		}
		tool := sortTools(tools)[0]
		example := buildExample(binaryName, serverName, tool)
		_, _ = fmt.Fprintln(w, example)
	}
	_, _ = fmt.Fprintln(w, "```")
}

func buildExample(binaryName, serverName string, tool mcpclient.ToolSchema) string {
	return strings.Join([]string{binaryName, serverName, tool.Name, "--help"}, " ")
}

func sortTools(tools []mcpclient.ToolSchema) []mcpclient.ToolSchema {
	sorted := make([]mcpclient.ToolSchema, len(tools))
	copy(sorted, tools)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	return sorted
}

func autoDescription(serverNames []string) string {
	if len(serverNames) == 0 {
		return "CLI tool wrapping MCP servers"
	}
	return "CLI tool to work with " + strings.Join(serverNames, ", ")
}

func toKebabCase(s string) string {
	lower := strings.Map(func(r rune) rune {
		return unicode.ToLower(r)
	}, s)
	return strings.Trim(nonAlphanumRe.ReplaceAllString(lower, "-"), "-")
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "\n"); idx >= 0 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}
