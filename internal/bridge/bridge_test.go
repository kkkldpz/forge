package bridge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestGlobalBridge(t *testing.T) {
	b1 := GlobalBridge()
	b2 := GlobalBridge()

	if b1 != b2 {
		t.Error("GlobalBridge should return the same instance")
	}
}

func TestBridge_EnableDisable(t *testing.T) {
	b := GlobalBridge()

	err := b.Enable(18792, "test-secret")
	if err != nil {
		t.Errorf("Enable failed: %v", err)
	}

	if !b.IsEnabled() {
		t.Error("Bridge should be enabled after Enable()")
	}

	err = b.Enable(18793, "another-secret")
	if err == nil {
		t.Error("Expected error when enabling already enabled bridge")
	}

	b.Disable()
	if b.IsEnabled() {
		t.Error("Bridge should be disabled after Disable()")
	}
}

func TestValidateToken(t *testing.T) {
	secret := "test-secret"
	peerID := "peer-123"

	token := GenerateToken(secret, peerID)

	if !ValidateToken(secret, peerID, token) {
		t.Error("Valid token should pass validation")
	}

	if ValidateToken(secret, peerID, "invalid-token") {
		t.Error("Invalid token should fail validation")
	}

	if ValidateToken("wrong-secret", peerID, token) {
		t.Error("Token with wrong secret should fail validation")
	}
}

func TestHub_RegisterUnregister(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	// Register a mock peer
	peer := &WSPeer{
		ID:   "test-peer",
		Name: "Test",
		send: make(chan []byte, 16),
		conn: nil,
		hub:  hub,
	}

	hub.register <- peer
	time.Sleep(50 * time.Millisecond) // wait for event loop

	if hub.PeerCount() != 1 {
		t.Errorf("Expected 1 peer, got %d", hub.PeerCount())
	}

	hub.unreg <- peer
	time.Sleep(50 * time.Millisecond)

	if hub.PeerCount() != 0 {
		t.Errorf("Expected 0 peers, got %d", hub.PeerCount())
	}
}

func TestHub_Broadcast(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	// Create two mock peers with buffered send channels
	peer1 := &WSPeer{
		ID:   "peer-1",
		Name: "Peer 1",
		send: make(chan []byte, 16),
		conn: nil,
		hub:  hub,
	}
	peer2 := &WSPeer{
		ID:   "peer-2",
		Name: "Peer 2",
		send: make(chan []byte, 16),
		conn: nil,
		hub:  hub,
	}

	hub.register <- peer1
	hub.register <- peer2
	time.Sleep(50 * time.Millisecond)

	// Broadcast from peer-1
	hub.Send(&Envelope{
		From:    "peer-1",
		To:      "",
		Type:    "test",
		Payload: json.RawMessage(`{"hello":"world"}`),
	})

	time.Sleep(50 * time.Millisecond)

	// peer-1 should NOT receive its own message
	select {
	case <-peer1.send:
		t.Error("peer-1 should not receive its own broadcast")
	default:
		// expected
	}

	// peer-2 should receive the message
	select {
	case msg := <-peer2.send:
		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}
		if env.Type != "test" {
			t.Errorf("Expected type 'test', got '%s'", env.Type)
		}
	case <-time.After(time.Second):
		t.Error("peer-2 should have received the broadcast")
	}
}

func TestHub_DirectMessage(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	peer1 := &WSPeer{
		ID:   "peer-1",
		Name: "Peer 1",
		send: make(chan []byte, 16),
		conn: nil,
		hub:  hub,
	}
	peer2 := &WSPeer{
		ID:   "peer-2",
		Name: "Peer 2",
		send: make(chan []byte, 16),
		conn: nil,
		hub:  hub,
	}

	hub.register <- peer1
	hub.register <- peer2
	time.Sleep(50 * time.Millisecond)

	// Direct message from peer-1 to peer-2
	hub.Send(&Envelope{
		From:    "peer-1",
		To:      "peer-2",
		Type:    "direct",
		Payload: json.RawMessage(`{"msg":"hello"}`),
	})

	time.Sleep(50 * time.Millisecond)

	// peer-1 should NOT receive the direct message
	select {
	case <-peer1.send:
		t.Error("peer-1 should not receive direct message to peer-2")
	default:
	}

	// peer-2 should receive the direct message
	select {
	case msg := <-peer2.send:
		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}
		if env.Type != "direct" {
			t.Errorf("Expected type 'direct', got '%s'", env.Type)
		}
	case <-time.After(time.Second):
		t.Error("peer-2 should have received the direct message")
	}
}

