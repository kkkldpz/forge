package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kkkldpz/forge/internal/tool"
)

func TestSkillTool_Name(t *testing.T) {
	skillTool := NewSkillTool()
	if skillTool.Name() != "skill" {
		t.Errorf("Expected name 'skill', got '%s'", skillTool.Name())
	}
}

func TestSkillTool_InputSchema(t *testing.T) {
	skillTool := NewSkillTool()
	schema := skillTool.InputSchema()

	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", schema.Type)
	}

	if _, ok := schema.Properties["name"]; !ok {
		t.Error("Expected 'name' property")
	}

	if _, ok := schema.Properties["action"]; !ok {
		t.Error("Expected 'action' property")
	}
}

func TestSkillTool_Call_Invoke(t *testing.T) {
	skillTool := NewSkillTool()
	input := `{"name":"test-skill","action":"invoke"}`
	result := skillTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}
}

func TestSkillTool_Call_List(t *testing.T) {
	skillTool := NewSkillTool()
	input := `{"name":"any","action":"list"}`
	result := skillTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}
}

func TestSkillTool_Call_Info(t *testing.T) {
	skillTool := NewSkillTool()
	input := `{"name":"my-skill","action":"info"}`
	result := skillTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}
}

func TestSkillTool_Call_UnknownAction(t *testing.T) {
	skillTool := NewSkillTool()
	input := `{"name":"test","action":"unknown"}`
	result := skillTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{})

	if !result.IsError {
		t.Error("Expected error for unknown action")
	}
}

func TestSkillTool_Call_EmptyParams(t *testing.T) {
	skillTool := NewSkillTool()
	input := `{"name":"test","action":""}`
	result := skillTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{})

	if !result.IsError {
		t.Error("Expected error for empty action")
	}
}

func TestSkillTool_IsReadOnly(t *testing.T) {
	skillTool := NewSkillTool()

	listInput := json.RawMessage(`{"name":"test","action":"list"}`)
	if !skillTool.IsReadOnly(listInput) {
		t.Error("Expected 'list' action to be read-only")
	}

	invokeInput := json.RawMessage(`{"name":"test","action":"invoke"}`)
	if skillTool.IsReadOnly(invokeInput) {
		t.Error("Expected 'invoke' action to not be read-only")
	}
}

func TestLSPTool_Name(t *testing.T) {
	lspTool := NewLSPTool()
	if lspTool.Name() != "lsp" {
		t.Errorf("Expected name 'lsp', got '%s'", lspTool.Name())
	}
}

func TestLSPTool_InputSchema(t *testing.T) {
	lspTool := NewLSPTool()
	schema := lspTool.InputSchema()

	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", schema.Type)
	}

	if _, ok := schema.Properties["action"]; !ok {
		t.Error("Expected 'action' property")
	}
}

func TestLSPTool_Call_Initialize(t *testing.T) {
	lspTool := NewLSPTool()
	input := `{"action":"initialize"}`
	result := lspTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}
}

func TestLSPTool_Call_Hover(t *testing.T) {
	lspTool := NewLSPTool()
	input := `{"action":"hover","document":"/test/file.go","line":10,"character":5}`
	result := lspTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}
}

func TestLSPTool_Call_Hover_NoDocument(t *testing.T) {
	lspTool := NewLSPTool()
	input := `{"action":"hover"}`
	result := lspTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{})

	if !result.IsError {
		t.Error("Expected error for hover without document")
	}
}

func TestLSPTool_Call_Completion(t *testing.T) {
	lspTool := NewLSPTool()
	input := `{"action":"completion","document":"/test/file.go","line":10,"character":5}`
	result := lspTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}
}

func TestLSPTool_Call_Diagnostics(t *testing.T) {
	lspTool := NewLSPTool()
	input := `{"action":"diagnostics","document":"/test/file.go"}`
	result := lspTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}
}

func TestLSPTool_Call_UnknownAction(t *testing.T) {
	lspTool := NewLSPTool()
	input := `{"action":"unknown"}`
	result := lspTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{})

	if !result.IsError {
		t.Error("Expected error for unknown action")
	}
}

func TestMCPProxyTool_Name(t *testing.T) {
	mcpTool := NewMCPProxyTool()
	if mcpTool.Name() != "mcp_proxy" {
		t.Errorf("Expected name 'mcp_proxy', got '%s'", mcpTool.Name())
	}
}

func TestMCPProxyTool_InputSchema(t *testing.T) {
	mcpTool := NewMCPProxyTool()
	schema := mcpTool.InputSchema()

	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", schema.Type)
	}

	if _, ok := schema.Properties["server"]; !ok {
		t.Error("Expected 'server' property")
	}

	if _, ok := schema.Properties["method"]; !ok {
		t.Error("Expected 'method' property")
	}
}

func TestMCPProxyTool_Call(t *testing.T) {
	mcpTool := NewMCPProxyTool()
	input := `{"server":"test-server","method":"tools/list"}`
	result := mcpTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}
}

func TestMCPProxyTool_Call_EmptyServer(t *testing.T) {
	mcpTool := NewMCPProxyTool()
	input := `{"server":"","method":"test"}`
	result := mcpTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{})

	if !result.IsError {
		t.Error("Expected error for empty server")
	}
}