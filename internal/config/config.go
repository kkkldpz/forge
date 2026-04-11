package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"
)

// GlobalConfig 用户级配置，来自 ~/.forge/settings.json。
type GlobalConfig struct {
	APIKey      string           `json:"apiKey,omitempty"`
	BaseURL     string           `json:"baseUrl,omitempty"`
	DefaultModel string          `json:"defaultModel,omitempty"`
	HaikuModel  string           `json:"haikuModel,omitempty"`
	SonnetModel string           `json:"sonnetModel,omitempty"`
	OpusModel   string           `json:"opusModel,omitempty"`
	Theme       string           `json:"theme,omitempty"`
	Permissions PermissionConfig `json:"permissions,omitempty"`
	MCPServers  map[string]any   `json:"mcpServers,omitempty"`
	Plugins     map[string]any   `json:"plugins,omitempty"`
}

// ProjectConfig 项目级配置，来自 .forge/settings.json。
type ProjectConfig struct {
	AllowedTools []string       `json:"allowedTools,omitempty"`
	DeniedTools  []string       `json:"deniedTools,omitempty"`
	MCPServers   map[string]any `json:"mcpServers,omitempty"`
	AppendPrompt string         `json:"appendSystemPrompt,omitempty"`
	CustomPrompt string         `json:"customSystemPrompt,omitempty"`
}

// PermissionConfig 权限相关配置。
type PermissionConfig struct {
	Mode           string   `json:"mode,omitempty"`
	AllowedTools   []string `json:"allowedTools,omitempty"`
	DeniedTools    []string `json:"deniedTools,omitempty"`
	AdditionalDirs []string `json:"additionalDirectories,omitempty"`
}

// Config 合并后的配置（全局 + 项目 + 环境变量覆盖）。
type Config struct {
	Global  *GlobalConfig
	Project *ProjectConfig

	mu sync.RWMutex
}

// Loader 分层配置加载器。
type Loader struct {
	homeDir    string
	workingDir string
}

// NewLoader 为指定目录创建配置加载器。
func NewLoader(homeDir, workingDir string) *Loader {
	return &Loader{homeDir: homeDir, workingDir: workingDir}
}

// Load 读取并合并所有层级的配置。
func (l *Loader) Load() (*Config, error) {
	cfg := &Config{}

	// 第一层：全局配置 (~/.forge/settings.json)
	cfg.Global = l.loadGlobalConfig()

	// 第二层：项目配置 (.forge/settings.json)
	cfg.Project = l.loadProjectConfig()

	// 第三层：环境变量覆盖
	l.applyEnvOverrides(cfg)

	return cfg, nil
}

func (l *Loader) loadGlobalConfig() *GlobalConfig {
	gc := &GlobalConfig{}
	path := filepath.Join(l.homeDir, ".forge", "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return gc
	}
	_ = json.Unmarshal(data, gc)
	return gc
}

func (l *Loader) loadProjectConfig() *ProjectConfig {
	pc := &ProjectConfig{}
	path := filepath.Join(l.workingDir, ".forge", "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return pc
	}
	_ = json.Unmarshal(data, pc)
	return pc
}

// applyEnvOverrides 用环境变量覆盖配置值。
func (l *Loader) applyEnvOverrides(cfg *Config) {
	if v := viper.GetString("api_key"); v != "" {
		cfg.Global.APIKey = v
	}
	if v := viper.GetString("base_url"); v != "" {
		cfg.Global.BaseURL = v
	}
	if v := viper.GetString("model"); v != "" {
		cfg.Global.DefaultModel = v
	}
	// Anthropic 专用环境变量
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		cfg.Global.APIKey = v
	}
	if v := os.Getenv("ANTHROPIC_BASE_URL"); v != "" {
		cfg.Global.BaseURL = v
	}
	// OpenAI 兼容模式环境变量
	if v := os.Getenv("CLAUDE_CODE_USE_OPENAI"); v == "1" {
		if v := os.Getenv("OPENAI_API_KEY"); v != "" {
			cfg.Global.APIKey = v
		}
		if v := os.Getenv("OPENAI_BASE_URL"); v != "" {
			cfg.Global.BaseURL = v
		}
	}
}
