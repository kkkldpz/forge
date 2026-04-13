package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/kkkldpz/forge/internal/permission"
)

// streamChunkMsg 表示从 API 收到的文本块。
type streamChunkMsg struct {
	Text string
}

// streamDoneMsg 表示流式接收结束。
type streamDoneMsg struct{}

// streamErrorMsg 表示流式接收出错。
type streamErrorMsg struct {
	Err error
}

// usageMsg 表示用量更新。
type usageMsg struct {
	InputTokens  int
	OutputTokens int
	CostUSD      float64
}

// permissionMsg 表示需要权限确认。
type permissionMsg struct {
	ToolName  string
	Input     string
	Decision permission.PermissionDecision
}

// Update 处理所有消息，返回更新后的模型和命令。
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.ready = true
		}
		// 调整视口和输入框尺寸
		m.updateLayout()

	case tea.KeyMsg:
		// 权限确认模式
		if m.state == stateWaitingPermission {
			return m.handlePermissionKey(msg)
		}

		// 流式接收中：只处理 Ctrl+C 中断
		if m.state == stateStreaming {
			if msg.String() == "ctrl+c" || msg.String() == "esc" {
				if m.onCancel != nil {
					m.onCancel()
				}
				m.EndStreaming()
				return m, nil
			}
			// 流式中忽略其他按键
			return m, nil
		}

		// 正常输入模式
		switch msg.String() {
		case "ctrl+c":
			// 无消息时退出
			if len(m.messages) == 0 {
				return m, tea.Quit
			}
			// 有消息时视为中断
			if m.onCancel != nil {
				m.onCancel()
			}
			return m, nil

		case "enter", "ctrl+enter":
			text := strings.TrimSpace(m.textarea.Value())
			if text == "" {
				return m, nil
			}

			// 记录到历史
			m.history = append(m.history, text)
			m.histIdx = -1

			// 添加用户消息
			m.AddMessage("user", text)

			// 清空输入框
			m.textarea.SetValue("")

			// 触发提交回调
			if m.onSubmit != nil {
				m.onSubmit(text)
			}
			return m, nil

		case "ctrl+up":
			m.browseHistory(-1)
			return m, nil

		case "ctrl+down":
			m.browseHistory(1)
			return m, nil

		case "ctrl+l":
			// 清屏
			m.viewport.SetContent("")
			return m, nil
		}

		// 委托给 textarea 处理普通输入
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

	case streamChunkMsg:
		m.AppendStream(msg.Text)
		return m, nil

	case streamDoneMsg:
		m.EndStreaming()
		return m, nil

	case streamErrorMsg:
		m.EndStreaming()
		m.AddError(msg.Err.Error())
		return m, nil

	case usageMsg:
		m.totalInput += msg.InputTokens
		m.totalOutput += msg.OutputTokens
		m.totalCostUSD += msg.CostUSD
		return m, nil

	case permissionMsg:
	m.ShowPermission(msg.ToolName, msg.Input, msg.Decision)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, tea.Batch(cmds...)
}

// handlePermissionKey 处理权限确认按键。
func (m Model) handlePermissionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y", "enter":
		toolName := ""
		if m.permissionRequest != nil {
			toolName = m.permissionRequest.ToolName
		}
		m.state = stateIdle
		m.permissionRequest = nil
		if m.onPermit != nil {
			m.onPermit(true)
		}
		m.AddToolResult(toolName, "已允许", false)
		return m, nil

	case "n", "esc":
		toolName := ""
		if m.permissionRequest != nil {
			toolName = m.permissionRequest.ToolName
		}
		m.state = stateIdle
		m.permissionRequest = nil
		if m.onPermit != nil {
			m.onPermit(false)
		}
		m.AddToolResult(toolName, "已拒绝", true)
		return m, nil

	case "a":
		// 始终允许 — 这里简化为允许并记录
		toolName := ""
		if m.permissionRequest != nil {
			toolName = m.permissionRequest.ToolName
		}
		m.state = stateIdle
		m.permissionRequest = nil
		if m.onPermit != nil {
			m.onPermit(true)
		}
		m.AddToolResult(toolName, "已始终允许", false)
		return m, nil
	}

	return m, nil
}

// browseHistory 浏览输入历史。
func (m *Model) browseHistory(direction int) {
	if len(m.history) == 0 {
		return
	}

	if m.histIdx == -1 {
		// 首次进入历史浏览，保存当前草稿
		m.draftSaved = m.textarea.Value()
		m.histIdx = len(m.history) - 1
	} else {
		m.histIdx += direction
		if m.histIdx < 0 {
			m.histIdx = -1
			m.textarea.SetValue(m.draftSaved)
			return
		}
		if m.histIdx >= len(m.history) {
			m.histIdx = -1
			m.textarea.SetValue(m.draftSaved)
			return
		}
	}

	m.textarea.SetValue(m.history[m.histIdx])
}

// updateLayout 根据终端尺寸更新组件布局。
func (m *Model) updateLayout() {
	statusBarHeight := 1
	inputHeight := m.textarea.Height() + 2 // textarea + 边框
	separatorHeight := 1

	viewportHeight := m.height - statusBarHeight - inputHeight - separatorHeight
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	m.viewport.Width = m.width
	m.viewport.Height = viewportHeight

	m.textarea.SetWidth(m.width - 4) // 留出边框空间
}
