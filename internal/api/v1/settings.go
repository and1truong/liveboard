package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/and1truong/liveboard/internal/web"
	"github.com/and1truong/liveboard/pkg/models"
)

func (d Deps) getBoardSettings(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	board, err := d.Workspace.LoadBoard(slug)
	if err != nil {
		writeError(w, err)
		return
	}
	global := web.LoadSettingsFromDir(d.Workspace.Dir)
	resolved := web.ResolveSettings(global, board.Settings)
	_ = json.NewEncoder(w).Encode(resolved)
}

// putBoardSettings replaces the board's per-board settings overrides with the
// supplied payload (true replace, not a merge). Fields absent from the payload
// will be nil in BoardSettings, which clears any existing per-board override for
// that field so it falls back to the global default at resolve time.
func (d Deps) putBoardSettings(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	slug := chi.URLParam(r, "slug")
	path, err := d.Workspace.BoardPath(slug)
	if err != nil {
		writeError(w, err)
		return
	}
	var patch models.BoardSettings
	if err := decodeJSON(r, &patch); err != nil {
		writeError(w, fmt.Errorf("%w: %v", errInvalid, err))
		return
	}
	if err := d.Engine.UpdateBoardSettings(path, patch); err != nil {
		writeError(w, err)
		return
	}
	if d.SSE != nil {
		d.SSE.Publish(slug)
	}
	w.WriteHeader(http.StatusNoContent)
}
