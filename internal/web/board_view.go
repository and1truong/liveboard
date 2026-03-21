package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/and1truong/liveboard/pkg/models"
)

// ResolvedSettings holds the effective settings for a board view,
// merging global defaults with per-board overrides.
type ResolvedSettings struct {
	ShowCheckbox   bool   `json:"show_checkbox"`
	NewLineTrigger string `json:"newline_trigger"`
	CardPosition   string `json:"card_position"`
	ExpandColumns  bool   `json:"expand_columns"`
	ViewMode       string `json:"view_mode"`
}

// BoardSettingsView holds pre-formatted per-board override values for the template.
// Empty string means "not set" (inherit global).
type BoardSettingsView struct {
	ShowCheckbox  string `json:"show_checkbox"`
	CardPosition  string `json:"card_position"`
	ExpandColumns string `json:"expand_columns"`
	ViewMode      string `json:"view_mode"`
}

// BoardViewModel is the state for the board view page.
type BoardViewModel struct {
	Title          string            `json:"title"`
	SiteName       string            `json:"site_name"`
	Board          *models.Board     `json:"board"`
	BoardName      string            `json:"board_name"`
	BoardSlug      string            `json:"board_slug"`
	Boards         []BoardSummary    `json:"boards"`
	Error          string            `json:"error,omitempty"`
	Settings       ResolvedSettings  `json:"settings"`
	BSView         BoardSettingsView `json:"bs_view"`
	GlobalSettings AppSettings       `json:"global_settings"`
}

// resolveSettings merges global defaults with per-board overrides.
func resolveSettings(global AppSettings, bs models.BoardSettings) ResolvedSettings {
	rs := ResolvedSettings{
		ShowCheckbox:   global.ShowCheckbox,
		NewLineTrigger: global.NewLineTrigger,
		CardPosition:   global.CardPosition,
		ExpandColumns:  false,
		ViewMode:       "board",
	}
	if bs.ShowCheckbox != nil {
		rs.ShowCheckbox = *bs.ShowCheckbox
	}
	if bs.CardPosition != nil {
		rs.CardPosition = *bs.CardPosition
	}
	if bs.ExpandColumns != nil {
		rs.ExpandColumns = *bs.ExpandColumns
	}
	if bs.ViewMode != nil {
		rs.ViewMode = *bs.ViewMode
	}
	return rs
}

// toBoardSettingsView converts pointer-based BoardSettings to string values for templates.
func toBoardSettingsView(bs models.BoardSettings) BoardSettingsView {
	v := BoardSettingsView{}
	if bs.ShowCheckbox != nil {
		if *bs.ShowCheckbox {
			v.ShowCheckbox = "true"
		} else {
			v.ShowCheckbox = "false"
		}
	}
	if bs.CardPosition != nil {
		v.CardPosition = *bs.CardPosition
	}
	if bs.ExpandColumns != nil {
		if *bs.ExpandColumns {
			v.ExpandColumns = "true"
		} else {
			v.ExpandColumns = "false"
		}
	}
	if bs.ViewMode != nil {
		v.ViewMode = *bs.ViewMode
	}
	return v
}

// boardViewModel loads a board by slug and returns a populated BoardViewModel.
func (h *Handler) boardViewModel(slug string) (BoardViewModel, error) {
	board, err := h.ws.LoadBoard(slug)
	if err != nil {
		return BoardViewModel{BoardSlug: slug, Error: err.Error()}, nil
	}
	allBoards, _ := h.ws.ListBoards()
	global := h.loadSettings()
	return BoardViewModel{
		Title:          board.Name + " — " + global.SiteName,
		SiteName:       global.SiteName,
		Board:          board,
		BoardName:      board.Name,
		BoardSlug:      slug,
		Boards:         toBoardSummaries(allBoards),
		Settings:       resolveSettings(global, board.Settings),
		BSView:         toBoardSettingsView(board.Settings),
		GlobalSettings: global,
	}, nil
}

// mutateBoard runs op, commits with msg, publishes SSE, and returns the view model.
func (h *Handler) mutateBoard(slug, msg string, op func(string) error) (BoardViewModel, error) {
	boardPath := h.ws.BoardPath(slug)
	if err := op(boardPath); err != nil {
		return BoardViewModel{BoardSlug: slug, Error: err.Error()}, nil
	}
	h.commitWithHandling(boardPath, msg)
	h.publishBoardEvent(slug)
	return h.boardViewModel(slug)
}

