package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// SSEHub manages active client event streams.
type SSEHub struct {
	mu      sync.RWMutex
	clients map[chan []byte]bool
}

// NewSSEHub creates a new event streaming hub.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[chan []byte]bool),
	}
}

// Broadcast sends a typed JSON event to all connected clients.
func (h *SSEHub) Broadcast(eventType string, data interface{}) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}

	msg := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, payload))

	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

// ServeHTTP handles incoming SSE connections.
func (h *SSEHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	msgCh := make(chan []byte, 64)

	h.mu.Lock()
	h.clients[msgCh] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, msgCh)
		h.mu.Unlock()
	}()

	// Initial heartbeat
	_, _ = w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			if _, err := w.Write(msg); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
