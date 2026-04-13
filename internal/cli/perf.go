package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kkkldpz/forge/internal/perf"
)

func init() {
	rootCmd.AddCommand(perfCmd)
	perfCmd.AddCommand(perfStatCmd)
	perfCmd.AddCommand(perfProfileCmd)
}

var perfCmd = &cobra.Command{
	Use:   "perf",
	Short: "性能监控和分析",
	Long:  "查看运行时性能统计和内存使用情况",
}

var perfStatCmd = &cobra.Command{
	Use:   "stat",
	Short: "查看性能统计",
	RunE:  runPerfStat,
}

func runPerfStat(cmd *cobra.Command, args []string) error {
	stats := perf.GetStats()

	fmt.Println("=== Forge 性能统计 ===")
	fmt.Printf("内存分配: %d bytes\n", stats.Alloc)
	fmt.Printf("累计分配: %d bytes\n", stats.TotalAlloc)
	fmt.Printf("系统内存: %d bytes\n", stats.Sys)
	fmt.Printf("Goroutines: %d\n", stats.NumGo)
	fmt.Printf("GC 次数: %d\n", stats.NumGC)

	return nil
}

var perfProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "生成性能报告",
	RunE:  runPerfProfile,
}

func runPerfProfile(cmd *cobra.Command, args []string) error {
	report := perf.GetProfile()
	fmt.Println(report.String())
	return nil
}