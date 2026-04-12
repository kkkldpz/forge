// Package context 实现系统提示词、用户上下文和 CLAUDE.md 加载。
package context

import (
	"log/slog"
	"time"

	"github.com/kkkldpz/forge/internal/api"
)

// GetLocalISODate 获取本地时区的 ISO 日期字符串 (YYYY-MM-DD)。
func GetLocalISODate() string {
	now := time.Now()
	return now.Format("2006-01-02")
}

// GetLocalMonthYear 获取本地时区的月份年份字符串 (如 "2026年4月")。
func GetLocalMonthYear() string {
	now := time.Now()
	return now.Format("2006年1月")
}

// SystemContext 包含系统级别的上下文信息（git 状态、环境变量等）。
type SystemContext struct {
	GitStatus    string `json:"gitStatus,omitempty"`
	CacheBreaker string `json:"cacheBreaker,omitempty"`
}

// UserContext 包含用户级别的上下文信息（CLAUDE.md、日期等）。
type UserContext struct {
	ClaudeMd   string `json:"claudeMd,omitempty"`
	CurrentDate string `json:"currentDate"`
}

// GetSystemContext 构建系统上下文，包含 git 状态等环境信息。
// 结果应在会话期间缓存，避免重复执行 git 命令。
func GetSystemContext(cwd string) SystemContext {
	slog.Debug("开始构建系统上下文")

	gitStatus := GetGitStatus(cwd)

	slog.Debug("系统上下文构建完成",
		"hasGitStatus", gitStatus != "",
	)

	return SystemContext{
		GitStatus: gitStatus,
	}
}

// GetUserContext 构建用户上下文，包含 CLAUDE.md 内容和当前日期。
// 结果应在会话期间缓存。
func GetUserContext(cwd string, homeDir string) UserContext {
	slog.Debug("开始构建用户上下文", "cwd", cwd)

	// 发现并加载 CLAUDE.md 文件
	memoryFiles := GetMemoryFiles(cwd, homeDir)
	claudeMd := GetClaudeMds(memoryFiles)

	slog.Debug("用户上下文构建完成",
		"claudeMdLength", len(claudeMd),
		"memoryFileCount", len(memoryFiles),
	)

	return UserContext{
		ClaudeMd:   claudeMd,
		CurrentDate: "Today's date is " + GetLocalISODate() + ".",
	}
}

// BuildSystemBlocks 将系统提示词部分和用户上下文组装为 API 所需的 SystemBlock 列表。
// userCtx 中的内容以 map 形式注入（key-value 格式），放在系统提示词之后。
func BuildSystemBlocks(
	promptParts []SystemPromptPart,
	userCtx UserContext,
	sysCtx SystemContext,
) []api.SystemBlock {
	var blocks []api.SystemBlock

	// 1. 系统提示词各部分（静态，大部分带缓存标记）
	for _, part := range promptParts {
		block := api.SystemBlock{
			Type: "text",
			Text: part.Text,
		}
		if part.CacheControl {
			block.CacheControl = &api.CacheControl{Type: "ephemeral"}
		}
		blocks = append(blocks, block)
	}

	// 2. 用户上下文注入点（动态内容）
	if userCtx.CurrentDate != "" {
		blocks = append(blocks, api.SystemBlock{
			Type: "text",
			Text: "# currentDate\n" + userCtx.CurrentDate,
		})
	}

	if userCtx.ClaudeMd != "" {
		blocks = append(blocks, api.SystemBlock{
			Type: "text",
			Text: userCtx.ClaudeMd,
		})
	}

	// 3. 系统上下文（git 状态等）
	if sysCtx.GitStatus != "" {
		blocks = append(blocks, api.SystemBlock{
			Type:         "text",
			Text:         "# gitStatus\n" + sysCtx.GitStatus,
			CacheControl: &api.CacheControl{Type: "ephemeral"},
		})
	}

	return blocks
}

// FetchContext 并行获取系统提示词、用户上下文和系统上下文。
// 这是对外的主入口函数，供 QueryEngine 调用。
func FetchContext(model string, cwd string, homeDir string, enabledTools []string) (
	promptParts []SystemPromptPart,
	userCtx UserContext,
	sysCtx SystemContext,
) {
	slog.Info("开始获取完整上下文", "model", model, "cwd", cwd)

	// 构建系统提示词模板
	promptParts = GetSystemPrompt(model, cwd, homeDir, enabledTools)

	// 获取用户上下文（CLAUDE.md + 日期）
	userCtx = GetUserContext(cwd, homeDir)

	// 获取系统上下文（git 状态）
	sysCtx = GetSystemContext(cwd)

	slog.Info("完整上下文获取完成",
		"promptParts", len(promptParts),
		"hasClaudeMd", userCtx.ClaudeMd != "",
		"hasGitStatus", sysCtx.GitStatus != "",
	)

	return
}
