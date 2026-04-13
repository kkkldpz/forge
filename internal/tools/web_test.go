package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kkkldpz/forge/internal/tool"
)

func TestWebFetchTool_InputSchema(t *testing.T) {
	webTool := NewWebFetchTool()
	if webTool.Name() != "webfetch" {
		t.Errorf("Expected name 'webfetch', got '%s'", webTool.Name())
	}

	schema := webTool.InputSchema()
	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", schema.Type)
	}
	if _, ok := schema.Properties["url"]; !ok {
		t.Error("Expected 'url' property")
	}
}

func TestWebFetchTool_Call_InvalidURL(t *testing.T) {
	webTool := NewWebFetchTool()
	input := `{"url":""}`
	result := webTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{WorkingDir: "/tmp"})

	if !result.IsError {
		t.Error("Expected error for empty URL")
	}
}

func TestWebFetchTool_Call_LocalhostURL(t *testing.T) {
	webTool := NewWebFetchTool()
	input := `{"url":"http://localhost:8080/test"}`
	result := webTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{WorkingDir: "/tmp"})

	if !result.IsError {
		t.Error("Expected error for localhost URL")
	}
}

func TestWebSearchTool_InputSchema(t *testing.T) {
	webTool := NewWebSearchTool()
	if webTool.Name() != "websearch" {
		t.Errorf("Expected name 'websearch', got '%s'", webTool.Name())
	}

	schema := webTool.InputSchema()
	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", schema.Type)
	}
	if _, ok := schema.Properties["query"]; !ok {
		t.Error("Expected 'query' property")
	}
}

func TestWebSearchTool_Call_EmptyQuery(t *testing.T) {
	webTool := NewWebSearchTool()
	input := `{"query":""}`
	result := webTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{WorkingDir: "/tmp"})

	if !result.IsError {
		t.Error("Expected error for empty query")
	}
}
