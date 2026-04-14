package v1

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (d Deps) getBoard(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	board, err := d.Workspace.LoadBoard(slug)
	if err != nil {
		writeError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(board)
}

func (d Deps) listBoards(w http.ResponseWriter, _ *http.Request) {
	boards, err := d.Workspace.ListBoards()
	if err != nil {
		writeError(w, err)
		return
	}
	if boards == nil {
		// Ensure we always emit `[]` rather than `null` for empty workspaces.
		_, _ = w.Write([]byte("[]\n"))
		return
	}
	_ = json.NewEncoder(w).Encode(boards)
}
