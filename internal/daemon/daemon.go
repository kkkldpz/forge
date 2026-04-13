// Package daemon 实现长驻守护进程模式和 worker 注册。
package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	SocketPath        = "/tmp/forge-daemon.sock"
	HeartbeatInterval = 30 * time.Second
	WorkerTimeout     = 90 * time.Second
)

type Daemon struct {
	mu       sync.RWMutex
	running  bool
	server   *http.Server
	listener net.Listener
	registry *Registry
}

var (
	globalDaemon     *Daemon
	globalDaemonOnce sync.Once
)

func GlobalDaemon() *Daemon {
	globalDaemonOnce.Do(func() {
		globalDaemon = &Daemon{
			registry: GlobalRegistry(),
		}
	})
	return globalDaemon
}

func (d *Daemon) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return fmt.Errorf("daemon 已在运行")
	}
	d.running = true
	d.mu.Unlock()

	// 清理旧的 socket 文件
	if _, err := net.Dial("unix", SocketPath); err == nil {
		slog.Warn("daemon socket 已存在，尝试清理")
		net.DialTimeout("unix", SocketPath, time.Second)
	}

	// 创建 Unix socket 监听器
	ln, err := net.Listen("unix", SocketPath)
	if err != nil {
		return fmt.Errorf("创建 socket 失败: %w", err)
	}
	d.listener = ln

	// 创建 HTTP 服务器
	d.server = &http.Server{
		Handler: d,
	}

	go func() {
		if err := d.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			slog.Error("daemon server error", "error", err)
		}
	}()

	// 启动 worker 健康检查
	go d.healthCheck()

	slog.Info("Forge daemon 已启动", "socket", SocketPath)
	return nil
}

func (d *Daemon) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return fmt.Errorf("daemon 未运行")
	}

	d.running = false

	if d.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := d.server.Shutdown(ctx); err != nil {
			slog.Error("关闭 daemon server 失败", "error", err)
		}
	}

	if d.listener != nil {
		d.listener.Close()
	}

	slog.Info("Forge daemon 已停止")
	return nil
}

func (d *Daemon) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/health":
		d.handleHealth(w, r)
	case "/workers":
		d.handleWorkers(w, r)
	case "/register":
		d.handleRegister(w, r)
	case "/unregister":
		d.handleUnregister(w, r)
	default:
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

func (d *Daemon) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (d *Daemon) handleWorkers(w http.ResponseWriter, r *http.Request) {
	workers := d.registry.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workers)
}

func (d *Daemon) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var worker Worker
	if err := json.NewDecoder(r.Body).Decode(&worker); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := d.registry.Register(&worker); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "registered"})
}

func (d *Daemon) handleUnregister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	d.registry.Unregister(req.ID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "unregistered"})
}

func (d *Daemon) healthCheck() {
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		d.mu.RLock()
		if !d.running {
			d.mu.RUnlock()
			return
		}
		d.mu.RUnlock()

		workers := d.registry.List()
		now := time.Now()

		for _, w := range workers {
			if now.Sub(w.LastBeat) > WorkerTimeout {
				slog.Warn("移除超时 worker", "id", w.ID)
				d.registry.Unregister(w.ID)
			}
		}
	}
}
