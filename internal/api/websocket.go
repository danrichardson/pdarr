package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/danrichardson/sqzarr/internal/queue"
)

// wsHub manages WebSocket connections and broadcasts events.
type wsHub struct {
	mu      sync.RWMutex
	clients map[*wsClient]struct{}
	eventCh chan queue.Event
	log     *slog.Logger
}

type wsClient struct {
	send chan []byte
	done chan struct{}
}

func newWSHub() *wsHub {
	return &wsHub{
		clients: make(map[*wsClient]struct{}),
		eventCh: make(chan queue.Event, 64),
	}
}

func (h *wsHub) broadcast(e queue.Event) {
	select {
	case h.eventCh <- e:
	default:
		// Drop if channel is full — non-critical.
	}
}

func (h *wsHub) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.mu.Lock()
			for c := range h.clients {
				close(c.done)
			}
			h.mu.Unlock()
			return
		case e := <-h.eventCh:
			data, err := json.Marshal(e)
			if err != nil {
				continue
			}
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.send <- data:
				default:
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *wsHub) addClient(c *wsClient) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *wsHub) removeClient(c *wsClient) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

// handleWebSocket upgrades the connection to a simple Server-Sent Events stream.
// Using SSE instead of WebSocket avoids an external dependency (gorilla/websocket)
// while supporting the same use case: one-way server→client events.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	client := &wsClient{
		send: make(chan []byte, 32),
		done: make(chan struct{}),
	}
	s.hub.addClient(client)
	defer s.hub.removeClient(client)

	// Send a heartbeat comment so the client knows we're alive.
	if _, err := w.Write([]byte(": connected\n\n")); err != nil {
		return
	}
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-client.done:
			return
		case msg := <-client.send:
			if _, err := w.Write(append([]byte("data: "), append(msg, '\n', '\n')...)); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
