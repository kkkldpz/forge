// Package cli 定义所有 Cobra 命令行子命令。
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(agentsCmd)
	agentsCmd.AddCommand(agentsListCmd)
	agentsCmd.AddCommand(agentsStopCmd)
}

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "管理子代理",
	Long:  "列出或停止正在运行的子代理",
}

var agentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有子代理",
	RunE:  runAgentsList,
}

func runAgentsList(cmd *cobra.Command, args []string) error {
	fmt.Println("子代理列表:")
	fmt.Println("  (暂无运行的代理)")

	fmt.Println("\n使用 'forge chat' 启动交互式会话后，可以通过 agent 工具创建子代理。")
	return nil
}

var agentsStopCmd = &cobra.Command{
	Use:   "stop <agent-id>",
	Short: "停止指定的子代理",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentsStop,
}

func runAgentsStop(cmd *cobra.Command, args []string) error {
	agentID := args[0]
	fmt.Printf("正在停止代理: %s\n", agentID)
	fmt.Println("注意: 停止子代理功能需要在交互式会话中使用。")
	return nil
}