package voice

import (
	"testing"
)

func TestGlobalRecorder(t *testing.T) {
	r1 := GlobalRecorder()
	r2 := GlobalRecorder()

	if r1 != r2 {
		t.Error("GlobalRecorder should return the same instance")
	}
}

func TestRecorder_StartStop(t *testing.T) {
	r := &Recorder{
		config: Config{Enabled: true},
	}

	err := r.Start()
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	if !r.IsRecording() {
		t.Error("Recorder should be recording after Start")
	}

	_, err = r.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	if r.IsRecording() {
		t.Error("Recorder should not be recording after Stop")
	}
}

func TestRecorder_Start_DoubleStart(t *testing.T) {
	r := &Recorder{
		config: Config{Enabled: true},
	}

	r.Start()
	err := r.Start()

	if err == nil {
		t.Error("Expected error for double Start")
	}

	r.Stop()
}

func TestRecorder_Stop_NotRecording(t *testing.T) {
	r := &Recorder{
		config: Config{Enabled: true},
	}

	_, err := r.Stop()

	if err == nil {
		t.Error("Expected error for Stop when not recording")
	}
}

func TestRecorder_SetGetConfig(t *testing.T) {
	r := GlobalRecorder()

	cfg := Config{
		Enabled:    true,
		SampleRate: 44100,
		Channels:   2,
		STTEngine:  "google",
	}

	r.SetConfig(cfg)
	got := r.GetConfig()

	if got.SampleRate != 44100 {
		t.Errorf("Expected sample rate 44100, got %d", got.SampleRate)
	}
}

func TestRecorder_ProcessAudio(t *testing.T) {
	r := GlobalRecorder()

	result, err := r.ProcessAudio([]byte("audio data"))
	if err != nil {
		t.Errorf("ProcessAudio failed: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty result")
	}
}

func TestRecorder_ProcessAudio_Empty(t *testing.T) {
	r := GlobalRecorder()

	_, err := r.ProcessAudio([]byte{})
	if err == nil {
		t.Error("Expected error for empty audio data")
	}
}

func TestEnableDisable(t *testing.T) {
	Enable()
	if !IsEnabled() {
		t.Error("Expected voice to be enabled")
	}

	Disable()
	if IsEnabled() {
		t.Error("Expected voice to be disabled")
	}
}