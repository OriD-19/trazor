package main

//go:generate go tool bpf2go -tags linux trazor_agent monitoring.c
import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
)

type HttpEvent struct {
	Timestamp uint64
	LatencyNs uint64
	ProcessId uint32
}

// Configuration constants
const (
	WindowDuration     = 10 * time.Second
	WebSocketServerURL = "ws://localhost:8080/monitoring"
	AgentID            = "trazor-agent-1"
)

func main() {
	// Parse command line flags
	testMode := flag.Bool("test", false, "Run component tests and exit")
	flag.Parse()

	if *testMode {
		runTests()
		return
	}
	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// boilerplate code
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal("Removing Memlock: ", err)
	}

	var objs trazor_agentObjects
	if err := loadTrazor_agentObjects(&objs, nil); err != nil {
		log.Fatal("Loading eBPF objects: ", err)
	}
	defer objs.Close()

	// attach the programs to their respective uprobes
	executable, err := link.OpenExecutable("/usr/sbin/nginx")
	if err != nil {
		log.Fatalf("opening executable: %v", err)
	}

	conn_start, err := executable.Uprobe("ngx_http_process_request", objs.GetConnStart, nil)
	if err != nil {
		log.Fatalf("opening uprobe 'ngx_http_process_request': %v", err)
	}
	defer conn_start.Close()

	conn_end, err := executable.Uprobe("ngx_http_free_request", objs.GetLatencyOnEnd, nil)
	if err != nil {
		log.Fatalf("opening uprobe 'ngx_http_free_request': %v", err)
	}
	defer conn_end.Close()

	// Initialize components
	metricsChannel := make(chan *WindowMetrics, 10) // Buffer for metrics
	windowAggregator := NewWindowAggregator(WindowDuration, metricsChannel)
	wsClient := NewWebSocketClient(WebSocketServerURL, AgentID)

	// Connect to WebSocket server (non-blocking)
	go func() {
		if err := wsClient.Connect(); err != nil {
			log.Printf("Failed to connect to WebSocket server: %v", err)
			log.Printf("Continuing with local processing only...")
		}
	}()

	// Start window ticker for periodic aggregation
	windowTicker := time.NewTicker(WindowDuration)
	defer windowTicker.Stop()

	// Start metrics sender goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case metrics := <-metricsChannel:
				if wsClient.IsConnected() {
					wsClient.SendMetrics(metrics)
					log.Printf("Sent metrics: %d requests, avg=%.2fμs, P50=%dμs, P95=%dμs, P99=%dμs",
						metrics.TotalRequests, metrics.AvgLatency,
						metrics.P50Latency, metrics.P95Latency, metrics.P99Latency)
				} else {
					log.Printf("WebSocket not connected, metrics dropped: %d requests", metrics.TotalRequests)
				}
			case <-sigChan:
				return
			}
		}
	}()

	// Start window rotation goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-windowTicker.C:
				windowAggregator.RotateWindow()
			case <-sigChan:
				return
			}
		}
	}()

	// Start ringbuf reader
	ringBuf, err := ringbuf.NewReader(objs.Events)
	if err != nil {
		log.Fatal("Opening ringbuf reader: ", err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer ringBuf.Close()

		for {
			select {
			case <-sigChan:
				return
			default:
			}

			record, err := ringBuf.Read()
			if err != nil {
				log.Printf("Reading ringbuf: %v", err)
				continue
			}

			var event HttpEvent
			if err := binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &event); err != nil {
				fmt.Printf("parsing event: %v", err)
				continue
			}

			// Add sample to current window
			windowAggregator.AddSample(event.ProcessId, event.LatencyNs, int64(event.Timestamp))

			// Optional: Keep console output for debugging
			fmt.Printf("Event: PID=%d, Latency=%dus\n", event.ProcessId, event.LatencyNs/1000)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Printf("Shutting down gracefully...")

	// Close WebSocket connection
	wsClient.Disconnect()

	// Wait for goroutines to finish
	wg.Wait()

	log.Printf("Shutdown complete")
}
