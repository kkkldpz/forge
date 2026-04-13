package transport

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// StdioTransport 通过子进程的 stdin/stdout 实现 MCP 消息传输。
// 使用换行分隔的 JSON (JSONL) 格式。
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	reader *bufio.Reader

	mu       sync.Mutex
	sendErr  error
	recvCh   chan []byte
	doneCh   chan struct{}
	cancelFn context.CancelFunc
}

// NewStdioTransport 创建 stdio 传输实例并启动子进程。
// command 是可执行文件路径，args 是命令行参数，env 是环境变量。
func NewStdioTransport(ctx context.Context, command string, args []string, env map[string]string) (*StdioTransport, error) {
	ctx, cancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = os.Environ()

	// 合并自定义环境变量（覆盖同名变量）
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("创建 stdin 管道失败: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		stdinPipe.Close()
		cancel()
		return nil, fmt.Errorf("创建 stdout 管道失败: %w", err)
	}

	// 将 stderr 指向父进程的 stderr，便于调试
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		stdinPipe.Close()
		stdoutPipe.Close()
		cancel()
		return nil, fmt.Errorf("启动子进程失败: %w", err)
	}

	t := &StdioTransport{
		cmd:      cmd,
		stdin:    stdinPipe,
		stdout:   stdoutPipe,
		reader:   bufio.NewReader(stdoutPipe),
		recvCh:   make(chan []byte, 64),
		doneCh:   make(chan struct{}),
		cancelFn: cancel,
	}

	// 启动读取协程
	go t.readLoop()

	return t, nil
}

// Send 发送 JSON-RPC 消息到子进程的 stdin。
func (t *StdioTransport) Send(msg []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.sendErr != nil {
		return fmt.Errorf("传输已关闭: %w", t.sendErr)
	}

	// 写入消息并追加换行符（JSONL 格式）
	_, err := t.stdin.Write(append(msg, '\n'))
	if err != nil {
		t.sendErr = err
		return fmt.Errorf("写入 stdin 失败: %w", err)
	}

	return nil
}

// Receive 返回消息接收通道。
func (t *StdioTransport) Receive() <-chan []byte {
	return t.recvCh
}

// Close 关闭传输并终止子进程。
func (t *StdioTransport) Close() error {
	// 停止读取协程
	t.cancelFn()

	// 关闭 stdin
	if t.stdin != nil {
		t.stdin.Close()
	}

	// 等待读取协程退出
	<-t.doneCh

	// 关闭 stdout
	if t.stdout != nil {
		t.stdout.Close()
	}

	// 终止子进程
	var cmdErr error
	if t.cmd.Process != nil {
		cmdErr = t.cmd.Wait()
	}

	close(t.recvCh)

	return cmdErr
}

// readLoop 持续从子进程 stdout 读取消息并推送到接收通道。
func (t *StdioTransport) readLoop() {
	defer close(t.doneCh)

	for {
		line, err := t.reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "[mcp:stdio] 读取错误: %v\n", err)
			}
			return
		}

		// 去掉末尾换行符
		msg := line
		if len(msg) > 0 && msg[len(msg)-1] == '\n' {
			msg = msg[:len(msg)-1]
		}
		if len(msg) > 0 && msg[len(msg)-1] == '\r' {
			msg = msg[:len(msg)-1]
		}

		if len(msg) == 0 {
			continue
		}

		// 非阻塞发送到接收通道
		select {
		case t.recvCh <- msg:
		default:
			fmt.Fprintf(os.Stderr, "[mcp:stdio] 警告: 接收通道已满，丢弃消息\n")
		}
	}
}
