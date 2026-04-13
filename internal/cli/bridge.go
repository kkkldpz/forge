package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kkkldpz/forge/internal/bridge"
)

func init() {
	rootCmd.AddCommand(bridgeCmd)
	bridgeCmd.AddCommand(bridgeStartCmd)
	bridgeCmd.AddCommand(bridgeStopCmd)
	bridgeCmd.AddCommand(bridgeStatusCmd)
}

var bridgeCmd = &cobra.Command{
	Use:   "bridge",
	Short: "管理 Forge 远程控制桥接",
	Long:  "启动、停止或查看 Forge 远程控制桥接状态",
}

var bridgeStartCmd = &cobra.Command{
	Use:   "start",
	Short: "启动远程控制桥接",
	RunE:  runBridgeStart,
}

func runBridgeStart(cmd *cobra.Command, args []string) error {
	b := bridge.GlobalBridge()

	port := 18792
	jwtSecret := ""

	if err := b.Enable(port, jwtSecret); err != nil {
		return fmt.Errorf("启用 bridge 失败: %w", err)
	}

	addr := fmt.Sprintf(":%d", port)
	if err := b.Start(addr); err != nil {
		return fmt.Errorf("启动 bridge 失败: %w", err)
	}

	fmt.Printf("Forge bridge 已启动，监听端口 %d\n", port)
	return nil
}

var bridgeStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止远程控制桥接",
	RunE:  runBridgeStop,
}

func runBridgeStop(cmd *cobra.Command, args []string) error {
	b := bridge.GlobalBridge()
	b.Disable()
	fmt.Println("Forge bridge 已停止")
	return nil
}

var bridgeStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看远程控制桥接状态",
	RunE:  runBridgeStatus,
}

func runBridgeStatus(cmd *cobra.Command, args []string) error {
	b := bridge.GlobalBridge()

	fmt.Printf("Forge Bridge 状态:\n")
	fmt.Printf("  启用: %v\n", b.IsEnabled())

	if b.IsEnabled() {
		peers := b.ListPeers()
		fmt.Printf("  已连接的 peer: %d\n", len(peers))

		if len(peers) > 0 {
			fmt.Println("\nPeers:")
			for _, p := range peers {
				fmt.Printf("  - %s (%s)\n", p.ID, p.Name)
			}
		}
	}

	return nil
}