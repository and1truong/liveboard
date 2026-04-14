package v1

import (
	"encoding/json"
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
		writeError(w, err)
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
