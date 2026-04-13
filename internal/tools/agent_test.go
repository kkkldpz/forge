package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

func TestAgentTool_InputSchema(t *testing.T) {
	at := NewAgentTool()

	if at.Name() != "agent" {
		t.Errorf("Expected name 'agent', got '%s'", at.Name())
	}

	schema := at.InputSchema()
	if schema.Type != "object" {
		t.Errorf("Expected schema type 'object', got '%s'", schema.Type)
	}

	// prompt is required
	found := false
	for _, r := range schema.Required {
		if r == "prompt" {
			found = true
		}
	}
	if !found {
		t.Error("'prompt' should be required")
	}
}

func TestAgentTool_EmptyPrompt(t *testing.T) {
	at := NewAgentTool()

	result := at.Call(context.Background(), json.RawMessage(`{}`), tool.ToolUseContext{})
	if !result.IsError {
		t.Error("Empty prompt should return error")
	}
}

func TestAgentTool_InvalidJSON(t *testing.T) {
	at := NewAgentTool()

	result := at.Call(context.Background(), json.RawMessage(`invalid`), tool.ToolUseContext{})
	if !result.IsError {
		t.Error("Invalid JSON should return error")
	}
}

func TestAgentTool_NoProvider(t *testing.T) {
	at := NewAgentTool()

	result := at.Call(context.Background(),
		json.RawMessage(`{"prompt":"do something"}`),
		tool.ToolUseContext{},
	)
	if !result.IsError {
		t.Error("Missing provider should return error")
	}
	if result.Content == "" {
		t.Error("Error should have a message")
	}
}

func TestAgentTool_BackgroundMode(t *testing.T) {
	at := NewAgentTool()

	// Background mode with no real provider should still start and fail gracefully
	result := at.Call(context.Background(),
		json.RawMessage(`{"prompt":"test task","run_in_background":true}`),
		tool.ToolUseContext{},
	)

	// Should fail because no provider
	if !result.IsError {
		t.Error("Should fail without provider")
	}
}

func TestAgentTool_IsNotReadOnly(t *testing.T) {
	at := NewAgentTool()
	if at.IsReadOnly(nil) {
		t.Error("Agent tool should not be read-only")
	}
}

func TestAgentTool_IsNotConcurrencySafe(t *testing.T) {
	at := NewAgentTool()
	if at.IsConcurrencySafe(nil) {
		t.Error("Agent tool should not be concurrency-safe")
	}
}

func TestAgentTool_ListAgents(t *testing.T) {
	at := NewAgentTool()

	// Initially empty
	agents := at.ListAgents()
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents, got %d", len(agents))
	}
}

func TestAgentTool_GetAgent_NotFound(t *testing.T) {
	at := NewAgentTool()

	_, ok := at.GetAgent("nonexistent")
	if ok {
		t.Error("Should not find nonexistent agent")
	}
}

func TestAgentTool_CancelAgent_NotFound(t *testing.T) {
	at := NewAgentTool()

	cancelled := at.CancelAgent("nonexistent")
	if cancelled {
		t.Error("Should not cancel nonexistent agent")
	}
}

func TestAgentTool_GetResult_NotFound(t *testing.T) {
	at := NewAgentTool()

	_, ok := at.GetResult("nonexistent")
	if ok {
		t.Error("Should not find nonexistent result")
	}
}

func TestFilterToolsForSubAgent(t *testing.T) {
	tools := []tool.Tool{
		&mockTool{name: "bash"},
		&mockTool{name: "agent"},
		&mockTool{name: "file_read"},
	}

	filtered := filterToolsForSubAgent(tools)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 tools after filtering, got %d", len(filtered))
	}
	for _, t2 := range filtered {
		if t2.Name() == "agent" {
			t.Error("Agent tool should be filtered out")
		}
	}
}

func TestExtractLastAssistantText(t *testing.T) {
	messages := []types.Message{
		{
			Type: types.MessageTypeUser,
			Message: &types.MessageContent{
				Role:    "user",
				Content: json.RawMessage(`[{"type":"text","text":"hello"}]`),
			},
		},
		{
			Type: types.MessageTypeAssistant,
			Message: &types.MessageContent{
				Role:    "assistant",
				Content: json.RawMessage(`[{"type":"text","text":"world"}]`),
			},
		},
	}

	text := extractLastAssistantText(messages)
	if text != "world" {
		t.Errorf("Expected 'world', got '%s'", text)
	}
}

func TestExtractLastAssistantText_Empty(t *testing.T) {
	text := extractLastAssistantText(nil)
	if text != "" {
		t.Errorf("Expected empty string, got '%s'", text)
	}
}

// mockTool is a minimal Tool implementation for testing.
type mockTool struct {
	name string
}

func (m *mockTool) Name() string                                          { return m.name }
func (m *mockTool) Description() string                                   { return "mock" }
func (m *mockTool) InputSchema() types.ToolInputJSONSchema                { return types.ToolInputJSONSchema{} }
func (m *mockTool) Call(ctx context.Context, input json.RawMessage, tuc tool.ToolUseContext) types.ToolResult {
	return types.ToolResult{Content: "mock result"}
}
func (m *mockTool) ValidateInput(ctx context.Context, input json.RawMessage) types.ValidationResult {
	return types.ValidationResult{Valid: true}
}
func (m *mockTool) IsReadOnly(input json.RawMessage) bool                 { return false }
func (m *mockTool) IsConcurrencySafe(input json.RawMessage) bool          { return false }
func (m *mockTool) IsEnabled() bool                                       { return true }
