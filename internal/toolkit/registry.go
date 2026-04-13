// Package toolkit 提供工具注册和管理功能。
package toolkit

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/kkkldpz/forge/internal/config"
)

// Registry 管理所有可用工具。
type Registry struct {
	tools map[string]Tool
}

// NewRegistry 创建新的工具注册表。
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register 注册一个工具。
func (r *Registry) Register(t Tool) {
	if t == nil {
		return
	}
	r.tools[t.Name()] = t
}

// Get 根据名称获取工具。
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// All 返回所有已注册的工具列表。
func (r *Registry) All() []Tool {
	list := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	// 按名称排序，保证输出稳定
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name() < list[j].Name()
	})
	return list
}

// AllEnabled 返回所有启用的工具。
func (r *Registry) AllEnabled() []Tool {
	list := make([]Tool, 0)
	for _, t := range r.tools {
		if t.IsEnabled() {
			list = append(list, t)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name() < list[j].Name()
	})
	return list
}

// FilterByFeatureFlags 根据功能开关过滤工具。
// 简单模式（CLAUDE_CODE_SIMPLE=1）只保留基础工具。
func (r *Registry) FilterByFeatureFlags() []Tool {
	if config.Feature("SIMPLE") {
		// 简单模式：只保留基础工具
		baseTools := []string{"bash", "file_read", "file_edit"}
		filtered := make([]Tool, 0, len(baseTools))
		for _, name := range baseTools {
			if t, ok := r.Get(name); ok && t.IsEnabled() {
				filtered = append(filtered, t)
			}
		}
		return filtered
	}
	return r.AllEnabled()
}

// FilterByDenyRules 根据拒绝规则过滤工具。
// denyList 是拒绝的工具名称列表。
func (r *Registry) FilterByDenyRules(denyList []string) []Tool {
	denySet := make(map[string]bool)
	for _, name := range denyList {
		denySet[name] = true
	}

	list := make([]Tool, 0)
	for _, t := range r.tools {
		if !denySet[t.Name()] && t.IsEnabled() {
			list = append(list, t)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name() < list[j].Name()
	})
	return list
}

// ToParams 将工具列表转换为 API 请求参数格式。
func ToParams(tools []Tool) []map[string]any {
	params := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		schema := t.InputSchema()
		params = append(params, map[string]any{
			"name":         t.Name(),
			"description":  t.Description(),
			"input_schema": schema,
		})
	}
	return params
}

// ToolConstructor 是工具构造函数类型。
type ToolConstructor func() Tool

// RegisterTools 批量注册工具。
func RegisterTools(r *Registry, constructors ...ToolConstructor) {
	for _, ctor := range constructors {
		if t := ctor(); t != nil {
			r.Register(t)
		}
	}
}

// ParseToolInput 解析工具输入为指定结构体。
func ParseToolInput(input json.RawMessage, v any) error {
	if len(input) == 0 {
		return fmt.Errorf("工具输入为空")
	}
	if err := json.Unmarshal(input, v); err != nil {
		return fmt.Errorf("解析工具输入失败: %w", err)
	}
	return nil
}

// IsSearchOrReadCommand 判断命令是否为搜索或读取类型。
// 用于 UI 折叠和权限检查。
func IsSearchOrReadCommand(toolName, command string) struct {
	IsSearch bool
	IsRead   bool
	IsList   bool
} {
	result := struct {
		IsSearch bool
		IsRead   bool
		IsList   bool
	}{}

	// 搜索类命令
	searchCommands := []string{"grep", "rg", "find", "locate", "ack", "ag"}
	for _, cmd := range searchCommands {
		if strings.HasPrefix(command, cmd+" ") {
			result.IsSearch = true
			return result
		}
	}

	// 读取类命令
	readCommands := []string{"cat", "head", "tail", "less", "more", "read"}
	for _, cmd := range readCommands {
		if strings.HasPrefix(command, cmd+" ") {
			result.IsRead = true
			return result
		}
	}

	// 列表类命令
	listCommands := []string{"ls", "ll", "la", "dir"}
	for _, cmd := range listCommands {
		if command == cmd || strings.HasPrefix(command, cmd+" ") {
			result.IsList = true
			return result
		}
	}

	return result
}
