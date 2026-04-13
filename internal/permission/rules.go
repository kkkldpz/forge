package permission

import (
	"fmt"
	"path/filepath"
)

// PermissionRuleSource 规则来源。
type PermissionRuleSource string

const (
	RuleSourceCLI       PermissionRuleSource = "cli"
	RuleSourceUser      PermissionRuleSource = "userSettings"
	RuleSourceProject   PermissionRuleSource = "projectSettings"
	RuleSourceLocal     PermissionRuleSource = "localSettings"
	RuleSourceSession   PermissionRuleSource = "session"
)

// PermissionRule 权限规则。
type PermissionRule struct {
	Source      PermissionRuleSource
	Behavior    PermissionBehavior
	ToolName    string
	PathPattern string // 文件路径匹配模式（仅文件工具）
}

// RuleEngine 规则引擎，按优先级评估权限。
type RuleEngine struct {
	rules []PermissionRule
}

// NewRuleEngine 创建规则引擎。
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules: make([]PermissionRule, 0),
	}
}

// AddRule 添加规则。
func (e *RuleEngine) AddRule(rule PermissionRule) {
	e.rules = append(e.rules, rule)
}

// Evaluate 评估工具调用的权限。返回 nil 表示无匹配规则。
func (e *RuleEngine) Evaluate(toolName string, inputPath string) *PermissionDecision {
	// 按来源优先级排序（CLI > Session > Project > User）
	// 实现简化：遍历所有规则，匹配的工具名返回第一个

	// 第一轮：查找 deny 规则（最高优先级）
	for _, rule := range e.rules {
		if rule.Behavior == BehaviorDeny && matchRule(rule, toolName, inputPath) {
			return &PermissionDecision{
				Behavior: BehaviorDeny,
				Message:  fmt.Sprintf("规则拒绝: %s", describeRule(rule)),
				Reason:   ReasonDeniedByRule,
			}
		}
	}

	// 第二轮：查找 allow 规则
	for _, rule := range e.rules {
		if rule.Behavior == BehaviorAllow && matchRule(rule, toolName, inputPath) {
			return &PermissionDecision{
				Behavior: BehaviorAllow,
				Message:  fmt.Sprintf("规则允许: %s", describeRule(rule)),
				Reason:   ReasonAllowedByRule,
			}
		}
	}

	// 第三轮：查找 ask 规则
	for _, rule := range e.rules {
		if rule.Behavior == BehaviorAsk && matchRule(rule, toolName, inputPath) {
			return &PermissionDecision{
				Behavior: BehaviorAsk,
				Message:  "此操作需要确认",
				Reason:   ReasonAskByRule,
			}
		}
	}

	return nil // 无匹配规则
}

// matchRule 检查规则是否匹配。
func matchRule(rule PermissionRule, toolName, inputPath string) bool {
	if rule.ToolName != toolName && rule.ToolName != "*" {
		return false
	}
	if rule.PathPattern != "" && inputPath != "" {
		// 简化的 glob 匹配：检查输入路径是否以模式开头
		// 完整实现应使用 filepath.Match
		matched, err := filepath.Match(rule.PathPattern, inputPath)
		return err == nil && matched
	}
	return true
}

func describeRule(rule PermissionRule) string {
	desc := rule.ToolName
	if rule.PathPattern != "" {
		desc += " (" + rule.PathPattern + ")"
	}
	return desc
}
