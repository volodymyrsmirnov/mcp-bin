package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestFormatJSON(t *testing.T) {
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: "hello",
			},
		},
	}

	var buf bytes.Buffer
	err := formatJSON(result, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"text"`) {
		t.Errorf("expected JSON with text, got %s", output)
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("expected JSON with hello, got %s", output)
	}

	// Should be valid JSON
	var parsed interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
}

func TestFormatTextPlain(t *testing.T) {
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: "Hello, World!",
			},
		},
	}

	var buf, errBuf bytes.Buffer
	err := formatText(result, &buf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.TrimSpace(buf.String()) != "Hello, World!" {
		t.Errorf("expected Hello, World!, got %q", buf.String())
	}
}

func TestFormatTextJSON(t *testing.T) {
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: `{"key": "value"}`,
			},
		},
	}

	var buf, errBuf bytes.Buffer
	err := formatText(result, &buf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should be pretty-printed
	if !strings.Contains(output, "  ") {
		t.Errorf("expected indented JSON, got %s", output)
	}
	if !strings.Contains(output, `"key"`) {
		t.Errorf("expected key in output, got %s", output)
	}
}

func TestFormatTextImage(t *testing.T) {
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.ImageContent{
				Type:     "image",
				MIMEType: "image/png",
				Data:     "base64data",
			},
		},
	}

	var buf, errBuf bytes.Buffer
	err := formatText(result, &buf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "image/png") {
		t.Errorf("expected image/png in output, got %s", output)
	}
	if !strings.Contains(output, "bytes") {
		t.Errorf("expected bytes in output, got %s", output)
	}
}

func TestFormatTextError(t *testing.T) {
	result := &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: "something went wrong",
			},
		},
	}

	var buf, errBuf bytes.Buffer
	err := formatText(result, &buf, &errBuf)
	if err == nil {
		t.Error("expected error for isError result")
	}
	// Error content should go to errW, not w
	if !strings.Contains(errBuf.String(), "something went wrong") {
		t.Errorf("expected error text in errW, got %q", errBuf.String())
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output in w for error result, got %q", buf.String())
	}
}

func TestFormatTextMultipleContent(t *testing.T) {
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: "line 1",
			},
			mcp.TextContent{
				Type: "text",
				Text: "line 2",
			},
		},
	}

	var buf, errBuf bytes.Buffer
	err := formatText(result, &buf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "line 1") || !strings.Contains(output, "line 2") {
		t.Errorf("expected both lines, got %s", output)
	}
}

func TestFormatTextEmptyContent(t *testing.T) {
	result := &mcp.CallToolResult{
		Content: []mcp.Content{},
	}

	var buf, errBuf bytes.Buffer
	err := formatText(result, &buf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.String() != "" {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

func TestFormatResultJSONMode(t *testing.T) {
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: "test",
			},
		},
	}

	// FormatResult writes to os.Stdout, so we just verify no error
	err := FormatResult(result, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatResultTextMode(t *testing.T) {
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: "test",
			},
		},
	}

	err := FormatResult(result, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatResultNilResult(t *testing.T) {
	err := FormatResult(nil, false)
	if err == nil {
		t.Error("expected error for nil result")
	}
	if !strings.Contains(err.Error(), "nil result") {
		t.Errorf("expected 'nil result' error, got %v", err)
	}
}

func TestFormatResultToWriters(t *testing.T) {
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: "output text",
			},
		},
	}

	var out, errOut bytes.Buffer
	err := FormatResultTo(&out, &errOut, result, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "output text") {
		t.Errorf("expected output in writer, got %q", out.String())
	}
}

func TestFormatTextEmbeddedResource(t *testing.T) {
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.EmbeddedResource{
				Type: "resource",
				Resource: mcp.TextResourceContents{
					URI:      "file:///test.txt",
					MIMEType: "text/plain",
					Text:     "content",
				},
			},
		},
	}

	var buf, errBuf bytes.Buffer
	err := formatText(result, &buf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "resource") {
		t.Errorf("expected resource in output, got %s", output)
	}
}
