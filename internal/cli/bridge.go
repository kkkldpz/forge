package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/kkkldpz/forge/internal/bridge"
)

func init() {
	rootCmd.AddCommand(bridgeCmd)
	bridgeCmd.AddCommand(bridgeStartCmd)
	bridgeCmd.AddCommand(bridgeStopCmd)
	bridgeCmd.AddCommand(bridgeStatusCmd)
	bridgeCmd.AddCommand(bridgePeersCmd)
	bridgeCmd.AddCommand(bridgeSendCmd)

	bridgeStartCmd.Flags().IntP("port", "p", 18792, "桥接监听端口")
	bridgeStartCmd.Flags().String("jwt-secret", "", "JWT 认证密钥")
}

var bridgeCmd = &cobra.Command{
	Use:   "bridge",
	Short: "管理 Forge 远程控制桥接",
	Long:  "启动、停止或查看 Forge 远程控制桥接服务器",
}

var bridgeStartCmd = &cobra.Command{
	Use:   "start",
	Short: "启动远程控制桥接",
	RunE:  runBridgeStart,
}

var bridgeStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止远程控制桥接",
	RunE:  runBridgeStop,
}

var bridgeStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看桥接状态",
	RunE:  runBridgeStatus,
}

var bridgePeersCmd = &cobra.Command{
	Use:   "peers",
	Short: "列出已连接的对等端",
	RunE:  runBridgePeers,
}

var bridgeSendCmd = &cobra.Command{
	Use:   "send [对等端ID] [消息类型] [载荷]",
	Short: "向指定对等端发送消息",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runBridgeSend,
}

func runBridgeStart(cmd *cobra.Command, args []string) error {
	b := bridge.GlobalBridge()

	port, _ := cmd.Flags().GetInt("port")
	jwtSecret, _ := cmd.Flags().GetString("jwt-secret")

	if err := b.Enable(port, jwtSecret); err != nil {
		return fmt.Errorf("启用桥接失败: %w", err)
	}

	addr := fmt.Sprintf(":%d", port)
	if err := b.Start(addr); err != nil {
		return fmt.Errorf("启动桥接失败: %w", err)
	}

	fmt.Printf("Forge bridge 已启动，端口 %d\n", port)
	fmt.Printf("  WebSocket: ws://localhost:%d/ws\n", port)
	fmt.Printf("  状态接口:  http://localhost:%d/api/status\n", port)
	fmt.Println()
	fmt.Println("按 Ctrl+C 停止")

	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\n正在关闭桥接...")
	b.Disable()
	return nil
}

func runBridgeStop(cmd *cobra.Command, args []string) error {
	b := bridge.GlobalBridge()
	b.Disable()
	fmt.Println("Forge bridge 已停止")
	return nil
}

func runBridgeStatus(cmd *cobra.Command, args []string) error {
	b := bridge.GlobalBridge()

	fmt.Println("Forge Bridge 状态:")
	fmt.Printf("  已启用: %v\n", b.IsEnabled())

	if b.IsEnabled() {
		fmt.Printf("  对等端: %d\n", b.PeerCount())
	}
	return nil
}

func runBridgePeers(cmd *cobra.Command, args []string) error {
	b := bridge.GlobalBridge()
	peers := b.ListPeers()

	if len(peers) == 0 {
		fmt.Println("无已连接的对等端")
		return nil
	}

	fmt.Printf("%-12s %-20s %-s\n", "ID", "名称", "连接时间")
	fmt.Printf("%-12s %-20s %-s\n", "---", "----", "---------")
	for _, p := range peers {
		fmt.Printf("%-12s %-20s %s\n", p.ID, p.Name,
			p.Connected.Format(time.RFC3339))
	}
	return nil
}

func runBridgeSend(cmd *cobra.Command, args []string) error {
	b := bridge.GlobalBridge()

	peerID := args[0]
	msgType := args[1]
	payload := "{}"
	if len(args) > 2 {
		payload = args[2]
	}

	// 校验 JSON 格式
	if !json.Valid([]byte(payload)) {
		return fmt.Errorf("载荷必须是合法的 JSON")
	}

	if err := b.SendToPeer("cli", peerID, msgType, json.RawMessage(payload)); err != nil {
		return fmt.Errorf("发送消息失败: %w", err)
	}

	fmt.Printf("消息已发送至 %s（类型: %s）\n", peerID, msgType)
	return nil
}