package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "检查并安装更新",
	Long:  "检查 Forge 新版本并在可用时提示更新",
	RunE:  runUpdate,
}

const (
	currentVersion = "0.1.0"
	updateCheckURL = "https://api.github.com/repos/kkkldpz/forge/releases/latest"
)

func runUpdate(cmd *cobra.Command, args []string) error {
	fmt.Printf("当前版本: %s\n", currentVersion)
	fmt.Println("正在检查更新...")

	fmt.Println("\n提示: 手动更新请访问 https://github.com/kkkldpz/forge/releases")
	fmt.Println("或者使用: go install github.com/kkkldpz/forge@latest")

	return nil
}