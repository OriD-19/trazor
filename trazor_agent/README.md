# Trazor Agent with WebSocket Streaming and Windowing

This enhanced monitoring agent extends the original eBPF-based nginx latency monitoring with real-time WebSocket streaming and time-based windowing aggregation.

## Features

### Original Functionality
- eBPF uprobes on nginx for HTTP request latency monitoring
- Ring buffer communication between kernel and userspace
- Process-level latency tracking

### New Features
- **Time-based Windowing**: 10-second aggregation windows
- **Percentile Calculation**: P50, P95, P99 latency percentiles
- **WebSocket Streaming**: Real-time metrics transmission to central server
- **Efficient Algorithms**: Quickselect algorithm for percentile calculation
- **Graceful Degradation**: Drop-on-failure approach for network issues
- **Structured Metrics**: JSON-formatted aggregated data

## Architecture

```
eBPF Ringbuffer → Event Collector → Window Aggregator → WebSocket Client → Monitoring Server
                     ↓                          ↓              ↓
              Individual Events           Aggregated Metrics   JSON Serialization
```

## Components

### Core Files

- `main.go` - Main application entry point and orchestration
- `metrics.go` - Data structures for windowing and metrics
- `percentile_calculator.go` - Efficient percentile calculation algorithms
- `window_aggregator.go` - Time-based window management and aggregation
- `websocket_client.go` - WebSocket communication with monitoring server
- `monitoring.c` - eBPF programs (unchanged from original)

### Data Flow

1. **Event Collection**: eBPF programs capture HTTP request start/end times
2. **Windowing**: Events are aggregated into 10-second windows
3. **Metrics Calculation**: Percentiles, averages, and statistics computed
4. **WebSocket Streaming**: Aggregated metrics sent to central server
5. **Error Handling**: Network failures result in dropped data (graceful degradation)

## Configuration

Key configuration constants in `main.go`:

```go
const (
    WindowDuration      = 10 * time.Second     // Aggregation window size
    WebSocketServerURL  = "ws://localhost:8080/monitoring"  // Monitoring server
    AgentID             = "trazor-agent-1"      // Agent identifier
)
```

## Building and Running

### Prerequisites
- Linux kernel with eBPF support
- Go 1.21+ 
- nginx installation at `/usr/sbin/nginx`
- `clang` for eBPF compilation

### Build Steps

1. **Generate eBPF object files:**
   ```bash
   go generate .
   ```

2. **Build the agent:**
   ```bash
   go build -o trazor_agent .
   ```

3. **Run component tests:**
   ```bash
   go run . -test
   ```

4. **Run the agent:**
   ```bash
   sudo ./trazor_agent  # Requires root for eBPF
   ```

### WebSocket Server Testing

A test WebSocket server is included in the `test_server/` directory:

```bash
cd test_server
go run .
```

This starts a server on `ws://localhost:8080/monitoring` that receives and logs metrics.

## Metrics Format

The agent sends JSON messages with the following structure:

```json
{
  "window_start": 1640995200000000000,
  "window_end": 1640995210000000000,
  "total_requests": 1234,
  "avg_latency_us": 250.5,
  "min_latency_us": 10,
  "max_latency_us": 5000,
  "p50_latency_us": 200,
  "p95_latency_us": 800,
  "p99_latency_us": 1500,
  "process_breakdown": {
    "1234": 600,
    "5678": 634
  },
  "agent_id": "trazor-agent-1",
  "timestamp": "2026-01-26T02:02:01.90176566Z"
}
```

## Performance Characteristics

- **Memory Usage**: ~1-2MB for latency samples per window
- **CPU Overhead**: Minimal percentiles calculation every 10 seconds
- **Network Efficiency**: One JSON payload every 10 seconds vs continuous events
- **Latency**: ~10 second maximum reporting delay due to windowing

## Algorithm Details

### Percentile Calculation
- Small datasets (≤1000 items): Full sort approach
- Large datasets (>1000 items): Quickselect algorithm for O(n) average performance
- Multiple percentiles: Optimized batch calculation when possible

### Window Management
- Aligned to 10-second boundaries for consistency
- Non-blocking rotation to prevent event loss
- Buffer management to handle high event rates

### WebSocket Protocol
- Text messages with JSON payloads
- Ping/pong for connection health monitoring
- Graceful shutdown and connection handling

## Error Handling Strategy

Following the "drop-on-failure" requirement:
- WebSocket connection failures → metrics dropped, logging only
- Channel buffer full → metrics dropped with log message
- Network timeouts → automatic reconnection attempts disabled (configurable)

## Monitoring and Observability

The agent provides logging for:
- Connection status changes
- Metrics transmission success/failure
- eBPF program loading and attachment
- Window rotation and sample counts

## Future Enhancements

Potential areas for extension:
- Configurable window durations
- Multiple monitoring server support
- Authentication headers for WebSocket connections
- Local buffering for offline scenarios
- Additional metric types (error rates, request sizes)
- Circuit breaker pattern for resilience