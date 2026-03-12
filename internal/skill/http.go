package skill

import (
	"fmt"
	"io"
	"strings"

	mcpclient "github.com/volodymyrsmirnov/mcp-bin/internal/mcp"
	"github.com/volodymyrsmirnov/mcp-bin/internal/version"
)

// GenerateHTTP writes an HTTP-mode markdown skill document to w.
func GenerateHTTP(w io.Writer, manifest *mcpclient.Manifest, skillName, description, baseURL, token string) {
	serverNames := sortedServerNames(manifest)

	if description == "" {
		description = autoHTTPDescription(serverNames)
	}

	baseURL = strings.TrimRight(baseURL, "/")

	// YAML front matter
	_, _ = fmt.Fprintln(w, "---")
	_, _ = fmt.Fprintf(w, "name: %s\n", toKebabCase(skillName))
	_, _ = fmt.Fprintf(w, "description: %s\n", description)
	_, _ = fmt.Fprintln(w, "metadata:")
	_, _ = fmt.Fprintf(w, "  version: %s\n", version.Version)
	_, _ = fmt.Fprintln(w, "---")
	_, _ = fmt.Fprintln(w)

	// Header
	_, _ = fmt.Fprintf(w, "# %s\n\n", skillName)
	_, _ = fmt.Fprintln(w, "HTTP API for interacting with MCP server tools.")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Base URL: `%s`\n\n", baseURL)

	// Authentication section
	if token != "" {
		_, _ = fmt.Fprintln(w, "## Authentication")
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "All requests require a Bearer token in the Authorization header:")
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "```")
		_, _ = fmt.Fprintf(w, "Authorization: Bearer %s\n", token)
		_, _ = fmt.Fprintln(w, "```")
		_, _ = fmt.Fprintln(w)
	}

	// Endpoints overview
	_, _ = fmt.Fprintln(w, "## Endpoints")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "### List Servers")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "```")
	_, _ = fmt.Fprintln(w, "GET /")
	_, _ = fmt.Fprintln(w, "```")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Returns a JSON array of available servers with their names and descriptions.")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "### List Tools")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "```")
	_, _ = fmt.Fprintln(w, "GET /{server}/")
	_, _ = fmt.Fprintln(w, "```")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Returns a JSON array of tools for the specified server, including parameters.")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "### Tool Details")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "```")
	_, _ = fmt.Fprintln(w, "GET /{server}/{tool}")
	_, _ = fmt.Fprintln(w, "```")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Returns detailed information about a specific tool, including its parameters.")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "### Execute Tool")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "```")
	_, _ = fmt.Fprintln(w, "POST /{server}/{tool}")
	_, _ = fmt.Fprintln(w, "Content-Type: application/json")
	_, _ = fmt.Fprintln(w, "```")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Executes a tool with the provided parameters in the JSON body. Returns the MCP tool result.")
	_, _ = fmt.Fprintln(w)

	// Server/tool sections
	for _, serverName := range serverNames {
		tools := manifest.Servers[serverName]
		writeHTTPServerSection(w, serverName, tools)
	}

	// Usage examples
	writeHTTPUsageExamples(w, baseURL, token, serverNames, manifest)
}

func writeHTTPServerSection(w io.Writer, serverName string, tools []mcpclient.ToolSchema) {
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
				_, _ = fmt.Fprintf(w, "  - `%s` %s%s\n", name, typ, req)
			}
		}
	}
	_, _ = fmt.Fprintln(w)
}

func writeHTTPUsageExamples(w io.Writer, baseURL, token string, serverNames []string, manifest *mcpclient.Manifest) {
	_, _ = fmt.Fprintln(w, "## Usage")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "```")

	authHeader := ""
	if token != "" {
		authHeader = fmt.Sprintf(" \\\n  -H \"Authorization: Bearer %s\"", token)
	}

	// List servers
	_, _ = fmt.Fprintf(w, "# List all servers\ncurl %s/%s\n\n", baseURL, authHeader)

	// List tools example
	if len(serverNames) > 0 {
		first := serverNames[0]
		_, _ = fmt.Fprintf(w, "# List tools in a server\ncurl %s/%s/%s\n\n", baseURL, first, authHeader)

		// Tool detail example
		tools := manifest.Servers[first]
		if len(tools) > 0 {
			sorted := sortTools(tools)
			_, _ = fmt.Fprintf(w, "# Get tool details\ncurl %s/%s/%s%s\n\n", baseURL, first, sorted[0].Name, authHeader)
		}
	}

	// POST examples
	for _, serverName := range serverNames {
		tools := manifest.Servers[serverName]
		if len(tools) == 0 {
			continue
		}
		tool := sortTools(tools)[0]
		example := buildHTTPExample(baseURL, serverName, tool, authHeader)
		_, _ = fmt.Fprintln(w, example)
	}
	_, _ = fmt.Fprintln(w, "```")
}

func buildHTTPExample(baseURL, serverName string, tool mcpclient.ToolSchema, authHeader string) string {
	schema := mcpclient.ParseInputSchema(tool.InputSchema)
	requiredSet := make(map[string]bool)
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	// Build JSON body from required params
	var bodyParts []string
	names := mcpclient.SortedKeys(schema.Properties)
	for _, name := range names {
		if !requiredSet[name] {
			continue
		}
		bodyParts = append(bodyParts, fmt.Sprintf("%q: \"<%s>\"", name, name))
	}
	body := "{" + strings.Join(bodyParts, ", ") + "}"

	return fmt.Sprintf("# Execute %s/%s\ncurl -X POST %s/%s/%s%s \\\n  -H \"Content-Type: application/json\" \\\n  -d '%s'",
		serverName, tool.Name, baseURL, serverName, tool.Name, authHeader, body)
}

func autoHTTPDescription(serverNames []string) string {
	if len(serverNames) == 0 {
		return "HTTP API wrapping MCP servers"
	}
	return "HTTP API to work with " + strings.Join(serverNames, ", ")
}
