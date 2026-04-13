package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(sshCmd)
}

var sshCmd = &cobra.Command{
	Use:   "ssh [user@]host",
	Short: "通过 SSH 连接到远程主机",
	Long:  "建立 SSH 连接以便在远程主机上执行 Forge 命令",
	Args:  cobra.ExactArgs(1),
	RunE:  runSSH,
}

func runSSH(cmd *cobra.Command, args []string) error {
	target := args[0]

	fmt.Printf("SSH 连接功能\n")
	fmt.Printf("目标: %s\n\n", target)

	fmt.Println("提示: Forge 支持远程执行，可以通过以下方式使用:")
	fmt.Println("  1. 在远程主机上安装 Forge")
	fmt.Println("  2. 使用 'forge chat --remote' 启动远程会话")
	fmt.Println("  3. 使用 Bridge 模式进行远程控制")

	return nil
}