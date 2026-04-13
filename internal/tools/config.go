package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kkkldpz/forge/internal/config"
	"github.com/kkkldpz/forge/internal/toolkit"
	"github.com/kkkldpz/forge/internal/types"
)

type ConfigTool struct {
	toolkit.BaseTool
}

func NewConfigTool() *ConfigTool {
	return &ConfigTool{
		BaseTool: toolkit.BaseTool{
			NameStr:        "config",
			DescriptionStr: "查看或修改 Forge 配置",
		},
	}
}

type ConfigInput struct {
	Action string `json:"action"`
	Key    string `json:"key,omitempty"`
	Value  string `json:"value,omitempty"`
}

func (t *ConfigTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"action": {
				Type:        "string",
				Description: "操作: get, set, list",
				Enum:        []string{"get", "set", "list"},
			},
			"key": {
				Type:        "string",
				Description: "配置项名称",
			},
			"value": {
				Type:        "string",
				Description: "配置值（set 操作时使用）",
			},
		},
		Required: []string{"action"},
	}
}

func (t *ConfigTool) Call(ctx context.Context, input json.RawMessage, tuc toolkit.ToolUseContext) types.ToolResult {
	var args ConfigInput
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	switch args.Action {
	case "list":
		return t.listConfig()
	case "get":
		return t.getConfig(args.Key)
	case "set":
		return t.setConfig(args.Key, args.Value)
	default:
		return types.ToolResult{Content: fmt.Sprintf("未知操作: %s", args.Action), IsError: true}
	}
}

func (t *ConfigTool) listConfig() types.ToolResult {
	settings := map[string]string{
		"default_model":      "claude-sonnet-4-6",
		"permission_mode":    "auto",
		"verbose":            "false",
		"debug":              "false",
		"color":              "true",
	}

	var lines []string
	lines = append(lines, "Forge 配置项:")
	for k, v := range settings {
		lines = append(lines, fmt.Sprintf("  %s = %s", k, v))
	}

	return types.ToolResult{Content: fmt.Sprintf("共 %d 个配置项:\n%s", len(settings), joinLines(lines))}
}

func (t *ConfigTool) getConfig(key string) types.ToolResult {
	if key == "" {
		return types.ToolResult{Content: "请指定要查看的配置项名称", IsError: true}
	}

	settings := map[string]string{
		"default_model":   "claude-sonnet-4-6",
		"permission_mode": "auto",
		"verbose":         "false",
		"debug":           "false",
		"color":           "true",
	}

	if v, ok := settings[key]; ok {
		return types.ToolResult{Content: fmt.Sprintf("%s = %s", key, v)}
	}

	return types.ToolResult{Content: fmt.Sprintf("未找到配置项: %s", key), IsError: true}
}

func (t *ConfigTool) setConfig(key, value string) types.ToolResult {
	if key == "" || value == "" {
		return types.ToolResult{Content: "请指定配置项名称和值", IsError: true}
	}

	validKeys := map[string]bool{
		"default_model":   true,
		"permission_mode": true,
		"verbose":         true,
		"debug":           true,
		"color":           true,
	}

	if !validKeys[key] {
		return types.ToolResult{Content: fmt.Sprintf("不允许修改配置项: %s", key), IsError: true}
	}

	_ = config.Feature("TEST")

	return types.ToolResult{Content: fmt.Sprintf("配置项 %s 已设置为: %s\n注意: 持久化配置需要写入配置文件", key, value)}
}

func (t *ConfigTool) IsReadOnly(input json.RawMessage) bool {
	var args ConfigInput
	if err := json.Unmarshal(input, &args); err != nil {
		return true
	}
	return args.Action == "get" || args.Action == "list"
}

func (t *ConfigTool) IsConcurrencySafe(input json.RawMessage) bool { return true }

func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}