func TestSessionManager(t *testing.T) {
	sm := NewSessionManager()

	s := sm.Create("sess-1", "peer-1")
	if s.ID != "sess-1" {
		t.Errorf("Expected session ID 'sess-1', got '%s'", s.ID)
	}

	got, ok := sm.Get("sess-1")
	if !ok || got.ID != "sess-1" {
		t.Error("Should retrieve created session")
	}

	sm.Touch("sess-1")
	got2, _ := sm.Get("sess-1")
	if got2.LastActivity.Before(s.LastActivity) {
		t.Error("Touch should update LastActivity")
	}

	list := sm.List()
	if len(list) != 1 {
		t.Errorf("Expected 1 session, got %d", len(list))
	}

	sm.Remove("sess-1")
	if _, ok := sm.Get("sess-1"); ok {
		t.Error("Session should be removed")
	}
}

func TestSessionManager_ExpireStale(t *testing.T) {
	sm := NewSessionManager()

	s1 := sm.Create("sess-1", "peer-1")
	_ = s1
	s2 := sm.Create("sess-2", "peer-2")
	_ = s2

	// Manually age sess-1
	sm.mu.Lock()
	sm.sessions["sess-1"].LastActivity = time.Now().Add(-time.Hour)
	sm.mu.Unlock()

	expired := sm.ExpireStale(30 * time.Minute)
	if expired != 1 {
		t.Errorf("Expected 1 expired session, got %d", expired)
	}

	if _, ok := sm.Get("sess-1"); ok {
		t.Error("sess-1 should be expired")
	}
	if _, ok := sm.Get("sess-2"); !ok {
		t.Error("sess-2 should still exist")
	}
}

func TestBridge_HTTPHandlers(t *testing.T) {
	b := GlobalBridge()
	b.Enable(0, "")

	// Status endpoint (public)
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	b.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var status map[string]any
	json.NewDecoder(w.Body).Decode(&status)
	if status["enabled"] != true {
		t.Error("Status should show enabled=true")
	}

	b.Disable()
}

func TestBridge_JWTMiddleware(t *testing.T) {
	b := &Bridge{
		config: &Config{
			Enabled:   true,
			JWTSecret: "my-secret",
		},
		hub:    NewHub(),
		logger: nil,
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := b.jwtMiddleware(inner)

	// No token -> 401
	req := httptest.NewRequest(http.MethodGet, "/api/peers", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 without token, got %d", w.Code)
	}

	// Valid token -> 200
	token := GenerateToken("my-secret", "peer-1")
	req = httptest.NewRequest(http.MethodGet, "/api/peers", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Peer-ID", "peer-1")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 with valid token, got %d", w.Code)
	}

	// /ws path bypasses JWT
	req = httptest.NewRequest(http.MethodGet, "/ws", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for /ws, got %d", w.Code)
	}

	// /api/status is public
	req = httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for /api/status, got %d", w.Code)
	}
}

func TestWebSocketConnect(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	// Start a test HTTP server with WebSocket support
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWS(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?name=test-client&id=ws-test-1"

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.CloseNow()

	time.Sleep(50 * time.Millisecond)

	if hub.PeerCount() != 1 {
		t.Errorf("Expected 1 peer, got %d", hub.PeerCount())
	}

	peers := hub.PeerList()
	if len(peers) != 1 || peers[0].Name != "test-client" {
		t.Errorf("Expected peer named 'test-client', got %v", peers)
	}
}

func TestWebSocketMessaging(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	// Track broadcast messages
	var received []Envelope
	var recvMu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWS(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// Connect peer 2 first (will receive messages)
	conn2, _, err := websocket.Dial(ctx, wsURL+"?name=receiver&id=peer-2", nil)
	if err != nil {
		t.Fatalf("Failed to connect peer-2: %v", err)
	}
	defer conn2.CloseNow()

	// Read goroutine for peer-2
	go func() {
		for {
			_, data, err := conn2.Read(ctx)
			if err != nil {
				return
			}
			var env Envelope
			if err := json.Unmarshal(data, &env); err == nil {
				recvMu.Lock()
				received = append(received, env)
				recvMu.Unlock()
			}
		}
	}()

	time.Sleep(50 * time.Millisecond)

	// Connect peer 1 (sender)
	conn1, _, err := websocket.Dial(ctx, wsURL+"?name=sender&id=peer-1", nil)
	if err != nil {
		t.Fatalf("Failed to connect peer-1: %v", err)
	}
	defer conn1.CloseNow()

	time.Sleep(50 * time.Millisecond)

	// Send a broadcast from peer-1
	msg, _ := json.Marshal(&Envelope{
		From:    "peer-1",
		To:      "",
		Type:    "chat",
		Payload: json.RawMessage(`{"text":"hello"}`),
	})
	conn1.Write(ctx, websocket.MessageText, msg)

	time.Sleep(100 * time.Millisecond)

	recvMu.Lock()
	defer recvMu.Unlock()
	if len(received) == 0 {
		t.Fatal("peer-2 should have received a message")
	}
	if received[0].Type != "chat" {
		t.Errorf("Expected type 'chat', got '%s'", received[0].Type)
	}
}
