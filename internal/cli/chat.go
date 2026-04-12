package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/kkkldpz/forge/internal/api"
	"github.com/kkkldpz/forge/internal/config"
	promptctx "github.com/kkkldpz/forge/internal/context"
	"github.com/kkkldpz/forge/internal/engine"
	"github.com/kkkldpz/forge/internal/query"
	"github.com/kkkldpz/forge/internal/tui"
	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/tools"
)

func init() {
	rootCmd.AddCommand(chatCmd)
}

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "启动交互式对话",
	Long:  "启动交互式 REPL 对话界面，连接 Anthropic API。",
	RunE:  runChat,
}

// runChat 是 chat 子命令的入口。
func runChat(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取工作目录失败: %w", err)
	}

	// 加载配置
	loader := config.NewLoader(homeDir, cwd)
	cfg, err := loader.Load()
	if err != nil {
		slog.Warn("加载配置失败，使用默认值", "error", err)
	}

	// 确定模型
	model := cfg.Global.DefaultModel
	if model == "" {
		model = "claude-sonnet-4-6"
	}

	// 创建 API 客户端
	apiKey := cfg.Global.APIKey
	if apiKey == "" {
		return fmt.Errorf("未配置 API Key，请设置 ANTHROPIC_API_KEY 环境变量或在 ~/.forge/settings.json 中配置")
	}

	apiClient := api.NewClient(apiKey, cfg.Global.BaseURL, model)

	// 注册工具
	toolRegistry := tool.NewRegistry()
	toolRegistry.Register(tools.NewBashTool())
	toolRegistry.Register(tools.NewFileReadTool())
	toolRegistry.Register(tools.NewFileWriteTool())
	toolRegistry.Register(tools.NewFileEditTool())
	toolRegistry.Register(tools.NewGlobTool())
	toolRegistry.Register(tools.NewGrepTool())
	allTools := toolRegistry.AllEnabled()

	// 启动 TUI
	m := tui.NewModel(model)

	// 创建 QueryEngine
	qe := engine.NewQueryEngine(engine.QueryEngineConfig{
		Cwd:       cwd,
		HomeDir:   homeDir,
		Tools:     allTools,
		APIClient: apiClient,
		Model:     model,
		MaxTurns:  50,
	})

	// 设置 TUI 回调
	m.SetCallbacks(
		// onSubmit: 用户提交消息
		func(prompt string) {
			slog.Debug("用户提交消息", "length", len(prompt))
			go processQuery(cmd.Context(), &m, qe, prompt)
		},
		// onPermit: 权限确认（当前简化为始终允许）
		func(allow bool) {
			slog.Debug("权限确认", "allow", allow)
		},
		// onCancel: 取消操作
		func() {
			slog.Debug("用户取消操作")
			qe.Interrupt()
		},
	)

	// 启动 Bubble Tea 程序
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// processQuery 在 goroutine 中处理一次查询，通过 channel 向 TUI 发送事件。
func processQuery(ctx context.Context, m *tui.Model, qe *engine.QueryEngine, prompt string) {
	// 提交消息
	ch := qe.SubmitMessage(ctx, prompt)

	// 开始流式接收
	m.StartStreaming()

	for event := range ch {
		switch event.Type {
		case "assistant":
			// 助手消息开始
			continue
		case "tool_start":
			// 工具调用开始
			continue
		case "tool_result":
			// 工具调用完成
			continue
		case "tool_stop":
			// 所有工具调用结束
			continue
		case "complete":
			m.EndStreaming()
			if event.Usage != nil {
				m.UpdateUsage(event.Usage)
			}
			return
		case "error":
			m.EndStreaming()
			m.AddError(fmt.Sprintf("%v", event.Error))
			return
		case "cancelled":
			m.EndStreaming()
			m.AddMessage("system", "已取消")
			return
		case "max_turns":
			m.EndStreaming()
			m.AddMessage("system", "已达到最大轮次限制")
			return
		}
	}

	m.EndStreaming()
}

// 注入工具名称的辅助函数
func getToolNames(tools []tool.Tool) []string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name()
	}
	return names
}

// collectEnabledTools 收集已启用工具的名称。
func collectEnabledTools(tools []tool.Tool) []string {
	return getToolNames(tools)
}

// _ = 确保函数被引用（防止编译器删除未使用的 import）
var _ = collectEnabledTools
var _ = getToolNames
var _ = strings.Join
var _ = promptctx.FetchContext
var _ = query.Query
var _ = tools.NewBashTool
