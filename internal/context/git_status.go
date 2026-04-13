// Package context 实现系统提示词、用户上下文和 CLAUDE.md 加载。
package context

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

const maxStatusChars = 2000

// gitExe 返回 git 可执行文件路径。
func gitExe() string {
	return "git"
}

// runGit 在指定目录执行 git 命令并返回 stdout。
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command(gitExe(), args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// isGitDir 检查指定目录是否在 git 仓库内。
func isGitDir(dir string) bool {
	_, err := runGit(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil
}

// getGitBranch 获取当前分支名。
func getGitBranch(dir string) string {
	out, err := runGit(dir, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		// detached HEAD 场景
		out, err2 := runGit(dir, "rev-parse", "--short", "HEAD")
		if err2 != nil {
			return "unknown"
		}
		return out
	}
	return out
}

// getDefaultBranch 获取默认分支名（main 或 master）。
func getDefaultBranch(dir string) string {
	// 优先检查 main
	out, err := runGit(dir, "symbolic-ref", "refs/remotes/origin/main")
	if err == nil && out != "" {
		return strings.TrimPrefix(out, "refs/remotes/origin/")
	}
	// 回退到 master
	out, err = runGit(dir, "symbolic-ref", "refs/remotes/origin/master")
	if err == nil && out != "" {
		return strings.TrimPrefix(out, "refs/remotes/origin/")
	}
	return "main"
}

// GetGitStatus 获取 git 状态信息，格式化为系统提示词片段。
// 返回空字符串表示不在 git 仓库中。
func GetGitStatus(dir string) string {
	slog.Debug("开始获取 git 状态", "dir", dir)

	if !isGitDir(dir) {
		slog.Debug("当前目录不在 git 仓库中", "dir", dir)
		return ""
	}

	branch := getGitBranch(dir)
	mainBranch := getDefaultBranch(dir)

	status, _ := runGit(dir, "--no-optional-locks", "status", "--short")
	log, _ := runGit(dir, "--no-optional-locks", "log", "--oneline", "-n", "5")
	userName, _ := runGit(dir, "config", "user.name")

	var truncated string
	if len(status) > maxStatusChars {
		slog.Debug("git status 超长，进行截断", "length", len(status))
		truncated = status[:maxStatusChars] +
			"\n... (truncated because it exceeds 2k characters. If you need more information, run \"git status\" using BashTool)"
	} else {
		truncated = status
	}

	if status == "" {
		truncated = "(clean)"
	}

	parts := []string{
		"This is the git status at the start of the conversation. Note that this status is a snapshot in time, and will not update during the conversation.",
		fmt.Sprintf("Current branch: %s", branch),
		fmt.Sprintf("Main branch (you will usually use this for PRs): %s", mainBranch),
	}
	if userName != "" {
		parts = append(parts, fmt.Sprintf("Git user: %s", userName))
	}
	parts = append(parts,
		fmt.Sprintf("Status:\n%s", truncated),
		fmt.Sprintf("Recent commits:\n%s", log),
	)

	return strings.Join(parts, "\n\n")
}
