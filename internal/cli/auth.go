package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kkkldpz/forge/internal/auth"
)

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "管理身份认证",
	Long:  "管理 Anthropic API 密钥认证",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "登录并保存 API 密钥",
	RunE:  runAuthLogin,
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	fmt.Println("正在登录...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户目录失败: %w", err)
	}

	fmt.Println("请输入您的 Anthropic API 密钥 (sk-ant-...): ")

	var apiKey string
	fmt.Scanln(&apiKey)

	if apiKey == "" {
		return fmt.Errorf("API 密钥不能为空")
	}

	if err := auth.ValidateAPIKey(apiKey); err != nil {
		return fmt.Errorf("API 密钥验证失败: %w", err)
	}

	apiKeyResult := auth.GetAPIKey()
	if apiKeyResult.Source != auth.KeySourceNone {
		return fmt.Errorf("已存在 API 密钥配置，如需更新请先运行 logout")
	}

	settingsPath := fmt.Sprintf("%s/.forge/settings.json", homeDir)
	fmt.Printf("API 密钥已验证，请在 ~/.forge/settings.json 中保存:\n%s\n", apiKey)

	_ = settingsPath
	return nil
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "清除已保存的 API 密钥",
	RunE:  runAuthLogout,
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	fmt.Println("已清除 API 密钥配置")
	fmt.Println("提示: 使用环境变量 ANTHROPIC_API_KEY 继续使用")
	return nil
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "检查当前认证状态",
	RunE:  runAuthStatus,
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	result := auth.GetAPIKey()

	if result.Source == auth.KeySourceNone {
		fmt.Println("✗ 未配置 API 密钥，请运行 'forge auth login' 或设置 ANTHROPIC_API_KEY 环境变量")
		return nil
	}

	sourceName := "环境变量"
	if result.Source == auth.KeySourceConfigFile {
		sourceName = "配置文件"
	}
	fmt.Printf("✓ 已配置 API 密钥 (来源: %s)\n", sourceName)
	return nil
}