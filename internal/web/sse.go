package web

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
)

// SSEBroker manages Server-Sent Events connections for real-time board updates.
type SSEBroker struct {
	mu        sync.RWMutex
	clients   map[string]map[chan string]struct{} // boardSlug -> set of channels
	global    map[chan SSEEvent]struct{}          // global subscribers (for reminders)
	boardList map[chan SSEEvent]struct{}          // workspace-list subscribers (board.list.updated)
	allBoards map[chan SSEEvent]struct{}          // workspace-wide board.updated subscribers
}

// SSEEvent carries an event type and JSON payload for global SSE.
type SSEEvent struct {
	Type    string
	Payload string
}

// NewSSEBroker creates a new SSE broker.
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients:   make(map[string]map[chan string]struct{}),
		global:    make(map[chan SSEEvent]struct{}),
		boardList: make(map[chan SSEEvent]struct{}),
		allBoards: make(map[chan SSEEvent]struct{}),
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

// SubscribeGlobal registers a channel to receive global events (e.g. reminders).
func (b *SSEBroker) SubscribeGlobal() chan SSEEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan SSEEvent, 4)
	b.global[ch] = struct{}{}
	return ch
}

// UnsubscribeGlobal removes a global subscriber.
func (b *SSEBroker) UnsubscribeGlobal(ch chan SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.global, ch)
}

// PublishGlobal sends an event to all global subscribers.
func (b *SSEBroker) PublishGlobal(event SSEEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.global {
		select {
		case ch <- event:
		default:
		}
	}
}

// Shutdown closes all subscriber channels so SSE handlers exit promptly.
// Must be called before http.Server.Shutdown to unblock long-lived connections.
func (b *SSEBroker) Shutdown() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for slug, subs := range b.clients {
		for ch := range subs {
			close(ch)
		}
		delete(b.clients, slug)
	}
	for ch := range b.global {
		close(ch)
		delete(b.global, ch)
	}
	for ch := range b.boardList {
		close(ch)
		delete(b.boardList, ch)
	}
	for ch := range b.allBoards {
		close(ch)
		delete(b.allBoards, ch)
	}
}

// Publish sends a notification to all subscribers of a board slug, and to
// workspace-wide all-boards subscribers.
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
	for ch := range b.allBoards {
		select {
		case ch <- SSEEvent{Type: "board.updated", Payload: slug}:
		default:
		}
	}
}

// SubscribeBoardList registers a channel for workspace-list events
// (board.list.updated). The returned cancel func unregisters and closes the channel.
func (b *SSEBroker) SubscribeBoardList() (chan SSEEvent, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan SSEEvent, 8)
	b.boardList[ch] = struct{}{}
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.boardList[ch]; ok {
			delete(b.boardList, ch)
			close(ch)
		}
	}
}

// PublishBoardList fans out a board.list.updated event to all workspace-list subscribers.
func (b *SSEBroker) PublishBoardList() {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.boardList {
		select {
		case ch <- SSEEvent{Type: "board.list.updated"}:
		default:
		}
	}
}

// SubscribeAllBoards registers a channel that receives every board.updated event
// regardless of slug. The returned cancel func unregisters and closes the channel.
func (b *SSEBroker) SubscribeAllBoards() (chan SSEEvent, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan SSEEvent, 32)
	b.allBoards[ch] = struct{}{}
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.allBoards[ch]; ok {
			delete(b.allBoards, ch)
			close(ch)
		}
	}
}

// ServeGlobalSSE handles SSE connections for global events (reminders).
func (b *SSEBroker) ServeGlobalSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := b.SubscribeGlobal()
	defer b.UnsubscribeGlobal(ch)

	_, _ = fmt.Fprintf(w, "event: connected\ndata: ok\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, evt.Payload)
			flusher.Flush()
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
	_, _ = fmt.Fprintf(w, "event: connected\ndata: ok\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case _, ok := <-ch:
			if !ok {
				// Channel closed by Shutdown — exit cleanly.
				return
			}
			_, _ = fmt.Fprintf(w, "event: board-update\ndata: refresh\n\n")
			flusher.Flush()
		}
	}
}
