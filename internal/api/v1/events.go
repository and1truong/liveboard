package v1

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// getEvents serves a workspace-wide SSE stream, fanning in per-board
// `board.updated` events and workspace-list `board.list.updated` events.
func (d Deps) getEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	listCh, cancelList := d.SSE.SubscribeBoardList()
	defer cancelList()
	boardCh, cancelBoards := d.SSE.SubscribeAllBoards()
	defer cancelBoards()

	_, _ = fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			_, _ = fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case ev, ok := <-listCh:
			if !ok {
				return
			}
			_, _ = fmt.Fprintf(w, "event: %s\ndata: null\n\n", ev.Type)
			flusher.Flush()
		case ev, ok := <-boardCh:
			if !ok {
				return
			}
			slug := ev.Payload
			version := 0
			if b, err := d.Workspace.LoadBoard(slug); err != nil {
				log.Printf("api/v1/events: load %q failed: %v", slug, err)
			} else {
				version = b.Version
			}
			payload, _ := json.Marshal(struct {
				BoardID string `json:"board_id"`
				Version int    `json:"version"`
			}{BoardID: slug, Version: version})
			_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, payload)
			flusher.Flush()
		}
	}
}
