package main

import (
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

func statsHandler(w http.ResponseWriter, _ *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	allocMB := float64(m.Alloc) / 1024 / 1024
	sysMB := float64(m.Sys) / 1024 / 1024
	var pct float64
	if sysMB > 0 {
		pct = allocMB / sysMB * 100
	}

	sendSuccess(w, map[string]any{
		"active_players":   0,
		"memory_usage_mb":  allocMB,
		"memory_total_mb":  sysMB,
		"memory_usage_pct": pct,
		"goroutines":       runtime.NumGoroutine(),
		"uptime_seconds":   time.Since(s.startTime).Seconds(),
		"response_mb":      float64(atomic.LoadInt64(&s.responseBytes)) / 1024 / 1024,
	})
}
