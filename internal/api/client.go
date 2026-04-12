package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/kkkldpz/forge/internal/types"
)

// Client 是 Anthropic Messages API 的客户端。
type Client struct {
	httpClient *http.Client
	builder    *RequestBuilder
	logger     *slog.Logger
}

// NewClient 创建 API 客户端。
func NewClient(apiKey, baseURL, model string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Minute, // 流式请求需要较长超时
		},
		builder: NewRequestBuilder(apiKey, baseURL, model),
		logger:  slog.Default().With("component", "api"),
	}
}

// QueryModel 发送流式请求到 Anthropic Messages API。
// 返回一个只读 channel，持续产出 StreamEvent。
// channel 关闭表示流结束或出错（错误通过 QueryEventError 发送）。
func (c *Client) QueryModel(
	ctx context.Context,
	messages []MessageParam,
	system []SystemBlock,
	tools []ToolParam,
	opts *RequestOptions,
) <-chan types.StreamEvent {
	ch := make(chan types.StreamEvent, 128)

	go func() {
		defer close(ch)
		c.streamWithRetry(ctx, ch, messages, system, tools, opts)
	}()

	return ch
}

// streamWithRetry 执行流式请求，失败时按策略重试。
func (c *Client) streamWithRetry(
	ctx context.Context,
	ch chan<- types.StreamEvent,
	messages []MessageParam,
	system []SystemBlock,
	tools []ToolParam,
	opts *RequestOptions,
) {
	retry := NewRetryConfig()

	for {
		// 检查上下文是否已取消
		if ctx.Err() != nil {
			ch <- &types.ErrorEvent{
				Error: types.ErrorDetail{
					Type:    "context_cancelled",
					Message: ctx.Err().Error(),
				},
			}
			return
		}

		// 构建请求
		req, err := c.builder.BuildStreamRequest(ctx, messages, system, tools, opts)
		if err != nil {
			ch <- &types.ErrorEvent{
				Error: types.ErrorDetail{
					Type:    "request_build_failed",
					Message: err.Error(),
				},
			}
			return
		}

		// 发送请求
		resp, err := c.httpClient.Do(req)
		if err != nil {
			if retry.ShouldRetry(&APIError{StatusCode: 0, Message: err.Error()}) {
				delay := retry.BackoffDuration()
				c.logger.Warn("请求失败，准备重试",
					"attempt", retry.Attempt,
					"delay_ms", delay,
					"error", err,
				)
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(delay) * time.Millisecond):
				}
				continue
			}
			ch <- &types.ErrorEvent{
				Error: types.ErrorDetail{
					Type:    "connection_failed",
					Message: err.Error(),
				},
			}
			return
		}

		// 检查响应状态码
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			apiErr := parseAPIError(resp.StatusCode, body)

			if retry.ShouldRetry(apiErr) {
				delay := retry.BackoffDuration()
				// 如果有 Retry-After header，使用它
				if ra := resp.Header.Get("Retry-After"); ra != "" {
					if raMs, e := strconv.Atoi(ra); e == nil && raMs > 0 {
						delay = raMs * 1000
					}
				}
				c.logger.Warn("API 返回非 200，准备重试",
					"status", resp.StatusCode,
					"attempt", retry.Attempt,
					"delay_ms", delay,
				)
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(delay) * time.Millisecond):
				}
				continue
			}

			// 不可重试的错误，尝试降级到非流式
			if resp.StatusCode == 404 {
				c.logger.Info("流式请求 404，降级为非流式请求")
				c.nonStreamFallback(ctx, ch, messages, system, tools, opts)
				return
			}

			ch <- &types.ErrorEvent{Error: types.ErrorDetail{
				Type:    "api_error",
				Message: apiErr.Error(),
			}}
			return
		}

		// 成功，解析 SSE 流
		c.processStream(ctx, ch, resp)
		return
	}
}

// processStream 从 HTTP 响应中解析 SSE 事件并发送到 channel。
func (c *Client) processStream(
	ctx context.Context,
	ch chan<- types.StreamEvent,
	resp *http.Response,
) {
	defer resp.Body.Close()

	for rawEvent := range ParseSSE(resp.Body) {
		// 检查上下文取消
		select {
		case <-ctx.Done():
			return
		default:
		}

		event, err := StreamEventFromRaw(rawEvent)
		if err != nil {
			c.logger.Debug("解析 SSE 事件失败", "error", err, "event", rawEvent.Event)
			continue
		}

		// nil 表示未知事件类型，跳过
		if event == nil {
			continue
		}

		// 检查是否为错误事件
		if errEvt, ok := event.(*types.ErrorEvent); ok {
			ch <- errEvt
			return
		}

		ch <- event

		// message_stop 表示流结束
		if _, ok := event.(*types.MessageStopEvent); ok {
			return
		}
	}
}

// nonStreamFallback 降级为非流式请求。
func (c *Client) nonStreamFallback(
	ctx context.Context,
	ch chan<- types.StreamEvent,
	messages []MessageParam,
	system []SystemBlock,
	tools []ToolParam,
	opts *RequestOptions,
) {
	req, err := c.builder.BuildNonStreamRequest(ctx, messages, system, tools, opts)
	if err != nil {
		ch <- &types.ErrorEvent{Error: types.ErrorDetail{
			Type: "request_build_failed", Message: err.Error(),
		}}
		return
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		ch <- &types.ErrorEvent{Error: types.ErrorDetail{
			Type: "connection_failed", Message: err.Error(),
		}}
		return
	}

	apiResp, err := ReadNonStreamResponse(resp)
	if err != nil {
		ch <- &types.ErrorEvent{Error: types.ErrorDetail{
			Type: "api_error", Message: err.Error(),
		}}
		return
	}

	// 将非流式响应转换为流式事件序列发送
	ch <- &types.MessageStartEvent{
		ID: apiResp.ID, Type: apiResp.Type,
		Role: apiResp.Role, Model: apiResp.Model,
	}
	if apiResp.Usage != nil {
		usage, _ := json.Marshal(apiResp.Usage)
		ch <- &types.MessageStartEvent{Usage: usage}
	}
	ch <- &types.MessageStopEvent{}
}

// parseAPIError 从 HTTP 响应解析 API 错误。
func parseAPIError(statusCode int, body []byte) *APIError {
	apiErr := &APIError{
		StatusCode: statusCode,
		Message:    string(body),
	}

	// 尝试解析结构化的错误响应
	var errResp struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		apiErr.Type = errResp.Error.Type
		apiErr.Message = errResp.Error.Message
	}

	return apiErr
}
