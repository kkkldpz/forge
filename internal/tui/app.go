package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/kkkldpz/forge/internal/api"
)

// ChatMsg 表示聊天中的一条消息。
type ChatMsg struct {
	Role       string // "user", "assistant", "tool_use", "tool_result", "error", "system"
	Content    string
	IsStreaming bool
	Timestamp  time.Time
	ToolName   string // 仅 tool_use/tool_result
}

// PermissionRequest 表示权限请求。
type PermissionRequest struct {
	ToolName string
	Input    string
}

// appState 表示 TUI 应用状态。
type appState int

const (
	stateIdle appState = iota
	stateStreaming
	stateWaitingPermission
	stateError
)

// Model 是 TUI 的主模型（Bubble Tea Elm 架构）。
type Model struct {
	// 终端尺寸
	width  int
	height int
	ready  bool

	// 核心组件
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model

	// 应用状态
	state appState

	// 消息历史
	messages []ChatMsg

	// 流式累积
	streamingContent string

	// 权限请求
	permissionRequest *PermissionRequest

	// 输入历史
	history    []string
	histIdx    int
	draftSaved string // 进入历史浏览前保存的草稿

	// 用量统计
	totalCostUSD  float64
	totalInput    int
	totalOutput   int

	// 当前模型
	model string

	// QueryEngine 回调
	onSubmit func(prompt string)
	onPermit func(allow bool)
	onCancel func()
}

// NewModel 创建新的 TUI 模型。
func NewModel(model string) Model {
	ta := textarea.New()
	ta.Placeholder = "输入消息..."
	ta.Focus()
	ta.CharLimit = 100000
	ta.SetWidth(80)
	ta.SetHeight(3)

	// 自定义键绑定：Ctrl+Enter 提交
	keyMap := ta.KeyMap
	keyMap.InsertNewline.SetEnabled(false) // 禁用默认的换行行为

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = toolUseStyle

	vp := viewport.New(80, 20)

	return Model{
		textarea: ta,
		spinner:  s,
		viewport: vp,
		model:    model,
		state:    stateIdle,
	}
}

// SetCallbacks 设置外部回调函数。
func (m *Model) SetCallbacks(
	onSubmit func(string),
	onPermit func(bool),
	onCancel func(),
) {
	m.onSubmit = onSubmit
	m.onPermit = onPermit
	m.onCancel = onCancel
}

// AddMessage 添加一条消息到历史。
func (m *Model) AddMessage(role, content string) {
	m.messages = append(m.messages, ChatMsg{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	m.scrollToBottom()
}

// AddToolUse 添加工具调用消息。
func (m *Model) AddToolUse(toolName, content string) {
	m.messages = append(m.messages, ChatMsg{
		Role:      "tool_use",
		Content:   content,
		ToolName:  toolName,
		Timestamp: time.Now(),
	})
	m.scrollToBottom()
}

// AddToolResult 添加工具结果消息。
func (m *Model) AddToolResult(toolName, content string, isError bool) {
	role := "tool_result"
	if isError {
		role = "tool_error"
	}
	m.messages = append(m.messages, ChatMsg{
		Role:      role,
		Content:   content,
		ToolName:  toolName,
		Timestamp: time.Now(),
	})
	m.scrollToBottom()
}

// AddError 添加错误消息。
func (m *Model) AddError(err string) {
	m.messages = append(m.messages, ChatMsg{
		Role:      "error",
		Content:   err,
		Timestamp: time.Now(),
	})
	m.scrollToBottom()
}

// StartStreaming 开始流式接收助手回复。
func (m *Model) StartStreaming() {
	m.state = stateStreaming
	m.streamingContent = ""
	// 添加空的流式消息占位
	m.messages = append(m.messages, ChatMsg{
		Role:       "assistant",
		IsStreaming: true,
		Timestamp:  time.Now(),
	})
	m.scrollToBottom()
}

// AppendStream 追加流式文本块。
func (m *Model) AppendStream(text string) {
	m.streamingContent += text
	if len(m.messages) > 0 {
		last := &m.messages[len(m.messages)-1]
		if last.IsStreaming {
			last.Content = m.streamingContent
		}
	}
	m.scrollToBottom()
}

// EndStreaming 结束流式接收。
func (m *Model) EndStreaming() {
	m.state = stateIdle
	if len(m.messages) > 0 {
		last := &m.messages[len(m.messages)-1]
		if last.IsStreaming {
			last.IsStreaming = false
			if strings.TrimSpace(last.Content) == "" {
				// 移除空消息
				m.messages = m.messages[:len(m.messages)-1]
			}
		}
	}
	m.streamingContent = ""
}

// ShowPermission 显示权限请求对话框。
func (m *Model) ShowPermission(toolName, input string) {
	m.state = stateWaitingPermission
	m.permissionRequest = &PermissionRequest{
		ToolName: toolName,
		Input:    input,
	}
	m.scrollToBottom()
}

// UpdateUsage 更新用量统计。
func (m *Model) UpdateUsage(usage *api.Usage) {
	if usage == nil {
		return
	}
	m.totalInput += usage.InputTokens
	m.totalOutput += usage.OutputTokens
}

// UpdateCost 更新费用统计。
func (m *Model) UpdateCost(cost float64) {
	m.totalCostUSD += cost
}

// Init 初始化模型。
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
	)
}

