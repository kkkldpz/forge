package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kkkldpz/forge/internal/config"
	"github.com/kkkldpz/forge/internal/engine"
	"github.com/kkkldpz/forge/internal/provider"
	"github.com/kkkldpz/forge/internal/toolkit"
	"github.com/kkkldpz/forge/internal/tools"
)

func init() {
	rootCmd.AddCommand(pipeCmd)
}

var pipeCmd = &cobra.Command{
	Use:   "pipe [message]",
	Short: "非交互式管道模式",
	Long:  "从 stdin 或命令行参数读取消息，处理后输出结果",
	RunE:  runPipe,
}

func runPipe(cmd *cobra.Command, args []string) error {
	var input string

	if len(args) > 0 {
		input = args[0]
	} else {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("读取 stdin 失败: %w", err)
		}
		input = strings.TrimSpace(string(data))
	}

	if input == "" {
		return fmt.Errorf("输入不能为空")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取工作目录失败: %w", err)
	}

	loader := config.NewLoader(homeDir, cwd)
	cfg, err := loader.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	model := cfg.Global.DefaultModel
	if model == "" {
		model = "claude-sonnet-4-6"
	}

	prov, err := provider.GetProvider(cfg)
	if err != nil {
		return fmt.Errorf("创建 Provider 失败: %w", err)
	}

	toolRegistry := toolkit.NewRegistry()
	toolRegistry.Register(tools.NewBashTool())
	toolRegistry.Register(tools.NewFileReadTool())
	toolRegistry.Register(tools.NewFileWriteTool())
	toolRegistry.Register(tools.NewFileEditTool())
	toolRegistry.Register(tools.NewGlobTool())
	toolRegistry.Register(tools.NewGrepTool())
	allTools := toolRegistry.AllEnabled()

	qe := engine.NewQueryEngine(engine.QueryEngineConfig{
		Cwd:      cwd,
		HomeDir:  homeDir,
		Tools:    allTools,
		Provider: prov,
		Model:    model,
		MaxTurns: 10,
	})

	ctx := context.Background()
	ch := qe.SubmitMessage(ctx, input)

	for event := range ch {
		if event.Error != nil {
			return fmt.Errorf("处理失败: %w", event.Error)
		}

		if event.Type == "complete" {
			if event.Usage != nil {
				fmt.Printf("\n[Token 使用: 输入 %d, 输出 %d]\n",
					event.Usage.InputTokens, event.Usage.OutputTokens)
			}
			return nil
		}
	}

	return nil
}