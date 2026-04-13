// Package permission 实现完整的权限系统：模式、规则、路径验证、命令安全。
package permission

// PermissionMode 权限模式。
type PermissionMode string

const (
	ModeDefault     PermissionMode = "default"
	ModeBypass      PermissionMode = "bypassPermissions"
	ModeDontAsk     PermissionMode = "dontAsk"
	ModeAcceptEdits PermissionMode = "acceptEdits"
	ModePlan        PermissionMode = "plan"
	ModeAuto        PermissionMode = "auto"
)

// PermissionBehavior 权限行为。
type PermissionBehavior string

const (
	BehaviorAllow PermissionBehavior = "allow"
	BehaviorAsk   PermissionBehavior = "ask"
	BehaviorDeny   PermissionBehavior = "deny"
)

// PermissionDecision 权限决策结果。
type PermissionDecision struct {
	Behavior PermissionBehavior
	Message  string // 供 UI 显示的提示文本
	Reason   DecisionReason
}

// DecisionReason 决策原因，用于日志和审计。
type DecisionReason string

const (
	ReasonAllowedByRule      DecisionReason = "allowed_by_rule"
	ReasonDeniedByRule       DecisionReason = "denied_by_rule"
	ReasonAskByRule          DecisionReason = "ask_by_rule"
	ReasonReadOnlyTool       DecisionReason = "read_only_tool"
	ReasonOutsideWorkingDir  DecisionReason = "outside_working_dir"
	ReasonDangerousPath      DecisionReason = "dangerous_path"
	ReasonDangerousCommand   DecisionReason = "dangerous_command"
	ReasonBypassMode         DecisionReason = "bypass_mode"
	ReasonAcceptEditsMode    DecisionReason = "accept_edits_mode"
	ReasonDontAskMode        DecisionReason = "dont_ask_mode"
	ReasonAutoAllowed        DecisionReason = "auto_allowed"
	ReasonAutoBlocked        DecisionReason = "auto_blocked"
)

// AllowDecision 创建允许决策。
func AllowDecision(reason DecisionReason) PermissionDecision {
	return PermissionDecision{
		Behavior: BehaviorAllow,
		Reason:   reason,
	}
}

// AskDecision 创建请求确认决策。
func AskDecision(message string, reason DecisionReason) PermissionDecision {
	return PermissionDecision{
		Behavior: BehaviorAsk,
		Message:  message,
		Reason:   reason,
	}
}

// DenyDecision 创建拒绝决策。
func DenyDecision(message string, reason DecisionReason) PermissionDecision {
	return PermissionDecision{
		Behavior: BehaviorDeny,
		Message:  message,
		Reason:   reason,
	}
}

// ToolPermissionContext 工具执行权限上下文。
type ToolPermissionContext struct {
	Mode                  PermissionMode
	CWD                   string
	AdditionalWorkingDirs map[string]bool
	AlwaysAllowTools      map[string]bool // 工具名 → 始终允许
	AlwaysDenyTools       map[string]bool // 工具名 → 始终拒绝
}

// NewToolPermissionContext 创建默认的权限上下文。
func NewToolPermissionContext(cwd string, mode PermissionMode) *ToolPermissionContext {
	return &ToolPermissionContext{
		Mode:                  mode,
		CWD:                   cwd,
		AdditionalWorkingDirs: make(map[string]bool),
		AlwaysAllowTools:      make(map[string]bool),
		AlwaysDenyTools:       make(map[string]bool),
	}
}

// IsToolAlwaysAllowed 检查工具是否在始终允许列表中。
func (c *ToolPermissionContext) IsToolAlwaysAllowed(toolName string) bool {
	return c.AlwaysAllowTools[toolName]
}

// IsToolAlwaysDenied 检查工具是否在始终拒绝列表中。
func (c *ToolPermissionContext) IsToolAlwaysDenied(toolName string) bool {
	return c.AlwaysDenyTools[toolName]
}
