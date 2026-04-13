package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/kkkldpz/forge/internal/daemon"
)

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "管理 Forge 后台服务",
	Long:  "启动、停止或查看 Forge 后台服务状态",
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "启动后台服务",
	RunE:  runDaemonStart,
}

func runDaemonStart(cmd *cobra.Command, args []string) error {
	d := daemon.GlobalDaemon()

	ctx := context.Background()
	if err := d.Start(ctx); err != nil {
		return fmt.Errorf("启动 daemon 失败: %w", err)
	}

	fmt.Println("Forge daemon 已启动")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	d.Stop()
	fmt.Println("Forge daemon 已停止")
	return nil
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止后台服务",
	RunE:  runDaemonStop,
}

func runDaemonStop(cmd *cobra.Command, args []string) error {
	d := daemon.GlobalDaemon()
	if err := d.Stop(); err != nil {
		return fmt.Errorf("停止 daemon 失败: %w", err)
	}
	fmt.Println("Forge daemon 已停止")
	return nil
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看后台服务状态",
	RunE:  runDaemonStatus,
}

func runDaemonStatus(cmd *cobra.Command, args []string) error {
	registry := daemon.GlobalRegistry()
	workers := registry.List()

	fmt.Printf("Forge Daemon 状态:\n")
	fmt.Printf("  运行中的 worker: %d\n", len(workers))

	if len(workers) > 0 {
		fmt.Println("\nWorkers:")
		for _, w := range workers {
			fmt.Printf("  - %s (%s): %s\n", w.ID, w.Name, w.Status)
		}
	}

	return nil
}