package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/and1truong/liveboard/internal/web"
	"github.com/and1truong/liveboard/pkg/models"
)

// boardSummary is the v1 JSON shape for a board identity returned by
// create/rename. Keys match the renderer's BoardSummary type.
type boardSummary struct {
	ID          string   `json:"id"`
	Folder      string   `json:"folder,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	IconColor   string   `json:"icon_color,omitempty"`
	Version     int      `json:"version"`
	Tags        []string `json:"tags,omitempty"`
	UpdatedAgo  string   `json:"updatedAgo,omitempty"`
	CardCount   int      `json:"cardCount,omitempty"`
	DoneCount   int      `json:"doneCount,omitempty"`
	Pinned      bool     `json:"pinned,omitempty"`
}

// relativeTime returns a human-readable relative time string.
func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}

// boardFileSlug returns the canonical board id: the path relative to the
// workspace dir, without the .md suffix, using forward slashes.
// Root-level boards: "ideas". Nested: "work/ideas".
// LoadBoard / BoardPath identify boards by this id.
func boardFileSlug(workspaceDir string, b *models.Board) string {
	if b.FilePath == "" {
		return b.Name
	}
	rel, err := filepath.Rel(workspaceDir, b.FilePath)
	if err != nil {
		return strings.TrimSuffix(filepath.Base(b.FilePath), ".md")
	}
	return strings.TrimSuffix(filepath.ToSlash(rel), ".md")
}

// splitBoardID returns the folder (may be "") and the file stem of a board id.
func splitBoardID(id string) (folder, name string) {
	i := strings.LastIndex(id, "/")
	if i < 0 {
		return "", id
	}
	return id[:i], id[i+1:]
}

// boardPathParam extracts the catch-all board id from the URL (the "*" param).
func boardPathParam(r *http.Request) string {
	return chi.URLParam(r, "*")
}

func (d Deps) toBoardSummary(b *models.Board) boardSummary {
	cardCount, doneCount := 0, 0
	for _, col := range b.Columns {
		for _, card := range col.Cards {
			cardCount++
			if card.Completed {
				doneCount++
			}
		}
	}
	id := boardFileSlug(d.Dir, b)
	folder, _ := splitBoardID(id)
	return boardSummary{
		ID:          id,
		Folder:      folder,
		Name:        b.Name,
		Description: b.Description,
		Icon:        b.Icon,
		IconColor:   b.IconColor,
		Version:     b.Version,
		Tags:        b.Tags,
		UpdatedAgo:  relativeTime(b.UpdatedAt),
		CardCount:   cardCount,
		DoneCount:   doneCount,
	}
}

func (d Deps) getBoard(w http.ResponseWriter, r *http.Request) {
	id := boardPathParam(r)
	board, err := d.Workspace.LoadBoard(id)
	if err != nil {
		writeError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(board)
}

// composeBoardID joins an optional folder with a bare name.
func composeBoardID(folder, name string) string {
	folder = strings.Trim(folder, "/")
	name = strings.TrimSpace(name)
	if folder == "" {
		return name
	}
	return folder + "/" + name
}

func (d Deps) createBoard(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		Folder string `json:"folder,omitempty"`
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
	id := composeBoardID(req.Folder, name)
	b, err := d.Workspace.CreateBoard(id)
	if err != nil {
		writeError(w, err)
		return
	}
	if d.SSE != nil {
		d.SSE.PublishBoardList()
	}
	if d.Search != nil && b != nil {
		_ = d.Search.UpdateBoard(boardFileSlug(d.Dir, b), b)
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(d.toBoardSummary(b))
}

func (d Deps) renameBoard(w http.ResponseWriter, r *http.Request) {
	id := boardPathParam(r)
	var req struct {
		NewName string  `json:"new_name"`
		Folder  *string `json:"folder,omitempty"` // optional: move to this folder
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, fmt.Errorf("%w: %v", errInvalid, err))
		return
	}
	newName := strings.TrimSpace(req.NewName)
	if newName == "" {
		writeError(w, fmt.Errorf("%w: new_name required", errInvalid))
		return
	}
	// If folder is provided, use it; otherwise keep the current folder.
	var folder string
	if req.Folder != nil {
		folder = *req.Folder
	} else {
		folder, _ = splitBoardID(id)
	}
	newID := composeBoardID(folder, newName)
	b, err := d.Workspace.RenameBoard(id, newID)
	if err != nil {
		writeError(w, err)
		return
	}
	// Rewrite pins so any pin pointing at the old id survives.
	if id != newID {
		_ = web.RewritePinsOnRename(d.Dir, id, newID)
	}
	if d.SSE != nil {
		d.SSE.PublishBoardList()
	}
	if d.Search != nil && b != nil {
		if id != newID {
			_ = d.Search.DeleteBoard(id)
		}
		_ = d.Search.UpdateBoard(boardFileSlug(d.Dir, b), b)
	}
	_ = json.NewEncoder(w).Encode(d.toBoardSummary(b))
}

func (d Deps) deleteBoard(w http.ResponseWriter, r *http.Request) {
	id := boardPathParam(r)
	if err := d.Workspace.DeleteBoard(id); err != nil {
		writeError(w, err)
		return
	}
	if d.SSE != nil {
		d.SSE.PublishBoardList()
	}
	if d.Search != nil {
		_ = d.Search.DeleteBoard(id)
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
		entries = append(entries, boardListLiteEntry{Slug: boardFileSlug(d.Dir, b), Name: b.Name, Columns: cols})
	}
	_ = json.NewEncoder(w).Encode(entries)
}

func (d Deps) listBoards(w http.ResponseWriter, _ *http.Request) {
	boards, err := d.Workspace.ListBoards()
	if err != nil {
		writeError(w, err)
		return
	}

	settings := web.LoadSettingsFromDir(d.Dir)
	pinnedIdx := make(map[string]int, len(settings.PinnedBoards))
	for i, slug := range settings.PinnedBoards {
		pinnedIdx[slug] = i
	}

	summaries := make([]boardSummary, 0, len(boards))
	for i := range boards {
		s := d.toBoardSummary(&boards[i])
		if _, ok := pinnedIdx[s.ID]; ok {
			s.Pinned = true
		}
		summaries = append(summaries, s)
	}

	sort.SliceStable(summaries, func(i, j int) bool {
		pi, iok := pinnedIdx[summaries[i].ID]
		pj, jok := pinnedIdx[summaries[j].ID]
		if iok && jok {
			return pi < pj
		}
		if iok {
			return true
		}
		if jok {
			return false
		}
		// Group by folder first (root boards first, then folders alphabetically),
		// then by name within each group.
		if summaries[i].Folder != summaries[j].Folder {
			return summaries[i].Folder < summaries[j].Folder
		}
		return summaries[i].Name < summaries[j].Name
	})

	_ = json.NewEncoder(w).Encode(summaries)
}

func (d Deps) toggleBoardPin(w http.ResponseWriter, r *http.Request) {
	id := boardPathParam(r)
	err := web.MutateSettings(d.Dir, func(s *web.AppSettings) {
		found := false
		filtered := s.PinnedBoards[:0]
		for _, p := range s.PinnedBoards {
			if p == id {
				found = true
			} else {
				filtered = append(filtered, p)
			}
		}
		if found {
			s.PinnedBoards = filtered
		} else {
			s.PinnedBoards = append(s.PinnedBoards, id)
		}
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if d.SSE != nil {
		d.SSE.PublishBoardList()
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- folder CRUD ---

func (d Deps) listFolders(w http.ResponseWriter, _ *http.Request) {
	folders, err := d.Workspace.ListFolders()
	if err != nil {
		writeError(w, err)
		return
	}
	if folders == nil {
		folders = []string{}
	}
	_ = json.NewEncoder(w).Encode(folders)
}

func (d Deps) createFolder(w http.ResponseWriter, r *http.Request) {
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
	if err := d.Workspace.CreateFolder(name); err != nil {
		writeError(w, err)
		return
	}
	if d.SSE != nil {
		d.SSE.PublishBoardList()
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(struct {
		Name string `json:"name"`
	}{Name: name})
}

func (d Deps) renameFolder(w http.ResponseWriter, r *http.Request) {
	oldName := boardPathParam(r)
	var req struct {
		NewName string `json:"new_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, fmt.Errorf("%w: %v", errInvalid, err))
		return
	}
	newName := strings.TrimSpace(req.NewName)
	if newName == "" {
		writeError(w, fmt.Errorf("%w: new_name required", errInvalid))
		return
	}
	if err := d.Workspace.RenameFolder(oldName, newName); err != nil {
		writeError(w, err)
		return
	}
	// Rewrite any pins that pointed into the old folder.
	_ = web.RewritePinsOnFolderRename(d.Dir, oldName, newName)
	if d.SSE != nil {
		d.SSE.PublishBoardList()
	}
	w.WriteHeader(http.StatusNoContent)
}

func (d Deps) deleteFolder(w http.ResponseWriter, r *http.Request) {
	name := boardPathParam(r)
	if err := d.Workspace.DeleteFolder(name); err != nil {
		writeError(w, err)
		return
	}
	if d.SSE != nil {
		d.SSE.PublishBoardList()
	}
	w.WriteHeader(http.StatusNoContent)
}
