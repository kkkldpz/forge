package provider

import (
	"context"

	"github.com/kkkldpz/forge/internal/api"
	"github.com/kkkldpz/forge/internal/types"
)

// FirstPartyProvider 是 Anthropic 直连 Provider，封装已有的 api.Client。
type FirstPartyProvider struct {
	client *api.Client
}

// NewFirstPartyProvider 创建 Anthropic 直连 Provider。
func NewFirstPartyProvider(client *api.Client) *FirstPartyProvider {
	return &FirstPartyProvider{client: client}
}

// Name 返回 Provider 名称。
func (p *FirstPartyProvider) Name() string {
	return "anthropic"
}

// QueryModel 调用 Anthropic Messages API。
func (p *FirstPartyProvider) QueryModel(
	ctx context.Context,
	messages []api.MessageParam,
	system []api.SystemBlock,
	tools []api.ToolParam,
	opts *api.RequestOptions,
) <-chan types.StreamEvent {
	return p.client.QueryModel(ctx, messages, system, tools, opts)
}
