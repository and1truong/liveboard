package v1

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func (d Deps) getEvents(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("board")
	if slug == "" {
		writeError(w, fmt.Errorf("%w: board query param required", errInvalid))
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

	ch := d.SSE.Subscribe(slug)
	defer d.SSE.Unsubscribe(slug, ch)

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
		case _, ok := <-ch:
			if !ok {
				return
			}
			version := 0
			if board, err := d.Workspace.LoadBoard(slug); err != nil {
				log.Printf("api/v1/events: load %q failed: %v", slug, err)
			} else {
				version = board.Version
			}
			payload, _ := json.Marshal(struct {
				BoardID string `json:"board_id"`
				Version int    `json:"version"`
			}{BoardID: slug, Version: version})
			_, _ = fmt.Fprintf(w, "event: board.updated\ndata: %s\n\n", payload)
			flusher.Flush()
		}
	}
}
