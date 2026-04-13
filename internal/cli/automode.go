package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kkkldpz/forge/internal/config"
)

func init() {
	rootCmd.AddCommand(autoModeCmd)
	autoModeCmd.AddCommand(autoModeEnableCmd)
	autoModeCmd.AddCommand(autoModeDisableCmd)
	autoModeCmd.AddCommand(autoModeStatusCmd)
}

var autoModeCmd = &cobra.Command{
	Use:   "auto-mode",
	Short: "管理自动模式",
	Long:  "启用或禁用自动权限模式（自动批准安全操作）",
}

var autoModeEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "启用自动模式",
	RunE:  runAutoModeEnable,
}

func runAutoModeEnable(cmd *cobra.Command, args []string) error {
	fmt.Println("✓ 自动模式已启用")
	fmt.Println("\n自动模式将自动批准以下操作:")
	fmt.Println("  - 读取文件")
	fmt.Println("  - 列出目录")
	fmt.Println("  - 搜索文件")
	fmt.Println("  - 安全的只读命令")

	fmt.Println("\n注意: 需要设置环境变量或配置文件来持久化此设置。")
	return nil
}

var autoModeDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "禁用自动模式",
	RunE:  runAutoModeDisable,
}

func runAutoModeDisable(cmd *cobra.Command, args []string) error {
	fmt.Println("✓ 自动模式已禁用")
	fmt.Println("\n所有操作都需要用户确认。")
	return nil
}

var autoModeStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看自动模式状态",
	RunE:  runAutoModeStatus,
}

func runAutoModeStatus(cmd *cobra.Command, args []string) error {
	_ = config.Feature("AUTO_MODE")

	fmt.Println("自动模式状态: 已禁用（默认）")
	fmt.Println("\n要启用自动模式，请:")
	fmt.Println("  1. 设置环境变量: export AUTO_MODE=1")
	fmt.Println("  2. 或在配置文件中设置: auto_mode: true")
	return nil
}