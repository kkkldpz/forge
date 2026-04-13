// Package mcp 实现 Model Context Protocol (MCP) 客户端。
// 用于与 MCP 服务器通信，发现和调用外部工具。
package mcp

import "encoding/json"

// ServerConfig 定义 MCP 服务器的连接配置。
type ServerConfig struct {
	// Command 可执行文件路径（stdio 模式）
	Command string `json:"command"`
	// Args 传递给可执行文件的参数（stdio 模式）
	Args []string `json:"args,omitempty"`
	// URL 服务器地址（SSE/HTTP 模式）
	URL string `json:"url,omitempty"`
	// Type 传输类型: "stdio", "sse", "http"
	Type string `json:"type,omitempty"`
	// Env 环境变量
	Env map[string]string `json:"env,omitempty"`
	// Headers HTTP 请求头（SSE/HTTP 模式）
	Headers map[string]string `json:"headers,omitempty"`
}

// ToolDef 描述 MCP 服务器暴露的工具。
type ToolDef struct {
	// Name 工具名称
	Name string `json:"name"`
	// Description 工具描述
	Description string `json:"description"`
	// InputSchema 工具输入的 JSON Schema
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ResourceDef 描述 MCP 服务器暴露的资源。
type ResourceDef struct {
	// URI 资源标识符
	URI string `json:"uri"`
	// Name 资源名称
	Name string `json:"name"`
	// Description 资源描述
	Description string `json:"description,omitempty"`
	// MimeType 资源 MIME 类型
	MimeType string `json:"mimeType,omitempty"`
}

// CallToolResult 是工具调用的返回结果。
type CallToolResult struct {
	// Content 结果内容列表
	Content []ContentItem `json:"content"`
	// IsError 是否执行出错
	IsError bool `json:"isError"`
}

// ContentItem 表示工具结果中的单个内容块。
type ContentItem struct {
	// Type 内容类型: "text", "image", "resource"
	Type string `json:"type"`
	// Text 文本内容
	Text string `json:"text,omitempty"`
	// Data 二进制或编码数据
	Data string `json:"data,omitempty"`
}

// jsonRPCRequest 是 JSON-RPC 2.0 请求消息。
type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// jsonRPCResponse 是 JSON-RPC 2.0 响应消息。
type jsonRPCResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      int64            `json:"id"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *jsonRPCError    `json:"error,omitempty"`
}

// jsonRPCError 是 JSON-RPC 2.0 错误对象。
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// initializeParams 是 initialize 请求的参数。
type initializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    clientCaps     `json:"capabilities"`
	ClientInfo      clientInfo     `json:"clientInfo"`
}

type clientCaps struct {
	Roots struct {
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"roots,omitempty"`
}

type clientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// initializeResult 是 initialize 响应的结果。
type initializeResult struct {
	ProtocolVersion string      `json:"protocolVersion"`
	Capabilities    serverCaps  `json:"capabilities"`
	ServerInfo      serverInfo  `json:"serverInfo"`
}

type serverCaps struct {
	Tools   *toolsCap `json:"tools,omitempty"`
	Resources *resourcesCap `json:"resources,omitempty"`
}

type toolsCap struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type resourcesCap struct {
	Subscribe bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// listToolsResult 是 tools/list 响应的结果。
type listToolsResult struct {
	Tools []ToolDef `json:"tools"`
}

// callToolParams 是 tools/call 请求的参数。
type callToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// listResourcesResult 是 resources/list 响应的结果。
type listResourcesResult struct {
	Resources []ResourceDef `json:"resources"`
}
