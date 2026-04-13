package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(completionCmd)
	completionCmd.AddCommand(bashCompletionCmd)
	completionCmd.AddCommand(zshCompletionCmd)
	completionCmd.AddCommand(fishCompletionCmd)
	completionCmd.AddCommand(powershellCompletionCmd)
}

var completionCmd = &cobra.Command{
	Use:   "completion [shell]",
	Short: "生成 Shell 自动补全脚本",
	Long: `为 Bash、Zsh、Fish 和 PowerShell 生成自动补全脚本。

用法:
  forge completion bash      # 生成 Bash 补全
  forge completion zsh       # 生成 Zsh 补全
  forge completion fish      # 生成 Fish 补全
  forge completion powershell # 生成 PowerShell 补全

安装说明:

Bash (Linux):
  # 系统级
  forge completion bash > /etc/bash_completion.d/forge
  
  # 用户级
  mkdir -p ~/.config/fish/completions
  forge completion fish > ~/.config/fish/completions/forge.fish

Zsh:
  # 确保在 ~/.zshrc 中启用补全
  autoload -Uz compinit
  compinit
  
  # 添加到补全路径
  forge completion zsh > ~/.zshrc.d/_forge

Fish:
  mkdir -p ~/.config/fish/completions
  forge completion fish > ~/.config/fish/completions/forge.fish
`,
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Args:      cobra.ExactValidArgs(1),
	RunE:      runCompletion,
}

func runCompletion(cmd *cobra.Command, args []string) error {
	shell := args[0]

	switch shell {
	case "bash":
		return rootCmd.GenBashCompletion(os.Stdout)
	case "zsh":
		return rootCmd.GenZshCompletion(os.Stdout)
	case "fish":
		return rootCmd.GenFishCompletion(os.Stdout, false)
	case "powershell":
		return rootCmd.GenPowerShellCompletion(os.Stdout)
	default:
		return fmt.Errorf("不支持的 shell 类型: %s", shell)
	}
}

var bashCompletionCmd = &cobra.Command{
	Use:   "bash",
	Short: "生成 Bash 自动补全脚本",
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenBashCompletion(os.Stdout)
	},
}

var zshCompletionCmd = &cobra.Command{
	Use:   "zsh",
	Short: "生成 Zsh 自动补全脚本",
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenZshCompletion(os.Stdout)
	},
}

var fishCompletionCmd = &cobra.Command{
	Use:   "fish",
	Short: "生成 Fish 自动补全脚本",
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenFishCompletion(os.Stdout, false)
	},
}

var powershellCompletionCmd = &cobra.Command{
	Use:   "powershell",
	Short: "生成 PowerShell 自动补全脚本",
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenPowerShellCompletion(os.Stdout)
	},
}