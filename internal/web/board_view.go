package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/reminder"
	"github.com/and1truong/liveboard/pkg/models"
)

// ResolvedSettings holds the effective settings for a board view,
// merging global defaults with per-board overrides.
type ResolvedSettings struct {
	ShowCheckbox    bool   `json:"show_checkbox"`
	NewLineTrigger  string `json:"newline_trigger"`
	CardPosition    string `json:"card_position"`
	ExpandColumns   bool   `json:"expand_columns"`
	ViewMode        string `json:"view_mode"`
	CardDisplayMode string `json:"card_display_mode"`
	WeekStart       string `json:"week_start"`
}

// BoardSettingsView holds pre-formatted per-board override values for the template.
// Empty string means "not set" (inherit global).
type BoardSettingsView struct {
	ShowCheckbox    string `json:"show_checkbox"`
	CardPosition    string `json:"card_position"`
	ExpandColumns   string `json:"expand_columns"`
	ViewMode        string `json:"view_mode"`
	CardDisplayMode string `json:"card_display_mode"`
	WeekStart       string `json:"week_start"`
}

// CardWithPosition holds a card along with its column/card indices for template use.
type CardWithPosition struct {
	models.Card
	ColIdx     int    `json:"col_idx"`
	CardIdx    int    `json:"card_idx"`
	ColumnName string `json:"column_name"`
}

