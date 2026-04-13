// Package types 定义核心数据类型，包括消息、权限、流事件和工具相关类型。
package types

// SessionID 唯一标识一个 Forge 会话。
type SessionID string

// NewSessionID 从字符串创建 SessionID。
func NewSessionID(s string) SessionID { return SessionID(s) }

// String 返回字符串表示。
func (s SessionID) String() string { return string(s) }

// AgentID 唯一标识一个子代理。
type AgentID string

// NewAgentID 从字符串创建 AgentID。
func NewAgentID(s string) AgentID { return AgentID(s) }

// String 返回字符串表示。
func (a AgentID) String() string { return string(a) }
