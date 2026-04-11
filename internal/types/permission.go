package types

// PermissionMode 控制工具权限的处理方式。
type PermissionMode string

const (
	PermissionModeDefault     PermissionMode = "default"
	PermissionModeBypass      PermissionMode = "bypassPermissions"
	PermissionModeDontAsk     PermissionMode = "dontAsk"
	PermissionModeAcceptEdits PermissionMode = "acceptEdits"
	PermissionModePlan        PermissionMode = "plan"
	PermissionModeAuto        PermissionMode = "auto"
)

// PermissionBehavior 表示权限检查的动作。
type PermissionBehavior string

const (
	PermissionBehaviorAllow PermissionBehavior = "allow"
	PermissionBehaviorDeny  PermissionBehavior = "deny"
	PermissionBehaviorAsk   PermissionBehavior = "ask"
)

// PermissionRuleSource 表示权限规则的来源。
type PermissionRuleSource string

const (
	RuleSourceUserSettings    PermissionRuleSource = "userSettings"
	RuleSourceProjectSettings PermissionRuleSource = "projectSettings"
	RuleSourceLocalSettings   PermissionRuleSource = "localSettings"
	RuleSourceFlagSettings    PermissionRuleSource = "flagSettings"
	RuleSourcePolicySettings  PermissionRuleSource = "policySettings"
	RuleSourceCLIArg          PermissionRuleSource = "cliArg"
	RuleSourceCommand         PermissionRuleSource = "command"
	RuleSourceSession         PermissionRuleSource = "session"
)

// PermissionRuleValue 描述工具权限的匹配模式。
type PermissionRuleValue struct {
	ToolName    string `json:"toolName"`
	RuleContent string `json:"ruleContent,omitempty"`
}

// PermissionRule 是来自特定来源的单条权限规则。
type PermissionRule struct {
	Source       PermissionRuleSource `json:"source"`
	RuleBehavior PermissionBehavior   `json:"ruleBehavior"`
	RuleValue    PermissionRuleValue  `json:"ruleValue"`
}

// PermissionResult 是权限检查的结果。
type PermissionResult struct {
	Behavior    PermissionBehavior `json:"behavior"`
	Message     string             `json:"message,omitempty"`
	Rule        *PermissionRule    `json:"rule,omitempty"`
	Suggestions []PermissionUpdate `json:"suggestions,omitempty"`
}

// PermissionAllow 创建允许结果。
func PermissionAllow() PermissionResult {
	return PermissionResult{Behavior: PermissionBehaviorAllow}
}

// PermissionDeny 创建带消息的拒绝结果。
func PermissionDeny(msg string) PermissionResult {
	return PermissionResult{Behavior: PermissionBehaviorDeny, Message: msg}
}

// PermissionAsk 创建询问结果。
func PermissionAsk(msg string) PermissionResult {
	return PermissionResult{Behavior: PermissionBehaviorAsk, Message: msg}
}

// PermissionUpdateDestination 指定权限更新的应用位置。
type PermissionUpdateDestination string

const (
	UpdateDestUserSettings    PermissionUpdateDestination = "userSettings"
	UpdateDestProjectSettings PermissionUpdateDestination = "projectSettings"
	UpdateDestLocalSettings   PermissionUpdateDestination = "localSettings"
	UpdateDestSession         PermissionUpdateDestination = "session"
	UpdateDestCLIArg          PermissionUpdateDestination = "cliArg"
)

// PermissionUpdate 表示对权限规则的变更。
type PermissionUpdate struct {
	Type        PermissionUpdateType        `json:"type"`
	Destination PermissionUpdateDestination `json:"destination"`
	Rules       []PermissionRuleValue       `json:"rules,omitempty"`
	Behavior    PermissionBehavior          `json:"behavior,omitempty"`
	Mode        PermissionMode              `json:"mode,omitempty"`
	Directories []string                    `json:"directories,omitempty"`
}

// PermissionUpdateType 描述权限更新的类型。
type PermissionUpdateType string

const (
	UpdateTypeAddRules     PermissionUpdateType = "addRules"
	UpdateTypeReplaceRules PermissionUpdateType = "replaceRules"
	UpdateTypeRemoveRules  PermissionUpdateType = "removeRules"
	UpdateTypeSetMode      PermissionUpdateType = "setMode"
	UpdateTypeAddDirs      PermissionUpdateType = "addDirectories"
	UpdateTypeRemoveDirs   PermissionUpdateType = "removeDirectories"
)

// AdditionalWorkingDirectory 表示项目根目录之外的工作目录。
type AdditionalWorkingDirectory struct {
	Path   string `json:"path"`
	Source string `json:"source"`
}
