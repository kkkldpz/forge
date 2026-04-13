package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kkkldpz/forge/internal/api"
	"github.com/kkkldpz/forge/internal/types"
)

// defaultTimeout 是 HTTP 请求的默认超时时间。
const defaultTimeout = 5 * time.Minute

// OpenAIProvider 是 OpenAI Chat Completions API Provider。
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewOpenAIProvider 创建 OpenAI 兼容 Provider。
func NewOpenAIProvider(apiKey, baseURL, model string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API Key 未设置")
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	// 移除末尾斜杠
	baseURL = strings.TrimRight(baseURL, "/")

	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: defaultTimeout},
	}, nil
}

// Name 返回 Provider 名称。
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// QueryModel 调用 OpenAI Chat Completions API 并将响应流转换为 Anthropic 格式事件。
func (p *OpenAIProvider) QueryModel(
	ctx context.Context,
	messages []api.MessageParam,
	system []api.SystemBlock,
	tools []api.ToolParam,
	opts *api.RequestOptions,
) <-chan types.StreamEvent {
	ch := make(chan types.StreamEvent, 64)
	go func() {
		defer close(ch)
		p.queryModel(ctx, messages, system, tools, opts, ch)
	}()
	return ch
}

// queryModel 执行实际的 API 调用和流转换。
func (p *OpenAIProvider) queryModel(
	ctx context.Context,
	messages []api.MessageParam,
	system []api.SystemBlock,
	tools []api.ToolParam,
	opts *api.RequestOptions,
	ch chan<- types.StreamEvent,
) {
	// 1. 转换消息格式
	oaiMessages := ConvertMessages(system, messages)
	oaiTools := ConvertTools(tools)

	// 2. 确定使用的模型
	model := p.model
	if opts != nil && opts.Model != "" {
		model = opts.Model
	}

	// 3. 构建请求
	req := BuildRequest(model, oaiMessages, oaiTools, opts)
	if err := ValidateRequest(req); err != nil {
		ch <- &types.ErrorEvent{Error: types.ErrorDetail{
			Type:    "invalid_request",
			Message: fmt.Sprintf("请求参数校验失败: %v", err),
		}}
		return
	}

	// 4. 序列化请求体
	body, err := json.Marshal(req)
	if err != nil {
		ch <- &types.ErrorEvent{Error: types.ErrorDetail{
			Type:    "serialize_error",
			Message: fmt.Sprintf("序列化请求体失败: %v", err),
		}}
		return
	}

	// 5. 构建 HTTP 请求
	url := p.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		ch <- &types.ErrorEvent{Error: types.ErrorDetail{
			Type:    "request_error",
			Message: fmt.Sprintf("创建 HTTP 请求失败: %v", err),
		}}
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	// 6. 发送请求
	slog.Info("发送 OpenAI 请求", "url", url, "model", model)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		ch <- &types.ErrorEvent{Error: types.ErrorDetail{
			Type:    "http_error",
			Message: fmt.Sprintf("HTTP 请求失败: %v", err),
		}}
		return
	}
	defer resp.Body.Close()

	// 7. 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		ch <- &types.ErrorEvent{Error: types.ErrorDetail{
			Type:    "api_error",
			Message: fmt.Sprintf("OpenAI API 返回错误 (HTTP %d): %s", resp.StatusCode, string(errBody)),
		}}
		return
	}

	// 8. 适配流式响应
	for event := range AdaptStream(resp.Body) {
		ch <- event
	}
}
