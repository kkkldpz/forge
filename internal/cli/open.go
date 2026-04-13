package cli

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(openCmd)
}

var openCmd = &cobra.Command{
	Use:   "open [url]",
	Short: "在浏览器中打开 URL 或本地文件",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runOpen,
}

func runOpen(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("请提供要打开的 URL 或路径")
	}

	target := args[0]

	var err error
	switch runtime.GOOS {
	case "darwin":
		err = exec.Command("open", target).Run()
	case "linux":
		err = exec.Command("xdg-open", target).Run()
	case "windows":
		err = exec.Command("cmd", "/c", "start", target).Run()
	default:
		return fmt.Errorf("不支持的平台: %s", runtime.GOOS)
	}

	if err != nil {
		return fmt.Errorf("打开失败: %w", err)
	}

	fmt.Printf("已打开: %s\n", target)
	return nil
}