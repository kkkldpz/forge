package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func init() {
	rootCmd.AddCommand(genManPageCmd)
	genManPageCmd.Flags().StringP("dir", "d", "man/man1", "输出目录")
}

var genManPageCmd = &cobra.Command{
	Use:   "gen-man-page [dir]",
	Short: "生成 Man Page 文档",
	Long: `为 Forge 生成 man page 文档。

默认输出到 man/man1 目录。

示例:
  forge gen-man-page
  forge gen-man-page --dir /usr/local/share/man/man1
`,
	Args: cobra.RangeArgs(0, 1),
	RunE: runGenManPage,
}

func runGenManPage(cmd *cobra.Command, args []string) error {
	dir := "man/man1"

	if len(args) == 1 {
		dir = args[0]
	} else {
		if d, err := cmd.Flags().GetString("dir"); err == nil && d != "" {
			dir = d
		}
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	fmt.Printf("正在生成 man page 到 %s...\n", dir)

	header := &doc.GenManHeader{
		Title:   "FORGE",
		Section: "1",
		Source:  "Forge",
		Manual:  "Forge Manual",
	}

	if err := doc.GenManTree(rootCmd, header, dir); err != nil {
		return fmt.Errorf("生成 man page 失败: %w", err)
	}

	fmt.Printf("✓ Man page 已生成到 %s\n", dir)
	fmt.Println("\n可以使用以下命令查看:")
	fmt.Printf("  man -M %s forge\n", dir)
	fmt.Printf("  或者将文件复制到系统 man 目录:\n")
	fmt.Printf("  cp %s/*.1 /usr/share/man/man1/\n", dir)

	return nil
}

func GenManPages(dir string) error {
	header := &doc.GenManHeader{
		Title:   "FORGE",
		Section: "1",
		Source:  "Forge",
		Manual:  "Forge Manual",
	}

	return doc.GenManTree(rootCmd, header, dir)
}

func GetManPagePath(name string) string {
	return filepath.Join("man/man1", name+".1")
}