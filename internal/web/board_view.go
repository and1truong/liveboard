package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/pkg/models"
)

// BoardViewHandler handles board view page and all board mutations.
type BoardViewHandler struct {
	*Base
	boardViewTpl     *template.Template
	boardContentTpl  *template.Template
	onCardCompleted  func(slug, cardID string)      // callback: reminder auto-dismiss
	onDueDateChanged func(slug, cardID, due string) // callback: reminder recalculation
}

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
	BoardVersion   int                `json:"version"`
	Settings       ResolvedSettings   `json:"settings"`
	BSView         BoardSettingsView  `json:"bs_view"`
	GlobalSettings AppSettings        `json:"global_settings"`
	AllCards       []CardWithPosition `json:"all_cards,omitempty"`
}

// ResolveSettings merges global defaults with per-board overrides.
// Exported so the API layer can use it without duplicating logic.
func ResolveSettings(global AppSettings, bs models.BoardSettings) ResolvedSettings {
	return resolveSettings(global, bs)
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
func (bv *BoardViewHandler) boardViewModel(slug string) (BoardViewModel, error) {
	b, err := bv.ws.LoadBoard(slug)
	if err != nil {
		return BoardViewModel{BoardSlug: slug, Error: err.Error()}, err
	}
	allInfos, _ := bv.ws.ListBoardSummaries()
	global := bv.loadSettings()
	summaries := sortBoardsWithPins(toBoardSummariesFast(allInfos), global.PinnedBoards)
	tcMap := b.TagColors
	if tcMap == nil {
		tcMap = map[string]string{}
	}
	tcJSON, _ := json.Marshal(tcMap)
	resolved := resolveSettings(global, b.Settings)
	vm := BoardViewModel{
		LayoutSettings: bv.layoutSettings(global),
		Title:          b.Name + " — " + global.SiteName,
		SiteName:       global.SiteName,
		Board:          b,
		BoardName:      b.Name,
		BoardSlug:      slug,
		Boards:         summaries,
		AllTags:        collectAllTags(summaries),
		TagColorsJSON:  string(tcJSON),
		BoardVersion:   b.Version,
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
func (bv *BoardViewHandler) mutateBoard(slug string, clientVersion int, op func(*models.Board) error) (BoardViewModel, error) {
	boardPath, err := bv.ws.BoardPath(slug)
	if err != nil {
		return BoardViewModel{}, err
	}
	if err := bv.eng.MutateBoard(boardPath, clientVersion, op); err != nil {
		if errors.Is(err, board.ErrVersionConflict) {
			return BoardViewModel{}, board.ErrVersionConflict
		}
		return BoardViewModel{BoardSlug: slug, Error: err.Error()}, nil
	}
	bv.publishBoardEvent(slug)
	return bv.boardViewModel(slug)
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
func (bv *BoardViewHandler) renderBoardContent(w http.ResponseWriter, model BoardViewModel) {
	renderPartial(w, bv.boardContentTpl, "board-content", model)
}

// handleConflict sends a 409 response with fresh board HTML.
func (bv *BoardViewHandler) handleConflict(w http.ResponseWriter, slug string) {
	w.WriteHeader(http.StatusConflict)
	model, _ := bv.boardViewModel(slug)
	bv.renderBoardContent(w, model)
}

// BoardViewPage handles GET /board/{slug} — renders the full board view page.
func (bv *BoardViewHandler) BoardViewPage(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	if slug == "" {
		http.Error(w, "Board name is required", http.StatusBadRequest)
		return
	}

	model, err := bv.boardViewModel(slug)
	if err != nil {
		http.Redirect(w, r, "/?error="+url.QueryEscape(fmt.Sprintf("Board '%s' not found", slug)), http.StatusSeeOther)
		return
	}

	// Persist last-viewed board for desktop restore-on-launch.
	if bv.IsDesktop && !bv.ReadOnly {
		if s := bv.loadSettings(); s.LastBoard != slug {
			s.LastBoard = slug
			_ = bv.saveSettings(s)
		}
	}

	renderFullPage(w, bv.boardViewTpl, model)
}

// BoardContent handles GET /board/{slug}/content — returns the board content partial.
func (bv *BoardViewHandler) BoardContent(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	if slug == "" {
		http.Error(w, "Board name is required", http.StatusBadRequest)
		return
	}

	model, err := bv.boardViewModel(slug)
	if err != nil {
		w.Header().Set("HX-Redirect", "/?error="+url.QueryEscape(fmt.Sprintf("Board '%s' not found", slug)))
		w.WriteHeader(http.StatusNotFound)
		return
	}
	bv.renderBoardContent(w, model)
}