// mutateBoardRemove runs op, commits a removal, publishes SSE, and returns the view model.
func (h *Handler) mutateBoardRemove(slug, msg string, op func(string) error) (BoardViewModel, error) {
	boardPath := h.ws.BoardPath(slug)
	if err := op(boardPath); err != nil {
		return BoardViewModel{BoardSlug: slug, Error: err.Error()}, nil
	}
	h.commitRemoveWithHandling(boardPath, msg)
	h.publishBoardEvent(slug)
	return h.boardViewModel(slug)
}

// slugFromRequest extracts the board slug from chi URL params.
func slugFromRequest(r *http.Request) string {
	return chi.URLParam(r, "slug")
}

// formInt extracts an integer form value.
func formInt(r *http.Request, key string) (int, error) {
	s := r.FormValue(key)
	if s == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	return strconv.Atoi(s)
}

// renderBoardContent renders the board-content partial.
func (h *Handler) renderBoardContent(w http.ResponseWriter, model BoardViewModel) {
	renderPartial(w, h.boardContentTpl, "board-content", model)
}

// BoardViewPage handles GET /board/{slug} — renders the full board view page.
func (h *Handler) BoardViewPage(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	if slug == "" {
		http.Error(w, "Board name is required", http.StatusBadRequest)
		return
	}

	model, _ := h.boardViewModel(slug)
	renderFullPage(w, h.boardViewTpl, model)
}

// BoardContent handles GET /board/{slug}/content — returns the board content partial.
func (h *Handler) BoardContent(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	if slug == "" {
		http.Error(w, "Board name is required", http.StatusBadRequest)
		return
	}

	model, _ := h.boardViewModel(slug)
	h.renderBoardContent(w, model)
}

// HandleCreateCard handles POST /board/{slug}/cards.
func (h *Handler) HandleCreateCard(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	column := r.FormValue("column")
	title := r.FormValue("title")

	if column == "" || title == "" || slug == "" {
		model, _ := h.boardViewModel(slug)
		if slug == "" {
			model.Error = "Board name is required"
		} else if column == "" {
			model.Error = "Column name is required"
		} else {
			model.Error = "Card title is required"
		}
		h.renderBoardContent(w, model)
		return
	}

	// Resolve card position setting to determine prepend vs append.
	board, loadErr := h.ws.LoadBoard(slug)
	prepend := false
	if loadErr == nil {
		global := h.loadSettings()
		rs := resolveSettings(global, board.Settings)
		prepend = rs.CardPosition == "prepend"
	}

	model, _ := h.mutateBoard(slug, fmt.Sprintf("Add card \"%s\" to %s", title, column), func(boardPath string) error {
		_, err := h.eng.AddCard(boardPath, column, title, prepend)
		return err
	})
	h.renderBoardContent(w, model)
}

// HandleMoveCard handles POST /board/{slug}/cards/move.
func (h *Handler) HandleMoveCard(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colIdx, err := formInt(r, "col_idx")
	if err != nil {
		model, _ := h.boardViewModel(slug)
		model.Error = err.Error()
		h.renderBoardContent(w, model)
		return
	}

	cardIdx, err := formInt(r, "card_idx")
	if err != nil {
		model, _ := h.boardViewModel(slug)
		model.Error = err.Error()
		h.renderBoardContent(w, model)
		return
	}

	targetColumn := r.FormValue("target_column")
	if targetColumn == "" {
		model, _ := h.boardViewModel(slug)
		model.Error = "Target column is required"
		h.renderBoardContent(w, model)
		return
	}

	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		h.renderBoardContent(w, model)
		return
	}

	model, _ := h.mutateBoard(slug, fmt.Sprintf("Move card to %s", targetColumn), func(boardPath string) error {
		return h.eng.MoveCard(boardPath, colIdx, cardIdx, targetColumn)
	})
	h.renderBoardContent(w, model)
}

// HandleReorderCard handles POST /board/{slug}/cards/reorder.
func (h *Handler) HandleReorderCard(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colIdx, err := formInt(r, "col_idx")
	if err != nil {
		model, _ := h.boardViewModel(slug)
		model.Error = err.Error()
		h.renderBoardContent(w, model)
		return
	}

	cardIdx, err := formInt(r, "card_idx")
	if err != nil {
		model, _ := h.boardViewModel(slug)
		model.Error = err.Error()
		h.renderBoardContent(w, model)
		return
	}

	column := r.FormValue("column")
	if column == "" {
		model, _ := h.boardViewModel(slug)
		model.Error = "Column is required"
		h.renderBoardContent(w, model)
		return
	}

	beforeIdx := -1
	if s := r.FormValue("before_idx"); s != "" {
		beforeIdx, _ = strconv.Atoi(s)
	}

	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		h.renderBoardContent(w, model)
		return
	}

	model, _ := h.mutateBoard(slug, fmt.Sprintf("Reorder card in %s", column), func(boardPath string) error {
		return h.eng.ReorderCard(boardPath, colIdx, cardIdx, beforeIdx, column)
	})
	h.renderBoardContent(w, model)
}

