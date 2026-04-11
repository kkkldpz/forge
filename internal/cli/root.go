package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "forge",
	Short: "Forge — AI 驱动的编程助手",
	Long:  "Forge 是用 Go 语言实现的 AI 编程助手，完整复刻 Claude Code 的功能。",
}

func init() {
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate(fmt.Sprintf("forge v%s\n", Version))
}

// Execute 执行根命令。
func Execute() error {
	return rootCmd.Execute()
}

// RootCmd 返回根 cobra 命令，供子命令注册使用。
func RootCmd() *cobra.Command {
	return rootCmd
}
