package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/kkkldpz/forge/internal/auth"
)

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.AddCommand(serverStartCmd)
	serverCmd.AddCommand(serverStopCmd)
	serverCmd.AddCommand(serverStatusCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "HTTP 服务器模式",
	Long:  "启动 HTTP API 服务器接受请求",
}

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 HTTP 服务器",
	RunE:  runServerStart,
}

func runServerStart(cmd *cobra.Command, args []string) error {
	port := 18789

	fmt.Printf("正在启动 Forge HTTP 服务器，端口: %d\n", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/api/chat", handleChat)
	mux.HandleFunc("/api/tools", handleTools)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:     mux,
		ReadTimeout: 30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		fmt.Printf("Forge 服务器已启动: http://localhost:%d\n", port)
		fmt.Println("\n可用端点:")
		fmt.Println("  GET  /health - 健康检查")
		fmt.Println("  POST /api/chat - 聊天接口")
		fmt.Println("  GET  /api/tools - 工具列表")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("服务器错误: %v\n", err)
		}
	}()

	<-sigChan
	fmt.Println("\n正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	fmt.Println("服务器已关闭")
	return nil
}

var serverStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止 HTTP 服务器",
	RunE:  runServerStop,
}

func runServerStop(cmd *cobra.Command, args []string) error {
	fmt.Println("服务器停止功能需要实现进程间通信")
	return nil
}

var serverStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看服务器状态",
	RunE:  runServerStatus,
}

func runServerStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("HTTP 服务器状态: 未运行")
	fmt.Println("\n使用 'forge server start' 启动服务器")
	return nil
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","version":"0.1.0"}`))
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.Header.Get("Authorization")
	if apiKey == "" {
		apiKeyResult := auth.GetAPIKey()
		if apiKeyResult.Source == auth.KeySourceNone {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message":"Chat API 实现中","status":"ok"}`))
}

func handleTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"tools":["bash","read","write","edit","glob","grep","task_create","task_get","task_list","task_stop","agent","cron_create","cron_delete","cron_list"],"status":"ok"}`))
}