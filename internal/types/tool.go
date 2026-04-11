package types

import "encoding/json"

// ToolInputJSONSchema 描述工具输入的 JSON Schema。
type ToolInputJSONSchema struct {
	Type       string                        `json:"type"`
	Properties map[string]ToolSchemaProperty `json:"properties,omitempty"`
	Required   []string                      `json:"required,omitempty"`
}

// ToolSchemaProperty 描述工具输入 schema 中的单个属性。
type ToolSchemaProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Default     any      `json:"default,omitempty"`
}

// ToolResult 是工具执行的输出结果。
type ToolResult struct {
	Content string `json:"content"`
	IsError bool   `json:"isError,omitempty"`
}

// ToolUseContext 提供工具执行所需的上下文。
type ToolUseContext struct {
	AbortSignal    chan struct{} // 关闭此通道以中止操作
	SessionID      SessionID
	WorkingDir     string
	Debug          bool
	Verbose        bool
	NonInteractive bool
}

// ToolProgressData 是工具进度更新的接口。
type ToolProgressData interface {
	ProgressType() string
}

// BashProgress 跟踪 bash 命令执行进度。
type BashProgress struct {
	Command   string `json:"command"`
	Running   bool   `json:"running"`
	ExitCode  int    `json:"exitCode,omitempty"`
	StdoutLen int    `json:"stdoutLen"`
	StderrLen int    `json:"stderrLen"`
}

func (p *BashProgress) ProgressType() string { return "bash" }

// FileProgress 跟踪文件操作进度。
type FileProgress struct {
	FilePath string `json:"filePath"`
	Op       string `json:"op"` // read, edit, write
	Bytes    int64  `json:"bytes,omitempty"`
}

func (p *FileProgress) ProgressType() string { return "file" }

// SearchProgress 跟踪搜索操作进度。
type SearchProgress struct {
	Pattern string `json:"pattern"`
	Matches int    `json:"matches"`
}

func (p *SearchProgress) ProgressType() string { return "search" }

// ValidationResult 是工具输入验证的返回结果。
type ValidationResult struct {
	Valid bool   `json:"valid"`
	Msg   string `json:"message,omitempty"`
	Code  int    `json:"errorCode,omitempty"`
}

// ToJSON 将 JSON Schema 序列化为原始 JSON 字节。
func (s *ToolInputJSONSchema) ToJSON() (json.RawMessage, error) {
	return json.Marshal(s)
}