// scrollToBottom 将视口滚动到底部。
func (m *Model) scrollToBottom() {
	m.viewport.GotoBottom()
}

// renderMessages 渲染所有消息为字符串。
func (m *Model) renderMessages() string {
	var sb strings.Builder

	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			sb.WriteString(userMsgStyle.Render(fmt.Sprintf("> %s", msg.Content)))
			sb.WriteString("\n\n")

		case "assistant":
			content := msg.Content
			if msg.IsStreaming {
				content += "▌" // 光标效果
			}
			if content != "" {
				sb.WriteString(assistantMsgStyle.Render(content))
				sb.WriteString("\n\n")
			}

		case "tool_use":
			sb.WriteString(toolUseStyle.Render(fmt.Sprintf("⚡ %s", msg.ToolName)))
			sb.WriteString("\n")
			sb.WriteString(toolResultStyle.Render(msg.Content))
			sb.WriteString("\n\n")

		case "tool_result":
			sb.WriteString(toolResultStyle.Render(fmt.Sprintf("✓ %s 完成", msg.ToolName)))
			if msg.Content != "" {
				// 截断过长的结果
				display := msg.Content
				if len(display) > 500 {
					display = display[:500] + "\n... (已截断)"
				}
				sb.WriteString(toolResultStyle.Render(display))
			}
			sb.WriteString("\n\n")

		case "tool_error":
			sb.WriteString(errorMsgStyle.Render(fmt.Sprintf("✗ %s 失败: %s", msg.ToolName, msg.Content)))
			sb.WriteString("\n\n")

		case "error":
			sb.WriteString(errorMsgStyle.Render(fmt.Sprintf("错误: %s", msg.Content)))
			sb.WriteString("\n\n")

		case "system":
			sb.WriteString(separatorStyle.Render(msg.Content))
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// renderPermission 渲染权限对话框。
func (m *Model) renderPermission() string {
	if m.permissionRequest == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(permDialogStyle.Render(
		fmt.Sprintf("工具调用请求: %s\n\n", m.permissionRequest.ToolName) +
			toolResultStyle.Render(m.permissionRequest.Input) +
			"\n\n" +
			permAllowStyle.Render("[Y] 允许") + "  " +
			permDenyStyle.Render("[N] 拒绝") + "  " +
			"[A] 始终允许此工具",
	))
	return sb.String()
}

// renderStatusBar 渲染状态栏。
func (m *Model) renderStatusBar() string {
	// 左侧：状态
	var stateStr string
	switch m.state {
	case stateIdle:
		stateStr = " 就绪 "
	case stateStreaming:
		stateStr = " 思考中... "
	case stateWaitingPermission:
		stateStr = " 等待权限确认 "
	case stateError:
		stateStr = " 错误 "
	}

	left := statusActiveStyle.Render(stateStr)

	// 右侧：统计信息
	right := statusBarStyle.Render(fmt.Sprintf(" %s │ $%.4f │ ↑%d ↓%d ",
		m.model,
		m.totalCostUSD,
		m.totalInput,
		m.totalOutput,
	))

	// 中间填充
	middleWidth := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if middleWidth < 0 {
		middleWidth = 0
	}
	middle := statusBarStyle.Width(middleWidth).Render("")

	return lipgloss.JoinHorizontal(lipgloss.Bottom, left, middle, right)
}

// inputPrompt 返回输入框的提示文本。
func (m *Model) inputPrompt() string {
	return inputPromptStyle.Render("❯ ")
}
