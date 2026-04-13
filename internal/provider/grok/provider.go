package grok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/kkkldpz/forge/internal/api"
	"github.com/kkkldpz/forge/internal/provider/openai"
	"github.com/kkkldpz/forge/internal/types"
)

// defaultBaseURL 是 xAI Grok API 的默认地址。
const defaultBaseURL = "https://api.x.ai/v1"

// defaultTimeout 是 HTTP 请求的默认超时时间。
const defaultTimeout = 5 * time.Minute

// GrokProvider 是 xAI Grok API Provider，基于 OpenAI Chat Completions 协议。
type GrokProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewGrokProvider 创建 Grok 兼容 Provider。
func NewGrokProvider(apiKey, model string) (*GrokProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Grok API Key 未设置")
	}
	return &GrokProvider{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		model:   model,
		client:  &http.Client{Timeout: defaultTimeout},
	}, nil
}

// Name 返回 Provider 名称。
func (p *GrokProvider) Name() string {
	return "grok"
}

// QueryModel 调用 Grok Chat Completions API 并将响应流转换为 Anthropic 格式事件。
// 复用 OpenAI 的消息转换和流适配逻辑。
func (p *GrokProvider) QueryModel(
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
func (p *GrokProvider) queryModel(
	ctx context.Context,
	messages []api.MessageParam,
	system []api.SystemBlock,
	tools []api.ToolParam,
	opts *api.RequestOptions,
	ch chan<- types.StreamEvent,
) {
	// 1. 使用 OpenAI 转换器转换消息格式
	oaiMessages := openai.ConvertMessages(system, messages)
	oaiTools := openai.ConvertTools(tools)

	// 2. 确定使用的模型
	model := p.model
	if opts != nil && opts.Model != "" {
		model = opts.Model
	}

	// 3. 使用 OpenAI 构建器和校验器
	req := openai.BuildRequest(model, oaiMessages, oaiTools, opts)
	if err := openai.ValidateRequest(req); err != nil {
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

	// 5. 构建 HTTP 请求（发送到 xAI 端点）
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
	slog.Info("发送 Grok 请求", "url", url, "model", model)
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
			Message: fmt.Sprintf("Grok API 返回错误 (HTTP %d): %s", resp.StatusCode, string(errBody)),
		}}
		return
	}

	// 8. 复用 OpenAI 的流适配器
	for event := range openai.AdaptStream(resp.Body) {
		ch <- event
	}
}
