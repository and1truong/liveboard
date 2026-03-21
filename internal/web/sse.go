package web

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
)

// SSEBroker manages Server-Sent Events connections for real-time board updates.
type SSEBroker struct {
	mu      sync.RWMutex
	clients map[string]map[chan string]struct{} // boardSlug -> set of channels
}

// NewSSEBroker creates a new SSE broker.
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients: make(map[string]map[chan string]struct{}),
	}
}

// Subscribe registers a channel to receive events for a board slug.
func (b *SSEBroker) Subscribe(slug string) chan string {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan string, 1)
	if b.clients[slug] == nil {
		b.clients[slug] = make(map[chan string]struct{})
	}
	b.clients[slug][ch] = struct{}{}
	return ch
}

// Unsubscribe removes a channel from a board slug's subscribers.
func (b *SSEBroker) Unsubscribe(slug string, ch chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subs, ok := b.clients[slug]; ok {
		delete(subs, ch)
		if len(subs) == 0 {
			delete(b.clients, slug)
		}
	}
}

// Publish sends a notification to all subscribers of a board slug.
func (b *SSEBroker) Publish(slug string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if subs, ok := b.clients[slug]; ok {
		for ch := range subs {
			select {
			case ch <- slug:
			default:
				// Drop if channel is full (non-blocking)
			}
		}
	}
}

// ServeHTTP handles SSE connections for board updates.
func (b *SSEBroker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		http.Error(w, "board slug required", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := b.Subscribe(slug)
	defer b.Unsubscribe(slug, ch)

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: ok\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			fmt.Fprintf(w, "event: board-update\ndata: refresh\n\n")
			flusher.Flush()
		}
	}
}
