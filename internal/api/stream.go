package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/kkkldpz/forge/internal/types"
)

// RawSSEEvent 是从 SSE 流中解析出的原始事件。
type RawSSEEvent struct {
	Event string // event 字段的值（如 "message_start"）
	Data  string // data 字段的原始 JSON
}

// ParseSSE 从 HTTP 响应体中解析 SSE 事件流。
// 返回一个只读 channel，调用者应遍历它来接收事件。
// 当流结束或出错时，channel 会被关闭。
func ParseSSE(body io.Reader) <-chan RawSSEEvent {
	ch := make(chan RawSSEEvent, 64)
	go func() {
		defer close(ch)
		parseSSEStream(body, ch)
	}()
	return ch
}

// parseSSEStream 逐行读取 SSE 流，提取事件并发送到 channel。
func parseSSEStream(body io.Reader, ch chan<- RawSSEEvent) {
	scanner := bufio.NewScanner(body)
	// 增大缓冲区，防止长行被截断
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var event string
	var dataLines []string

	for scanner.Scan() {
		line := scanner.Text()

		// 空行表示事件结束，发送累积的事件
		if line == "" {
			if len(dataLines) > 0 {
				ch <- RawSSEEvent{
					Event: event,
					Data:  strings.Join(dataLines, "\n"),
				}
			}
			event = ""
			dataLines = nil
			continue
		}

		// 注释行（以冒号开头），忽略
		if strings.HasPrefix(line, ":") {
			continue
		}

		// 解析 "field: value" 格式
		field, value, ok := parseSSEField(line)
		if !ok {
			continue
		}

		switch field {
		case "event":
			event = value
		case "data":
			dataLines = append(dataLines, value)
		// id、retry 等其他字段忽略
		default:
			// 忽略不认识的字段
		}
	}

	// 流结束时发送最后一个事件（如果 buffer 中还有数据）
	if len(dataLines) > 0 {
		ch <- RawSSEEvent{
			Event: event,
			Data:  strings.Join(dataLines, "\n"),
		}
	}
}

// parseSSEField 将 "field: value" 拆分为字段名和值。
func parseSSEField(line string) (field, value string, ok bool) {
	field, value, found := strings.Cut(line, ":")
	if !found {
		return line, "", true // 没有冒号的行，整个作为字段名
	}
	// 去掉值开头的单个空格（SSE 规范）
	if len(value) > 0 && value[0] == ' ' {
		value = value[1:]
	}
	return field, value, true
}

// StreamEventFromRaw 将原始 SSE 事件解析为类型化的 StreamEvent。
func StreamEventFromRaw(raw RawSSEEvent) (types.StreamEvent, error) {
	eventType := raw.Event
	if eventType == "" {
		// 如果没有 event 字段，从 data 中提取 type
		var peek struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(raw.Data), &peek); err != nil {
			return nil, fmt.Errorf("解析 SSE 事件类型失败: %w", err)
		}
		eventType = peek.Type
	}

	switch eventType {
	case "message_start":
		var evt struct {
			Type    string `json:"type"`
			Message struct {
				ID    string `json:"id"`
				Type  string `json:"type"`
				Role  string `json:"role"`
				Model string `json:"model"`
				Usage json.RawMessage `json:"usage"`
			} `json:"message"`
		}
		if err := json.Unmarshal([]byte(raw.Data), &evt); err != nil {
			return nil, fmt.Errorf("解析 message_start 失败: %w", err)
		}
		return &types.MessageStartEvent{
			ID:    evt.Message.ID,
			Type:  evt.Message.Type,
			Role:  evt.Message.Role,
			Model: evt.Message.Model,
			Usage: evt.Message.Usage,
		}, nil

	case "content_block_start":
		var evt struct {
			Index        int    `json:"index"`
			ContentBlock struct {
				Type  string          `json:"type"`
				ID    string          `json:"id,omitempty"`
				Name  string          `json:"name,omitempty"`
				Text  string          `json:"text,omitempty"`
			} `json:"content_block"`
		}
		if err := json.Unmarshal([]byte(raw.Data), &evt); err != nil {
			return nil, fmt.Errorf("解析 content_block_start 失败: %w", err)
		}
		return &types.ContentBlockStartEvent{
			Index: evt.Index,
			Type:  evt.ContentBlock.Type,
			ID:    evt.ContentBlock.ID,
			Name:  evt.ContentBlock.Name,
			Text:  evt.ContentBlock.Text,
		}, nil

	case "content_block_delta":
		var evt struct {
			Index int `json:"index"`
			Delta struct {
				Type        string `json:"type"`
				Text        string `json:"text,omitempty"`
				PartialJSON string `json:"partial_json,omitempty"`
				Thinking    string `json:"thinking,omitempty"`
				Signature   string `json:"signature,omitempty"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(raw.Data), &evt); err != nil {
			return nil, fmt.Errorf("解析 content_block_delta 失败: %w", err)
		}
		return &types.ContentBlockDeltaEvent{
			Index:       evt.Index,
			DeltaType:   evt.Delta.Type,
			Text:        evt.Delta.Text,
			PartialJSON: evt.Delta.PartialJSON,
		}, nil

	case "content_block_stop":
		var evt struct {
			Index int `json:"index"`
		}
		if err := json.Unmarshal([]byte(raw.Data), &evt); err != nil {
			return nil, fmt.Errorf("解析 content_block_stop 失败: %w", err)
		}
		return &types.ContentBlockStopEvent{Index: evt.Index}, nil

	case "message_delta":
		var evt struct {
			Type  string `json:"type"`
			Delta struct {
				StopReason string `json:"stop_reason,omitempty"`
			} `json:"delta"`
			Usage struct {
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(raw.Data), &evt); err != nil {
			return nil, fmt.Errorf("解析 message_delta 失败: %w", err)
		}
		usage, _ := json.Marshal(evt.Usage)
		return &types.MessageDeltaEvent{
			StopReason: evt.Delta.StopReason,
			Usage:      usage,
		}, nil

	case "message_stop":
		return &types.MessageStopEvent{}, nil

	case "ping":
		return &types.PingEvent{}, nil

	case "error":
		var evt types.ErrorEvent
		if err := json.Unmarshal([]byte(raw.Data), &evt); err != nil {
			return nil, fmt.Errorf("解析 error 事件失败: %w", err)
		}
		return &evt, nil

	default:
		// 未知事件类型，忽略
		return nil, nil
	}
}
