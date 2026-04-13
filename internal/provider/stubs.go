package provider

import (
	"context"
	"os"

	"github.com/kkkldpz/forge/internal/api"
	"github.com/kkkldpz/forge/internal/types"
)

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// NewBedrockProvider 创建 AWS Bedrock Provider。
func NewBedrockProvider(model string) (*BedrockProvider, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	return &BedrockProvider{region: region, model: model}, nil
}

// BedrockProvider AWS Bedrock Anthropic Provider。
type BedrockProvider struct {
	region string
	model  string
}

func (p *BedrockProvider) Name() string { return "bedrock" }

func (p *BedrockProvider) QueryModel(
	_ context.Context, _ []api.MessageParam, _ []api.SystemBlock,
	_ []api.ToolParam, _ *api.RequestOptions,
) <-chan types.StreamEvent {
	ch := make(chan types.StreamEvent, 64)
	go func() {
		defer close(ch)
		ch <- &types.ErrorEvent{Error: types.ErrorDetail{
			Type: "not_implemented", Message: "Bedrock Provider 尚未实现 SigV4 签名",
		}}
	}()
	return ch
}

// NewFoundryProvider 创建 Azure Foundry Provider。
func NewFoundryProvider(apiKey, baseURL, model string) (*FoundryProvider, error) {
	if baseURL == "" {
		baseURL = os.Getenv("ANTHROPIC_BASE_URL")
	}
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_AUTH_TOKEN")
	}
	return &FoundryProvider{
		client: api.NewClient(apiKey, baseURL, model),
	}, nil
}

// FoundryProvider Azure Foundry Anthropic Provider。
type FoundryProvider struct {
	client *api.Client
}

func (p *FoundryProvider) Name() string { return "foundry" }

func (p *FoundryProvider) QueryModel(
	ctx context.Context, messages []api.MessageParam, system []api.SystemBlock,
	tools []api.ToolParam, opts *api.RequestOptions,
) <-chan types.StreamEvent {
	return p.client.QueryModel(ctx, messages, system, tools, opts)
}
