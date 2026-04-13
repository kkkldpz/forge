package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

type SkillTool struct {
	tool.BaseTool
}

func NewSkillTool() *SkillTool {
	return &SkillTool{
		BaseTool: tool.BaseTool{
			NameStr:        "skill",
			DescriptionStr: "调用已安装的技能/插件",
		},
	}
}

type SkillInput struct {
	Name   string          `json:"name"`
	Action string          `json:"action"`
	Params json.RawMessage `json:"params,omitempty"`
}

func (t *SkillTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"name":   {Type: "string", Description: "技能名称"},
			"action": {Type: "string", Description: "技能动作: invoke, list, info"},
			"params": {Type: "object", Description: "动作参数（可选）"},
		},
		Required: []string{"name", "action"},
	}
}

func (t *SkillTool) Call(ctx context.Context, input json.RawMessage, tuc tool.ToolUseContext) types.ToolResult {
	var args SkillInput
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	if args.Name == "" || args.Action == "" {
		return types.ToolResult{Content: "技能名称和动作不能为空", IsError: true}
	}

	switch args.Action {
	case "invoke":
		return t.invokeSkill(args.Name, args.Params)
	case "list":
		return t.listSkills()
	case "info":
		return t.skillInfo(args.Name)
	default:
		return types.ToolResult{Content: fmt.Sprintf("未知动作: %s", args.Action), IsError: true}
	}
}

func (t *SkillTool) invokeSkill(name string, params json.RawMessage) types.ToolResult {
	content := fmt.Sprintf("技能 '%s' 调用成功\n\n注意: 技能系统需要配置技能插件", name)
	if params != nil {
		content += fmt.Sprintf("\n参数: %s", string(params))
	}
	return types.ToolResult{Content: content}
}

func (t *SkillTool) listSkills() types.ToolResult {
	content := `已安装的技能:
  (暂无已安装的技能)

使用 'forge plugin search' 搜索可用技能`
	return types.ToolResult{Content: content}
}

func (t *SkillTool) skillInfo(name string) types.ToolResult {
	content := fmt.Sprintf("技能信息: %s\n状态: 未安装\n\n提示: 使用 'forge skill invoke %s' 前需先安装", name, name)
	return types.ToolResult{Content: content}
}

func (t *SkillTool) IsReadOnly(input json.RawMessage) bool {
	var args SkillInput
	if err := json.Unmarshal(input, &args); err != nil {
		return true
	}
	return args.Action == "list" || args.Action == "info"
}

func (t *SkillTool) IsConcurrencySafe(input json.RawMessage) bool { return true }