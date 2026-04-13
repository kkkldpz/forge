package bridge

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Enabled       bool
	Port          int
	JWTSecret     string
	AllowedOrigins []string
}

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type Bridge struct {
	mu      sync.RWMutex
	config  *Config
	peers   map[string]*Peer
	server  *http.Server
}

type Peer struct {
	ID        string
	Name      string
	Connected time.Time
	LastSeen  time.Time
	RemoteAddr string
}

var (
	globalBridge     *Bridge
	globalBridgeOnce sync.Once
)

func GlobalBridge() *Bridge {
	globalBridgeOnce.Do(func() {
		globalBridge = &Bridge{
			config: &Config{
				Enabled:       false,
				Port:          18792,
				JWTSecret:     "",
				AllowedOrigins: []string{"*"},
			},
			peers: make(map[string]*Peer),
		}
	})
	return globalBridge
}

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

func (b *Bridge) Disable() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		b.server.Shutdown(ctx)
	}

	b.config.Enabled = false
	b.peers = make(map[string]*Peer)
}

func (b *Bridge) IsEnabled() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.config.Enabled
}

func (b *Bridge) Start(addr string) error {
	b.mu.RLock()
	if !b.config.Enabled {
		b.mu.RUnlock()
		return fmt.Errorf("bridge 未启用")
	}
	b.mu.RUnlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/status", b.handleStatus)
	mux.HandleFunc("/connect", b.handleConnect)
	mux.HandleFunc("/message", b.handleMessage)
	mux.HandleFunc("/peer/", b.handlePeer)

	b.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := b.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Bridge server error: %v\n", err)
		}
	}()

	return nil
}

func (b *Bridge) handleStatus(w http.ResponseWriter, r *http.Request) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	status := map[string]interface{}{
		"enabled": b.config.Enabled,
		"peers":   len(b.peers),
	}

	if b.config.Enabled {
		status["port"] = b.config.Port
	}

	json.NewEncoder(w).Encode(status)
}

func (b *Bridge) handleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	peer := &Peer{
		ID:         req.ID,
		Name:       req.Name,
		Connected:  time.Now(),
		LastSeen:   time.Now(),
		RemoteAddr: r.RemoteAddr,
	}

	b.mu.Lock()
	b.peers[req.ID] = peer
	b.mu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"status": "connected"})
}

func (b *Bridge) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var msg Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	broadcast := msg.Payload
	for _, peer := range b.peers {
		peer.LastSeen = time.Now()
		_ = broadcast
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "broadcast"})
}

func (b *Bridge) handlePeer(w http.ResponseWriter, r *http.Request) {
	peerID := strings.TrimPrefix(r.URL.Path, "/peer/")

	b.mu.RLock()
	defer b.mu.RUnlock()

	peer, ok := b.peers[peerID]
	if !ok {
		http.Error(w, "Peer not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(peer)
}

func (b *Bridge) ListPeers() []*Peer {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]*Peer, 0, len(b.peers))
	for _, peer := range b.peers {
		copyPeer := *peer
		result = append(result, &copyPeer)
	}
	return result
}

func (b *Bridge) GetPeer(id string) (*Peer, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	peer, ok := b.peers[id]
	return peer, ok
}

func GenerateToken(secret, peerID string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(peerID))
	h.Write([]byte(time.Now().Format(time.RFC3339)))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func ValidateToken(secret, peerID, token string) bool {
	expected := GenerateToken(secret, peerID)
	return hmac.Equal([]byte(token), []byte(expected))
}

func (b *Bridge) ConnectSocket(addr string) (net.Conn, error) {
	return net.Dial("tcp", addr)
}

var _ context.Context = context.Background()