package perf

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

type Stats struct {
	mu sync.RWMutex

	Alloc      uint64
	TotalAlloc uint64
	Sys       uint64
	NumGC     uint32
	NumGo     int

	StartTime time.Time
	Requests  uint64
	Errors    uint64
}

var globalStats Stats

func GetStats() Stats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	globalStats.mu.RLock()
	defer globalStats.mu.RUnlock()

	stats := globalStats
	stats.Alloc = m.Alloc
	stats.TotalAlloc = m.TotalAlloc
	stats.Sys = m.Sys
	stats.NumGC = m.NumGC
	stats.NumGo = runtime.NumGoroutine()

	return stats
}

func RecordRequest() {
	atomic.AddUint64(&globalStats.Requests, 1)
}

func RecordError() {
	atomic.AddUint64(&globalStats.Errors, 1)
}

func ResetStats() {
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()
	globalStats.Requests = 0
	globalStats.Errors = 0
	globalStats.StartTime = time.Now()
}

type Profiler struct {
	enabled    atomic.Bool
	interval   time.Duration
	samples    []Sample
	mu         sync.RWMutex
	stopChan   chan struct{}
}

type Sample struct {
	Timestamp time.Time
	MemStats  runtime.MemStats
	GoRoutines int
}

var globalProfiler Profiler

func (p *Profiler) Start(interval time.Duration) {
	if p.enabled.Load() {
		return
	}

	p.enabled.Store(true)
	p.interval = interval
	p.stopChan = make(chan struct{})
	p.samples = make([]Sample, 0)

	go p.collect()
}

func (p *Profiler) Stop() {
	if !p.enabled.Load() {
		return
	}

	p.enabled.Store(false)
	close(p.stopChan)
}

func (p *Profiler) collect() {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			p.mu.Lock()
			p.samples = append(p.samples, Sample{
				Timestamp:  time.Now(),
				MemStats:   m,
				GoRoutines: runtime.NumGoroutine(),
			})
			p.mu.Unlock()
		}
	}
}

func (p *Profiler) GetSamples() []Sample {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]Sample, len(p.samples))
	copy(result, p.samples)
	return result
}

func EnableProfiling(interval time.Duration) {
	globalProfiler.Start(interval)
}

func DisableProfiling() {
	globalProfiler.Stop()
}

func GetProfile() ProfileReport {
	stats := GetStats()
	samples := globalProfiler.GetSamples()

	uptime := time.Since(stats.StartTime)
	var avgRequestsPerSec float64
	if uptime.Seconds() > 0 {
		avgRequestsPerSec = float64(stats.Requests) / uptime.Seconds()
	}

	var errorRate float64
	if stats.Requests > 0 {
		errorRate = float64(stats.Errors) / float64(stats.Requests)
	}

	return ProfileReport{
		Uptime:            uptime,
		TotalRequests:      stats.Requests,
		TotalErrors:       stats.Errors,
		ErrorRate:         errorRate,
		AvgRequestsPerSec: avgRequestsPerSec,
		CurrentMemAlloc:   stats.Alloc,
		TotalMemAlloc:     stats.TotalAlloc,
		SysMem:            stats.Sys,
		NumGoroutines:     stats.NumGo,
		NumGC:             stats.NumGC,
		Samples:           samples,
	}
}

type ProfileReport struct {
	Uptime            time.Duration
	TotalRequests     uint64
	TotalErrors      uint64
	ErrorRate         float64
	AvgRequestsPerSec float64
	CurrentMemAlloc   uint64
	TotalMemAlloc     uint64
	SysMem           uint64
	NumGoroutines     int
	NumGC            uint32
	Samples          []Sample
}

func (r ProfileReport) String() string {
	return formatProfileReport(r)
}

func formatProfileReport(r ProfileReport) string {
	lines := []string{
		"=== Forge 性能报告 ===",
		"",
		"运行时长: " + formatDuration(r.Uptime),
		"总请求数: " + formatUint64(r.TotalRequests),
		"错误数:   " + formatUint64(r.TotalErrors),
		"错误率:   " + formatFloat(r.ErrorRate*100) + "%",
		"平均 QPS: " + formatFloat(r.AvgRequestsPerSec),
		"",
		"内存统计:",
		"  当前分配: " + formatBytes(r.CurrentMemAlloc),
		"  累计分配: " + formatBytes(r.TotalMemAlloc),
		"  系统内存: " + formatBytes(r.SysMem),
		"",
		"运行时:",
		"  Goroutine 数量: " + formatInt(r.NumGoroutines),
		"  GC 次数: " + formatUint32(r.NumGC),
		"",
	}

	result := ""
	for _, line := range lines {
		if result != "" {
			result += "\n"
		}
		result += line
	}
	return result
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return formatInt(h) + "h " + formatInt(m) + "m " + formatInt(s) + "s"
}

func formatUint64(v uint64) string {
	return fmt.Sprintf("%d", v)
}

func formatUint32(v uint32) string {
	return fmt.Sprintf("%d", v)
}

func formatInt(v int) string {
	return fmt.Sprintf("%d", v)
}

func formatFloat(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

func formatBytes(v uint64) string {
	const KB = 1024
	const MB = KB * 1024
	const GB = MB * 1024

	if v >= GB {
		return fmt.Sprintf("%.2f GB", float64(v)/GB)
	}
	if v >= MB {
		return fmt.Sprintf("%.2f MB", float64(v)/MB)
	}
	if v >= KB {
		return fmt.Sprintf("%.2f KB", float64(v)/KB)
	}
	return fmt.Sprintf("%d B", v)
}

var _ = debug.SetGCPercent