// HandleDeleteCard handles POST /board/{slug}/cards/delete.
func (h *Handler) HandleDeleteCard(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colIdx, err := formInt(r, "col_idx")
	if err != nil {
		model, _ := h.boardViewModel(slug)
		model.Error = err.Error()
		h.renderBoardContent(w, model)
		return
	}

	cardIdx, err := formInt(r, "card_idx")
	if err != nil {
		model, _ := h.boardViewModel(slug)
		model.Error = err.Error()
		h.renderBoardContent(w, model)
		return
	}

	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		h.renderBoardContent(w, model)
		return
	}

	model, _ := h.mutateBoardRemove(slug, "Delete card", func(boardPath string) error {
		return h.eng.DeleteCard(boardPath, colIdx, cardIdx)
	})
	h.renderBoardContent(w, model)
}

// HandleToggleComplete handles POST /board/{slug}/cards/complete.
func (h *Handler) HandleToggleComplete(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colIdx, err := formInt(r, "col_idx")
	if err != nil {
		model, _ := h.boardViewModel(slug)
		model.Error = err.Error()
		h.renderBoardContent(w, model)
		return
	}

	cardIdx, err := formInt(r, "card_idx")
	if err != nil {
		model, _ := h.boardViewModel(slug)
		model.Error = err.Error()
		h.renderBoardContent(w, model)
		return
	}

	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		h.renderBoardContent(w, model)
		return
	}

	model, _ := h.mutateBoard(slug, "Toggle card complete", func(boardPath string) error {
		return h.eng.CompleteCard(boardPath, colIdx, cardIdx)
	})
	h.renderBoardContent(w, model)
}

// HandleEditCard handles POST /board/{slug}/cards/edit.
func (h *Handler) HandleEditCard(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colIdx, err := formInt(r, "col_idx")
	if err != nil {
		model, _ := h.boardViewModel(slug)
		model.Error = err.Error()
		h.renderBoardContent(w, model)
		return
	}

	cardIdx, err := formInt(r, "card_idx")
	if err != nil {
		model, _ := h.boardViewModel(slug)
		model.Error = err.Error()
		h.renderBoardContent(w, model)
		return
	}

	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		h.renderBoardContent(w, model)
		return
	}

	title := r.FormValue("title")
	body := r.FormValue("body")
	tagsRaw := r.FormValue("tags")
	priority := r.FormValue("priority")
	due := r.FormValue("due")
	assignee := r.FormValue("assignee")

	var tags []string
	for _, t := range strings.Split(tagsRaw, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}

	model, _ := h.mutateBoard(slug, "Edit card", func(boardPath string) error {
		if err := h.eng.EditCard(boardPath, colIdx, cardIdx, title, body, tags, priority, due, assignee); err != nil {
			return err
		}
		// If an assignee was set, ensure they're in the board's member list.
		if assignee != "" {
			board, err := h.eng.LoadBoard(boardPath)
			if err != nil {
				return nil // card saved, member sync is best-effort
			}
			found := false
			for _, m := range board.Members {
				if m == assignee {
					found = true
					break
				}
			}
			if !found {
				_ = h.eng.UpdateBoardMembers(boardPath, append(board.Members, assignee))
			}
		}
		return nil
	})
	h.renderBoardContent(w, model)
}

// HandleCreateColumn handles POST /board/{slug}/columns.
func (h *Handler) HandleCreateColumn(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colName := r.FormValue("column_name")

	if colName == "" {
		model, _ := h.boardViewModel(slug)
		model.Error = "Column name is required"
		h.renderBoardContent(w, model)
		return
	}

	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		h.renderBoardContent(w, model)
		return
	}

	model, _ := h.mutateBoard(slug, fmt.Sprintf("Add column: %s", colName), func(boardPath string) error {
		return h.eng.AddColumn(boardPath, colName)
	})
	h.renderBoardContent(w, model)
}

