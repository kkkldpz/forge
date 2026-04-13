package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kkkldpz/forge/internal/voice"
)

func init() {
	rootCmd.AddCommand(voiceCmd)
	voiceCmd.AddCommand(voiceStartCmd)
	voiceCmd.AddCommand(voiceStopCmd)
	voiceCmd.AddCommand(voiceStatusCmd)
}

var voiceCmd = &cobra.Command{
	Use:   "voice",
	Short: "语音模式",
	Long:  "启用或禁用语音输入模式",
}

var voiceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "启动语音模式",
	RunE:  runVoiceStart,
}

func runVoiceStart(cmd *cobra.Command, args []string) error {
	if err := voice.Enable(); err != nil {
		return fmt.Errorf("启动语音模式失败: %w", err)
	}

	fmt.Println("✓ 语音模式已启动")
	fmt.Println("\n提示: 使用麦克风进行语音输入")

	return nil
}

var voiceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止语音模式",
	RunE:  runVoiceStop,
}

func runVoiceStop(cmd *cobra.Command, args []string) error {
	voice.Disable()
	fmt.Println("✓ 语音模式已停止")
	return nil
}

var voiceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看语音模式状态",
	RunE:  runVoiceStatus,
}

func runVoiceStatus(cmd *cobra.Command, args []string) error {
	if voice.IsEnabled() {
		fmt.Println("语音模式: 已启用")
	} else {
		fmt.Println("语音模式: 已禁用")
	}

	fmt.Println("\n提示: 使用 'forge voice start' 启用语音模式")

	return nil
}