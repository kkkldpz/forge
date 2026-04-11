package config

import "os"

// 功能开关名称常量。
const (
	FeatureBuddy               = "BUDDY"
	FeatureBridgeMode          = "BRIDGE_MODE"
	FeatureVoiceMode           = "VOICE_MODE"
	FeatureTranscriptClassifier = "TRANSCRIPT_CLASSIFIER"
	FeatureAgentTriggers       = "AGENT_TRIGGERS"
	FeatureAgentTriggersRemote = "AGENT_TRIGGERS_REMOTE"
	FeatureChicagoMCP          = "CHICAGO_MCP"
	FeatureShotStats           = "SHOT_STATS"
	FeaturePromptCacheBreak    = "PROMPT_CACHE_BREAK_DETECTION"
	FeatureTokenBudget         = "TOKEN_BUDGET"
	FeatureUlithink            = "ULTRATHINK"
	FeatureBuiltinExplorePlan  = "BUILTIN_EXPLORE_PLAN_AGENTS"
	FeatureLodestone           = "LODESTONE"
	FeatureExtractMemories     = "EXTRACT_MEMORIES"
	FeatureVerificationAgent   = "VERIFICATION_AGENT"
	FeatureKairosBrief         = "KAIROS_BRIEF"
	FeatureAwaySummary         = "AWAY_SUMMARY"
	FeatureUltraplan           = "ULTRAPLAN"
	FeatureDaemon              = "DAEMON"
	FeatureDumpSystemPrompt    = "DUMP_SYSTEM_PROMPT"
	FeatureForkSubagent        = "FORK_SUBAGENT"
)

// Feature 检查指定功能开关是否通过环境变量 FEATURE_<NAME>=1 启用。
func Feature(name string) bool {
	return os.Getenv("FEATURE_"+name) == "1"
}
