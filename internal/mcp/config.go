package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// templateVarRegex 匹配 ${{VAR}} 格式的环境变量引用。
var templateVarRegex = regexp.MustCompile(`\$\{\{(\w+)\}\}`)

// mcpConfigFile 是配置文件的磁盘结构。
type mcpConfigFile struct {
	MCPServers map[string]ServerConfig `json:"mcpServers"`
}

// ConfigManager 管理 MCP 配置的加载和合并。
type ConfigManager struct {
	mu      sync.RWMutex
	servers map[string]ServerConfig
}

// NewConfigManager 创建新的配置管理器。
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		servers: make(map[string]ServerConfig),
	}
}

// Load 从全局和项目配置文件加载 MCP 服务器配置。
// 合并优先级: 项目配置 > 全局配置。
func (cm *ConfigManager) Load(projectDir string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	loaded := make(map[string]ServerConfig)

	// 1. 加载全局配置 ~/.forge/settings.json
	globalPath, err := globalConfigPath()
	if err != nil {
		return fmt.Errorf("获取全局配置路径失败: %w", err)
	}
	if data, err := os.ReadFile(globalPath); err == nil {
		if err := mergeConfig(loaded, data); err != nil {
			// 全局配置损坏不阻塞，仅记录警告
			fmt.Fprintf(os.Stderr, "[mcp] 警告: 全局配置解析失败: %v\n", err)
		}
	}

	// 2. 加载项目配置 .forge/settings.json
	if projectDir != "" {
		projectPath := filepath.Join(projectDir, ".forge", "settings.json")
		if data, err := os.ReadFile(projectPath); err == nil {
			if err := mergeConfig(loaded, data); err != nil {
				fmt.Fprintf(os.Stderr, "[mcp] 警告: 项目 .forge 配置解析失败: %v\n", err)
			}
		}

		// 3. 加载 .mcp.json（项目根目录）
		mcpJSONPath := filepath.Join(projectDir, ".mcp.json")
		if data, err := os.ReadFile(mcpJSONPath); err == nil {
			if err := mergeConfig(loaded, data); err != nil {
				fmt.Fprintf(os.Stderr, "[mcp] 警告: .mcp.json 配置解析失败: %v\n", err)
			}
		}
	}

	// 展开环境变量
	for name, cfg := range loaded {
		cfg = expandEnvVars(cfg)
		loaded[name] = cfg
	}

	cm.servers = loaded
	return nil
}

// Servers 返回所有已加载的服务器配置。
func (cm *ConfigManager) Servers() map[string]ServerConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	result := make(map[string]ServerConfig, len(cm.servers))
	for k, v := range cm.servers {
		result[k] = v
	}
	return result
}

// GetServer 获取指定名称的服务器配置。
func (cm *ConfigManager) GetServer(name string) (ServerConfig, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	cfg, ok := cm.servers[name]
	return cfg, ok
}

// mergeConfig 将配置文件内容合并到目标 map 中。
func mergeConfig(dst map[string]ServerConfig, data []byte) error {
	var cfg mcpConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("JSON 解析失败: %w", err)
	}
	for name, server := range cfg.MCPServers {
		dst[name] = server
	}
	return nil
}

// globalConfigPath 返回全局配置文件路径。
func globalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}
	return filepath.Join(home, ".forge", "settings.json"), nil
}

// expandEnvVars 展开 ServerConfig 中的环境变量引用。
// 支持 ${{ENV_VAR}} 和 $ENV_VAR 两种格式。
func expandEnvVars(cfg ServerConfig) ServerConfig {
	// 展开环境变量 map
	if cfg.Env != nil {
		expanded := make(map[string]string, len(cfg.Env))
		for k, v := range cfg.Env {
			expanded[k] = os.ExpandEnv(v)
		}
		cfg.Env = expanded
	}

	// 展开命令和参数中的 ${{ENV_VAR}} 引用
	cfg.Command = expandTemplateVars(cfg.Command)
	for i, arg := range cfg.Args {
		cfg.Args[i] = expandTemplateVars(arg)
	}
	cfg.URL = expandTemplateVars(cfg.URL)

	return cfg
}

// expandTemplateVars 将 ${{VAR}} 格式的环境变量引用替换为实际值。
// 同时支持标准 $VAR 格式。
func expandTemplateVars(s string) string {
	// 先将 ${{VAR}} 转换为 $VAR 格式，再由 os.ExpandEnv 统一处理
	converted := templateVarRegex.ReplaceAllStringFunc(s, func(match string) string {
		inner := strings.TrimSuffix(strings.TrimPrefix(match, "${{"), "}}")
		return "$" + inner
	})
	return os.ExpandEnv(converted)
}

// validateCommand 检查命令是否可以被执行。
func validateCommand(cfg ServerConfig) error {
	if cfg.Type == "stdio" {
		if cfg.Command == "" {
			return fmt.Errorf("stdio 模式需要指定 command")
		}
		_, err := exec.LookPath(cfg.Command)
		if err != nil {
			return fmt.Errorf("找不到可执行文件: %s: %w", cfg.Command, err)
		}
	}
	if (cfg.Type == "sse" || cfg.Type == "http") && cfg.URL == "" {
		return fmt.Errorf("%s 模式需要指定 url", cfg.Type)
	}
	return nil
}
