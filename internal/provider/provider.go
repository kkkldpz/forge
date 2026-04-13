// Package provider 定义多 Provider 抽象层，统一 Anthropic/OpenAI/Gemini/Grok/Bedrock/Vertex/Foundry 的 API 调用。
package provider

import (
	"context"

	"github.com/kkkldpz/forge/internal/api"
	"github.com/kkkldpz/forge/internal/types"
)

// Provider 是所有 API Provider 的统一接口。
// 每个 Provider 将其特定的 API 协议转换为 Forge 内部的 StreamEvent channel。
type Provider interface {
	// QueryModel 向模型发送请求并返回流式事件通道。
	QueryModel(
		ctx context.Context,
		messages []api.MessageParam,
		system []api.SystemBlock,
		tools []api.ToolParam,
		opts *api.RequestOptions,
	) <-chan types.StreamEvent

	// Name 返回 Provider 名称（如 "anthropic"、"openai"、"gemini"）。
	Name() string
}
