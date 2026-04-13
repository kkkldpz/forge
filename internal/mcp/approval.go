package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// approvalFile 是审批状态的磁盘存储结构。
type approvalFile struct {
	Approved map[string]bool `json:"approved"`
	Denied   map[string]bool `json:"denied"`
}

// ApprovalTracker 管理 MCP 服务器的审批状态。
// 用于跟踪哪些服务器已被用户批准或拒绝连接。
type ApprovalTracker struct {
	mu       sync.RWMutex
	approved map[string]bool
	denied   map[string]bool
	filePath string
}

// NewApprovalTracker 创建审批跟踪器。
// 审批状态持久化到 ~/.forge/mcp_approvals.json。
func NewApprovalTracker() *ApprovalTracker {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	dir := filepath.Join(home, ".forge")
	filePath := filepath.Join(dir, "mcp_approvals.json")

	at := &ApprovalTracker{
		approved: make(map[string]bool),
		denied:   make(map[string]bool),
		filePath: filePath,
	}

	// 尝试加载已有状态（文件不存在则忽略）
	_ = at.load()

	return at
}

// IsApproved 检查服务器是否已被批准。
func (at *ApprovalTracker) IsApproved(serverName string) bool {
	at.mu.RLock()
	defer at.mu.RUnlock()
	return at.approved[serverName]
}

// IsDenied 检查服务器是否已被拒绝。
func (at *ApprovalTracker) IsDenied(serverName string) bool {
	at.mu.RLock()
	defer at.mu.RUnlock()
	return at.denied[serverName]
}

// IsDecided 检查服务器是否已有审批决定（批准或拒绝）。
func (at *ApprovalTracker) IsDecided(serverName string) bool {
	at.mu.RLock()
	defer at.mu.RUnlock()
	return at.approved[serverName] || at.denied[serverName]
}

// Approve 批准指定服务器。
func (at *ApprovalTracker) Approve(serverName string) error {
	at.mu.Lock()
	defer at.mu.Unlock()

	at.approved[serverName] = true
	delete(at.denied, serverName)

	return at.save()
}

// Deny 拒绝指定服务器。
func (at *ApprovalTracker) Deny(serverName string) error {
	at.mu.Lock()
	defer at.mu.Unlock()

	at.denied[serverName] = true
	delete(at.approved, serverName)

	return at.save()
}

// Clear 清除指定服务器的审批决定。
func (at *ApprovalTracker) Clear(serverName string) error {
	at.mu.Lock()
	defer at.mu.Unlock()

	delete(at.approved, serverName)
	delete(at.denied, serverName)

	return at.save()
}

// AllApproved 返回所有已批准的服务器名称。
func (at *ApprovalTracker) AllApproved() []string {
	at.mu.RLock()
	defer at.mu.RUnlock()

	names := make([]string, 0, len(at.approved))
	for name := range at.approved {
		names = append(names, name)
	}
	return names
}

// load 从磁盘加载审批状态。
func (at *ApprovalTracker) load() error {
	data, err := os.ReadFile(at.filePath)
	if err != nil {
		// 文件不存在是正常情况
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("读取审批文件失败: %w", err)
	}

	if len(data) == 0 {
		return nil
	}

	var file approvalFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("解析审批文件失败: %w", err)
	}

	if file.Approved != nil {
		at.approved = file.Approved
	}
	if file.Denied != nil {
		at.denied = file.Denied
	}

	return nil
}

// save 将审批状态持久化到磁盘。
func (at *ApprovalTracker) save() error {
	dir := filepath.Dir(at.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建审批目录失败: %w", err)
	}

	file := approvalFile{
		Approved: at.approved,
		Denied:   at.denied,
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化审批数据失败: %w", err)
	}

	if err := os.WriteFile(at.filePath, data, 0644); err != nil {
		return fmt.Errorf("写入审批文件失败: %w", err)
	}

	return nil
}
