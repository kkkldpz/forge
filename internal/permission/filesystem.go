package permission

import (
	"path/filepath"
	"runtime"
	"strings"
)

// FileOperationType 文件操作类型。
type FileOperationType string

const (
	FileRead  FileOperationType = "read"
	FileWrite FileOperationType = "write"
	FileDelete FileOperationType = "delete"
)

// PathCheckResult 路径检查结果。
type PathCheckResult struct {
	Allowed bool
	Reason  string
}

// dangerousFiles 禁止操作的敏感文件。
var dangerousFiles = map[string]bool{
	".gitconfig":         true,
	".gitignore":         true,
	".gitmodules":        true,
	".gitattributes":     true,
	".bashrc":            true,
	".bash_profile":       true,
	".zshrc":             true,
	".zshenv":            true,
	".profile":           true,
	".bash_history":       true,
	".zsh_history":        true,
	".mcp.json":          true,
	".claude.json":       true,
	".claude-plugin/":     true,
	"credentials":        true,
	".env":               true,
	".npmrc":             true,
	".pypirc":            true,
	".netrc":             true,
	".ssh/":              true,
	"id_rsa":             true,
	"id_ed25519":         true,
	"known_hosts":        true,
}

// dangerousDirectories 禁止操作的敏感目录。
var dangerousDirectories = map[string]bool{
	".git":           true,
	".vscode":        true,
	".idea":         true,
	".claude":        true,
	".claude-plugin": true,
	".forge":         true,
}

// ValidateFilePath 验证文件路径是否允许操作。
func ValidateFilePath(path string, cwd string, op FileOperationType) PathCheckResult {
	if path == "" {
		return PathCheckResult{Allowed: false, Reason: "路径为空"}
	}

	// 规范化路径
	absPath := path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(cwd, absPath)
	}
	absPath = filepath.Clean(absPath)

	// Windows 安全检查
	if runtime.GOOS == "windows" {
		if result := checkSuspiciousWindowsPath(absPath); !result.Allowed {
			return result
		}
	}

	// 检查是否为敏感文件
	if isDangerousFile(absPath) {
		return PathCheckResult{
			Allowed: false,
			Reason:  "敏感文件，禁止操作: " + filepath.Base(absPath),
		}
	}

	// 检查是否为敏感目录
	if isDangerousDirectory(absPath) {
		return PathCheckResult{
			Allowed: false,
			Reason:  "敏感目录，禁止操作: " + filepath.Base(absPath),
		}
	}

	// 对于写入/删除操作，额外检查工作目录范围
	if op != FileRead {
		if !isWithinWorkingDir(absPath, cwd) {
			return PathCheckResult{
				Allowed: false,
				Reason:  "路径在工作目录之外: " + absPath,
			}
		}
	}

	return PathCheckResult{Allowed: true}
}

// isWithinWorkingDir 检查路径是否在工作目录内。
func isWithinWorkingDir(path, cwd string) bool {
	absPath := path
	absCwd := cwd
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(absCwd, absPath)
	}
	absCwd = filepath.Clean(absCwd)

	rel, err := filepath.Rel(absCwd, absPath)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}

// isDangerousFile 检查路径是否指向敏感文件。
func isDangerousFile(path string) bool {
	base := filepath.Base(path)

	// 精确匹配
	if dangerousFiles[base] {
		return true
	}

	// 前缀匹配（如 .env.production）
	for dangerousFile := range dangerousFiles {
		if strings.HasSuffix(dangerousFile, "/") {
			// 目录类型，检查路径前缀
			if strings.HasPrefix(filepath.Join(filepath.Dir(path), dangerousFile), path) {
				return true
			}
		}
	}

	return false
}

// isDangerousDirectory 检查路径是否在敏感目录内。
func isDangerousDirectory(path string) bool {
	for dir := range dangerousDirectories {
		if strings.Contains(path, string(filepath.Separator)+dir+string(filepath.Separator)) ||
			strings.HasPrefix(path, dir+string(filepath.Separator)) ||
			path == dir {
			return true
		}
	}
	return false
}

// checkSuspiciousWindowsPath 检测 Windows 平台的路径绕过手法。
func checkSuspiciousWindowsPath(path string) PathCheckResult {
	// 替代数据流: file.txt::$DATA
	if strings.Contains(path, "::$") {
		return PathCheckResult{Allowed: false, Reason: "检测到 Windows 替代数据流"}
	}

	// 短文件名: GIT~1, CLAUDE~1
	base := filepath.Base(path)
	if strings.Contains(base, "~1") {
		return PathCheckResult{Allowed: false, Reason: "检测到短文件名"}
	}

	// 长路径前缀: \\?\C:\, \\.\C:\
	if strings.HasPrefix(path, `\\?\`) || strings.HasPrefix(path, `\\.\`) {
		return PathCheckResult{Allowed: false, Reason: "检测到长路径前缀"}
	}

	// 尾部点号: .git., .claude.
	for _, suffix := range []string{".git.", ".claude.", ".vscode."} {
		if strings.HasSuffix(strings.ToLower(base), suffix) {
			return PathCheckResult{Allowed: false, Reason: "检测到伪装目录名"}
		}
	}

	// UNC 路径: \\server\share
	if strings.HasPrefix(path, `\\`) && !strings.HasPrefix(path, `\\?\`) && !strings.HasPrefix(path, `\\.\`) {
		return PathCheckResult{Allowed: false, Reason: "UNC 路径不受信任"}
	}

	// 三连点: .../file.txt
	if strings.Contains(path, "...") {
		return PathCheckResult{Allowed: false, Reason: "检测到可疑的路径模式"}
	}

	return PathCheckResult{Allowed: true}
}
