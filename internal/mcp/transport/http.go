package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// HTTPTransport 通过 HTTP POST 实现 MCP 消息传输。
// 支持标准 JSON 响应和 SSE 流式响应。
type HTTPTransport struct {
	url        string
	headers    map[string]string
	client     *http.Client
	sessionID  string

	mu      sync.Mutex
	sendErr error
	recvCh  chan []byte
}

// NewHTTPTransport 创建 HTTP 传输实例。
func NewHTTPTransport(url string, headers map[string]string) *HTTPTransport {
	return &HTTPTransport{
		url:     strings.TrimRight(url, "/"),
		headers: headers,
		client:  &http.Client{},
		recvCh:  make(chan []byte, 64),
	}
}

// Send 通过 POST 请求发送 JSON-RPC 消息并等待响应。
// HTTP 传输是请求-响应模式，响应会直接推送到接收通道。
func (t *HTTPTransport) Send(msg []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.sendErr != nil {
		return fmt.Errorf("传输已关闭: %w", t.sendErr)
	}

	req, err := http.NewRequest(http.MethodPost, t.url, bytes.NewReader(msg))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if t.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", t.sessionID)
	}
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 保存 session ID
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		t.sessionID = sid
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")

	if strings.Contains(contentType, "text/event-stream") {
		// SSE 流式响应：逐行解析并推送
		return t.handleSSEResponse(resp.Body)
	}

	// 标准 JSON 响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 推送到接收通道
	t.recvCh <- body
	return nil
}

// Receive 返回消息接收通道。
func (t *HTTPTransport) Receive() <-chan []byte {
	return t.recvCh
}

// Close 关闭 HTTP 传输。
func (t *HTTPTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sendErr = fmt.Errorf("已关闭")
	close(t.recvCh)
	return nil
}

// handleSSEResponse 处理 SSE 格式的响应体。
func (t *HTTPTransport) handleSSEResponse(body io.Reader) error {
	decoder := json.NewDecoder(body)
	for decoder.More() {
		var msg json.RawMessage
		if err := decoder.Decode(&msg); err != nil {
			break
		}
		t.recvCh <- []byte(msg)
	}
	return nil
}
