// Package transport 定义 MCP 传输层的抽象接口。
package transport

// Transport 是 MCP 消息传输的抽象接口。
// 所有传输实现必须支持发送和接收 JSON-RPC 消息。
type Transport interface {
	// Send 发送 JSON-RPC 消息到 MCP 服务器。
	Send(msg []byte) error

	// Receive 返回消息接收通道。
	// 调用方通过此通道读取服务器发来的消息。
	Receive() <-chan []byte

	// Close 关闭传输连接并释放资源。
	Close() error
}
