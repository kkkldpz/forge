// Package auth 处理 API 认证，包括 API Key 的获取和管理。
package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// KeySource 标识 API Key 的来源。
type KeySource string

const (
	KeySourceEnvVar     KeySource = "environment_variable"
	KeySourceConfigFile KeySource = "config_file"
	KeySourceNone       KeySource = "none"
)

// KeyResult 包含解析后的 API Key 及其来源。
type KeyResult struct {
	Key    string
	Source KeySource
}

// GetAPIKey 按优先级获取 Anthropic API Key：
// 1. ANTHROPIC_API_KEY 环境变量
// 2. CLAUDE_API_KEY 环境变量（兼容）
// 3. ~/.forge/settings.json 配置文件
func GetAPIKey() KeyResult {
	// 优先级 1：ANTHROPIC_API_KEY 环境变量
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		return KeyResult{Key: key, Source: KeySourceEnvVar}
	}

	// 优先级 2：CLAUDE_API_KEY 环境变量（兼容 CCB）
	if key := os.Getenv("CLAUDE_API_KEY"); key != "" {
		return KeyResult{Key: key, Source: KeySourceEnvVar}
	}

	// 优先级 3：配置文件
	home, err := os.UserHomeDir()
	if err == nil {
		key := loadKeyFromConfig(filepath.Join(home, ".forge", "settings.json"))
		if key != "" {
			return KeyResult{Key: key, Source: KeySourceConfigFile}
		}
	}

	return KeyResult{Key: "", Source: KeySourceNone}
}

// GetBaseURL 获取 API 基础 URL，优先使用环境变量覆盖。
func GetBaseURL() string {
	if url := os.Getenv("ANTHROPIC_BASE_URL"); url != "" {
		return url
	}
	if url := os.Getenv("CLAUDE_API_BASE_URL"); url != "" {
		return url
	}
	return "https://api.anthropic.com"
}

// loadKeyFromConfig 从 JSON 配置文件读取 API Key。
func loadKeyFromConfig(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var cfg struct {
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ""
	}
	return cfg.APIKey
}

// ValidateAPIKey 验证 API Key 格式是否合法。
func ValidateAPIKey(key string) error {
	if key == "" {
		return fmt.Errorf("API Key 未设置。请设置 ANTHROPIC_API_KEY 环境变量或在 ~/.forge/settings.json 中配置")
	}
	if len(key) < 10 {
		return fmt.Errorf("API Key 格式异常：长度过短")
	}
	return nil
}
