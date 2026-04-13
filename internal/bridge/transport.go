package bridge

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Hub 管理所有已连接的 WebSocket 对等端。
type Hub struct {
	mu        sync.RWMutex
	peers     map[string]*WSPeer
	register  chan *WSPeer
	unreg     chan *WSPeer
	broadcast chan *Envelope
	logger    *slog.Logger
}

// WSPeer 表示一个已连接的 WebSocket 客户端。
type WSPeer struct {
	ID        string
	Name      string
	Connected time.Time
	send      chan []byte
	conn      *websocket.Conn
	hub       *Hub
}

// Envelope 包装带目标对等端 ID 的路由消息。
type Envelope struct {
	From    string          `json:"from"`
	To      string          `json:"to,omitempty"` // 空 = 广播
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// NewHub 创建新的 WebSocket Hub。
func NewHub() *Hub {
	return &Hub{
		peers:     make(map[string]*WSPeer),
		register:  make(chan *WSPeer),
		unreg:     make(chan *WSPeer),
		broadcast: make(chan *Envelope, 64),
		logger:    slog.Default().With("component", "bridge-hub"),
	}
}

// Run 启动 Hub 事件循环，需在 goroutine 中调用。
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.closeAll()
			return
		case peer := <-h.register:
			h.mu.Lock()
			h.peers[peer.ID] = peer
			h.mu.Unlock()
			h.logger.Info("对等端已注册", "peer_id", peer.ID, "name", peer.Name)
		case peer := <-h.unreg:
			h.mu.Lock()
			delete(h.peers, peer.ID)
			h.mu.Unlock()
			close(peer.send)
			h.logger.Info("对等端已注销", "peer_id", peer.ID)
		case env := <-h.broadcast:
			data, err := json.Marshal(env)
			if err != nil {
				continue
			}
			h.route(env, data)
		}
	}
}

// Send 将消息加入路由队列。
func (h *Hub) Send(env *Envelope) {
	h.broadcast <- env
}

// route 将消息投递到目标对等端。
func (h *Hub) route(env *Envelope, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if env.To != "" {
		// 定向消息
		if peer, ok := h.peers[env.To]; ok {
			select {
			case peer.send <- data:
			default:
				h.logger.Warn("对等端发送缓冲区已满，丢弃消息", "peer_id", env.To)
			}
		}
		return
	}

	// 广播给除发送者外的所有对等端
	for id, peer := range h.peers {
		if id == env.From {
			continue
		}
		select {
		case peer.send <- data:
		default:
			h.logger.Warn("对等端发送缓冲区已满，丢弃消息", "peer_id", id)
		}
	}
}

// PeerList 返回所有已连接对等端的信息。
func (h *Hub) PeerList() []*PeerInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	list := make([]*PeerInfo, 0, len(h.peers))
	for _, p := range h.peers {
		list = append(list, &PeerInfo{
			ID:        p.ID,
			Name:      p.Name,
			Connected: p.Connected,
		})
	}
	return list
}

// PeerCount 返回已连接的对等端数量。
func (h *Hub) PeerCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.peers)
}

// closeAll 关闭所有对等端连接。
func (h *Hub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, p := range h.peers {
		close(p.send)
		if p.conn != nil {
			p.conn.Close(websocket.StatusNormalClosure, "hub shutdown")
		}
	}
	h.peers = make(map[string]*WSPeer)
}

// ServeWS 处理 /ws 路径的 WebSocket 升级请求。
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		h.logger.Error("WebSocket 升级失败", "error", err)
		return
	}

	peerID := generatePeerID(r)
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "peer-" + peerID[:4]
	}

	peer := &WSPeer{
		ID:        peerID,
		Name:      name,
		Connected: time.Now(),
		send:      make(chan []byte, 32),
		conn:      conn,
		hub:       h,
	}

	h.register <- peer

	// 使用独立 context，不使用 r.Context()（HTTP handler 返回后会被取消）
	peerCtx, peerCancel := context.WithCancel(context.Background())
	go func() {
		peer.writePump(peerCtx)
		peerCancel()
	}()
	go func() {
		peer.readPump(peerCtx)
		peerCancel()
	}()
}

// readPump 从 WebSocket 连接读取消息。
func (p *WSPeer) readPump(ctx context.Context) {
	defer func() {
		p.hub.unreg <- p
		p.conn.CloseNow()
	}()

	for {
		_, data, err := p.conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
				p.hub.logger.Debug("读取错误", "peer_id", p.ID, "error", err)
			}
			return
		}

		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			p.hub.logger.Warn("收到无效消息", "peer_id", p.ID, "error", err)
			continue
		}

		env.From = p.ID
		p.hub.Send(&env)
	}
}

// writePump 向 WebSocket 连接写入消息。
func (p *WSPeer) writePump(ctx context.Context) {
	defer p.conn.CloseNow()

	for {
		select {
		case <-ctx.Done():
			p.conn.Close(websocket.StatusNormalClosure, "context cancelled")
			return
		case msg, ok := <-p.send:
			if !ok {
				p.conn.Close(websocket.StatusNormalClosure, "hub closing")
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := p.conn.Write(writeCtx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

// generatePeerID 从请求参数生成对等端 ID。
func generatePeerID(r *http.Request) string {
	id := r.URL.Query().Get("id")
	if id != "" {
		return id
	}
	return "p-" + randomHex(6)
}

// PeerInfo 是已连接对等端的公开信息。
type PeerInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Connected time.Time `json:"connected"`
}
