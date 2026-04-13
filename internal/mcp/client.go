package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/kkkldpz/forge/internal/mcp/transport"
)

// Client 是 MCP 客户端，管理与单个 MCP 服务器的通信。
type Client struct {
	serverName string
	transport  transport.Transport

	mu    sync.Mutex
	idSeq atomic.Int64
	tools []ToolDef

	initialized bool
	cancelFn    context.CancelFunc
}

// Connect 建立与 MCP 服务器的连接并完成握手。
// 发送 initialize 请求，等待服务器返回能力声明。
func (c *Client) Connect(ctx context.Context, cfg ServerConfig) error {
	ctx, c.cancelFn = context.WithCancel(ctx)

	// 根据类型创建传输层
	tp, err := createTransport(ctx, cfg)
	if err != nil {
		return fmt.Errorf("创建传输失败: %w", err)
	}
	c.transport = tp

	// 发送 initialize 请求
	initResult, err := c.initialize(ctx)
	if err != nil {
		c.transport.Close()
		return fmt.Errorf("初始化握手失败: %w", err)
	}

	// 记录服务器信息
	c.serverName = initResult.ServerInfo.Name
	fmt.Fprintf(os.Stderr, "[mcp] 已连接服务器: %s (协议版本 %s)\n",
		initResult.ServerInfo.Name, initResult.ProtocolVersion)

	// 发送 initialized 通知（JSON-RPC notification，无 id）
	notify := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	notifyBytes, _ := json.Marshal(notify)
	if err := c.transport.Send(notifyBytes); err != nil {
		c.transport.Close()
		return fmt.Errorf("发送 initialized 通知失败: %w", err)
	}

	c.initialized = true
	return nil
}

// ListTools 请求服务器返回所有可用工具列表。
func (c *Client) ListTools(ctx context.Context) ([]ToolDef, error) {
	if !c.initialized {
		return nil, fmt.Errorf("客户端未初始化")
	}

	result := listToolsResult{}
	err := c.sendRequest(ctx, "tools/list", nil, &result)
	if err != nil {
		return nil, fmt.Errorf("获取工具列表失败: %w", err)
	}

	c.mu.Lock()
	c.tools = result.Tools
	c.mu.Unlock()

	return result.Tools, nil
}

// ListResources 请求服务器返回所有可用资源列表。
func (c *Client) ListResources(ctx context.Context) ([]ResourceDef, error) {
	if !c.initialized {
		return nil, fmt.Errorf("客户端未初始化")
	}

	result := listResourcesResult{}
	err := c.sendRequest(ctx, "resources/list", nil, &result)
	if err != nil {
		return nil, fmt.Errorf("获取资源列表失败: %w", err)
	}

	return result.Resources, nil
}

// CallTool 调用服务器上的指定工具。
func (c *Client) CallTool(ctx context.Context, name string, args json.RawMessage) (*CallToolResult, error) {
	if !c.initialized {
		return nil, fmt.Errorf("客户端未初始化")
	}

	params := callToolParams{
		Name:      name,
		Arguments: args,
	}

	result := CallToolResult{}
	err := c.sendRequest(ctx, "tools/call", params, &result)
	if err != nil {
		return nil, fmt.Errorf("调用工具 %s 失败: %w", name, err)
	}

	return &result, nil
}

// ServerName 返回已连接的服务器名称。
func (c *Client) ServerName() string {
	return c.serverName
}

// Tools 返回已缓存的工具列表。
func (c *Client) Tools() []ToolDef {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]ToolDef, len(c.tools))
	copy(result, c.tools)
	return result
}

// Close 关闭与 MCP 服务器的连接。
func (c *Client) Close() error {
	if c.cancelFn != nil {
		c.cancelFn()
	}
	if c.transport != nil {
		return c.transport.Close()
	}
	return nil
}

// initialize 发送 initialize 请求完成握手。
func (c *Client) initialize(ctx context.Context) (*initializeResult, error) {
	params := initializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo: clientInfo{
			Name:    "forge",
			Version: "0.1.0",
		},
	}

	result := initializeResult{}
	err := c.sendRequest(ctx, "initialize", params, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// sendRequest 发送 JSON-RPC 请求并等待响应。
func (c *Client) sendRequest(ctx context.Context, method string, params any, result any) error {
	id := c.idSeq.Add(1)

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	if err := c.transport.Send(reqBytes); err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}

	// 等待匹配 ID 的响应
	resp, err := c.waitForResponse(ctx, id)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("服务器错误 (code %d): %s", resp.Error.Code, resp.Error.Message)
	}

	if resp.Result == nil {
		return fmt.Errorf("服务器返回空结果")
	}

	if result != nil {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("解析响应失败: %w", err)
		}
	}

	return nil
}

// waitForResponse 等待指定 ID 的 JSON-RPC 响应。
func (c *Client) waitForResponse(ctx context.Context, id int64) (*jsonRPCResponse, error) {
	recvCh := c.transport.Receive()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("等待响应超时: %w", ctx.Err())
		case data, ok := <-recvCh:
			if !ok {
				return nil, fmt.Errorf("传输通道已关闭")
			}

			var resp jsonRPCResponse
			if err := json.Unmarshal(data, &resp); err != nil {
				// 跳过无法解析的消息
				continue
			}

			if resp.ID == id {
				return &resp, nil
			}
			// 忽略非匹配 ID 的响应（可能是通知或其他请求的响应）
		}
	}
}

// createTransport 根据配置创建对应的传输实例。
func createTransport(ctx context.Context, cfg ServerConfig) (transport.Transport, error) {
	transportType := cfg.Type
	if transportType == "" {
		// 默认使用 stdio 模式
		if cfg.Command != "" {
			transportType = "stdio"
		} else if cfg.URL != "" {
			transportType = "http"
		} else {
			return nil, fmt.Errorf("未指定传输类型且无法推断")
		}
	}

	switch transportType {
	case "stdio":
		return transport.NewStdioTransport(ctx, cfg.Command, cfg.Args, cfg.Env)
	case "sse":
		if cfg.URL == "" {
			return nil, fmt.Errorf("SSE 模式需要指定 url")
		}
		return transport.NewSSETransport(ctx, cfg.URL, cfg.Headers), nil
	case "http":
		if cfg.URL == "" {
			return nil, fmt.Errorf("HTTP 模式需要指定 url")
		}
		return transport.NewHTTPTransport(cfg.URL, cfg.Headers), nil
	default:
		return nil, fmt.Errorf("不支持的传输类型: %s", transportType)
	}
}
