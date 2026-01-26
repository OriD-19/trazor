package main

import (
	"time"
)

// WindowMetrics represents aggregated metrics for a time window
type WindowMetrics struct {
	WindowStart      int64             `json:"window_start"`
	WindowEnd        int64             `json:"window_end"`
	TotalRequests    uint64            `json:"total_requests"`
	AvgLatency       float64           `json:"avg_latency_us"`
	MinLatency       uint64            `json:"min_latency_us"`
	MaxLatency       uint64            `json:"max_latency_us"`
	P50Latency       uint64            `json:"p50_latency_us"`
	P95Latency       uint64            `json:"p95_latency_us"`
	P99Latency       uint64            `json:"p99_latency_us"`
	ProcessBreakdown map[uint32]uint64 `json:"process_breakdown"`
	AgentID          string            `json:"agent_id"`
	Timestamp        time.Time         `json:"timestamp"`
}

// NewWindowMetrics creates a new WindowMetrics instance
func NewWindowMetrics() *WindowMetrics {
	return &WindowMetrics{
		ProcessBreakdown: make(map[uint32]uint64),
		Timestamp:        time.Now().UTC(),
	}
}

// LatencySample represents a single latency measurement
type LatencySample struct {
	ProcessID uint32
	LatencyNs uint64
	Timestamp int64
}
