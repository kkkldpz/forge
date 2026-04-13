package permission

import "strings"

// BashCommandClass Bash 命令分类。
type BashCommandClass int

const (
	ClassUnknown BashCommandClass = iota
	ClassSearch
	ClassRead
	ClassList
	ClassWrite
	ClassDangerous
)

// ClassifyBashCommand 分类 Bash 命令。
func ClassifyBashCommand(command string) BashCommandClass {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return ClassUnknown
	}

	// 取第一个单词作为命令名
	parts := strings.Fields(trimmed)
	cmd := parts[0]

	// 搜索类命令
	searchPrefixes := []string{"grep", "rg", "ack", "ag", "find", "locate", "where", "which"}
	for _, prefix := range searchPrefixes {
		if cmd == prefix || strings.HasPrefix(cmd, prefix+" ") {
			return ClassSearch
		}
	}

	// 读取类命令
	readPrefixes := []string{"cat", "head", "tail", "less", "more", "file", "wc", "wc"}
	for _, prefix := range readPrefixes {
		if cmd == prefix || strings.HasPrefix(cmd, prefix+" ") {
			return ClassRead
		}
	}

	// 列表类命令
	listCommands := []string{"ls", "ll", "la", "dir", "tree", "du", "df"}
	for _, c := range listCommands {
		if cmd == c || strings.HasPrefix(cmd, c+" ") {
			return ClassList
		}
	}

	// 危险命令模式
	dangerousPatterns := []string{
		"rm -rf", "rm -r /", "mkfs", "dd if=", ":(){ :|:&",
		"chmod -R 777", "chown -R",
	"curl | sh", "wget | sh", "bash -c $(curl",
	"eval $(curl", "> /dev/", "mv /* /*",
	}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(trimmed, pattern) {
			return ClassDangerous
		}
	}

	// 默认为写入类
	return ClassWrite
}

// IsDangerousBashCommand 检查命令是否为危险操作。
func IsDangerousBashCommand(command string) bool {
	return ClassifyBashCommand(command) == ClassDangerous
}

// IsReadOnlyBashCommand 检查命令是否为只读操作。
func IsReadOnlyBashCommand(command string) bool {
	return ClassifyBashCommand(command) == ClassSearch ||
		ClassifyBashCommand(command) == ClassRead ||
		ClassifyBashCommand(command) == ClassList
}
