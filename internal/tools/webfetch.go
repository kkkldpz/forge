package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

type WebFetchTool struct {
	tool.BaseTool
	httpClient *http.Client
}

func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{
		BaseTool: tool.BaseTool{
			NameStr:        "webfetch",
			DescriptionStr: "获取网页内容",
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type WebFetchInput struct {
	URL     string `json:"url"`
	Timeout int    `json:"timeout_ms,omitempty"`
}

func (t *WebFetchTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"url": {Type: "string", Description: "要获取的网页 URL"},
		},
		Required: []string{"url"},
	}
}

func (t *WebFetchTool) Call(ctx context.Context, input json.RawMessage, tuc tool.ToolUseContext) types.ToolResult {
	var args WebFetchInput
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	if err := t.validateURL(args.URL); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("URL 验证失败: %v", err), IsError: true}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", args.URL, nil)
	if err != nil {
		return types.ToolResult{Content: fmt.Sprintf("创建请求失败: %v", err), IsError: true}
	}

	req.Header.Set("User-Agent", "Forge/1.0 (AI CLI Tool)")

	client := t.httpClient
	if args.Timeout > 0 {
		client = &http.Client{Timeout: time.Duration(args.Timeout) * time.Millisecond}
	}

	resp, err := client.Do(req)
	if err != nil {
		return types.ToolResult{Content: fmt.Sprintf("请求失败: %v", err), IsError: true}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return types.ToolResult{Content: fmt.Sprintf("HTTP 错误: %d %s", resp.StatusCode, resp.Status), IsError: true}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return types.ToolResult{Content: fmt.Sprintf("读取响应失败: %v", err), IsError: true}
	}

	content := string(body)
	if len(content) > 50000 {
		content = content[:50000] + "\n\n[内容已截断]"
	}

	return types.ToolResult{Content: content}
}

func (t *WebFetchTool) validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("无效的 URL 格式")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("仅支持 http/https 协议")
	}

	if u.Host == "" {
		return fmt.Errorf("URL 缺少主机名")
	}

	blockedDomains := []string{"localhost", "127.0.0.1", "0.0.0.0"}
	for _, domain := range blockedDomains {
		if strings.Contains(u.Host, domain) {
			return fmt.Errorf("不允许访问本地地址")
		}
	}

	return nil
}

func (t *WebFetchTool) IsReadOnly(input json.RawMessage) bool  { return true }
func (t *WebFetchTool) IsConcurrencySafe(input json.RawMessage) bool { return true }