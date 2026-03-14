package tools

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	openai "github.com/sashabaranov/go-openai"
)

func TestSanitizeMCPName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"My-Server", "my_server"},
		{"my server", "my_server"},
		{"dots.in.name", "dotsinname"},
		{"slashes/in/name", "slashesinname"},
		{"special!@#$chars", "specialchars"},
		{"  spaces  ", "spaces"},
		{"UPPER", "upper"},
		{"123numeric", "m123numeric"},
		{"", "mcp"},
		{"   ", "mcp"},
		{strings.Repeat("a", 50), strings.Repeat("a", 32)},
		{"a-b_c d.e/f", "a_b_c_def"},
		{"0start", "m0start"},
		{"_leading_underscore", "_leading_underscore"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeMCPName(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeMCPName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeMCPName_MaxLength(t *testing.T) {
	long := strings.Repeat("x", 100)
	got := SanitizeMCPName(long)
	if len(got) > 32 {
		t.Errorf("SanitizeMCPName should truncate to 32 chars, got %d", len(got))
	}
}

func TestExtractMCPText_SingleText(t *testing.T) {
	content := []mcp.Content{
		mcp.TextContent{Type: "text", Text: "hello world"},
	}
	got := ExtractMCPText(content)
	if got != "hello world" {
		t.Errorf("ExtractMCPText = %q, want %q", got, "hello world")
	}
}

func TestExtractMCPText_MultipleTexts(t *testing.T) {
	content := []mcp.Content{
		mcp.TextContent{Type: "text", Text: "line1"},
		mcp.TextContent{Type: "text", Text: "line2"},
		mcp.TextContent{Type: "text", Text: "line3"},
	}
	got := ExtractMCPText(content)
	want := "line1\nline2\nline3"
	if got != want {
		t.Errorf("ExtractMCPText = %q, want %q", got, want)
	}
}

func TestExtractMCPText_Empty(t *testing.T) {
	got := ExtractMCPText(nil)
	if got != "Sem conteudo." {
		t.Errorf("ExtractMCPText(nil) = %q, want %q", got, "Sem conteudo.")
	}

	got = ExtractMCPText([]mcp.Content{})
	if got != "Sem conteudo." {
		t.Errorf("ExtractMCPText([]) = %q, want %q", got, "Sem conteudo.")
	}
}

func TestExtractMCPText_NonTextContent(t *testing.T) {
	// ImageContent implements mcp.Content but is not TextContent
	content := []mcp.Content{
		mcp.ImageContent{Type: "image", MIMEType: "image/png", Data: "base64data"},
	}
	got := ExtractMCPText(content)
	if got != "Sem conteudo." {
		t.Errorf("ExtractMCPText with non-text = %q, want %q", got, "Sem conteudo.")
	}
}

func TestExtractMCPText_MixedContent(t *testing.T) {
	content := []mcp.Content{
		mcp.TextContent{Type: "text", Text: "some text"},
		mcp.ImageContent{Type: "image", MIMEType: "image/png", Data: "base64data"},
		mcp.TextContent{Type: "text", Text: "more text"},
	}
	got := ExtractMCPText(content)
	want := "some text\nmore text"
	if got != want {
		t.Errorf("ExtractMCPText mixed = %q, want %q", got, want)
	}
}

func TestMCPToolToOpenAI(t *testing.T) {
	mcpTool := mcp.Tool{
		Name:        "get_weather",
		Description: "Gets the weather for a city",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"city": map[string]interface{}{
					"type":        "string",
					"description": "City name",
				},
			},
		},
	}

	result := MCPToolToOpenAI("server_get_weather", mcpTool)

	if result.Type != openai.ToolTypeFunction {
		t.Errorf("Type = %v, want %v", result.Type, openai.ToolTypeFunction)
	}
	if result.Function == nil {
		t.Fatal("Function is nil")
	}
	if result.Function.Name != "server_get_weather" {
		t.Errorf("Name = %q, want %q", result.Function.Name, "server_get_weather")
	}
	if result.Function.Description != "Gets the weather for a city" {
		t.Errorf("Description = %q, want %q", result.Function.Description, "Gets the weather for a city")
	}

	// Verify parameters is valid JSON containing the schema
	params, ok := result.Function.Parameters.(json.RawMessage)
	if !ok {
		t.Fatalf("Parameters type = %T, want json.RawMessage", result.Function.Parameters)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(params, &schema); err != nil {
		t.Fatalf("Parameters is not valid JSON: %v", err)
	}
	if schema["type"] != "object" {
		t.Errorf("schema type = %v, want %q", schema["type"], "object")
	}
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("schema properties missing or wrong type")
	}
	if _, exists := props["city"]; !exists {
		t.Error("schema properties missing 'city'")
	}
}

func TestMCPToolToOpenAI_EmptySchema(t *testing.T) {
	mcpTool := mcp.Tool{
		Name:        "no_params",
		Description: "A tool with no params",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
		},
	}

	result := MCPToolToOpenAI("server_no_params", mcpTool)

	if result.Function == nil {
		t.Fatal("Function is nil")
	}
	params, ok := result.Function.Parameters.(json.RawMessage)
	if !ok {
		t.Fatalf("Parameters type = %T, want json.RawMessage", result.Function.Parameters)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(params, &schema); err != nil {
		t.Fatalf("Parameters is not valid JSON: %v", err)
	}
	if schema["type"] != "object" {
		t.Errorf("schema type = %v, want %q", schema["type"], "object")
	}
}
