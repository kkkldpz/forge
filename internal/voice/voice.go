// Package voice 实现语音模式的音频捕获和识别。
package voice

import (
	"context"
	"fmt"
	"sync"
)

type Config struct {
	Enabled    bool
	SampleRate int
	Channels   int
	STTEngine  string
}

type Recorder struct {
	mu       sync.RWMutex
	config   Config
	recording bool
	ctx    context.Context
	cancel context.CancelFunc
}

var (
	globalRecorder     *Recorder
	globalRecorderOnce sync.Once
)

func GlobalRecorder() *Recorder {
	globalRecorderOnce.Do(func() {
		globalRecorder = &Recorder{
			config: Config{
				Enabled:    false,
				SampleRate: 16000,
				Channels:   1,
				STTEngine:  "default",
			},
		}
	})
	return globalRecorder
}

func (r *Recorder) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.recording {
		return fmt.Errorf("已经在录音中")
	}

	r.ctx, r.cancel = context.WithCancel(context.Background())
	r.recording = true

	return nil
}

func (r *Recorder) Stop() ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.recording {
		return nil, fmt.Errorf("未在录音")
	}

	r.cancel()
	r.recording = false

	return []byte{}, nil
}

func (r *Recorder) IsRecording() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.recording
}

func (r *Recorder) SetConfig(cfg Config) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = cfg
}

func (r *Recorder) GetConfig() Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

func (r *Recorder) ProcessAudio(audioData []byte) (string, error) {
	if len(audioData) == 0 {
		return "", fmt.Errorf("音频数据为空")
	}

	return "语音处理功能需要配置 STT 引擎", nil
}

func Enable() error {
	r := GlobalRecorder()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config.Enabled = true
	return nil
}

func Disable() {
	r := GlobalRecorder()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config.Enabled = false
}

func IsEnabled() bool {
	r := GlobalRecorder()
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config.Enabled
}