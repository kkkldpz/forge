package transport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// SSETransport 通过 Server-Sent Events (SSE) 实现 MCP 消息传输。
// POST 发送消息，GET 接收 SSE 事件流。
type SSETransport struct {
	url      string
	headers  map[string]string
	client   *http.Client
	sessionID string

	mu       sync.Mutex
	sendErr  error
	recvCh   chan []byte
	doneCh   chan struct{}
	cancelFn context.CancelFunc
}

// NewSSETransport 创建 SSE 传输实例。
func NewSSETransport(ctx context.Context, url string, headers map[string]string) *SSETransport {
	ctx, cancel := context.WithCancel(ctx)

	t := &SSETransport{
		url:      strings.TrimRight(url, "/"),
		headers:  headers,
		client:   &http.Client{Timeout: 0}, // SSE 长连接不设超时
		recvCh:   make(chan []byte, 64),
		doneCh:   make(chan struct{}),
		cancelFn: cancel,
	}

	// 启动 SSE 事件流监听
	go t.sseLoop(ctx)

	return t
}

// Send 通过 POST 请求发送 JSON-RPC 消息。
func (t *SSETransport) Send(msg []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.sendErr != nil {
		return fmt.Errorf("传输已关闭: %w", t.sendErr)
	}

	endpoint := t.url
	if t.sessionID != "" {
		endpoint += "?sessionId=" + t.sessionID
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(msg))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// 检查响应中的 session ID
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		t.sessionID = sid
	}

	return nil
}

// Receive 返回消息接收通道。
func (t *SSETransport) Receive() <-chan []byte {
	return t.recvCh
}

// Close 关闭 SSE 传输连接。
func (t *SSETransport) Close() error {
	t.cancelFn()
	<-t.doneCh
	close(t.recvCh)
	return nil
}

// sseLoop 监听 SSE 事件流，解析 JSON-RPC 消息。
func (t *SSETransport) sseLoop(ctx context.Context) {
	defer close(t.doneCh)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := t.connectSSE(ctx)
		if err != nil {
			// 连接断开时自动重连
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
				fmt.Fprintf(os.Stderr, "[mcp:sse] 连接断开，2 秒后重连: %v\n", err)
				continue
			}
		}
	}
}

// connectSSE 建立 SSE 连接并读取事件。
func (t *SSETransport) connectSSE(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url, nil)
	if err != nil {
		return fmt.Errorf("创建 SSE 请求失败: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	if t.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", t.sessionID)
	}
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("SSE 连接失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SSE 连接返回 HTTP %d", resp.StatusCode)
	}

	// 保存 session ID
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		t.sessionID = sid
	}

	// 读取 SSE 事件
	scanner := bufio.NewScanner(resp.Body)
	// 增大缓冲区以处理大消息
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	var eventBuffer bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// 空行表示事件结束，处理缓冲区
			data := eventBuffer.String()
			eventBuffer.Reset()
			if data == "" {
				continue
			}
			t.handleSSEData(data)
			continue
		}

		// 解析 SSE 字段
		if strings.HasPrefix(line, "data: ") {
			if eventBuffer.Len() > 0 {
				eventBuffer.WriteByte('\n')
			}
			eventBuffer.WriteString(strings.TrimPrefix(line, "data: "))
		}
	}

	return scanner.Err()
}

// handleSSEData 解析 SSE data 内容为 JSON-RPC 消息。
func (t *SSETransport) handleSSEData(data string) {
	// 尝试解析为 JSON-RPC 消息
	var msg json.RawMessage
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		// 非 JSON 数据，跳过
		return
	}

	select {
	case t.recvCh <- []byte(data):
	default:
		fmt.Fprintf(os.Stderr, "[mcp:sse] 警告: 接收通道已满，丢弃消息\n")
	}
}
