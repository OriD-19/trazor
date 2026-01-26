package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for testing
	},
}

// WindowMetrics mirrors the structure from the agent
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

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("WebSocket connection established from %s", conn.RemoteAddr())

	for {
		// Read message from client
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			break
		}

		log.Printf("Received message type: %d, size: %d bytes", messageType, len(message))

		// Try to parse as WindowMetrics
		var metrics WindowMetrics
		if err := json.Unmarshal(message, &metrics); err == nil {
			log.Printf("=== Window Metrics Received ===")
			log.Printf("Agent ID: %s", metrics.AgentID)
			log.Printf("Window: %d - %d", metrics.WindowStart, metrics.WindowEnd)
			log.Printf("Total Requests: %d", metrics.TotalRequests)
			if metrics.TotalRequests > 0 {
				log.Printf("Latency Stats (μs): Avg=%.2f, Min=%d, Max=%d",
					metrics.AvgLatency, metrics.MinLatency, metrics.MaxLatency)
				log.Printf("Percentiles (μs): P50=%d, P95=%d, P99=%d",
					metrics.P50Latency, metrics.P95Latency, metrics.P99Latency)
			}
			log.Printf("Process Breakdown: %v", metrics.ProcessBreakdown)
			log.Printf("Timestamp: %s", metrics.Timestamp.Format(time.RFC3339))
			log.Printf("===============================")
		} else {
			log.Printf("Raw message: %s", string(message))
		}

		// Echo back a simple acknowledgment
		response := map[string]string{
			"status":    "received",
			"timestamp": time.Now().Format(time.RFC3339),
		}

		if err := conn.WriteJSON(response); err != nil {
			log.Printf("Write error: %v", err)
			break
		}
	}

	log.Printf("WebSocket connection closed")
}

func main() {
	http.HandleFunc("/monitoring", handleWebSocket)

	log.Printf("Starting WebSocket test server on :8080")
	log.Printf("Connect to: ws://localhost:8080/monitoring")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server error:", err)
	}
}
