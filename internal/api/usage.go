package api

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
)

// ModelPricing 存储各模型的每百万 token 价格（美元）。
type ModelPricing struct {
	InputPerMillion              float64
	OutputPerMillion             float64
	CacheReadPerMillion          float64
	CacheCreationPerMillion      float64
}

// 已知模型的定价表。
var modelPricing = map[string]ModelPricing{
	"claude-opus-4": {
		InputPerMillion: 15.0, OutputPerMillion: 75.0,
		CacheReadPerMillion: 1.875, CacheCreationPerMillion: 18.75,
	},
	"claude-sonnet-4": {
		InputPerMillion: 3.0, OutputPerMillion: 15.0,
		CacheReadPerMillion: 0.375, CacheCreationPerMillion: 3.75,
	},
	"claude-haiku-4.5": {
		InputPerMillion: 0.8, OutputPerMillion: 4.0,
		CacheReadPerMillion: 0.08, CacheCreationPerMillion: 1.0,
	},
}

// CalculateCost 根据 token 用量计算费用（美元）。
func CalculateCost(model string, usage *Usage) float64 {
	if usage == nil {
		return 0
	}

	pricing := getPricing(model)
	cost := 0.0

	cost += float64(usage.InputTokens) / 1_000_000 * pricing.InputPerMillion
	cost += float64(usage.OutputTokens) / 1_000_000 * pricing.OutputPerMillion
	cost += float64(usage.CacheReadInputTokens) / 1_000_000 * pricing.CacheReadPerMillion
	cost += float64(usage.CacheCreationInputTokens) / 1_000_000 * pricing.CacheCreationPerMillion

	return cost
}

// getPricing 获取模型定价，未知模型使用 sonnet 定价。
func getPricing(model string) ModelPricing {
	// 尝试精确匹配
	if p, ok := modelPricing[model]; ok {
		return p
	}
	// 尝试前缀匹配
	for prefix, p := range modelPricing {
		if len(model) >= len(prefix) && model[:len(prefix)] == prefix {
			return p
		}
	}
	// 默认使用 sonnet 定价
	return ModelPricing{
		InputPerMillion: 3.0, OutputPerMillion: 15.0,
		CacheReadPerMillion: 0.375, CacheCreationPerMillion: 3.75,
	}
}

// UpdateUsage 增量更新 token 用量。
// SSE 的 message_delta 事件只携带增量值，需要与之前的累积值合并。
func UpdateUsage(current *Usage, deltaData json.RawMessage) *Usage {
	if deltaData == nil {
		return current
	}

	var delta struct {
		OutputTokens             int `json:"output_tokens"`
		InputTokens              int `json:"input_tokens,omitempty"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	}
	if err := json.Unmarshal(deltaData, &delta); err != nil {
		return current
	}

	updated := *current
	updated.OutputTokens += delta.OutputTokens
	// input tokens 只在 message_start 中设置，不累加
	if delta.InputTokens > 0 {
		updated.InputTokens = delta.InputTokens
	}
	if delta.CacheReadInputTokens > 0 {
		updated.CacheReadInputTokens = delta.CacheReadInputTokens
	}
	if delta.CacheCreationInputTokens > 0 {
		updated.CacheCreationInputTokens = delta.CacheCreationInputTokens
	}

	return &updated
}

// RetryConfig 管理重试状态和退避策略。
type RetryConfig struct {
	MaxRetries int
	Attempt    int
}

// NewRetryConfig 创建默认重试配置。
func NewRetryConfig() *RetryConfig {
	return &RetryConfig{MaxRetries: DefaultMaxRetries}
}

// ShouldRetry 判断是否应该重试。
func (r *RetryConfig) ShouldRetry(err error) bool {
	if r.Attempt >= r.MaxRetries {
		return false
	}
	if apiErr, ok := err.(*APIError); ok {
		if !apiErr.IsRetryable() {
			return false
		}
		// 529 最多重试 3 次
		if apiErr.IsOverloaded() && r.Attempt >= Max529Retries {
			return false
		}
	}
	return true
}

// BackoffDuration 计算下次重试的等待时间（毫秒）。
// 使用指数退避 + 随机抖动。
func (r *RetryConfig) BackoffDuration() int {
	r.Attempt++
	base := float64(BaseDelayMs) * math.Pow(2, float64(r.Attempt-1))
	jitter := float64(rand.Intn(1000))
	delay := base + jitter
	return int(math.Min(delay, float64(MaxDelayMs)))
}

// FormatCost 格式化费用显示。
func FormatCost(costUSD float64) string {
	if costUSD < 0.01 {
		return fmt.Sprintf("$%.4f", costUSD)
	}
	return fmt.Sprintf("$%.2f", costUSD)
}
