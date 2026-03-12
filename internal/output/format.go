package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
)

// FormatResult writes the tool call result to stdout.
func FormatResult(result *mcp.CallToolResult, jsonMode bool) error {
	if result == nil {
		return fmt.Errorf("nil result")
	}
	if jsonMode {
		return formatJSON(result, os.Stdout)
	}
	return formatText(result, os.Stdout)
}

func formatJSON(result *mcp.CallToolResult, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func formatText(result *mcp.CallToolResult, w io.Writer) error {
	if result.IsError {
		for _, content := range result.Content {
			if tc, ok := content.(mcp.TextContent); ok {
				_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", tc.Text)
			}
		}
		return fmt.Errorf("tool returned an error")
	}

	for _, content := range result.Content {
		switch c := content.(type) {
		case mcp.TextContent:
			// Try to detect and pretty-print JSON
			var jsonObj interface{}
			if err := json.Unmarshal([]byte(c.Text), &jsonObj); err == nil {
				pretty, _ := json.MarshalIndent(jsonObj, "", "  ")
				_, _ = fmt.Fprintln(w, string(pretty))
			} else {
				_, _ = fmt.Fprintln(w, c.Text)
			}
		case mcp.ImageContent:
			_, _ = fmt.Fprintf(w, "[image: %s, %d bytes]\n", c.MIMEType, len(c.Data))
		case mcp.EmbeddedResource:
			data, _ := json.MarshalIndent(c.Resource, "", "  ")
			_, _ = fmt.Fprintf(w, "[resource: %s]\n", string(data))
		default:
			// Fallback: marshal as JSON
			data, _ := json.MarshalIndent(content, "", "  ")
			_, _ = fmt.Fprintln(w, string(data))
		}
	}
	return nil
}
