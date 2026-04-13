// Package bridge 实现远程控制桥接服务器，支持 WebSocket 双向通信和消息路由。
package bridge

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Config 保存桥接配置。
type Config struct {
	Enabled        bool
	Port           int
	JWTSecret      string
	AllowedOrigins []string
}

// Bridge 是远程控制桥接服务器。
type Bridge struct {
	mu      sync.RWMutex
	config  *Config
	hub     *Hub
	session *SessionManager
	server  *http.Server
	cancel  context.CancelFunc
	logger  *slog.Logger
}

var (
	globalBridge     *Bridge
	globalBridgeOnce sync.Once
)

// GlobalBridge 返回桥接单例。
func GlobalBridge() *Bridge {
	globalBridgeOnce.Do(func() {
		globalBridge = &Bridge{
			config: &Config{
				Enabled:        false,
				Port:           18792,
				JWTSecret:      "",
				AllowedOrigins: []string{"*"},
			},
			hub:     NewHub(),
			session: NewSessionManager(),
			logger:  slog.Default().With("component", "bridge"),
		}
	})
	return globalBridge
}

// Enable 启用桥接。
func (b *Bridge) Enable(port int, jwtSecret string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.config.Enabled {
		return fmt.Errorf("bridge 已启用")
	}

	b.config.Enabled = true
	b.config.Port = port
	b.config.JWTSecret = jwtSecret

	return nil
}

// Disable 关闭桥接。
func (b *Bridge) Disable() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.cancel != nil {
		b.cancel()
	}

	if b.server != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		b.server.Shutdown(shutdownCtx)
	}

	b.config.Enabled = false
}

// IsEnabled 返回桥接是否启用。
func (b *Bridge) IsEnabled() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.config.Enabled
}

// Start 启动 HTTP/WebSocket 服务器。
func (b *Bridge) Start(addr string) error {
	b.mu.RLock()
	if !b.config.Enabled {
		b.mu.RUnlock()
		return fmt.Errorf("bridge 未启用")
	}
	b.mu.RUnlock()

	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel

	// 启动 Hub 事件循环
	go b.hub.Run(ctx)

	// 启动会话清理协程
	go b.sessionCleanup(ctx)

	// 注册 HTTP 路由
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", b.hub.ServeWS)
	mux.HandleFunc("/api/status", b.handleStatus)
	mux.HandleFunc("/api/peers", b.handlePeers)
	mux.HandleFunc("/api/sessions", b.handleSessions)
	mux.HandleFunc("/api/message", b.handleMessage)

	b.server = &http.Server{
		Addr:    addr,
		Handler: b.jwtMiddleware(mux),
	}

	go func() {
		if err := b.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			b.logger.Error("服务器错误", "error", err)
		}
	}()

	b.logger.Info("bridge 已启动", "addr", addr)
	return nil
}

// SendToPeer 向指定对等端发送消息。
func (b *Bridge) SendToPeer(from, to, msgType string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	b.hub.Send(&Envelope{
		From:    from,
		To:      to,
		Type:    msgType,
		Payload: data,
	})
	return nil
}

// Broadcast 向所有已连接的对等端广播消息。
func (b *Bridge) Broadcast(from, msgType string, payload any) error {
	return b.SendToPeer(from, "", msgType, payload)
}

// ListPeers 返回已连接对等端的信息。
func (b *Bridge) ListPeers() []*PeerInfo {
	return b.hub.PeerList()
}

// PeerCount 返回已连接的对等端数量。
func (b *Bridge) PeerCount() int {
	return b.hub.PeerCount()
}

// --- HTTP 处理函数 ---

func (b *Bridge) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"enabled":       b.config.Enabled,
		"port":          b.config.Port,
		"peer_count":    b.hub.PeerCount(),
		"session_count": len(b.session.List()),
	})
}

func (b *Bridge) handlePeers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b.hub.PeerList())
}

func (b *Bridge) handleSessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(b.session.List())
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "缺少 id 参数", http.StatusBadRequest)
			return
		}
		b.session.Remove(id)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

func (b *Bridge) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		From    string          `json:"from"`
		To      string          `json:"to"`
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求", http.StatusBadRequest)
		return
	}

	if req.From == "" || req.Type == "" {
		http.Error(w, "from 和 type 为必填", http.StatusBadRequest)
		return
	}

	b.hub.Send(&Envelope{
		From:    req.From,
		To:      req.To,
		Type:    req.Type,
		Payload: req.Payload,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}

// --- JWT 中间件 ---

func (b *Bridge) jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// WebSocket 升级不需要 JWT
		if strings.HasPrefix(r.URL.Path, "/ws") {
			next.ServeHTTP(w, r)
			return
		}

		// 状态端点公开访问
		if r.URL.Path == "/api/status" {
			next.ServeHTTP(w, r)
			return
		}

		// JWT 验证
		if b.config.JWTSecret != "" {
			token := r.Header.Get("Authorization")
			token = strings.TrimPrefix(token, "Bearer ")
			if token == "" {
				http.Error(w, "未授权", http.StatusUnauthorized)
				return
			}

			peerID := r.Header.Get("X-Peer-ID")
			if !ValidateToken(b.config.JWTSecret, peerID, token) {
				http.Error(w, "令牌无效", http.StatusUnauthorized)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// sessionCleanup 定期清理过期会话。
func (b *Bridge) sessionCleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			expired := b.session.ExpireStale(30 * time.Minute)
			if expired > 0 {
				b.logger.Info("清理过期会话", "count", expired)
			}
		}
	}
}

// --- 令牌工具函数 ---

// GenerateToken 创建 HMAC-SHA256 令牌。
func GenerateToken(secret, peerID string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(peerID))
	h.Write([]byte(time.Now().Format(time.RFC3339)))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// ValidateToken 验证 HMAC-SHA256 令牌。
func ValidateToken(secret, peerID, token string) bool {
	expected := GenerateToken(secret, peerID)
	return hmac.Equal([]byte(token), []byte(expected))
}

// randomHex 生成指定字节数的随机十六进制字符串。
func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
