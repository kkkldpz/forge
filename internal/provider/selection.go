package provider

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/kkkldpz/forge/internal/api"
	"github.com/kkkldpz/forge/internal/config"
	"github.com/kkkldpz/forge/internal/provider/gemini"
	"github.com/kkkldpz/forge/internal/provider/grok"
	"github.com/kkkldpz/forge/internal/provider/openai"
)

// GetProvider 根据配置创建对应的 Provider 实例。
// 选择优先级：config.ModelType > 环境变量 > 默认 anthropic。
func GetProvider(cfg *config.Config) (Provider, error) {
	modelType := cfg.Global.ModelType
	if modelType == "" {
		modelType = detectModelType()
	}

	apiKey := cfg.Global.APIKey
	baseURL := cfg.Global.BaseURL
	model := cfg.Global.DefaultModel
	if model == "" {
		model = "claude-sonnet-4-6"
	}

	slog.Info("选择 Provider", "type", modelType, "model", model)

	switch modelType {
	case "openai":
		return newOpenAIProvider(apiKey, baseURL, model)
	case "gemini":
		return newGeminiProvider(apiKey, model)
	case "grok":
		return newGrokProvider(apiKey, model)
	case "bedrock":
		return newBedrockProvider(model)
	case "vertex":
		return newVertexProvider(model)
	case "foundry":
		return newFoundryProvider(apiKey, baseURL, model)
	case "anthropic", "":
		client := api.NewClient(apiKey, baseURL, model)
		return NewFirstPartyProvider(client), nil
	default:
		return nil, fmt.Errorf("未知的 Provider 类型: %s", modelType)
	}
}

// detectModelType 从环境变量推断 Provider 类型。
func detectModelType() string {
	envMap := map[string]string{
		"CLAUDE_CODE_USE_OPENAI":  "openai",
		"CLAUDE_CODE_USE_GEMINI":  "gemini",
		"CLAUDE_CODE_USE_GROK":    "grok",
		"CLAUDE_CODE_USE_BEDROCK": "bedrock",
		"CLAUDE_CODE_USE_VERTEX":  "vertex",
		"CLAUDE_CODE_USE_FOUNDRY": "foundry",
	}
	for env, pType := range envMap {
		if os.Getenv(env) == "1" {
			return pType
		}
	}
	return "anthropic"
}

// --- 占位构造函数，后续阶段实现 ---

func newOpenAIProvider(apiKey, baseURL, model string) (Provider, error) {
	return openai.NewOpenAIProvider(apiKey, baseURL, model)
}

func newGeminiProvider(apiKey, model string) (Provider, error) {
	return gemini.NewGeminiProvider(apiKey, model)
}

func newGrokProvider(apiKey, model string) (Provider, error) {
	return grok.NewGrokProvider(apiKey, model)
}

func newBedrockProvider(model string) (Provider, error) {
	return NewBedrockProvider(model)
}

func newVertexProvider(model string) (Provider, error) {
	return NewVertexProvider(model)
}

func newFoundryProvider(apiKey, baseURL, model string) (Provider, error) {
	return NewFoundryProvider(apiKey, baseURL, model)
}
