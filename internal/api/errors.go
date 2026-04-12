package api

import "fmt"

// APIError 表示 Anthropic API 返回的错误。
type APIError struct {
	StatusCode int    `json:"status"`
	Type       string `json:"type"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API 错误 %d [%s]: %s", e.StatusCode, e.Type, e.Message)
}

// IsRetryable 判断该错误是否可以重试。
func (e *APIError) IsRetryable() bool {
	switch e.StatusCode {
	case 429, 529, 502, 503, 504:
		return true
	default:
		return false
	}
}

// IsAuthError 判断是否为认证错误。
func (e *APIError) IsAuthError() bool {
	return e.StatusCode == 401
}

// IsOverloaded 判断是否为服务过载。
func (e *APIError) IsOverloaded() bool {
	return e.StatusCode == 529
}

// IsRateLimited 判断是否为速率限制。
func (e *APIError) IsRateLimited() bool {
	return e.StatusCode == 429
}

// IsContextOverflow 判断是否为上下文溢出。
func (e *APIError) IsContextOverflow() bool {
	return e.StatusCode == 400 && (e.Type == "invalid_request_error" ||
		containsStr(e.Message, "context") ||
		containsStr(e.Message, "too many tokens"))
}

// CannotRetryError 表示无法重试的错误，应直接返回给调用方。
type CannotRetryError struct {
	OriginalError error
	Context       string
}

func (e *CannotRetryError) Error() string {
	return fmt.Sprintf("无法重试 [%s]: %v", e.Context, e.OriginalError)
}

func (e *CannotRetryError) Unwrap() error {
	return e.OriginalError
}

// FallbackTriggeredError 表示需要切换到备用模型的错误。
type FallbackTriggeredError struct {
	OriginalModel string
	FallbackModel string
	Cause         error
}

func (e *FallbackTriggeredError) Error() string {
	return fmt.Sprintf("模型 %s 失败，需要降级到 %s: %v", e.OriginalModel, e.FallbackModel, e.Cause)
}

func (e *FallbackTriggeredError) Unwrap() error {
	return e.Cause
}

// containsStr 检查字符串是否包含子串（大小写不敏感）。
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > 0 && len(substr) > 0 &&
				findSubstr(s, substr)))
}

func findSubstr(s, substr string) bool {
	sl := len(s)
	subl := len(substr)
	for i := 0; i <= sl-subl; i++ {
		match := true
		for j := 0; j < subl; j++ {
			sc := s[i+j]
			bc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if bc >= 'A' && bc <= 'Z' {
				bc += 32
			}
			if sc != bc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