// BoardViewModel is the state for the board view page.
type BoardViewModel struct {
	LayoutSettings
	Title          string             `json:"title"`
	SiteName       string             `json:"site_name"`
	Board          *models.Board      `json:"board"`
	BoardName      string             `json:"board_name"`
	BoardSlug      string             `json:"board_slug"`
	Boards         []BoardSummary     `json:"boards"`
	AllTags        []string           `json:"all_tags,omitempty"`
	TagColorsJSON  string             `json:"tag_colors_json,omitempty"`
	Error          string             `json:"error,omitempty"`
	Version        int                `json:"version"`
	Settings       ResolvedSettings   `json:"settings"`
	BSView         BoardSettingsView  `json:"bs_view"`
	GlobalSettings AppSettings        `json:"global_settings"`
	AllCards       []CardWithPosition `json:"all_cards,omitempty"`
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
	if rs.ViewMode == "table" {
		rs.ViewMode = "list"
	}
	rs.CardDisplayMode = global.CardDisplayMode
	if rs.CardDisplayMode == "" {
		rs.CardDisplayMode = "full"
	}
	if bs.CardDisplayMode != nil {
		rs.CardDisplayMode = *bs.CardDisplayMode
	}
	rs.WeekStart = global.WeekStart
	if rs.WeekStart == "" {
		rs.WeekStart = "sunday"
	}
	if bs.WeekStart != nil {
		rs.WeekStart = *bs.WeekStart
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
	if bs.CardDisplayMode != nil {
		v.CardDisplayMode = *bs.CardDisplayMode
	}
	if bs.WeekStart != nil {
		v.WeekStart = *bs.WeekStart
	}
	return v
}

// boardViewModel loads a board by slug and returns a populated BoardViewModel.
func (h *Handler) boardViewModel(slug string) (BoardViewModel, error) {
	b, err := h.ws.LoadBoard(slug)
	if err != nil {
		return BoardViewModel{BoardSlug: slug, Error: err.Error()}, err
	}
	allInfos, _ := h.ws.ListBoardSummaries()
	global := h.loadSettings()
	summaries := sortBoardsWithPins(toBoardSummariesFast(allInfos), global.PinnedBoards)
	tcMap := b.TagColors
	if tcMap == nil {
		tcMap = map[string]string{}
	}
	tcJSON, _ := json.Marshal(tcMap)
	resolved := resolveSettings(global, b.Settings)
	vm := BoardViewModel{
		LayoutSettings: h.layoutSettings(global),
		Title:          b.Name + " — " + global.SiteName,
		SiteName:       global.SiteName,
		Board:          b,
		BoardName:      b.Name,
		BoardSlug:      slug,
		Boards:         summaries,
		AllTags:        collectAllTags(summaries),
		TagColorsJSON:  string(tcJSON),
		Version:        b.Version,
		Settings:       resolved,
		BSView:         toBoardSettingsView(b.Settings),
		GlobalSettings: global,
	}
	if resolved.ViewMode == "calendar" {
		vm.AllCards = flattenCards(b)
	}
	return vm, nil
}

// mutateBoard runs a versioned mutation via eng.MutateBoard, publishes SSE,
// and returns the refreshed view model. Returns ErrVersionConflict on stale version.
func (h *Handler) mutateBoard(slug string, clientVersion int, op func(*models.Board) error) (BoardViewModel, error) {
	boardPath, err := h.ws.BoardPath(slug)
	if err != nil {
		return BoardViewModel{}, err
	}
	if err := h.eng.MutateBoard(boardPath, clientVersion, op); err != nil {
		if errors.Is(err, board.ErrVersionConflict) {
			return BoardViewModel{}, board.ErrVersionConflict
		}
		return BoardViewModel{BoardSlug: slug, Error: err.Error()}, nil
	}
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

	model, err := h.boardViewModel(slug)
	if err != nil {
		http.Redirect(w, r, "/?error="+url.QueryEscape(fmt.Sprintf("Board '%s' not found", slug)), http.StatusSeeOther)
		return
	}

	// Persist last-viewed board for desktop restore-on-launch.
	if h.IsDesktop && !h.ReadOnly {
		if s := h.loadSettings(); s.LastBoard != slug {
			s.LastBoard = slug
			_ = h.saveSettings(s)
		}
	}

	renderFullPage(w, h.boardViewTpl, model)
}

// BoardContent handles GET /board/{slug}/content — returns the board content partial.
func (h *Handler) BoardContent(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	if slug == "" {
		http.Error(w, "Board name is required", http.StatusBadRequest)
		return
	}

	model, err := h.boardViewModel(slug)
	if err != nil {
		w.Header().Set("HX-Redirect", "/?error="+url.QueryEscape(fmt.Sprintf("Board '%s' not found", slug)))
		w.WriteHeader(http.StatusNotFound)
		return
	}
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
	model, err := h.mutateBoard(slug, version, func(b *models.Board) error {
		card := models.Card{
			Title:    title,
			Metadata: map[string]string{"id": reminder.GenerateID()},
		}
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
	model, mutErr := h.mutateBoard(slug, version, func(b *models.Board) error {
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
	model, mutErr := h.mutateBoard(slug, version, func(b *models.Board) error {
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
			cards = slices.Insert(cards, beforeIdx, card)
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
	model, mutErr := h.mutateBoard(slug, version, func(b *models.Board) error {
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
	model, mutErr := h.mutateBoard(slug, version, func(b *models.Board) error {
		if err := validateIndices(b, colIdx, cardIdx); err != nil {
			return err
		}
		card := &b.Columns[colIdx].Cards[cardIdx]
		card.Completed = !card.Completed
		// Auto-dismiss reminder when card is completed
		if card.Completed {
			if cardID := reminder.GetCardID(card); cardID != "" {
				_ = h.ReminderStore.RemoveByCardID(slug, cardID)
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

// splitTags splits a comma-separated string into trimmed, non-empty tags.
func splitTags(raw string) []string {
	var tags []string
	for _, t := range strings.Split(raw, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// ensureMember adds member to the board's member list if not already present.
func ensureMember(b *models.Board, member string) {
	for _, m := range b.Members {
		if m == member {
			return
		}
	}
	b.Members = append(b.Members, member)
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
	tags := splitTags(r.FormValue("tags"))
	priority := r.FormValue("priority")
	due := r.FormValue("due")
	assignee := r.FormValue("assignee")

	version := formVersion(r)
	model, mutErr := h.mutateBoard(slug, version, func(b *models.Board) error {
		if err := validateIndices(b, colIdx, cardIdx); err != nil {
			return err
		}
		card := &b.Columns[colIdx].Cards[cardIdx]

		// Ensure card has an ID
		if card.Metadata == nil {
			card.Metadata = map[string]string{}
		}
		if card.Metadata["id"] == "" {
			card.Metadata["id"] = reminder.GenerateID()
		}

		oldDue := card.Due

		if title != "" {
			card.Title = title
		}
		card.Body = body
		card.Tags = tags
		card.Priority = priority
		card.Due = due
		card.Assignee = assignee
		if assignee != "" {
			ensureMember(b, assignee)
		}

		// Auto-recalculate relative reminder if due date changed
		if oldDue != due && due != "" {
			cardID := card.Metadata["id"]
			settings := h.loadSettings()
			tz := settings.ReminderTimezone
			if tz == "" {
				tz = "Local"
			}
			_ = h.ReminderStore.RecalculateRelativeReminder(slug, cardID, due, tz)
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
	model, mutErr := h.mutateBoard(slug, version, func(b *models.Board) error {
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
	model, mutErr := h.mutateBoard(slug, version, func(b *models.Board) error {
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
	model, mutErr := h.mutateBoard(slug, version, func(b *models.Board) error {
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
	tagColorsRaw := r.FormValue("tag_colors")

	parts := strings.Split(tagsRaw, ",")
	tags := make([]string, 0, len(parts))
	for _, t := range parts {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}

	version := formVersion(r)
	model, mutErr := h.mutateBoard(slug, version, func(b *models.Board) error {
		if name != "" {
			b.Name = name
		}
		b.Description = description
		b.Tags = tags
		if tagColorsRaw != "" {
			var tagColors map[string]string
			if err := json.Unmarshal([]byte(tagColorsRaw), &tagColors); err == nil {
				if len(tagColors) == 0 {
					b.TagColors = nil
				} else {
					b.TagColors = tagColors
				}
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
	model, mutErr := h.mutateBoard(slug, version, func(b *models.Board) error {
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
	model, mutErr := h.mutateBoard(slug, version, func(b *models.Board) error {
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

// reorderColumns moves colName after afterCol (or to front if afterCol is empty)
// and rebuilds ListCollapse to match the new order.
func reorderColumns(b *models.Board, colName, afterCol string) error {
	// Align ListCollapse with columns.
	for len(b.ListCollapse) < len(b.Columns) {
		b.ListCollapse = append(b.ListCollapse, false)
	}

	// Build collapse state map.
	collapseByName := make(map[string]bool, len(b.Columns))
	for i, col := range b.Columns {
		collapseByName[col.Name] = b.ListCollapse[i]
	}

	// Find and remove the column being moved.
	var movingCol *models.Column
	var remaining []models.Column
	for _, col := range b.Columns {
		if col.Name == colName {
			c := col
			movingCol = &c
		} else {
			remaining = append(remaining, col)
		}
	}
	if movingCol == nil {
		return fmt.Errorf("column %q not found", colName)
	}

	// Rebuild: if afterCol is empty, prepend; otherwise insert after afterCol.
	var reordered []models.Column
	if afterCol == "" {
		reordered = append([]models.Column{*movingCol}, remaining...)
	} else {
		for _, col := range remaining {
			reordered = append(reordered, col)
			if col.Name == afterCol {
				reordered = append(reordered, *movingCol)
			}
		}
	}

	b.Columns = reordered

	// Rebuild ListCollapse to match new order.
	b.ListCollapse = make([]bool, len(b.Columns))
	for i, col := range b.Columns {
		b.ListCollapse[i] = collapseByName[col.Name]
	}
	return nil
}

// HandleMoveColumn handles POST /board/{slug}/columns/move.
func (h *Handler) HandleMoveColumn(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colName := r.FormValue("column")
	afterCol := r.FormValue("after_column")

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
	model, mutErr := h.mutateBoard(slug, version, func(b *models.Board) error {
		return reorderColumns(b, colName, afterCol)
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		h.handleConflict(w, slug)
		return
	}
	h.renderBoardContent(w, model)
}

// parseBoardSettingsForm extracts board settings from form values.
func parseBoardSettingsForm(r *http.Request) models.BoardSettings {
	var s models.BoardSettings
	if v := r.FormValue("show_checkbox"); v != "" {
		b := v == "true"
		s.ShowCheckbox = &b
	}
	if v := r.FormValue("card_position"); v == "prepend" || v == "append" {
		s.CardPosition = &v
	}
	if v := r.FormValue("expand_columns"); v != "" {
		b := v == "true"
		s.ExpandColumns = &b
	}
	if v := r.FormValue("view_mode"); v == "board" || v == "list" || v == "table" || v == "calendar" {
		s.ViewMode = &v
	}
	if v := r.FormValue("week_start"); v == "sunday" || v == "monday" {
		s.WeekStart = &v
	}
	if v := r.FormValue("card_display_mode"); v == "full" || v == "hide" || v == "trim" {
		s.CardDisplayMode = &v
	}
	return s
}

// HandleUpdateBoardSettings handles POST /board/{slug}/settings.
func (h *Handler) HandleUpdateBoardSettings(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		h.renderBoardContent(w, model)
		return
	}

	settings := parseBoardSettingsForm(r)

	version := formVersion(r)
	model, mutErr := h.mutateBoard(slug, version, func(b *models.Board) error {
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
	model, mutErr := h.mutateBoard(slug, version, func(b *models.Board) error {
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

// flattenCards collects all cards from all columns with their position indices.
func flattenCards(b *models.Board) []CardWithPosition {
	n := 0
	for _, col := range b.Columns {
		n += len(col.Cards)
	}
	all := make([]CardWithPosition, 0, n)
	for ci, col := range b.Columns {
		for ci2, card := range col.Cards {
			all = append(all, CardWithPosition{
				Card:       card,
				ColIdx:     ci,
				CardIdx:    ci2,
				ColumnName: col.Name,
			})
		}
	}
	return all
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
