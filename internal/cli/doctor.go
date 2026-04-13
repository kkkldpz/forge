package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kkkldpz/forge/internal/auth"
	"github.com/kkkldpz/forge/internal/config"
)

func init() {
	rootCmd.AddCommand(doctorCmd)
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "检查系统环境和配置",
	Long:  "运行诊断检查，验证 Go 环境、API 密钥、工具依赖等",
	RunE:  runDoctor,
}

type DoctorCheck struct {
	Name    string
	Status  string
	Message string
}

func runDoctor(cmd *cobra.Command, args []string) error {
	var checks []DoctorCheck

	checks = append(checks, checkGoVersion())
	checks = append(checks, checkOS())
	checks = append(checks, checkAPIKey())
	checks = append(checks, checkConfig())
	checks = append(checks, checkTools())
	checks = append(checks, checkNetwork())

	fmt.Println("\n=== Forge 诊断检查 ===")

	allPassed := true
	for _, check := range checks {
		statusIcon := "✓"
		if check.Status == "FAIL" {
			statusIcon = "✗"
			allPassed = false
		} else if check.Status == "WARN" {
			statusIcon = "⚠"
		}
		fmt.Printf("[%s] %s: %s\n", statusIcon, check.Name, check.Message)
	}

	fmt.Println()
	if allPassed {
		fmt.Println("✓ 所有检查通过")
	} else {
		fmt.Println("✗ 部分检查失败，请查看上述信息")
	}

	return nil
}

func checkGoVersion() DoctorCheck {
	version := runtime.Version()
	name := "Go 版本"

	if strings.HasPrefix(version, "go1.") {
		return DoctorCheck{Name: name, Status: "OK", Message: version}
	}
	return DoctorCheck{Name: name, Status: "WARN", Message: version + " (建议使用 Go 1.21+)"}
}

func checkOS() DoctorCheck {
	return DoctorCheck{
		Name:    "操作系统",
		Status:  "OK",
		Message: runtime.GOOS + "/" + runtime.GOARCH,
	}
}

func checkAPIKey() DoctorCheck {
	name := "API 密钥"

	result := auth.GetAPIKey()
	if result.Source == auth.KeySourceNone {
		return DoctorCheck{Name: name, Status: "FAIL", Message: "未配置 API 密钥"}
	}

	return DoctorCheck{Name: name, Status: "OK", Message: "已配置"}
}

func checkConfig() DoctorCheck {
	homeDir, err := os.UserHomeDir()
	name := "配置文件"

	if err != nil {
		return DoctorCheck{Name: name, Status: "WARN", Message: "无法获取用户目录"}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return DoctorCheck{Name: name, Status: "WARN", Message: "无法获取工作目录"}
	}

	loader := config.NewLoader(homeDir, cwd)
	_, err = loader.Load()
	if err != nil {
		return DoctorCheck{Name: name, Status: "WARN", Message: "配置加载有警告: " + err.Error()}
	}

	return DoctorCheck{Name: name, Status: "OK", Message: "配置正常"}
}

func checkTools() DoctorCheck {
	name := "系统工具"

	requiredTools := []string{"git"}
	var missing []string

	for _, tool := range requiredTools {
		if _, err := exec.LookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}

	if len(missing) > 0 {
		return DoctorCheck{Name: name, Status: "WARN", Message: "缺少工具: " + strings.Join(missing, ", ")}
	}

	return DoctorCheck{Name: name, Status: "OK", Message: "必要工具已安装"}
}

func checkNetwork() DoctorCheck {
	name := "网络连接"

	cmd := exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "https://api.anthropic.com", "--max-time", "5")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return DoctorCheck{Name: name, Status: "WARN", Message: "无法连接到 Anthropic API"}
	}

	code := strings.TrimSpace(string(output))
	if code == "200" {
		return DoctorCheck{Name: name, Status: "OK", Message: "Anthropic API 可达"}
	}

	return DoctorCheck{Name: name, Status: "WARN", Message: "API 返回状态码: " + code}
}
