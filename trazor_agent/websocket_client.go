package main

import (
	"encoding/json"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketClient handles communication with the monitoring server
type WebSocketClient struct {
	conn           *websocket.Conn
	serverURL      string
	connected      bool
	mutex          sync.RWMutex
	sendChannel    chan *WindowMetrics
	done           chan struct{}
	reconnectDelay time.Duration
	maxMessageSize int64
	writeWait      time.Duration
	pongWait       time.Duration
	pingPeriod     time.Duration
	agentID        string
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(serverURL, agentID string) *WebSocketClient {
	return &WebSocketClient{
		serverURL:      serverURL,
		sendChannel:    make(chan *WindowMetrics, 100), // Buffer for outgoing metrics
		done:           make(chan struct{}),
		reconnectDelay: 5 * time.Second,
		maxMessageSize: 512, // Max message size in bytes
		writeWait:      10 * time.Second,
		pongWait:       60 * time.Second,
		pingPeriod:     (54 * time.Second), // Must be less than pongWait (60s)
		agentID:        agentID,
	}
}

// Connect establishes connection to the WebSocket server
func (wsc *WebSocketClient) Connect() error {
	wsc.mutex.Lock()
	defer wsc.mutex.Unlock()

	if wsc.connected {
		return nil
	}

	u, err := url.Parse(wsc.serverURL)
	if err != nil {
		return err
	}

	log.Printf("Connecting to WebSocket server: %s", wsc.serverURL)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	wsc.conn = conn
	wsc.connected = true

	// Set connection limits
	wsc.conn.SetReadLimit(wsc.maxMessageSize)
	wsc.conn.SetReadDeadline(time.Now().Add(wsc.pongWait))
	wsc.conn.SetPongHandler(func(string) error {
		wsc.conn.SetReadDeadline(time.Now().Add(wsc.pongWait))
		return nil
	})

	// Start reader goroutine
	go wsc.readPump()

	// Start writer goroutine
	go wsc.writePump()

	log.Printf("Successfully connected to WebSocket server")
	return nil
}

// Disconnect closes the WebSocket connection
func (wsc *WebSocketClient) Disconnect() {
	wsc.mutex.Lock()
	defer wsc.mutex.Unlock()

	if !wsc.connected {
		return
	}

	close(wsc.done)
	if wsc.conn != nil {
		wsc.conn.Close()
	}

	wsc.connected = false
	log.Printf("Disconnected from WebSocket server")
}

// IsConnected returns the connection status
func (wsc *WebSocketClient) IsConnected() bool {
	wsc.mutex.RLock()
	defer wsc.mutex.RUnlock()
	return wsc.connected
}

// SendMetrics sends metrics to the WebSocket server (non-blocking)
func (wsc *WebSocketClient) SendMetrics(metrics *WindowMetrics) {
	if metrics == nil {
		return
	}

	// Set the agent ID if not already set
	if metrics.AgentID == "" {
		metrics.AgentID = wsc.agentID
	}

	select {
	case wsc.sendChannel <- metrics:
	default:
		// Channel is full, drop the metrics (as requested)
		log.Printf("Metrics send channel full, dropping metrics")
	}
}

// readPump handles incoming messages from the WebSocket server
func (wsc *WebSocketClient) readPump() {
	defer func() {
		wsc.mutex.Lock()
		if wsc.conn != nil {
			wsc.conn.Close()
		}
		wsc.connected = false
		wsc.mutex.Unlock()
	}()

	for {
		select {
		case <-wsc.done:
			return
		default:
		}

		wsc.conn.SetReadDeadline(time.Now().Add(wsc.pongWait))
		_, _, err := wsc.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}
	}
}

// writePump handles outgoing messages to the WebSocket server
func (wsc *WebSocketClient) writePump() {
	ticker := time.NewTicker(wsc.pingPeriod)
	defer func() {
		ticker.Stop()
		wsc.mutex.Lock()
		if wsc.conn != nil {
			wsc.conn.Close()
		}
		wsc.connected = false
		wsc.mutex.Unlock()
	}()

	for {
		select {
		case <-wsc.done:
			return
		case metrics, ok := <-wsc.sendChannel:
			if !ok {
				wsc.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := wsc.writeMetrics(metrics); err != nil {
				log.Printf("Error sending metrics: %v", err)
				return
			}

		case <-ticker.C:
			if err := wsc.sendPing(); err != nil {
				return
			}
		}
	}
}

// writeMetrics serializes and sends a metrics message
func (wsc *WebSocketClient) writeMetrics(metrics *WindowMetrics) error {
	wsc.mutex.RLock()
	if !wsc.connected || wsc.conn == nil {
		wsc.mutex.RUnlock()
		return nil // Not connected, drop the message
	}
	conn := wsc.conn
	wsc.mutex.RUnlock()

	conn.SetWriteDeadline(time.Now().Add(wsc.writeWait))

	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}

// sendPing sends a ping message to keep the connection alive
func (wsc *WebSocketClient) sendPing() error {
	wsc.mutex.RLock()
	if !wsc.connected || wsc.conn == nil {
		wsc.mutex.RUnlock()
		return nil
	}
	conn := wsc.conn
	wsc.mutex.RUnlock()

	conn.SetWriteDeadline(time.Now().Add(wsc.writeWait))
	return conn.WriteMessage(websocket.PingMessage, nil)
}

// StartReconnectLoop starts a background goroutine to handle reconnections
// Note: This is not used in the current drop-on-failure approach, but included for future use
func (wsc *WebSocketClient) StartReconnectLoop() {
	go func() {
		for {
			time.Sleep(wsc.reconnectDelay)

			wsc.mutex.RLock()
			connected := wsc.connected
			wsc.mutex.RUnlock()

			if !connected {
				log.Printf("Attempting to reconnect...")
				if err := wsc.Connect(); err != nil {
					log.Printf("Reconnection failed: %v", err)
				} else {
					log.Printf("Reconnection successful")
				}
			}
		}
	}()
}
