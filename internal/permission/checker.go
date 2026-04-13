package permission

import (
	"fmt"
	"log/slog"
	"strings"
)

// ToolInfo 工具信息，供权限检查器使用。
type ToolInfo struct {
	Name      string
	InputPath string // 工具操作的目标路径（文件工具）
	InputCmd  string // Bash 工具的命令内容
}

// CheckToolPermission 检查工具调用是否被允许。
// 返回需要用户确认的决策。
func CheckToolPermission(tool ToolInfo, ctx *ToolPermissionContext, rules *RuleEngine) PermissionDecision {
	// 1. Bypass 模式：全部允许
	if ctx.Mode == ModeBypass {
		return AllowDecision(ReasonBypassMode)
	}

	// 2. 检查始终允许/拒绝列表
	if ctx.IsToolAlwaysDenied(tool.Name) {
		return DenyDecision("工具被始终拒绝", ReasonDeniedByRule)
	}
	if ctx.IsToolAlwaysAllowed(tool.Name) {
		return AllowDecision(ReasonAllowedByRule)
	}

	// 3. 规则引擎评估
	if rules != nil {
		if decision := rules.Evaluate(tool.Name, tool.InputPath); decision != nil {
			// dontAsk 模式下将 ask 转为 deny
			if decision.Behavior == BehaviorAsk && ctx.Mode == ModeDontAsk {
				return DenyDecision("dontAsk 模式下拒绝需要确认的操作", ReasonDontAskMode)
			}
			return *decision
		}
	}

	// 4. 根据工具类型进行安全检查
	switch {
	case strings.HasPrefix(tool.Name, "bash"):
		return checkBashPermission(tool, ctx)
	case strings.HasPrefix(tool.Name, "file_"):
		return checkFilePermission(tool, ctx)
	default:
		return AskDecision(fmt.Sprintf("工具 %s 需要确认", tool.Name), ReasonAskByRule)
	}
}

// checkBashPermission 检查 Bash 工具权限。
func checkBashPermission(tool ToolInfo, ctx *ToolPermissionContext) PermissionDecision {
	cmd := tool.InputCmd

	if IsDangerousBashCommand(cmd) {
		slog.Warn("检测到危险 Bash 命令", "command", cmd)
		return AskDecision(
			fmt.Sprintf("危险命令需要确认:\n  %s", cmd),
			ReasonDangerousCommand,
		)
	}

	if IsReadOnlyBashCommand(cmd) {
		return AllowDecision(ReasonReadOnlyTool)
	}

	if ctx.Mode == ModeAcceptEdits {
		return AllowDecision(ReasonAcceptEditsMode)
	}

	return AskDecision(
		fmt.Sprintf("Bash 命令需要确认:\n  %s", cmd),
		ReasonAskByRule,
	)
}

// checkFilePermission 检查文件工具权限。
func checkFilePermission(tool ToolInfo, ctx *ToolPermissionContext) PermissionDecision {
	targetPath := tool.InputPath

	if targetPath == "" {
		return AskDecision(fmt.Sprintf("文件操作需要确认: %s", tool.Name), ReasonAskByRule)
	}

	// 确定操作类型
	op := FileWrite
	if tool.Name == "file_read" {
		op = FileRead
	}

	// 只读操作自动允许
	if op == FileRead && ctx.Mode != ModeDontAsk {
		return AllowDecision(ReasonReadOnlyTool)
	}

	// 路径安全验证
	result := ValidateFilePath(targetPath, ctx.CWD, op)
	if !result.Allowed {
		slog.Warn("文件路径检查失败", "path", targetPath, "reason", result.Reason)

		if ctx.Mode == ModeDontAsk {
			return DenyDecision(result.Reason, ReasonDangerousPath)
		}

		return AskDecision(result.Reason, ReasonDangerousPath)
	}

	if ctx.Mode == ModeAcceptEdits {
		return AllowDecision(ReasonAcceptEditsMode)
	}

	return AskDecision(
		fmt.Sprintf("文件写入需要确认:\n  %s\n  工具: %s", targetPath, tool.Name),
		ReasonOutsideWorkingDir,
	)
}

// IsReadOnlyTool 判断工具是否为只读工具。
func IsReadOnlyTool(toolName string) bool {
	readOnlyTools := map[string]bool{
		"file_read": true,
		"glob":      true,
		"grep":      true,
	}
	return readOnlyTools[toolName]
}
