package context

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MemoryType 表示 CLAUDE.md 文件的来源类型。
type MemoryType string

const (
	MemoryTypeUser    MemoryType = "User"
	MemoryTypeProject MemoryType = "Project"
	MemoryTypeLocal   MemoryType = "Local"
)

// MemoryFileInfo 表示一个已加载的 CLAUDE.md 文件信息。
type MemoryFileInfo struct {
	Path    string
	Type    MemoryType
	Content string
	Parent  string // 引用此文件的父文件路径
}

const maxIncludeDepth = 5

// includeRegex 匹配 @include 指令中的路径。
// 匹配 @path, @./path, @~/path, @/path
var includeRegex = regexp.MustCompile(`(?:^|\s)@((?:[^\s\\]|\\ )+)`)

// GetMemoryFiles 从 cwd 向上遍历目录树发现所有 CLAUDE.md 文件。
// 加载顺序：User → Project（从根到 cwd）→ Local（从根到 cwd）
func GetMemoryFiles(cwd string, homeDir string) []MemoryFileInfo {
	slog.Debug("开始发现 CLAUDE.md 文件", "cwd", cwd)
	var result []MemoryFileInfo
	processed := make(map[string]bool)

	// 1. 用户级 CLAUDE.md
	userClaudeMd := filepath.Join(homeDir, ".claude", "CLAUDE.md")
	result = append(result, processMemoryFile(userClaudeMd, MemoryTypeUser, processed, 0)...)

	// 用户级 rules
	userRulesDir := filepath.Join(homeDir, ".claude", "rules")
	result = append(result, processMdRules(userRulesDir, MemoryTypeUser, processed)...)

	// 2. 项目级和本地级 — 从根到 cwd（根优先加载，cwd 后加载 = 更高优先级）
	dirs := collectAncestorDirs(cwd)
	for _, dir := range dirs {
		// Project: CLAUDE.md
		projectPath := filepath.Join(dir, "CLAUDE.md")
		result = append(result, processMemoryFile(projectPath, MemoryTypeProject, processed, 0)...)

		// Project: .claude/CLAUDE.md
		dotClaudePath := filepath.Join(dir, ".claude", "CLAUDE.md")
		result = append(result, processMemoryFile(dotClaudePath, MemoryTypeProject, processed, 0)...)

		// Project: .claude/rules/*.md
		rulesDir := filepath.Join(dir, ".claude", "rules")
		result = append(result, processMdRules(rulesDir, MemoryTypeProject, processed)...)

		// Local: CLAUDE.local.md
		localPath := filepath.Join(dir, "CLAUDE.local.md")
		result = append(result, processMemoryFile(localPath, MemoryTypeLocal, processed, 0)...)
	}

	slog.Debug("CLAUDE.md 文件发现完成", "文件数", len(result))
	return result
}

// collectAncestorDirs 从 cwd 向上收集目录路径（不含 cwd 本身的根目录以上的部分）。
func collectAncestorDirs(cwd string) []string {
	var dirs []string
	dir := cwd
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dirs = append(dirs, dir)
		dir = parent
	}
	// 反转：从根到 cwd
	for i, j := 0, len(dirs)-1; i < j; i, j = i+1, j-1 {
		dirs[i], dirs[j] = dirs[j], dirs[i]
	}
	return dirs
}

// processMemoryFile 读取并解析单个 CLAUDE.md 文件，递归处理 @include。
func processMemoryFile(path string, memType MemoryType, processed map[string]bool, depth int) []MemoryFileInfo {
	normalized := normalizePath(path)
	if processed[normalized] || depth >= maxIncludeDepth {
		return nil
	}
	processed[normalized] = true

	content, err := os.ReadFile(path)
	if err != nil {
		// ENOENT/EISDIR 是预期情况，静默忽略
		return nil
	}

	contentStr := string(content)
	if strings.TrimSpace(contentStr) == "" {
		return nil
	}

	info := MemoryFileInfo{
		Path:    path,
		Type:    memType,
		Content: contentStr,
	}

	result := []MemoryFileInfo{info}

	// 解析 @include 指令
	includePaths := extractIncludePaths(contentStr, path)
	for _, incPath := range includePaths {
		included := processMemoryFile(incPath, memType, processed, depth+1)
		for _, f := range included {
			f.Parent = path
			result = append(result, f)
		}
	}

	return result
}

