package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kkkldpz/forge/internal/session"
)

func init() {
	rootCmd.AddCommand(sessionCmd)
	sessionCmd.AddCommand(sessionPsCmd)
	sessionCmd.AddCommand(sessionLogsCmd)
	sessionCmd.AddCommand(sessionAttachCmd)
	sessionCmd.AddCommand(sessionKillCmd)
}

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "管理后台会话",
	Long:  "列出、附加到或终止后台会话",
}

var sessionPsCmd = &cobra.Command{
	Use:   "ps",
	Short: "列出所有后台会话",
	RunE:  runSessionPs,
}

func runSessionPs(cmd *cobra.Command, args []string) error {
	m := session.GlobalManager()
	sessions := m.List()

	if len(sessions) == 0 {
		fmt.Println("没有运行中的后台会话")
		return nil
	}

	fmt.Printf("共 %d 个后台会话:\n\n", len(sessions))
	fmt.Println("ID                  PID     状态      启动时间")
	fmt.Println("-------------------------------------------------")

	for _, s := range sessions {
		fmt.Printf("%s  %-6d  %-8s  %s\n",
			s.ID[:8], s.PID, s.Status, s.StartedAt.Format("2006-01-02 15:04"))
	}

	return nil
}

var sessionLogsCmd = &cobra.Command{
	Use:   "logs <session-id>",
	Short: "查看会话日志",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionLogs,
}

func runSessionLogs(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	m := session.GlobalManager()
	if _, ok := m.Get(sessionID); !ok {
		return fmt.Errorf("会话 %s 不存在", sessionID)
	}

	fmt.Printf("会话 %s 的日志:\n\n", sessionID)
	fmt.Println("(日志功能需要实现日志持久化)")
	fmt.Println("提示: 使用 'forge session attach <id>' 附加到会话")

	return nil
}

var sessionAttachCmd = &cobra.Command{
	Use:   "attach <session-id>",
	Short: "附加到后台会话",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionAttach,
}

func runSessionAttach(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	m := session.GlobalManager()
	if _, ok := m.Get(sessionID); !ok {
		return fmt.Errorf("会话 %s 不存在", sessionID)
	}

	fmt.Printf("正在附加到会话 %s...\n", sessionID)
	fmt.Println("\n提示: 附加会话功能需要实现 TUI 集成")

	return nil
}

var sessionKillCmd = &cobra.Command{
	Use:   "kill <session-id>",
	Short: "终止后台会话",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionKill,
}

func runSessionKill(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	m := session.GlobalManager()
	if err := m.Remove(sessionID); err != nil {
		return fmt.Errorf("终止会话失败: %w", err)
	}

	fmt.Printf("会话 %s 已终止\n", sessionID)
	return nil
}