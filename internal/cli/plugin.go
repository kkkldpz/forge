package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kkkldpz/forge/internal/plugin"
)

func init() {
	rootCmd.AddCommand(pluginCmd)
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginLoadCmd)
	pluginCmd.AddCommand(pluginUnloadCmd)
	pluginCmd.AddCommand(pluginSearchCmd)
}

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "插件管理",
	Long:  "管理 Forge 插件",
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出已加载的插件",
	RunE:  runPluginList,
}

func runPluginList(cmd *cobra.Command, args []string) error {
	m := plugin.GlobalManager()
	plugins := m.List()

	if len(plugins) == 0 {
		fmt.Println("没有已加载的插件")
	} else {
		fmt.Printf("已加载的插件 (%d):\n", len(plugins))
		for _, name := range plugins {
			fmt.Printf("  - %s\n", name)
		}
	}

	fmt.Println("\n使用 'forge plugin search' 搜索可用插件")
	return nil
}

var pluginLoadCmd = &cobra.Command{
	Use:   "load <name>",
	Short: "加载插件",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginLoad,
}

func runPluginLoad(cmd *cobra.Command, args []string) error {
	name := args[0]
	m := plugin.GlobalManager()

	homeDir, _ := os.UserHomeDir()
	pluginPath := fmt.Sprintf("%s/.forge/plugins", homeDir)
	m.AddSearchPath(pluginPath)

	if err := m.Load(name); err != nil {
		return fmt.Errorf("加载插件失败: %w", err)
	}

	fmt.Printf("✓ 插件 %s 已加载\n", name)
	return nil
}

var pluginUnloadCmd = &cobra.Command{
	Use:   "unload <name>",
	Short: "卸载插件",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginUnload,
}

func runPluginUnload(cmd *cobra.Command, args []string) error {
	name := args[0]
	m := plugin.GlobalManager()

	if err := m.Unload(name); err != nil {
		return fmt.Errorf("卸载插件失败: %w", err)
	}

	fmt.Printf("✓ 插件 %s 已卸载\n", name)
	return nil
}

var pluginSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "搜索可用插件",
	RunE:  runPluginSearch,
}

func runPluginSearch(cmd *cobra.Command, args []string) error {
	fmt.Println("搜索插件...")
	fmt.Println("\n提示: Forge 插件系统支持 Go 插件 (.so 文件)")
	fmt.Println("插件搜索需要配置插件仓库 URL")

	fmt.Println("\n可用插件:")
	fmt.Println("  (暂无可用插件)")

	return nil
}