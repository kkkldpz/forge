package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kkkldpz/forge/internal/toolkit"
)

func TestConfigTool_Name(t *testing.T) {
	cfgTool := NewConfigTool()
	if cfgTool.Name() != "config" {
		t.Errorf("Expected name 'config', got '%s'", cfgTool.Name())
	}
}

func TestConfigTool_InputSchema(t *testing.T) {
	cfgTool := NewConfigTool()
	schema := cfgTool.InputSchema()

	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", schema.Type)
	}

	if _, ok := schema.Properties["action"]; !ok {
		t.Error("Expected 'action' property")
	}
}

func TestConfigTool_Call_List(t *testing.T) {
	cfgTool := NewConfigTool()
	input := `{"action":"list"}`
	result := cfgTool.Call(context.Background(), json.RawMessage(input), toolkit.ToolUseContext{})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}
}

func TestConfigTool_Call_Get(t *testing.T) {
	cfgTool := NewConfigTool()
	input := `{"action":"get","key":"default_model"}`
	result := cfgTool.Call(context.Background(), json.RawMessage(input), toolkit.ToolUseContext{})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}
}

func TestConfigTool_Call_Get_UnknownKey(t *testing.T) {
	cfgTool := NewConfigTool()
	input := `{"action":"get","key":"unknown_key"}`
	result := cfgTool.Call(context.Background(), json.RawMessage(input), toolkit.ToolUseContext{})

	if !result.IsError {
		t.Error("Expected error for unknown key")
	}
}

func TestConfigTool_Call_Set(t *testing.T) {
	cfgTool := NewConfigTool()
	input := `{"action":"set","key":"verbose","value":"true"}`
	result := cfgTool.Call(context.Background(), json.RawMessage(input), toolkit.ToolUseContext{})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}
}

func TestConfigTool_IsReadOnly(t *testing.T) {
	cfgTool := NewConfigTool()

	listInput := json.RawMessage(`{"action":"list"}`)
	if !cfgTool.IsReadOnly(listInput) {
		t.Error("Expected 'list' action to be read-only")
	}

	setInput := json.RawMessage(`{"action":"set","key":"test","value":"test"}`)
	if cfgTool.IsReadOnly(setInput) {
		t.Error("Expected 'set' action to not be read-only")
	}
}

func TestSendTool_Name(t *testing.T) {
	sendTool := NewSendTool()
	if sendTool.Name() != "send" {
		t.Errorf("Expected name 'send', got '%s'", sendTool.Name())
	}
}

func TestSendTool_InputSchema(t *testing.T) {
	sendTool := NewSendTool()
	schema := sendTool.InputSchema()

	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", schema.Type)
	}

	if _, ok := schema.Properties["channel"]; !ok {
		t.Error("Expected 'channel' property")
	}

	if _, ok := schema.Properties["message"]; !ok {
		t.Error("Expected 'message' property")
	}
}

func TestSendTool_Call(t *testing.T) {
	sendTool := NewSendTool()
	input := `{"channel":"test-channel","message":"hello"}`
	result := sendTool.Call(context.Background(), json.RawMessage(input), toolkit.ToolUseContext{})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}
}

func TestSendTool_Call_EmptyChannel(t *testing.T) {
	sendTool := NewSendTool()
	input := `{"channel":"","message":"hello"}`
	result := sendTool.Call(context.Background(), json.RawMessage(input), toolkit.ToolUseContext{})

	if !result.IsError {
		t.Error("Expected error for empty channel")
	}
}