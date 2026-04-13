// Package logging 封装 slog 日志功能，提供统一的日志初始化。
package logging

import (
	"log/slog"
	"os"
)

// 日志级别别名。
const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// Logger 封装 slog.Logger，提供 Forge 默认配置。
type Logger struct {
	*slog.Logger
}

// New 创建带指定名称的日志器。
func New(name string) *Logger {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return &Logger{
		Logger: slog.New(handler).With("component", name),
	}
}

// NewDebug 创建启用 debug 级别的日志器。
func NewDebug(name string) *Logger {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	return &Logger{
		Logger: slog.New(handler).With("component", name),
	}
}

// Default 返回全局 Forge 日志器。
func Default() *Logger {
	return New("forge")
}