// HandleRenameColumn handles POST /board/{slug}/columns/rename.
func (h *Handler) HandleRenameColumn(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	oldName := r.FormValue("old_name")
	newName := r.FormValue("new_name")

	if oldName == "" {
		model, _ := h.boardViewModel(slug)
		model.Error = "Old column name is required"
		h.renderBoardContent(w, model)
		return
	}

	if newName == "" {
		model, _ := h.boardViewModel(slug)
		model.Error = "New column name is required"
		h.renderBoardContent(w, model)
		return
	}

	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		h.renderBoardContent(w, model)
		return
	}

	model, _ := h.mutateBoard(slug, fmt.Sprintf("Rename column %q to %q", oldName, newName), func(boardPath string) error {
		return h.eng.RenameColumn(boardPath, oldName, newName)
	})
	h.renderBoardContent(w, model)
}

// HandleDeleteColumn handles POST /board/{slug}/columns/delete.
func (h *Handler) HandleDeleteColumn(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colName := r.FormValue("column_name")

	if colName == "" {
		model, _ := h.boardViewModel(slug)
		model.Error = "Column name is required"
		h.renderBoardContent(w, model)
		return
	}

	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		h.renderBoardContent(w, model)
		return
	}

	model, _ := h.mutateBoardRemove(slug, fmt.Sprintf("Delete column: %s", colName), func(boardPath string) error {
		return h.eng.DeleteColumn(boardPath, colName)
	})
	h.renderBoardContent(w, model)
}

// HandleUpdateBoardMeta handles POST /board/{slug}/meta.
func (h *Handler) HandleUpdateBoardMeta(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		h.renderBoardContent(w, model)
		return
	}

	name := r.FormValue("board_name")
	description := r.FormValue("description")
	tagsRaw := r.FormValue("tags")

	var tags []string
	for _, t := range strings.Split(tagsRaw, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}

	model, _ := h.mutateBoard(slug, fmt.Sprintf("Update board meta: %s", name), func(boardPath string) error {
		return h.eng.UpdateBoardMeta(boardPath, name, description, tags)
	})
	h.renderBoardContent(w, model)
}

// HandleToggleColumnCollapse handles POST /board/{slug}/columns/collapse.
func (h *Handler) HandleToggleColumnCollapse(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		h.renderBoardContent(w, model)
		return
	}

	colIndex, err := formInt(r, "col_index")
	if err != nil {
		model, _ := h.boardViewModel(slug)
		model.Error = err.Error()
		h.renderBoardContent(w, model)
		return
	}

	model, _ := h.mutateBoard(slug, "Toggle column collapse", func(boardPath string) error {
		return h.eng.ToggleColumnCollapse(boardPath, colIndex)
	})
	h.renderBoardContent(w, model)
}

// HandleSortColumn handles POST /board/{slug}/columns/sort.
func (h *Handler) HandleSortColumn(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colIdx, err := formInt(r, "col_idx")
	if err != nil {
		model, _ := h.boardViewModel(slug)
		model.Error = err.Error()
		h.renderBoardContent(w, model)
		return
	}

	sortBy := r.FormValue("sort_by")
	if sortBy == "" {
		model, _ := h.boardViewModel(slug)
		model.Error = "sort_by is required"
		h.renderBoardContent(w, model)
		return
	}

	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		h.renderBoardContent(w, model)
		return
	}

	model, _ := h.mutateBoard(slug, fmt.Sprintf("Sort column by %s", sortBy), func(boardPath string) error {
		return h.eng.SortColumn(boardPath, colIdx, sortBy)
	})
	h.renderBoardContent(w, model)
}

// HandleUpdateBoardSettings handles POST /board/{slug}/settings.
func (h *Handler) HandleUpdateBoardSettings(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		h.renderBoardContent(w, model)
		return
	}

	var settings models.BoardSettings

	if v := r.FormValue("show_checkbox"); v != "" {
		b := v == "true"
		settings.ShowCheckbox = &b
	}
	if v := r.FormValue("card_position"); v == "prepend" || v == "append" {
		settings.CardPosition = &v
	}
	if v := r.FormValue("expand_columns"); v != "" {
		b := v == "true"
		settings.ExpandColumns = &b
	}
	if v := r.FormValue("view_mode"); v == "board" || v == "table" {
		settings.ViewMode = &v
	}

	model, _ := h.mutateBoard(slug, "Update board settings", func(boardPath string) error {
		return h.eng.UpdateBoardSettings(boardPath, settings)
	})
	h.renderBoardContent(w, model)
}

// HandleSetBoardIcon handles POST /board/{slug}/icon.
func (h *Handler) HandleSetBoardIcon(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		h.renderBoardContent(w, model)
		return
	}

	icon := r.FormValue("icon")

	model, _ := h.mutateBoard(slug, "Set board icon", func(boardPath string) error {
		return h.eng.UpdateBoardIcon(boardPath, icon)
	})
	h.renderBoardContent(w, model)
}
