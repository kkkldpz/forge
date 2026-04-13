package perf

import (
	"testing"
	"time"
)

func TestGetStats(t *testing.T) {
	stats := GetStats()

	if stats.Alloc == 0 && stats.TotalAlloc == 0 {
		t.Log("Initial stats recorded (may be 0 in test environment)")
	}
}

func TestRecordRequest(t *testing.T) {
	initial := GetStats().Requests
	RecordRequest()
	after := GetStats().Requests

	if after <= initial {
		t.Errorf("Expected request count to increase, got before=%d after=%d", initial, after)
	}
}

func TestRecordError(t *testing.T) {
	initial := GetStats().Errors
	RecordError()
	after := GetStats().Errors

	if after <= initial {
		t.Errorf("Expected error count to increase, got before=%d after=%d", initial, after)
	}
}

func TestResetStats(t *testing.T) {
	RecordRequest()
	RecordError()
	ResetStats()

	stats := GetStats()
	if stats.Requests != 0 || stats.Errors != 0 {
		t.Errorf("Expected stats to be reset, got Requests=%d Errors=%d", stats.Requests, stats.Errors)
	}
}

func TestProfiler_StartStop(t *testing.T) {
	p := &Profiler{}

	p.Start(100 * time.Millisecond)
	if !p.enabled.Load() {
		t.Error("Expected profiler to be enabled after Start")
	}

	p.Stop()
	if p.enabled.Load() {
		t.Error("Expected profiler to be disabled after Stop")
	}
}

func TestProfiler_DoubleStart(t *testing.T) {
	p := &Profiler{}

	p.Start(100 * time.Millisecond)
	p.Start(100 * time.Millisecond)

	p.Stop()
}

func TestProfiler_GetSamples(t *testing.T) {
	p := &Profiler{}

	p.Start(10 * time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	p.Stop()

	samples := p.GetSamples()
	if len(samples) == 0 {
		t.Error("Expected samples to be collected")
	}
}

func TestGetProfile(t *testing.T) {
	ResetStats()
	RecordRequest()
	RecordRequest()
	RecordError()

	report := GetProfile()

	if report.TotalRequests != 2 {
		t.Errorf("Expected 2 requests, got %d", report.TotalRequests)
	}

	if report.TotalErrors != 1 {
		t.Errorf("Expected 1 error, got %d", report.TotalErrors)
	}

	if report.ErrorRate == 0 {
		t.Error("Expected non-zero error rate")
	}
}

func TestProfileReport_String(t *testing.T) {
	report := ProfileReport{
		Uptime:            1*time.Hour + 30*time.Minute + 45*time.Second,
		TotalRequests:     1000,
		TotalErrors:      10,
		ErrorRate:         0.01,
		AvgRequestsPerSec: 0.28,
		CurrentMemAlloc:   1024 * 1024,
		TotalMemAlloc:     10 * 1024 * 1024,
		SysMem:           50 * 1024 * 1024,
		NumGoroutines:     10,
		NumGC:            5,
	}

	output := report.String()
	if output == "" {
		t.Error("Expected non-empty output")
	}
}

func TestEnableDisableProfiling(t *testing.T) {
	EnableProfiling(100 * time.Millisecond)
	DisableProfiling()
}

func TestGlobalProfiler_StartStop(t *testing.T) {
	EnableProfiling(100 * time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	DisableProfiling()
}

func TestGetProfile_Empty(t *testing.T) {
	ResetStats()

	report := GetProfile()

	if report.TotalRequests != 0 {
		t.Errorf("Expected 0 requests, got %d", report.TotalRequests)
	}
}