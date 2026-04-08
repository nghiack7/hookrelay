package sse

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
)

type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan []byte]struct{} // endpointID -> set of channels
}

func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[string]map[chan []byte]struct{}),
	}
}

func (h *Hub) Subscribe(endpointID string) chan []byte {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan []byte, 16)
	if h.subscribers[endpointID] == nil {
		h.subscribers[endpointID] = make(map[chan []byte]struct{})
	}
	h.subscribers[endpointID][ch] = struct{}{}
	return ch
}

func (h *Hub) Unsubscribe(endpointID string, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subs, ok := h.subscribers[endpointID]; ok {
		delete(subs, ch)
		if len(subs) == 0 {
			delete(h.subscribers, endpointID)
		}
	}
	close(ch)
}

func (h *Hub) Broadcast(endpointID string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		slog.Error("sse marshal", "error", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.subscribers[endpointID] {
		select {
		case ch <- payload:
		default:
			// slow consumer, drop message
		}
	}
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request, endpointID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := h.Subscribe(endpointID)
	defer h.Unsubscribe(endpointID, ch)

	// Send initial keepalive
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}
