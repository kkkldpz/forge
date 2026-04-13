package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

type WebSearchTool struct {
	tool.BaseTool
}

func NewWebSearchTool() *WebSearchTool {
	return &WebSearchTool{
		BaseTool: tool.BaseTool{
			NameStr:        "websearch",
			DescriptionStr: "搜索网页内容",
		},
	}
}

type WebSearchInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

func (t *WebSearchTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"query": {Type: "string", Description: "搜索查询"},
			"limit": {Type: "number", Description: "返回结果数量限制（默认 5）"},
		},
		Required: []string{"query"},
	}
}

func (t *WebSearchTool) Call(ctx context.Context, input json.RawMessage, tuc tool.ToolUseContext) types.ToolResult {
	var args WebSearchInput
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	if args.Query == "" {
		return types.ToolResult{Content: "搜索查询不能为空", IsError: true}
	}

	limit := args.Limit
	if limit <= 0 || limit > 20 {
		limit = 5
	}

	results, err := t.search(ctx, args.Query, limit)
	if err != nil {
		return types.ToolResult{Content: fmt.Sprintf("搜索失败: %v", err), IsError: true}
	}

	if len(results) == 0 {
		return types.ToolResult{Content: fmt.Sprintf("未找到与 '%s' 相关的结果", args.Query)}
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("搜索结果 (%s):\n", args.Query))

	for i, r := range results {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, r.Title))
		lines = append(lines, fmt.Sprintf("   %s", r.URL))
		if r.Snippet != "" {
			lines = append(lines, fmt.Sprintf("   %s", r.Snippet))
		}
		lines = append(lines, "")
	}

	return types.ToolResult{Content: strings.Join(lines, "\n")}
}

type SearchResult struct {
	Title   string
	URL     string
	Snippet string
}

func (t *WebSearchTool) search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	_ = ctx

	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	return []SearchResult{
		{
			Title:   fmt.Sprintf("关于 '%s' 的搜索结果", query),
			URL:     searchURL,
			Snippet: "这是一个模拟搜索结果。在实际实现中，可以使用 DuckDuckGo HTML 或其他搜索引擎 API。",
		},
	}, nil
}

func (t *WebSearchTool) IsReadOnly(input json.RawMessage) bool  { return true }
func (t *WebSearchTool) IsConcurrencySafe(input json.RawMessage) bool { return true }

var _ = time.Sleep