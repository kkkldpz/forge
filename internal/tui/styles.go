// Package tui 实现基于 Bubble Tea 的交互式终端 UI。
package tui

import "github.com/charmbracelet/lipgloss"

// 颜色常量。
const (
	colorPrimary   = "62"  // 蓝色
	colorSuccess   = "64"  // 绿色
	colorWarning   = "214" // 橙色
	colorError     = "196" // 红色
	colorMuted     = "240" // 灰色
	colorSubtle    = "236" // 浅灰背景
	colorText      = "252" // 主文本
	colorAssistant = "69"  // 助手文本
	colorToolUse   = "180" // 工具调用
)

var (
	// 标题栏样式
	titleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorPrimary)).
				Bold(true).
				Padding(0, 1)

	// 用户消息样式
	userMsgStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorText)).
				Padding(0, 1)

	// 助手消息样式
	assistantMsgStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorAssistant)).
				Padding(0, 1)

	// 工具调用样式
	toolUseStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorToolUse)).
				Padding(0, 1)

	// 工具结果样式
	toolResultStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorMuted)).
				Padding(0, 2)

	// 错误消息样式
	errorMsgStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorError)).
				Bold(true).
				Padding(0, 1)

	// 状态栏样式
	statusBarStyle = lipgloss.NewStyle().
				Background(lipgloss.Color(colorSubtle)).
				Foreground(lipgloss.Color(colorMuted)).
				Padding(0, 1)

	statusActiveStyle = lipgloss.NewStyle().
				Background(lipgloss.Color(colorPrimary)).
				Foreground(lipgloss.Color("15")).
				Padding(0, 1)

	inputPromptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorPrimary)).
		Bold(true)

	// 权限对话框样式
	permDialogStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color(colorWarning)).
			Padding(1, 2)

	permAllowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSuccess)).
			Bold(true)

	permDenyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorError)).
			Bold(true)

	// 分隔线
	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSubtle)).
			Padding(0, 1)
)
