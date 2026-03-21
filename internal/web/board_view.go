package web

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/and1truong/liveboard/internal/board"
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
	Version        int               `json:"version"`
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
	b, err := h.ws.LoadBoard(slug)
	if err != nil {
		return BoardViewModel{BoardSlug: slug, Error: err.Error()}, nil
	}
	allBoards, _ := h.ws.ListBoards()
	global := h.loadSettings()
	return BoardViewModel{
		Title:          b.Name + " — " + global.SiteName,
		SiteName:       global.SiteName,
		Board:          b,
		BoardName:      b.Name,
		BoardSlug:      slug,
		Boards:         toBoardSummaries(allBoards),
		Version:        b.Version,
		Settings:       resolveSettings(global, b.Settings),
		BSView:         toBoardSettingsView(b.Settings),
		GlobalSettings: global,
	}, nil
}

// mutateBoard runs a versioned mutation via eng.MutateBoard, commits, publishes SSE,
// and returns the refreshed view model. Returns ErrVersionConflict on stale version.
func (h *Handler) mutateBoard(slug, msg string, clientVersion int, op func(*models.Board) error) (BoardViewModel, error) {
	boardPath := h.ws.BoardPath(slug)
	if err := h.eng.MutateBoard(boardPath, clientVersion, op); err != nil {
		if errors.Is(err, board.ErrVersionConflict) {
			return BoardViewModel{}, board.ErrVersionConflict
		}
		return BoardViewModel{BoardSlug: slug, Error: err.Error()}, nil
	}
	h.commitWithHandling(boardPath, msg)
	h.publishBoardEvent(slug)
	return h.boardViewModel(slug)
}