// processMdRules 处理 .claude/rules/ 目录下所有 .md 文件。
func processMdRules(rulesDir string, memType MemoryType, processed map[string]bool) []MemoryFileInfo {
	var result []MemoryFileInfo

	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		entryPath := filepath.Join(rulesDir, entry.Name())

		if entry.IsDir() {
			result = append(result, processMdRules(entryPath, memType, processed)...)
			continue
		}

		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		result = append(result, processMemoryFile(entryPath, memType, processed, 0)...)
	}

	return result
}

// extractIncludePaths 从文件内容中提取 @include 路径。
func extractIncludePaths(content string, basePath string) []string {
	matches := includeRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var paths []string
	baseDir := filepath.Dir(basePath)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		refPath := match[1]
		if refPath == "" {
			continue
		}

		// 去除 fragment (#heading)
		if idx := strings.Index(refPath, "#"); idx != -1 {
			refPath = refPath[:idx]
		}
		if refPath == "" {
			continue
		}

		// 反转义空格
		refPath = strings.ReplaceAll(refPath, "\\ ", " ")

		// 验证路径格式
		if !isValidIncludePath(refPath) {
			continue
		}

		resolved := resolveIncludePath(refPath, baseDir)
		if resolved == "" || seen[resolved] {
			continue
		}
		seen[resolved] = true
		paths = append(paths, resolved)
	}

	return paths
}

// isValidIncludePath 检查 @include 路径是否有效。
func isValidIncludePath(path string) bool {
	if strings.HasPrefix(path, "./") {
		return true
	}
	if strings.HasPrefix(path, "~/") {
		return true
	}
	if strings.HasPrefix(path, "/") && path != "/" {
		return true
	}
	// 相对路径（无前缀）
	if len(path) > 0 && !strings.HasPrefix(path, "@") {
		// 必须以字母、数字、.、-、_ 开头
		first := path[0]
		if (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') ||
			(first >= '0' && first <= '9') || first == '.' || first == '-' || first == '_' {
			return true
		}
	}
	return false
}

// resolveIncludePath 将 @include 路径解析为绝对路径。
func resolveIncludePath(refPath, baseDir string) string {
	if strings.HasPrefix(refPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, refPath[2:])
	}
	if filepath.IsAbs(refPath) {
		return filepath.Clean(refPath)
	}
	return filepath.Join(baseDir, refPath)
}

// normalizePath 规范化路径用于去重比较。
func normalizePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return filepath.Clean(abs)
}

// GetClaudeMds 将 MemoryFileInfo 列表格式化为系统提示词片段。
func GetClaudeMds(files []MemoryFileInfo) string {
	if len(files) == 0 {
		return ""
	}

	var memories []string
	for _, file := range files {
		if strings.TrimSpace(file.Content) == "" {
			continue
		}

		desc := memoryTypeDescription(file.Type)
		memories = append(memories,
			fmt.Sprintf("Contents of %s%s:\n\n%s", file.Path, desc, strings.TrimSpace(file.Content)),
		)
	}

	if len(memories) == 0 {
		return ""
	}

	return memoryInstructionPrompt + "\n\n" + strings.Join(memories, "\n\n")
}

// memoryTypeDescription 返回记忆类型的描述文本。
func memoryTypeDescription(memType MemoryType) string {
	switch memType {
	case MemoryTypeProject:
		return " (project instructions, checked into the codebase)"
	case MemoryTypeLocal:
		return " (user's private project instructions, not checked in)"
	default:
		return " (user's private global instructions for all projects)"
	}
}

const memoryInstructionPrompt = "Codebase and user instructions are shown below. Be sure to adhere to these instructions. IMPORTANT: These instructions OVERRIDE any default behavior and you MUST follow them exactly as written."
