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
	slug := chi.URLParam(r, "*")
	board, err := d.Workspace.LoadBoard(slug)
	if err != nil {
		writeError(w, err)
		return
	}
	global := web.LoadSettingsFromDir(d.Workspace.Dir)
	resolved := web.ResolveSettings(global, board.Settings)
	_ = json.NewEncoder(w).Encode(resolved)
}

// putBoardSettings applies a partial update to a board's per-board settings
// overrides. Non-nil fields in the payload replace existing overrides; absent
// fields are left untouched. This matches the TS Partial<BoardSettings>
// contract and the LocalAdapter's { ...existing, ...patch } semantics.
// Invalid enum values in the payload are dropped by SanitizeBoardSettings
// before merge.
func (d Deps) putBoardSettings(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	slug := chi.URLParam(r, "*")
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
	web.SanitizeBoardSettings(&patch)
	if err := d.Engine.PatchBoardSettings(path, patch); err != nil {
		writeError(w, err)
		return
	}
	if d.SSE != nil {
		d.SSE.Publish(slug)
	}
	w.WriteHeader(http.StatusNoContent)
}
