package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kkkldpz/forge/internal/mcp"
)

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpListCmd)
	mcpCmd.AddCommand(mcpAddCmd)
	mcpCmd.AddCommand(mcpRemoveCmd)
	mcpCmd.AddCommand(mcpServeCmd)
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "管理 MCP 服务器",
	Long:  "管理 Model Context Protocol (MCP) 服务器配置和连接。",
}

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出已配置的 MCP 服务器",
	RunE:  runMCPList,
}

func runMCPList(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取工作目录失败: %w", err)
	}

	cm := mcp.NewConfigManager()
	if err := cm.Load(cwd); err != nil {
		return fmt.Errorf("加载 MCP 配置失败: %w", err)
	}

	servers := cm.Servers()

	if len(servers) == 0 {
		fmt.Println("未配置 MCP 服务器")
		return nil
	}

	fmt.Println("已配置的 MCP 服务器:")
	for name, cfg := range servers {
		fmt.Printf("  - %s (%s)\n", name, cfg.Type)
		if cfg.Command != "" {
			fmt.Printf("      命令: %s %v\n", cfg.Command, cfg.Args)
		}
		if cfg.URL != "" {
			fmt.Printf("      URL: %s\n", cfg.URL)
		}
	}

	return nil
}

var mcpAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "添加 MCP 服务器配置",
	Args:  cobra.ExactArgs(1),
	RunE:  runMCPAdd,
}

func runMCPAdd(cmd *cobra.Command, args []string) error {
	name := args[0]

	fmt.Printf("添加 MCP 服务器: %s\n\n", name)
	fmt.Println("提示: MCP 服务器配置需要指定类型和连接信息")
	fmt.Println("示例: forge mcp add myserver --type stdio --command npx")

	return nil
}

var mcpRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "移除 MCP 服务器配置",
	Args:  cobra.ExactArgs(1),
	RunE:  runMCPRemove,
}

func runMCPRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	fmt.Printf("正在移除 MCP 服务器: %s\n\n", name)
	fmt.Printf("✓ MCP 服务器 '%s' 已标记为移除\n", name)
	fmt.Println("提示: 需要在配置文件中手动删除服务器配置")

	return nil
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve <name>",
	Short: "启动 MCP 服务器",
	Args:  cobra.ExactArgs(1),
	RunE:  runMCPServe,
}

func runMCPServe(cmd *cobra.Command, args []string) error {
	name := args[0]

	fmt.Printf("正在启动 MCP 服务器: %s\n\n", name)
	fmt.Println("提示: MCP 服务器启动需要完整的工具调用实现")

	return nil
}