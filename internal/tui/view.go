package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// View 渲染整个 TUI 界面。
func (m Model) View() string {
	if !m.ready {
		return "\n  正在初始化..."
	}

	// 消息区域内容
	content := m.renderMessages()

	// 流式状态时在底部追加 spinner
	if m.state == stateStreaming {
		content += toolUseStyle.Render(m.spinner.View() + " 正在思考...")
	}

	// 权限对话框
	permView := ""
	if m.state == stateWaitingPermission {
		permView = m.renderPermission()
	}

	// 更新视口内容
	m.viewport.SetContent(content)

	// 组装最终视图
	header := titleStyle.Width(m.width).Render(fmt.Sprintf(" Forge — %s ", m.model))

	inputView := m.inputPrompt() + m.textarea.View()

	statusView := m.renderStatusBar()

	// 垂直拼接
	view := lipgloss.JoinVertical(lipgloss.Left,
		header,
		m.viewport.View(),
		inputView,
		statusView,
	)

	// 权限对话框覆盖层
	if permView != "" {
		view += "\n" + permView
	}

	return view
}
