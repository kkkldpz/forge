// Package tool 提供工具执行和并发控制。
package tool

import (
	"context"
	"fmt"
	"sync"

	"github.com/kkkldpz/forge/internal/types"
)

// ToolCall 表示一个待执行的工具调用。
type ToolCall struct {
	ID    string
	Name  string
	Input []byte
}

// ToolCallResult 表示工具调用的执行结果。
type ToolCallResult struct {
	CallID  string
	Result  types.ToolResult
	Error   error
}

// ExecuteTools 批量执行工具调用。
// 并发安全的工具并行执行，不安全的串行执行。
func ExecuteTools(
	ctx context.Context,
	calls []ToolCall,
	tools []Tool,
	tuc ToolUseContext,
) []ToolCallResult {
	// 建立工具名称到工具的映射
	toolMap := make(map[string]Tool)
	for _, tool := range tools {
		toolMap[tool.Name()] = tool
	}

	// 按并发安全性分组
	safeCalls := make([]ToolCall, 0)
	unsafeCalls := make([]ToolCall, 0)

	for _, call := range calls {
		tool, ok := toolMap[call.Name]
		if !ok {
			// 工具不存在，作为不安全处理（会报错）
			unsafeCalls = append(unsafeCalls, call)
			continue
		}

		if tool.IsConcurrencySafe(call.Input) {
			safeCalls = append(safeCalls, call)
		} else {
			unsafeCalls = append(unsafeCalls, call)
		}
	}

	results := make([]ToolCallResult, 0, len(calls))
	resultChan := make(chan ToolCallResult, len(calls))

	// 并发执行安全的工具
	var wg sync.WaitGroup
	for _, call := range safeCalls {
		wg.Add(1)
		go func(c ToolCall) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				resultChan <- ToolCallResult{
					CallID: c.ID,
					Error:  ctx.Err(),
				}
				return
			default:
			}

			tool, ok := toolMap[c.Name]
			if !ok {
				resultChan <- ToolCallResult{
					CallID: c.ID,
					Error:  fmt.Errorf("工具 %s 未找到", c.Name),
				}
				return
			}

			result := tool.Call(ctx, c.Input, tuc)
			resultChan <- ToolCallResult{
				CallID: c.ID,
				Result: result,
			}
		}(call)
	}

	// 等待所有并发执行完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集并发执行结果
	for r := range resultChan {
		results = append(results, r)
	}

	// 串行执行不安全的工具
	for _, call := range unsafeCalls {
		select {
		case <-ctx.Done():
			results = append(results, ToolCallResult{
				CallID: call.ID,
				Error:  ctx.Err(),
			})
			continue
		default:
		}

		tool, ok := toolMap[call.Name]
		if !ok {
			results = append(results, ToolCallResult{
				CallID: call.ID,
				Error:  fmt.Errorf("工具 %s 未找到", call.Name),
			})
			continue
		}

		result := tool.Call(ctx, call.Input, tuc)
		results = append(results, ToolCallResult{
			CallID: call.ID,
			Result: result,
		})
	}

	// 按调用 ID 排序结果，保持顺序一致
	for i := range results {
		for j := i + 1; j < len(results); j++ {
			if results[j].CallID < results[i].CallID {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

// ExecuteSingleTool 执行单个工具调用。
func ExecuteSingleTool(
	ctx context.Context,
	call ToolCall,
	tool Tool,
	tuc ToolUseContext,
) ToolCallResult {
	select {
	case <-ctx.Done():
		return ToolCallResult{
			CallID: call.ID,
			Error:  ctx.Err(),
		}
	default:
	}

	result := tool.Call(ctx, call.Input, tuc)
	return ToolCallResult{
		CallID: call.ID,
		Result: result,
	}
}

// ValidateToolInput 验证工具输入。
func ValidateToolInput(
	ctx context.Context,
	tool Tool,
	input []byte,
) types.ValidationResult {
	return tool.ValidateInput(ctx, input)
}
