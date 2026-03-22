package web

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/and1truong/liveboard/pkg/models"
)

// BoardListModel is the state for the board list page.
type BoardListModel struct {
	LayoutSettings
	Title     string         `json:"title"`
	SiteName  string         `json:"site_name"`
	Boards    []BoardSummary `json:"boards"`
	AllTags   []string       `json:"all_tags,omitempty"`
	BoardSlug string         `json:"board_slug"` // always empty; shared with layout template
	Error     string         `json:"error,omitempty"`
}

// BoardSummary represents a simplified board for list display.
type BoardSummary struct {
	Name        string    `json:"name"`
	Slug        string    `json:"slug"` // filename stem, used for URLs
	Description string    `json:"description,omitempty"`
	Icon        string    `json:"icon,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Pinned      bool      `json:"pinned"`
	CardCount   int       `json:"card_count"`
	DoneCount   int       `json:"done_count"`
	ColumnCount int       `json:"column_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedAgo  string    `json:"created_ago"`
	UpdatedAgo  string    `json:"updated_ago"`
	CreatedFull string    `json:"created_full"`
	UpdatedFull string    `json:"updated_full"`
}

// boardSlug extracts the filename stem from a board's FilePath.
func boardSlug(b models.Board) string {
	return strings.TrimSuffix(filepath.Base(b.FilePath), ".md")
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

// toBoardSummaries converts a slice of boards to BoardSummary.
func toBoardSummaries(boards []models.Board) []BoardSummary {
	summaries := make([]BoardSummary, len(boards))
	for i, b := range boards {
		cardCount := 0
		doneCount := 0
		for _, col := range b.Columns {
			cardCount += len(col.Cards)
			for _, c := range col.Cards {
				if c.Completed {
					doneCount++
				}
			}
		}
		summaries[i] = BoardSummary{
			Name:        b.Name,
			Slug:        boardSlug(b),
			Description: b.Description,
			Icon:        b.Icon,
			Tags:        b.Tags,
			CardCount:   cardCount,
			DoneCount:   doneCount,
			ColumnCount: len(b.Columns),
			CreatedAt:   b.CreatedAt,
			UpdatedAt:   b.UpdatedAt,
			CreatedAgo:  relativeTime(b.CreatedAt),
			UpdatedAgo:  relativeTime(b.UpdatedAt),
			CreatedFull: b.CreatedAt.Format("Created: Jan 2, 2006 3:04 PM"),
			UpdatedFull: b.UpdatedAt.Format("Updated: Jan 2, 2006 3:04 PM"),
		}
	}
	return summaries
}

// collectAllTags returns sorted unique tags across all board summaries.
func collectAllTags(boards []BoardSummary) []string {
	seen := make(map[string]struct{})
	for _, b := range boards {
		for _, t := range b.Tags {
			seen[t] = struct{}{}
		}
	}
	tags := make([]string, 0, len(seen))
	for t := range seen {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

// sortBoardsWithPins marks pinned boards and sorts them first (in pin order),
// then unpinned boards alphabetically by name.
func sortBoardsWithPins(boards []BoardSummary, pinned []string) []BoardSummary {
	pinIndex := make(map[string]int, len(pinned))
	for i, slug := range pinned {
		pinIndex[slug] = i
	}
	for i := range boards {
		_, boards[i].Pinned = pinIndex[boards[i].Slug]
	}
	sort.SliceStable(boards, func(i, j int) bool {
		pi, iPinned := pinIndex[boards[i].Slug]
		pj, jPinned := pinIndex[boards[j].Slug]
		if iPinned && jPinned {
			return pi < pj
		}
		if iPinned != jPinned {
			return iPinned
		}
		return strings.ToLower(boards[i].Name) < strings.ToLower(boards[j].Name)
	})
	return boards
}

// boardListModel loads the board list and returns a populated BoardListModel.
func (h *Handler) boardListModel() (BoardListModel, error) {
	boards, err := h.ws.ListBoards()
	if err != nil {
		return BoardListModel{Error: err.Error()}, nil
	}
	settings := h.loadSettings()
	summaries := toBoardSummaries(boards)
	summaries = sortBoardsWithPins(summaries, settings.PinnedBoards)
	return BoardListModel{LayoutSettings: layoutSettingsFrom(settings), Title: settings.SiteName, SiteName: settings.SiteName, Boards: summaries, AllTags: collectAllTags(summaries)}, nil
}

// BoardListPage handles GET / — renders the full board list page.
func (h *Handler) BoardListPage(w http.ResponseWriter, _ *http.Request) {
	model, _ := h.boardListModel()
	renderFullPage(w, h.boardListTpl, model)
}

// HandleCreateBoard handles POST /boards/new — creates a board and returns the boards grid partial.
func (h *Handler) HandleCreateBoard(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if name == "" {
		model, _ := h.boardListModel()
		model.Error = "Board name is required"
		renderPartial(w, h.boardGridTpl, "boards-grid", model)
		return
	}

	if _, err := h.ws.CreateBoard(name); err != nil {
		model, _ := h.boardListModel()
		model.Error = err.Error()
		renderPartial(w, h.boardGridTpl, "boards-grid", model)
		return
	}

	w.Header().Set("HX-Redirect", "/board/"+name)
}

// HandleDeleteBoard handles POST /boards/{slug}/delete — deletes a board.
func (h *Handler) HandleDeleteBoard(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if name == "" {
		model, _ := h.boardListModel()
		model.Error = "Board name is required"
		renderPartial(w, h.boardGridTpl, "boards-grid", model)
		return
	}

	if err := h.ws.DeleteBoard(name); err != nil {
		model, _ := h.boardListModel()
		model.Error = err.Error()
		renderPartial(w, h.boardGridTpl, "boards-grid", model)
		return
	}

	model, _ := h.boardListModel()
	renderPartial(w, h.boardGridTpl, "boards-grid", model)
}

// HandleSetBoardIconList handles POST /boards/{slug}/icon from the board list page.
func (h *Handler) HandleSetBoardIconList(w http.ResponseWriter, r *http.Request) {
	slug := r.FormValue("name")
	if slug == "" {
		model, _ := h.boardListModel()
		model.Error = "Board name is required"
		renderPartial(w, h.boardGridTpl, "boards-grid", model)
		return
	}

	icon := r.FormValue("icon")

	boardPath := h.ws.BoardPath(slug)
	if err := h.eng.UpdateBoardIcon(boardPath, icon); err != nil {
		model, _ := h.boardListModel()
		model.Error = err.Error()
		renderPartial(w, h.boardGridTpl, "boards-grid", model)
		return
	}
	model, _ := h.boardListModel()
	renderPartial(w, h.boardGridTpl, "boards-grid", model)
}

// HandleTogglePin handles POST /api/boards/pin — toggles a board's pinned state.
func (h *Handler) HandleTogglePin(w http.ResponseWriter, r *http.Request) {
	slug := r.FormValue("slug")
	if slug == "" {
		http.Error(w, "slug is required", http.StatusBadRequest)
		return
	}

	s := h.loadSettings()
	found := false
	for i, p := range s.PinnedBoards {
		if p == slug {
			s.PinnedBoards = append(s.PinnedBoards[:i], s.PinnedBoards[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		s.PinnedBoards = append(s.PinnedBoards, slug)
	}
	if err := h.saveSettings(s); err != nil {
		http.Error(w, "save failed", http.StatusInternalServerError)
		return
	}

	model, _ := h.boardListModel()
	renderPartial(w, h.sidebarBoardsTpl, "sidebar-boards", model)
}

// HandleSidebarBoards handles GET /api/boards/sidebar — returns the sidebar board list partial.
func (h *Handler) HandleSidebarBoards(w http.ResponseWriter, r *http.Request) {
	model, _ := h.boardListModel()
	if slug := r.URL.Query().Get("slug"); slug != "" {
		model.BoardSlug = slug
	}
	renderPartial(w, h.sidebarBoardsTpl, "sidebar-boards", model)
}
