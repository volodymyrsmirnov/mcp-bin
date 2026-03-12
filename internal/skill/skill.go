package skill

import (
	"fmt"
	"io"
	"sort"
	"strings"

	mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
	"github.com/volodymyrsmirnov/mcp-bin/internal/version"
)

// Generate writes a markdown skill document to w.
func Generate(w io.Writer, manifest *mcpclient.Manifest, binaryName, skillName, description string) {
	serverNames := sortedServerNames(manifest)

	if description == "" {
		description = autoDescription(serverNames)
	}

	// YAML front matter
	_, _ = fmt.Fprintln(w, "---")
	_, _ = fmt.Fprintf(w, "name: %s\n", skillName)
	_, _ = fmt.Fprintf(w, "description: %s\n", description)
	_, _ = fmt.Fprintf(w, "version: %s\n", version.Version)
	_, _ = fmt.Fprintln(w, "---")
	_, _ = fmt.Fprintln(w)

	// Header
	_, _ = fmt.Fprintf(w, "# %s\n\n", skillName)
	_, _ = fmt.Fprintf(w, "Use `%s <server> --help` to list tools for a server.\n", binaryName)
	_, _ = fmt.Fprintf(w, "Use `%s <server> <tool> --help` for detailed flag descriptions.\n\n", binaryName)

	// Server sections
	for _, serverName := range serverNames {
		tools := manifest.Servers[serverName]
		writeServerSection(w, serverName, tools)
	}

	// Usage examples
	writeUsageExamples(w, binaryName, serverNames, manifest)
}

func writeServerSection(w io.Writer, serverName string, tools []mcpclient.ToolSchema) {
	_, _ = fmt.Fprintf(w, "## %s\n\n", serverName)

	for _, tool := range sortTools(tools) {
		desc := firstLine(tool.Description)
		_, _ = fmt.Fprintf(w, "- `%s` - %s\n", tool.Name, desc)

		schema := mcpclient.ParseInputSchema(tool.InputSchema)
		if len(schema.Properties) > 0 {
			requiredSet := make(map[string]bool)
			for _, r := range schema.Required {
				requiredSet[r] = true
			}
			names := mcpclient.SortedKeys(schema.Properties)
			for _, name := range names {
				prop := schema.Properties[name]
				typ := prop.Type
				if typ == "" {
					typ = "string"
				}
				req := ""
				if requiredSet[name] {
					req = " (required)"
				}
				_, _ = fmt.Fprintf(w, "  - `--%s` %s%s\n", name, typ, req)
			}
		}
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
	parts := []string{binaryName, serverName, tool.Name}

	schema := mcpclient.ParseInputSchema(tool.InputSchema)
	requiredSet := make(map[string]bool)
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	names := mcpclient.SortedKeys(schema.Properties)
	for _, name := range names {
		if !requiredSet[name] {
			continue
		}
		parts = append(parts, "--"+name, "<"+name+">")
	}

	return strings.Join(parts, " ")
}

func sortTools(tools []mcpclient.ToolSchema) []mcpclient.ToolSchema {
	sorted := make([]mcpclient.ToolSchema, len(tools))
	copy(sorted, tools)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	return sorted
}

func sortedServerNames(manifest *mcpclient.Manifest) []string {
	names := make([]string, 0, len(manifest.Servers))
	for name := range manifest.Servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func autoDescription(serverNames []string) string {
	if len(serverNames) == 0 {
		return "CLI tool wrapping MCP servers"
	}
	return "CLI tool to work with " + strings.Join(serverNames, ", ")
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "\n"); idx >= 0 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}
