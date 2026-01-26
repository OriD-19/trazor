package main

import (
	"sync"
	"time"
)

// WindowAggregator manages time-based windowing of latency data
type WindowAggregator struct {
	mutex          sync.RWMutex
	currentWindow  map[uint32][]uint64 // PID → latencies
	windowStart    int64
	windowDuration time.Duration
	metricsChannel chan *WindowMetrics
	samplesBuffer  []LatencySample
	maxSamples     int
}

// NewWindowAggregator creates a new WindowAggregator
func NewWindowAggregator(windowDuration time.Duration, metricsChannel chan *WindowMetrics) *WindowAggregator {
	now := time.Now().UnixNano()
	alignedStart := (now / int64(windowDuration)) * int64(windowDuration)

	return &WindowAggregator{
		currentWindow:  make(map[uint32][]uint64),
		windowStart:    alignedStart,
		windowDuration: windowDuration,
		metricsChannel: metricsChannel,
		samplesBuffer:  make([]LatencySample, 0, 1000),
		maxSamples:     1000,
	}
}

// AddSample adds a latency sample to the current window
func (wa *WindowAggregator) AddSample(processID uint32, latencyNs uint64, timestamp int64) {
	wa.mutex.Lock()
	defer wa.mutex.Unlock()

	wa.currentWindow[processID] = append(wa.currentWindow[processID], latencyNs)

	sample := LatencySample{
		ProcessID: processID,
		LatencyNs: latencyNs,
		Timestamp: timestamp,
	}

	wa.samplesBuffer = append(wa.samplesBuffer, sample)

	if len(wa.samplesBuffer) >= wa.maxSamples {
		wa.samplesBuffer = wa.samplesBuffer[len(wa.samplesBuffer)/2:]
	}
}

// RotateWindow rotates to the next time window and emits metrics for the completed window
func (wa *WindowAggregator) RotateWindow() {
	wa.mutex.Lock()
	defer wa.mutex.Unlock()

	if len(wa.currentWindow) == 0 {
		wa.windowStart += int64(wa.windowDuration)
		return
	}

	metrics := wa.calculateMetrics()

	select {
	case wa.metricsChannel <- metrics:
	default:
	}

	wa.currentWindow = make(map[uint32][]uint64)
	wa.windowStart += int64(wa.windowDuration)
}

// calculateMetrics computes aggregated metrics for the current window
func (wa *WindowAggregator) calculateMetrics() *WindowMetrics {
	metrics := NewWindowMetrics()
	metrics.WindowStart = wa.windowStart
	metrics.WindowEnd = wa.windowStart + int64(wa.windowDuration)

	var allLatencies []uint64
	var totalLatency uint64
	totalRequests := uint64(0)

	minLatency := ^uint64(0) // max uint64
	maxLatency := uint64(0)

	for processID, latencies := range wa.currentWindow {
		processRequests := uint64(len(latencies))
		metrics.ProcessBreakdown[processID] = processRequests
		totalRequests += processRequests

		for _, latency := range latencies {
			allLatencies = append(allLatencies, latency)
			totalLatency += latency

			if latency < minLatency {
				minLatency = latency
			}
			if latency > maxLatency {
				maxLatency = latency
			}
		}
	}

	metrics.TotalRequests = totalRequests
	metrics.MinLatency = minLatency / 1000 // Convert to microseconds
	metrics.MaxLatency = maxLatency / 1000 // Convert to microseconds

	if totalRequests > 0 {
		metrics.AvgLatency = float64(totalLatency) / float64(totalRequests) / 1000.0 // Convert to microseconds
	}

	if len(allLatencies) > 0 {
		metrics.P50Latency = CalculatePercentile(allLatencies, 50) / 1000
		metrics.P95Latency = CalculatePercentile(allLatencies, 95) / 1000
		metrics.P99Latency = CalculatePercentile(allLatencies, 99) / 1000
	}

	return metrics
}

// GetCurrentWindowStart returns the start time of the current window
func (wa *WindowAggregator) GetCurrentWindowStart() int64 {
	wa.mutex.RLock()
	defer wa.mutex.RUnlock()
	return wa.windowStart
}

// GetSampleCount returns the number of samples in the current window
func (wa *WindowAggregator) GetSampleCount() int {
	wa.mutex.RLock()
	defer wa.mutex.RUnlock()

	count := 0
	for _, latencies := range wa.currentWindow {
		count += len(latencies)
	}
	return count
}
