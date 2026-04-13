// Package session 实现会话持久化，包括 JSONL 转录读写和会话恢复。
package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SessionMeta 会话元数据。
type SessionMeta struct {
	SessionID   string    `json:"sessionId"`
	ProjectPath string    `json:"projectPath"`
	Model       string    `json:"model"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	MessageCount int     `json:"messageCount"`
}

// TranscriptEntry JSONL 转录中的一条记录。
type TranscriptEntry struct {
	Type      string          `json:"type"` // "user", "assistant", "tool_use", "tool_result", "system", "compact"
	Role      string          `json:"role,omitempty"`
	Content   string          `json:"content,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	ToolName  string          `json:"toolName,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

// CostState 费用状态持久化。
type CostState struct {
	TotalCostUSD            float64          `json:"totalCostUSD"`
	TotalInputTokens        int64            `json:"totalInputTokens"`
	TotalOutputTokens       int64            `json:"totalOutputTokens"`
	ModelUsage              map[string]*ModelCost `json:"modelUsage,omitempty"`
	TotalAPIDuration        time.Duration    `json:"totalAPIDuration"`
	TotalLinesAdded         int              `json:"totalLinesAdded"`
	TotalLinesRemoved       int              `json:"totalLinesRemoved"`
	SavedAt                 time.Time        `json:"savedAt"`
}

// ModelCost 单个模型的费用统计。
type ModelCost struct {
	InputTokens  int64   `json:"inputTokens"`
	OutputTokens int64   `json:"outputTokens"`
	CostUSD      float64 `json:"costUSD"`
}

// Storage 会话存储管理器。
type Storage struct {
	baseDir string // 存储根目录 (~/.forge/sessions/)
}

// NewStorage 创建会话存储管理器。
func NewStorage(homeDir string) *Storage {
	return &Storage{
		baseDir: filepath.Join(homeDir, ".forge", "sessions"),
	}
}

// SaveTranscript 保存会话转录到 JSONL 文件。
func (s *Storage) SaveTranscript(sessionID string, entries []TranscriptEntry) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return fmt.Errorf("创建会话目录失败: %w", err)
	}

	path := s.transcriptPath(sessionID)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建转录文件失败: %w", err)
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	encoder := json.NewEncoder(writer)

	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			slog.Warn("写入转录条目失败", "error", err)
			continue
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("刷新写入缓冲失败: %w", err)
	}

	slog.Debug("转录保存成功", "sessionID", sessionID, "条目数", len(entries), "path", path)
	return nil
}

// AppendTranscript 追加转录条目到已有文件。
func (s *Storage) AppendTranscript(sessionID string, entries []TranscriptEntry) error {
	path := s.transcriptPath(sessionID)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开转录文件失败: %w", err)
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	encoder := json.NewEncoder(writer)

	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			slog.Warn("追加转录条目失败", "error", err)
			continue
		}
	}

	return writer.Flush()
}

// LoadTranscript 加载会话转录。
func (s *Storage) LoadTranscript(sessionID string) ([]TranscriptEntry, error) {
	path := s.transcriptPath(sessionID)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开转录文件失败: %w", err)
	}
	defer f.Close()

	var entries []TranscriptEntry
	scanner := bufio.NewScanner(f)
	// 增大缓冲区以支持长行
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry TranscriptEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			slog.Warn("解析转录行失败", "error", err, "line", line[:min(len(line), 100)])
			continue
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取转录文件失败: %w", err)
	}

	slog.Debug("转录加载成功", "sessionID", sessionID, "条目数", len(entries))
	return entries, nil
}

// SaveCostState 保存费用状态。
func (s *Storage) SaveCostState(sessionID string, state *CostState) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return err
	}

	state.SavedAt = time.Now()
	path := s.costPath(sessionID)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化费用状态失败: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// LoadCostState 加载费用状态。
func (s *Storage) LoadCostState(sessionID string) (*CostState, error) {
	path := s.costPath(sessionID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state CostState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("反序列化费用状态失败: %w", err)
	}

	return &state, nil
}

// SaveMeta 保存会话元数据。
func (s *Storage) SaveMeta(meta *SessionMeta) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return err
	}

	meta.UpdatedAt = time.Now()
	path := s.metaPath(meta.SessionID)

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// ListSessions 列出所有会话。
func (s *Storage) ListSessions() ([]*SessionMeta, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []*SessionMeta
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".meta.json") {
			continue
		}

		path := filepath.Join(s.baseDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var meta SessionMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}

		sessions = append(sessions, &meta)
	}

	// 按更新时间降序排列
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

// DeleteSession 删除指定会话的所有文件。
func (s *Storage) DeleteSession(sessionID string) error {
	patterns := []string{
		s.transcriptPath(sessionID),
		s.metaPath(sessionID),
		s.costPath(sessionID),
	}

	for _, p := range patterns {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			slog.Warn("删除会话文件失败", "path", p, "error", err)
		}
	}

	return nil
}

// --- 路径辅助函数 ---

func (s *Storage) transcriptPath(sessionID string) string {
	return filepath.Join(s.baseDir, sessionID+".jsonl")
}

func (s *Storage) metaPath(sessionID string) string {
	return filepath.Join(s.baseDir, sessionID+".meta.json")
}

func (s *Storage) costPath(sessionID string) string {
	return filepath.Join(s.baseDir, sessionID+".cost.json")
}