// mutateBoardRemove is like mutateBoard but uses commitRemoveWithHandling.
func (h *Handler) mutateBoardRemove(slug, msg string, clientVersion int, op func(*models.Board) error) (BoardViewModel, error) {
	boardPath := h.ws.BoardPath(slug)
	if err := h.eng.MutateBoard(boardPath, clientVersion, op); err != nil {
		if errors.Is(err, board.ErrVersionConflict) {
			return BoardViewModel{}, board.ErrVersionConflict
		}
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

// formVersion extracts the board version from the request. Returns -1 if absent (skip check).
func formVersion(r *http.Request) int {
	s := r.FormValue("version")
	if s == "" {
		return -1
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return v
}

// renderBoardContent renders the board-content partial.
func (h *Handler) renderBoardContent(w http.ResponseWriter, model BoardViewModel) {
	renderPartial(w, h.boardContentTpl, "board-content", model)
}

// handleConflict sends a 409 response with fresh board HTML.
func (h *Handler) handleConflict(w http.ResponseWriter, slug string) {
	w.WriteHeader(http.StatusConflict)
	model, _ := h.boardViewModel(slug)
	h.renderBoardContent(w, model)
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
	loadedBoard, loadErr := h.ws.LoadBoard(slug)
	prepend := false
	if loadErr == nil {
		global := h.loadSettings()
		rs := resolveSettings(global, loadedBoard.Settings)
		prepend = rs.CardPosition == "prepend"
	}

	version := formVersion(r)
	model, err := h.mutateBoard(slug, fmt.Sprintf("Add card \"%s\" to %s", title, column), version, func(b *models.Board) error {
		card := models.Card{Title: title}
		for i := range b.Columns {
			if b.Columns[i].Name == column {
				if prepend {
					b.Columns[i].Cards = append([]models.Card{card}, b.Columns[i].Cards...)
				} else {
					b.Columns[i].Cards = append(b.Columns[i].Cards, card)
				}
				return nil
			}
		}
		return fmt.Errorf("column %q not found", column)
	})
	if errors.Is(err, board.ErrVersionConflict) {
		h.handleConflict(w, slug)
		return
	}
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

	version := formVersion(r)
	model, mutErr := h.mutateBoard(slug, fmt.Sprintf("Move card to %s", targetColumn), version, func(b *models.Board) error {
		if err := validateIndices(b, colIdx, cardIdx); err != nil {
			return err
		}
		card := b.Columns[colIdx].Cards[cardIdx]
		b.Columns[colIdx].Cards = removeCardAt(b.Columns[colIdx].Cards, cardIdx)
		for i := range b.Columns {
			if b.Columns[i].Name == targetColumn {
				b.Columns[i].Cards = append(b.Columns[i].Cards, card)
				return nil
			}
		}
		return fmt.Errorf("target column %q not found", targetColumn)
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		h.handleConflict(w, slug)
		return
	}
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

	version := formVersion(r)
	model, mutErr := h.mutateBoard(slug, fmt.Sprintf("Reorder card in %s", column), version, func(b *models.Board) error {
		if err := validateIndices(b, colIdx, cardIdx); err != nil {
			return err
		}
		card := b.Columns[colIdx].Cards[cardIdx]
		b.Columns[colIdx].Cards = removeCardAt(b.Columns[colIdx].Cards, cardIdx)

		targetIdx := -1
		for i := range b.Columns {
			if b.Columns[i].Name == column {
				targetIdx = i
				break
			}
		}
		if targetIdx < 0 {
			return fmt.Errorf("target column %q not found", column)
		}

		cards := b.Columns[targetIdx].Cards
		if beforeIdx < 0 || beforeIdx >= len(cards) {
			cards = append(cards, card)
		} else {
			cards = append(cards[:beforeIdx], append([]models.Card{card}, cards[beforeIdx:]...)...)
		}
		b.Columns[targetIdx].Cards = cards
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		h.handleConflict(w, slug)
		return
	}
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

	version := formVersion(r)
	model, mutErr := h.mutateBoardRemove(slug, "Delete card", version, func(b *models.Board) error {
		if err := validateIndices(b, colIdx, cardIdx); err != nil {
			return err
		}
		b.Columns[colIdx].Cards = removeCardAt(b.Columns[colIdx].Cards, cardIdx)
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		h.handleConflict(w, slug)
		return
	}
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

	version := formVersion(r)
	model, mutErr := h.mutateBoard(slug, "Toggle card complete", version, func(b *models.Board) error {
		if err := validateIndices(b, colIdx, cardIdx); err != nil {
			return err
		}
		b.Columns[colIdx].Cards[cardIdx].Completed = !b.Columns[colIdx].Cards[cardIdx].Completed
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		h.handleConflict(w, slug)
		return
	}
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

	version := formVersion(r)
	model, mutErr := h.mutateBoard(slug, "Edit card", version, func(b *models.Board) error {
		if err := validateIndices(b, colIdx, cardIdx); err != nil {
			return err
		}
		card := &b.Columns[colIdx].Cards[cardIdx]
		if title != "" {
			card.Title = title
		}
		card.Body = body
		card.Tags = tags
		card.Priority = priority
		card.Due = due
		card.Assignee = assignee

		// If an assignee was set, ensure they're in the board's member list.
		if assignee != "" {
			found := false
			for _, m := range b.Members {
				if m == assignee {
					found = true
					break
				}
			}
			if !found {
				b.Members = append(b.Members, assignee)
			}
		}
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		h.handleConflict(w, slug)
		return
	}
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

	version := formVersion(r)
	model, mutErr := h.mutateBoard(slug, fmt.Sprintf("Add column: %s", colName), version, func(b *models.Board) error {
		b.Columns = append(b.Columns, models.Column{Name: colName})
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		h.handleConflict(w, slug)
		return
	}
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

	version := formVersion(r)
	model, mutErr := h.mutateBoard(slug, fmt.Sprintf("Rename column %q to %q", oldName, newName), version, func(b *models.Board) error {
		for i := range b.Columns {
			if b.Columns[i].Name == oldName {
				b.Columns[i].Name = newName
				return nil
			}
		}
		return fmt.Errorf("column %q not found", oldName)
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		h.handleConflict(w, slug)
		return
	}
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

	version := formVersion(r)
	model, mutErr := h.mutateBoardRemove(slug, fmt.Sprintf("Delete column: %s", colName), version, func(b *models.Board) error {
		var cols []models.Column
		found := false
		for j, col := range b.Columns {
			if col.Name == colName {
				found = true
				if j < len(b.ListCollapse) {
					b.ListCollapse = append(b.ListCollapse[:j], b.ListCollapse[j+1:]...)
				}
				continue
			}
			cols = append(cols, col)
		}
		if !found {
			return fmt.Errorf("column %q not found", colName)
		}
		b.Columns = cols
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		h.handleConflict(w, slug)
		return
	}
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

	version := formVersion(r)
	model, mutErr := h.mutateBoard(slug, fmt.Sprintf("Update board meta: %s", name), version, func(b *models.Board) error {
		if name != "" {
			b.Name = name
		}
		b.Description = description
		b.Tags = tags
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		h.handleConflict(w, slug)
		return
	}
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

	version := formVersion(r)
	model, mutErr := h.mutateBoard(slug, "Toggle column collapse", version, func(b *models.Board) error {
		if colIndex < 0 || colIndex >= len(b.Columns) {
			return fmt.Errorf("column index %d out of range", colIndex)
		}
		for len(b.ListCollapse) < len(b.Columns) {
			b.ListCollapse = append(b.ListCollapse, false)
		}
		b.ListCollapse[colIndex] = !b.ListCollapse[colIndex]
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		h.handleConflict(w, slug)
		return
	}
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

	version := formVersion(r)
	model, mutErr := h.mutateBoard(slug, fmt.Sprintf("Sort column by %s", sortBy), version, func(b *models.Board) error {
		if colIdx < 0 || colIdx >= len(b.Columns) {
			return fmt.Errorf("column index %d out of range", colIdx)
		}
		cards := b.Columns[colIdx].Cards
		switch sortBy {
		case "name":
			sortCardsByName(cards)
		case "priority":
			sortCardsByPriority(cards)
		case "due":
			sortCardsByDue(cards)
		default:
			return fmt.Errorf("unknown sort key %q", sortBy)
		}
		b.Columns[colIdx].Cards = cards
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		h.handleConflict(w, slug)
		return
	}
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

	version := formVersion(r)
	model, mutErr := h.mutateBoard(slug, "Update board settings", version, func(b *models.Board) error {
		b.Settings = settings
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		h.handleConflict(w, slug)
		return
	}
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

	version := formVersion(r)
	model, mutErr := h.mutateBoard(slug, "Set board icon", version, func(b *models.Board) error {
		b.Icon = icon
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		h.handleConflict(w, slug)
		return
	}
	h.renderBoardContent(w, model)
}

// validateIndices checks that column and card indices are within bounds.
func validateIndices(b *models.Board, colIdx, cardIdx int) error {
	if colIdx < 0 || colIdx >= len(b.Columns) {
		return fmt.Errorf("column index %d out of range", colIdx)
	}
	if cardIdx < 0 || cardIdx >= len(b.Columns[colIdx].Cards) {
		return fmt.Errorf("card index %d out of range in column %q", cardIdx, b.Columns[colIdx].Name)
	}
	return nil
}

// removeCardAt removes a card at the given index.
func removeCardAt(cards []models.Card, idx int) []models.Card {
	return append(cards[:idx], cards[idx+1:]...)
}

// Sort helpers (moved from board package since mutations are now inline).

func sortCardsByName(cards []models.Card) {
	sort.SliceStable(cards, func(i, j int) bool {
		return strings.ToLower(cards[i].Title) < strings.ToLower(cards[j].Title)
	})
}

func sortCardsByPriority(cards []models.Card) {
	sort.SliceStable(cards, func(i, j int) bool {
		return priorityRank(cards[i].Priority) > priorityRank(cards[j].Priority)
	})
}

func sortCardsByDue(cards []models.Card) {
	sort.SliceStable(cards, func(i, j int) bool {
		a, b := cards[i].Due, cards[j].Due
		if a == "" && b == "" {
			return false
		}
		if a == "" {
			return false
		}
		if b == "" {
			return true
		}
		return a < b
	})
}

func priorityRank(p string) int {
	switch strings.ToLower(p) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}
