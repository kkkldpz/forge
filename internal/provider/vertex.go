package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/kkkldpz/forge/internal/api"
	"github.com/kkkldpz/forge/internal/types"
	"golang.org/x/oauth2/google"
)

// VertexProvider Google Vertex AI Anthropic Provider。
type VertexProvider struct {
	project string
	region  string
	model   string
}

// NewVertexProvider 创建 Google Vertex AI Provider。
func NewVertexProvider(model string) (*VertexProvider, error) {
	project := getEnvOrDefault("GOOGLE_CLOUD_PROJECT", "")
	region := getEnvOrDefault("GOOGLE_CLOUD_REGION", "us-east5")
	if project == "" {
		return nil, fmt.Errorf("GOOGLE_CLOUD_PROJECT 未设置")
	}
	return &VertexProvider{
		project: project,
		region:  region,
		model:   model,
	}, nil
}

// Name 返回 Provider 名称。
func (p *VertexProvider) Name() string {
	return "vertex"
}

// QueryModel 调用 Vertex AI streamRawPredict 端点。
func (p *VertexProvider) QueryModel(
	ctx context.Context,
	messages []api.MessageParam,
	system []api.SystemBlock,
	tools []api.ToolParam,
	opts *api.RequestOptions,
) <-chan types.StreamEvent {
	ch := make(chan types.StreamEvent, 128)

	go func() {
		defer close(ch)

		// 1. 获取 Google OAuth token (ADC)
		credentials, err := google.FindDefaultCredentials(ctx)
		if err != nil {
			ch <- &types.ErrorEvent{Error: types.ErrorDetail{
				Type:    "auth_failed",
				Message: fmt.Sprintf("获取 Google 凭据失败: %v", err),
			}}
			return
		}

		token, err := credentials.TokenSource.Token()
		if err != nil {
			ch <- &types.ErrorEvent{Error: types.ErrorDetail{
				Type:    "auth_failed",
				Message: fmt.Sprintf("获取 Google access token 失败: %v", err),
			}}
			return
		}

		// 2. 构造请求体
		model := p.model
		maxTokens := 16384
		if opts != nil {
			if opts.Model != "" {
				model = opts.Model
			}
			if opts.MaxTokens > 0 {
				maxTokens = opts.MaxTokens
			}
		}

		reqBody := map[string]any{
			"anthropic_version": "vertex-2023-10-16",
			"messages":          messages,
			"system":            system,
			"max_tokens":        maxTokens,
			"tools":             tools,
			"stream":            true,
		}
		if opts != nil && opts.Thinking != nil {
			reqBody["thinking"] = opts.Thinking
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			ch <- &types.ErrorEvent{Error: types.ErrorDetail{
				Type:    "request_build_failed",
				Message: err.Error(),
			}}
			return
		}

		// 3. 构造 URL
		url := fmt.Sprintf(
			"https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/anthropic/models/%s:streamRawPredict",
			p.region, p.project, p.region, model,
		)

		// 4. 发送请求
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			ch <- &types.ErrorEvent{Error: types.ErrorDetail{
				Type:    "request_build_failed",
				Message: err.Error(),
			}}
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)

		client := &http.Client{Timeout: 10 * time.Minute}
		resp, err := client.Do(req)
		if err != nil {
			ch <- &types.ErrorEvent{Error: types.ErrorDetail{
				Type:    "connection_failed",
				Message: err.Error(),
			}}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			ch <- &types.ErrorEvent{Error: types.ErrorDetail{
				Type:    "api_error",
				Message: fmt.Sprintf("Vertex AI 返回 %d", resp.StatusCode),
			}}
			return
		}

		// 5. 解析 SSE 流（Vertex 返回的格式与 Anthropic 相同）
		for rawEvent := range api.ParseSSE(resp.Body) {
			select {
			case <-ctx.Done():
				return
			default:
			}

			event, err := api.StreamEventFromRaw(rawEvent)
			if err != nil {
				slog.Debug("解析 Vertex SSE 事件失败", "error", err)
				continue
			}
			if event == nil {
				continue
			}

			if errEvt, ok := event.(*types.ErrorEvent); ok {
				ch <- errEvt
				return
			}

			ch <- event

			if _, ok := event.(*types.MessageStopEvent); ok {
				return
			}
		}
	}()

	return ch
}
