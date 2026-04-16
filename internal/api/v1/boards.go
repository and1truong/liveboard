package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/and1truong/liveboard/pkg/models"
)

// boardSummary is the v1 JSON shape for a board identity returned by
// create/rename. Keys match the renderer's BoardSummary type.
type boardSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Icon    string `json:"icon,omitempty"`
	Version int    `json:"version"`
}

// boardFileSlug returns the canonical board id: the filename stem.
// LoadBoard / BoardPath identify boards by filename, not frontmatter name —
// using b.Name for `id` breaks getBoard whenever filename != name.
func boardFileSlug(b *models.Board) string {
	if b.FilePath == "" {
		return b.Name
	}
	return strings.TrimSuffix(filepath.Base(b.FilePath), ".md")
}

func toBoardSummary(b *models.Board) boardSummary {
	return boardSummary{ID: boardFileSlug(b), Name: b.Name, Icon: b.Icon, Version: b.Version}
}

func (d Deps) getBoard(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	board, err := d.Workspace.LoadBoard(slug)
	if err != nil {
		writeError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(board)
}

func (d Deps) createBoard(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, fmt.Errorf("%w: %v", errInvalid, err))
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, fmt.Errorf("%w: name required", errInvalid))
		return
	}
	b, err := d.Workspace.CreateBoard(name)
	if err != nil {
		writeError(w, err)
		return
	}
	if d.SSE != nil {
		d.SSE.PublishBoardList()
	}
	if d.Search != nil && b != nil {
		_ = d.Search.UpdateBoard(b.Name, b)
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toBoardSummary(b))
}

func (d Deps) renameBoard(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	var req struct {
		NewName string `json:"new_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, fmt.Errorf("%w: %v", errInvalid, err))
		return
	}
	b, err := d.Workspace.RenameBoard(slug, req.NewName)
	if err != nil {
		writeError(w, err)
		return
	}
	if d.SSE != nil {
		d.SSE.PublishBoardList()
	}
	if d.Search != nil && b != nil {
		if slug != b.Name {
			_ = d.Search.DeleteBoard(slug)
		}
		_ = d.Search.UpdateBoard(b.Name, b)
	}
	_ = json.NewEncoder(w).Encode(toBoardSummary(b))
}

func (d Deps) deleteBoard(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if err := d.Workspace.DeleteBoard(slug); err != nil {
		writeError(w, err)
		return
	}
	if d.SSE != nil {
		d.SSE.PublishBoardList()
	}
	if d.Search != nil {
		_ = d.Search.DeleteBoard(slug)
	}
	w.WriteHeader(http.StatusNoContent)
}

// boardListLiteEntry is a lightweight board descriptor for cascading selects
// in the renderer (e.g. move-to-board picker).
type boardListLiteEntry struct {
	Slug    string   `json:"slug"`
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
}

func (d Deps) listBoardsLite(w http.ResponseWriter, _ *http.Request) {
	boards, err := d.Workspace.ListBoards()
	if err != nil {
		writeError(w, err)
		return
	}
	entries := make([]boardListLiteEntry, 0, len(boards))
	for i := range boards {
		b := &boards[i]
		cols := make([]string, 0, len(b.Columns))
		for _, c := range b.Columns {
			cols = append(cols, c.Name)
		}
		entries = append(entries, boardListLiteEntry{Slug: boardFileSlug(b), Name: b.Name, Columns: cols})
	}
	_ = json.NewEncoder(w).Encode(entries)
}

func (d Deps) listBoards(w http.ResponseWriter, _ *http.Request) {
	boards, err := d.Workspace.ListBoards()
	if err != nil {
		writeError(w, err)
		return
	}
	summaries := make([]boardSummary, 0, len(boards))
	for i := range boards {
		summaries = append(summaries, toBoardSummary(&boards[i]))
	}
	_ = json.NewEncoder(w).Encode(summaries)
}
