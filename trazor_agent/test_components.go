package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// This is a simple test to verify our data structures and JSON serialization
func testDataStructures() {
	fmt.Printf("=== Testing Data Structures ===\n")

	// Create a sample WindowMetrics
	metrics := NewWindowMetrics()
	metrics.WindowStart = time.Now().Add(-10 * time.Second).UnixNano()
	metrics.WindowEnd = time.Now().UnixNano()
	metrics.TotalRequests = 1000
	metrics.AvgLatency = 250.5
	metrics.MinLatency = 10
	metrics.MaxLatency = 5000
	metrics.P50Latency = 200
	metrics.P95Latency = 800
	metrics.P99Latency = 1500
	metrics.AgentID = "test-agent-1"
	metrics.ProcessBreakdown[1234] = 500
	metrics.ProcessBreakdown[5678] = 500

	// Test JSON serialization
	jsonData, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		log.Printf("JSON marshaling error: %v", err)
		return
	}

	fmt.Printf("Serialized WindowMetrics:\n%s\n", string(jsonData))

	// Test JSON deserialization
	var parsedMetrics WindowMetrics
	if err := json.Unmarshal(jsonData, &parsedMetrics); err != nil {
		log.Printf("JSON unmarshaling error: %v", err)
		return
	}

	fmt.Printf("Parsed WindowMetrics:\n")
	fmt.Printf("  Agent ID: %s\n", parsedMetrics.AgentID)
	fmt.Printf("  Total Requests: %d\n", parsedMetrics.TotalRequests)
	fmt.Printf("  Avg Latency: %.2f μs\n", parsedMetrics.AvgLatency)
	fmt.Printf("  P95 Latency: %d μs\n", parsedMetrics.P95Latency)
	fmt.Printf("  Process Count: %d\n", len(parsedMetrics.ProcessBreakdown))
	fmt.Printf("===============================\n")
}

// Test percentile calculation
func testPercentileCalculation() {
	fmt.Printf("=== Testing Percentile Calculation ===\n")

	// Create sample latency data in nanoseconds
	latencies := make([]uint64, 100)
	for i := 0; i < 100; i++ {
		latencies[i] = uint64((i + 1) * 1000) // 1ms to 100ms
	}

	p50 := CalculatePercentile(latencies, 50)
	p95 := CalculatePercentile(latencies, 95)
	p99 := CalculatePercentile(latencies, 99)

	fmt.Printf("P50: %d μs (expected: ~50000)\n", p50/1000)
	fmt.Printf("P95: %d μs (expected: ~95000)\n", p95/1000)
	fmt.Printf("P99: %d μs (expected: ~99000)\n", p99/1000)

	// Test multiple percentiles at once
	percentiles := []float64{50, 95, 99}
	results := CalculateMultiplePercentiles(latencies, percentiles)

	fmt.Printf("Multiple calculation results:\n")
	for _, p := range percentiles {
		fmt.Printf("  P%.0f: %d μs\n", p, results[p]/1000)
	}
	fmt.Printf("===================================\n")
}

// Test window aggregation
func testWindowAggregator() {
	fmt.Printf("=== Testing Window Aggregator ===\n")

	metricsChannel := make(chan *WindowMetrics, 10)
	aggregator := NewWindowAggregator(1*time.Second, metricsChannel)

	// Add some sample data
	for i := 0; i < 10; i++ {
		aggregator.AddSample(1234, uint64((i+1)*10000), time.Now().UnixNano())
		aggregator.AddSample(5678, uint64((i+1)*15000), time.Now().UnixNano())
	}

	fmt.Printf("Sample count before rotation: %d\n", aggregator.GetSampleCount())

	// Rotate window to generate metrics
	aggregator.RotateWindow()

	// Check if metrics were generated
	select {
	case metrics := <-metricsChannel:
		fmt.Printf("Generated metrics:\n")
		fmt.Printf("  Total Requests: %d\n", metrics.TotalRequests)
		fmt.Printf("  Avg Latency: %.2f μs\n", metrics.AvgLatency)
		fmt.Printf("  Process Count: %d\n", len(metrics.ProcessBreakdown))
	default:
		fmt.Printf("No metrics generated\n")
	}

	fmt.Printf("=================================\n")
}

func runTests() {
	fmt.Printf("Running Go component tests...\n\n")

	testDataStructures()
	testPercentileCalculation()
	testWindowAggregator()

	fmt.Printf("All tests completed!\n")
}
