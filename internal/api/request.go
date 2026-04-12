package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/kkkldpz/forge/internal/auth"
)

// RequestBuilder 组装 Anthropic Messages API 的 HTTP 请求。
type RequestBuilder struct {
	APIKey  string
	BaseURL string
	Model   string
}

// NewRequestBuilder 创建请求构建器。
func NewRequestBuilder(apiKey, baseURL, model string) *RequestBuilder {
	return &RequestBuilder{
		APIKey:  apiKey,
		BaseURL: strings.TrimRight(baseURL, "/"),
		Model:   model,
	}
}

// MessagesRequest 是发送到 /v1/messages 的完整请求体。
type MessagesRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []MessageParam  `json:"messages"`
	System    []SystemBlock   `json:"system,omitempty"`
	Tools     []ToolParam     `json:"tools,omitempty"`
	Stream    bool            `json:"stream"`
	Thinking  *ThinkingConfig `json:"thinking,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	ToolChoice any            `json:"tool_choice,omitempty"`
	Metadata  any             `json:"metadata,omitempty"`
}

// BuildStreamRequest 构建流式 Messages API 请求。
func (rb *RequestBuilder) BuildStreamRequest(
	ctx context.Context,
	messages []MessageParam,
	system []SystemBlock,
	tools []ToolParam,
	opts *RequestOptions,
) (*http.Request, error) {
	reqBody := MessagesRequest{
		Model:     rb.modelOrDefault(opts),
		MaxTokens: rb.maxTokensOrDefault(opts),
		Messages:  messages,
		System:    system,
		Tools:     tools,
		Stream:    true,
		Thinking:  rb.thinkingConfig(opts),
	}

	if opts != nil {
		reqBody.Temperature = &opts.Temperature
		reqBody.ToolChoice = opts.ToolChoice
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	url := rb.BaseURL + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Anthropic-Version", "2023-06-01")
	req.Header.Set("x-api-key", rb.APIKey)

	// 设置 beta headers
	betas := rb.betas(opts)
	if len(betas) > 0 {
		req.Header.Set("Anthropic-Beta", strings.Join(betas, ","))
	}

	return req, nil
}

// BuildNonStreamRequest 构建非流式 Messages API 请求（用于降级）。
func (rb *RequestBuilder) BuildNonStreamRequest(
	ctx context.Context,
	messages []MessageParam,
	system []SystemBlock,
	tools []ToolParam,
	opts *RequestOptions,
) (*http.Request, error) {
	reqBody := MessagesRequest{
		Model:     rb.modelOrDefault(opts),
		MaxTokens: rb.maxTokensOrDefault(opts),
		Messages:  messages,
		System:    system,
		Tools:     tools,
		Stream:    false,
		Thinking:  rb.thinkingConfig(opts),
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	url := rb.BaseURL + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Anthropic-Version", "2023-06-01")
	req.Header.Set("x-api-key", rb.APIKey)

	betas := rb.betas(opts)
	if len(betas) > 0 {
		req.Header.Set("Anthropic-Beta", strings.Join(betas, ","))
	}

	return req, nil
}

// ReadNonStreamResponse 读取非流式响应。
func ReadNonStreamResponse(resp *http.Response) (*APIResponse, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误 %d: %s", resp.StatusCode, string(body))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应 JSON 失败: %w", err)
	}
	return &apiResp, nil
}

// 辅助方法

func (rb *RequestBuilder) modelOrDefault(opts *RequestOptions) string {
	if opts != nil && opts.Model != "" {
		return opts.Model
	}
	return rb.Model
}

func (rb *RequestBuilder) maxTokensOrDefault(opts *RequestOptions) int {
	if opts != nil && opts.MaxTokens > 0 {
		return opts.MaxTokens
	}
	return 16384
}

func (rb *RequestBuilder) thinkingConfig(opts *RequestOptions) *ThinkingConfig {
	if opts != nil && opts.Thinking != nil {
		return opts.Thinking
	}
	return ThinkingDisabled()
}

func (rb *RequestBuilder) betas(opts *RequestOptions) []string {
	if opts == nil {
		return nil
	}
	return opts.Betas
}

// ResolveModel 从配置和环境变量解析要使用的模型名称。
func ResolveModel(defaultModel string) string {
	if m := os.Getenv("ANTHROPIC_MODEL"); m != "" {
		return m
	}
	if m := os.Getenv("CLAUDE_MODEL"); m != "" {
		return m
	}
	if defaultModel != "" {
		return defaultModel
	}
	return "claude-sonnet-4-20250514"
}

// ResolveAuth 从环境变量和配置解析认证信息。
func ResolveAuth() (apiKey, baseURL string, err error) {
	kr := auth.GetAPIKey()
	if err := auth.ValidateAPIKey(kr.Key); err != nil {
		return "", "", err
	}
	return kr.Key, auth.GetBaseURL(), nil
}
