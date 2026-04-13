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
