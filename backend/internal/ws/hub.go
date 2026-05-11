package wshub

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"xray2wg/backend/internal/telemetry"

	"github.com/gorilla/websocket"
)

type TunnelStats struct {
	ID     int64 `json:"id"`
	RxRate int64 `json:"rx_rate"`
	TxRate int64 `json:"tx_rate"`
}

type Message struct {
	Type      string        `json:"type"`
	Tunnels   []TunnelStats `json:"tunnels"`
	Timestamp int64         `json:"ts"`
}

// Hub fans out stats to WebSocket clients. Call Shutdown then Close on process exit.
type Hub struct {
	ctx    context.Context
	cancel context.CancelFunc

	shutdownOnce sync.Once

	wgRun sync.WaitGroup

	mu         sync.Mutex
	clients    map[*websocket.Conn]*time.Timer
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  chan Message
}

// NewHub starts the hub loop; cancel parent (or call Shutdown) to stop.
func NewHub(parent context.Context) *Hub {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	h := &Hub{
		ctx:        ctx,
		cancel:     cancel,
		clients:    make(map[*websocket.Conn]*time.Timer),
		register:   make(chan *websocket.Conn, 32),
		unregister: make(chan *websocket.Conn, 32),
		broadcast:  make(chan Message, 64),
	}
	h.wgRun.Add(1)
	go h.run()
	return h
}

func (h *Hub) syncWSGaugeLocked() {
	if telemetry.ActiveWSConns == nil {
		return
	}
	if h.clients == nil {
		telemetry.ActiveWSConns.Set(0)
		return
	}
	telemetry.ActiveWSConns.Set(float64(len(h.clients)))
}

func (h *Hub) run() {
	defer h.wgRun.Done()
	for {
		select {
		case <-h.ctx.Done():
			h.shutdown()
			return
		case c := <-h.register:
			h.mu.Lock()
			if h.clients != nil {
				h.clients[c] = nil
			}
			h.syncWSGaugeLocked()
			h.mu.Unlock()
		case c := <-h.unregister:
			h.mu.Lock()
			if h.clients != nil {
				delete(h.clients, c)
			}
			h.syncWSGaugeLocked()
			h.mu.Unlock()
			_ = c.Close()
		case m := <-h.broadcast:
			m.Type = "stats_update"
			b, err := json.Marshal(m)
			if err != nil {
				continue
			}
			h.mu.Lock()
			for c := range h.clients {
				_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if err := c.WriteMessage(websocket.TextMessage, b); err != nil {
					_ = c.Close()
					delete(h.clients, c)
				}
			}
			h.syncWSGaugeLocked()
			h.mu.Unlock()
		}
	}
}

func (h *Hub) shutdown() {
	h.mu.Lock()
	for c := range h.clients {
		_ = c.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutting down"),
			time.Now().Add(time.Second))
		_ = c.Close()
		delete(h.clients, c)
	}
	h.clients = nil
	h.syncWSGaugeLocked()
	h.mu.Unlock()
}

// Shutdown stops accepting work and signals the run loop to exit (idempotent).
func (h *Hub) Shutdown() {
	h.shutdownOnce.Do(func() {
		if h.cancel != nil {
			h.cancel()
		}
	})
}

// Run blocks until the hub fan-out goroutine exits (parent context cancelled, Shutdown+Close, or loop return).
// It is equivalent to waiting on the internal run loop completion counter.
func (h *Hub) Run() {
	h.wgRun.Wait()
}

// Close waits for the hub goroutine to finish, up to 5 seconds.
func (h *Hub) Close() error {
	h.Shutdown()
	done := make(chan struct{})
	go func() {
		h.wgRun.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("hub: close timeout")
	}
}

func (h *Hub) Register(c *websocket.Conn) {
	select {
	case <-h.ctx.Done():
		_ = c.Close()
		return
	case h.register <- c:
	}
}

func (h *Hub) Unregister(c *websocket.Conn) {
	select {
	case <-h.ctx.Done():
		return
	case h.unregister <- c:
	}
}

func (h *Hub) Broadcast(m Message) {
	select {
	case <-h.ctx.Done():
		return
	case h.broadcast <- m:
	default:
	}
}
