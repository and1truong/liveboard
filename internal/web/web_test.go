package web

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jfyne/live"

	"github.com/and1truong/liveboard/internal/board"
	gitpkg "github.com/and1truong/liveboard/internal/git"
	"github.com/and1truong/liveboard/internal/workspace"
	"github.com/and1truong/liveboard/pkg/models"
)

// setupHandlerWithBoard creates a Handler with a real workspace containing one board.
// Returns the handler and the board slug.
func setupHandlerWithBoard(t *testing.T) (*Handler, string) {
	t.Helper()
	dir := t.TempDir()
	ws := workspace.Open(dir)
	eng := board.New()
	ctx := context.Background()

	h := &Handler{
		ws:     ws,
		eng:    eng,
		pubsub: live.NewPubSub(ctx, live.NewLocalTransport()),
	}

	// Create a board via workspace so it has default columns.
	if _, err := ws.CreateBoard("test-board"); err != nil {
		t.Fatal(err)
	}

	return h, "test-board"
}

func TestBoardSlug(t *testing.T) {
	cases := []struct {
		filePath string
		want     string
	}{
		{"/path/to/my-board.md", "my-board"},
		{"/path/to/sprint.md", "sprint"},
		{"simple.md", "simple"},
		{"/path/to/no-ext", "no-ext"},
	}
	for _, tc := range cases {
		got := boardSlug(models.Board{FilePath: tc.filePath})
		if got != tc.want {
			t.Errorf("boardSlug(%q) = %q, want %q", tc.filePath, got, tc.want)
		}
	}
}

func TestToBoardSummaries(t *testing.T) {
	boards := []models.Board{
		{
			Name:        "Sprint 1",
			Description: "First sprint",
			Icon:        "🏃",
			FilePath:    "/boards/sprint-1.md",
			Columns: []models.Column{
				{Name: "Todo", Cards: []models.Card{{Title: "A"}, {Title: "B"}}},
				{Name: "Done", Cards: []models.Card{{Title: "C"}}},
			},
		},
		{
			Name:     "Empty Board",
			FilePath: "/boards/empty.md",
			Columns:  []models.Column{},
		},
	}

	summaries := toBoardSummaries(boards)

	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}

	s := summaries[0]
	if s.Name != "Sprint 1" {
		t.Errorf("name = %q", s.Name)
	}
	if s.Slug != "sprint-1" {
		t.Errorf("slug = %q", s.Slug)
	}
	if s.Description != "First sprint" {
		t.Errorf("description = %q", s.Description)
	}
	if s.Icon != "🏃" {
		t.Errorf("icon = %q", s.Icon)
	}
	if s.CardCount != 3 {
		t.Errorf("card_count = %d, want 3", s.CardCount)
	}

	if summaries[1].CardCount != 0 {
		t.Errorf("empty board card_count = %d", summaries[1].CardCount)
	}
}

func TestToBoardSummariesEmpty(t *testing.T) {
	summaries := toBoardSummaries(nil)
	if len(summaries) != 0 {
		t.Errorf("expected 0 summaries for nil input, got %d", len(summaries))
	}
}

func TestResolveSettings(t *testing.T) {
	global := AppSettings{
		ShowCheckbox:   true,
		NewLineTrigger: "shift-enter",
		CardPosition:   "append",
	}

	// No overrides
	rs := resolveSettings(global, models.BoardSettings{})
	if !rs.ShowCheckbox {
		t.Error("expected ShowCheckbox=true from global")
	}
	if rs.CardPosition != "append" {
		t.Errorf("expected CardPosition=append, got %q", rs.CardPosition)
	}
	if rs.ExpandColumns {
		t.Error("expected ExpandColumns=false default")
	}

	// With overrides
	showCheckbox := false
	cardPos := "prepend"
	expandCols := true
	bs := models.BoardSettings{
		ShowCheckbox:  &showCheckbox,
		CardPosition:  &cardPos,
		ExpandColumns: &expandCols,
	}

	rs = resolveSettings(global, bs)
	if rs.ShowCheckbox {
		t.Error("expected ShowCheckbox=false from board override")
	}
	if rs.CardPosition != "prepend" {
		t.Errorf("expected CardPosition=prepend, got %q", rs.CardPosition)
	}
	if !rs.ExpandColumns {
		t.Error("expected ExpandColumns=true from board override")
	}
}

func TestToBoardSettingsView(t *testing.T) {
	// All nil
	v := toBoardSettingsView(models.BoardSettings{})
	if v.ShowCheckbox != "" || v.CardPosition != "" || v.ExpandColumns != "" {
		t.Errorf("expected empty strings for nil settings, got %+v", v)
	}

	// All set
	showTrue := true
	showFalse := false
	cardPos := "prepend"

	v = toBoardSettingsView(models.BoardSettings{
		ShowCheckbox:  &showTrue,
		CardPosition:  &cardPos,
		ExpandColumns: &showFalse,
	})
	if v.ShowCheckbox != "true" {
		t.Errorf("ShowCheckbox = %q, want 'true'", v.ShowCheckbox)
	}
	if v.CardPosition != "prepend" {
		t.Errorf("CardPosition = %q, want 'prepend'", v.CardPosition)
	}
	if v.ExpandColumns != "false" {
		t.Errorf("ExpandColumns = %q, want 'false'", v.ExpandColumns)
	}
}

func TestIntParam(t *testing.T) {
	// Valid
	params := map[string]interface{}{"col_idx": "5"}
	v, err := intParam(params, "col_idx")
	if err != nil {
		t.Fatal(err)
	}
	if v != 5 {
		t.Errorf("got %d, want 5", v)
	}

	// Missing key
	_, err = intParam(params, "missing")
	if err == nil {
		t.Error("expected error for missing key")
	}

	// Non-numeric
	params = map[string]interface{}{"col_idx": "abc"}
	_, err = intParam(params, "col_idx")
	if err == nil {
		t.Error("expected error for non-numeric value")
	}

	// Empty string
	params = map[string]interface{}{"col_idx": ""}
	_, err = intParam(params, "col_idx")
	if err == nil {
		t.Error("expected error for empty string")
	}
}

func TestSlugFromParams(t *testing.T) {
	// Valid
	params := map[string]interface{}{"name": "my-board"}
	slug, ok := slugFromParams(params)
	if !ok || slug != "my-board" {
		t.Errorf("got slug=%q, ok=%v", slug, ok)
	}

	// Missing
	params = map[string]interface{}{}
	_, ok = slugFromParams(params)
	if ok {
		t.Error("expected ok=false for missing name")
	}

	// Empty
	params = map[string]interface{}{"name": ""}
	_, ok = slugFromParams(params)
	if ok {
		t.Error("expected ok=false for empty name")
	}

	// Wrong type
	params = map[string]interface{}{"name": 123}
	_, ok = slugFromParams(params)
	if ok {
		t.Error("expected ok=false for wrong type")
	}
}

func TestDefaultSettings(t *testing.T) {
	s := defaultSettings()
	if s.Theme != "system" {
		t.Errorf("Theme = %q", s.Theme)
	}
	if s.ColumnWidth != 280 {
		t.Errorf("ColumnWidth = %d", s.ColumnWidth)
	}
	if len(s.DefaultColumns) != 3 {
		t.Errorf("DefaultColumns = %v", s.DefaultColumns)
	}
	if s.ColorTheme != "aqua" {
		t.Errorf("ColorTheme = %q", s.ColorTheme)
	}
	if s.CardPosition != "append" {
		t.Errorf("CardPosition = %q", s.CardPosition)
	}
}

func TestSettingsAPIHandler(t *testing.T) {
	dir := t.TempDir()

	// Manually construct a Handler with minimal dependencies for settings tests.
	h := &Handler{
		ws: &workspace.Workspace{Dir: dir},
	}

	handler := h.SettingsAPIHandler()

	// GET returns defaults when no file exists
	req := httptest.NewRequest("GET", "/api/settings", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("GET: expected 200, got %d", w.Code)
	}

	var got AppSettings
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Theme != "system" {
		t.Errorf("default theme = %q", got.Theme)
	}

	// POST saves settings
	body := `{"theme":"dark","color_theme":"github","column_width":300,"sidebar_position":"right","show_checkbox":false,"newline_trigger":"enter","card_position":"prepend"}`
	req = httptest.NewRequest("POST", "/api/settings", strings.NewReader(body))
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("POST: expected 200, got %d", w.Code)
	}

	var saved AppSettings
	if err := json.NewDecoder(w.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.Theme != "dark" {
		t.Errorf("saved theme = %q", saved.Theme)
	}
	if saved.ColorTheme != "github" {
		t.Errorf("saved color_theme = %q", saved.ColorTheme)
	}
	if saved.CardPosition != "prepend" {
		t.Errorf("saved card_position = %q", saved.CardPosition)
	}

	// Verify it persisted to disk
	req = httptest.NewRequest("GET", "/api/settings", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var reloaded AppSettings
	if err := json.NewDecoder(w.Body).Decode(&reloaded); err != nil {
		t.Fatal(err)
	}
	if reloaded.Theme != "dark" {
		t.Errorf("reloaded theme = %q", reloaded.Theme)
	}

	// POST invalid JSON
	req = httptest.NewRequest("POST", "/api/settings", strings.NewReader("{bad"))
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("invalid JSON: expected 400, got %d", w.Code)
	}

	// Method not allowed
	req = httptest.NewRequest("PUT", "/api/settings", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 405 {
		t.Fatalf("PUT: expected 405, got %d", w.Code)
	}
}

func TestSettingsValidation(t *testing.T) {
	dir := t.TempDir()
	h := &Handler{
		ws: &workspace.Workspace{Dir: dir},
	}

	handler := h.SettingsAPIHandler()

	// Out of range column width should be reset to 280
	body := `{"theme":"light","color_theme":"aqua","column_width":50,"sidebar_position":"left","newline_trigger":"shift-enter","card_position":"append"}`
	req := httptest.NewRequest("POST", "/api/settings", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var s AppSettings
	if err := json.NewDecoder(w.Body).Decode(&s); err != nil {
		t.Fatal(err)
	}
	if s.ColumnWidth != 280 {
		t.Errorf("expected column_width reset to 280, got %d", s.ColumnWidth)
	}

	// Invalid theme should fall back to "system"
	body = `{"theme":"invalid","color_theme":"invalid_theme","column_width":300,"sidebar_position":"center","newline_trigger":"invalid","card_position":"invalid"}`
	req = httptest.NewRequest("POST", "/api/settings", strings.NewReader(body))
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if err := json.NewDecoder(w.Body).Decode(&s); err != nil {
		t.Fatal(err)
	}
	if s.Theme != "system" {
		t.Errorf("theme = %q, want 'system'", s.Theme)
	}
	if s.ColorTheme != "default" {
		t.Errorf("color_theme = %q, want 'default'", s.ColorTheme)
	}
	if s.SidebarPosition != "left" {
		t.Errorf("sidebar_position = %q, want 'left'", s.SidebarPosition)
	}
	if s.NewLineTrigger != "shift-enter" {
		t.Errorf("newline_trigger = %q, want 'shift-enter'", s.NewLineTrigger)
	}
	if s.CardPosition != "append" {
		t.Errorf("card_position = %q, want 'append'", s.CardPosition)
	}
}

func TestLoadSaveSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	h := &Handler{
		ws: &workspace.Workspace{Dir: dir},
	}

	// Load with no file → defaults
	s := h.loadSettings()
	if s.Theme != "system" {
		t.Errorf("default theme = %q", s.Theme)
	}

	// Save custom settings
	s.Theme = "dark"
	s.ColumnWidth = 400
	if err := h.saveSettings(s); err != nil {
		t.Fatal(err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, "settings.json")); err != nil {
		t.Fatalf("settings file not found: %v", err)
	}

	// Load back
	loaded := h.loadSettings()
	if loaded.Theme != "dark" {
		t.Errorf("loaded theme = %q", loaded.Theme)
	}
	if loaded.ColumnWidth != 400 {
		t.Errorf("loaded column_width = %d", loaded.ColumnWidth)
	}
}

func TestValidColorThemes(t *testing.T) {
	expected := []string{"default", "github", "gitlab", "emerald", "rose", "sunset", "aqua", "graphite", "macos"}
	for _, theme := range expected {
		if !validColorThemes[theme] {
			t.Errorf("expected %q to be valid", theme)
		}
	}
	if validColorThemes["nonexistent"] {
		t.Error("expected 'nonexistent' to be invalid")
	}
}

func TestSettingsAPIEmptyColumns(t *testing.T) {
	h := &Handler{ws: &workspace.Workspace{Dir: t.TempDir()}}
	handler := h.SettingsAPIHandler()

	body := `{"theme":"dark","color_theme":"aqua","column_width":280,"sidebar_position":"left","default_columns":[],"newline_trigger":"shift-enter","card_position":"append"}`
	req := httptest.NewRequest("POST", "/api/settings", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var s AppSettings
	if err := json.NewDecoder(w.Body).Decode(&s); err != nil {
		t.Fatal(err)
	}
	if len(s.DefaultColumns) == 0 {
		t.Error("expected default columns to be restored when empty")
	}
}

func TestSettingsAPIHighColumnWidth(t *testing.T) {
	h := &Handler{ws: &workspace.Workspace{Dir: t.TempDir()}}
	handler := h.SettingsAPIHandler()

	body := `{"theme":"light","color_theme":"aqua","column_width":999,"sidebar_position":"left","newline_trigger":"shift-enter","card_position":"append"}`
	req := httptest.NewRequest("POST", "/api/settings", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var s AppSettings
	if err := json.NewDecoder(w.Body).Decode(&s); err != nil {
		t.Fatal(err)
	}
	if s.ColumnWidth != 280 {
		t.Errorf("expected column_width=280 for out-of-range value, got %d", s.ColumnWidth)
	}
}

// --- Handler / event handler tests ---

func TestBoardListModel(t *testing.T) {
	h, _ := setupHandlerWithBoard(t)

	model, err := h.boardListModel()
	if err != nil {
		t.Fatal(err)
	}
	if model.Title != "LiveBoard" {
		t.Errorf("title = %q", model.Title)
	}
	if len(model.Boards) != 1 {
		t.Fatalf("boards = %d, want 1", len(model.Boards))
	}
	if model.Boards[0].Name != "test-board" {
		t.Errorf("board name = %q", model.Boards[0].Name)
	}
}

func TestMountBoardList(t *testing.T) {
	h, _ := setupHandlerWithBoard(t)

	result, err := h.mountBoardList(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	model, ok := result.(BoardListModel)
	if !ok {
		t.Fatalf("unexpected type %T", result)
	}
	if len(model.Boards) != 1 {
		t.Errorf("boards = %d", len(model.Boards))
	}
}

func TestHandleParams(t *testing.T) {
	h, _ := setupHandlerWithBoard(t)

	result, err := h.handleParams(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	model := result.(BoardListModel)
	if len(model.Boards) != 1 {
		t.Errorf("boards = %d", len(model.Boards))
	}
}

func TestHandleCreateBoard(t *testing.T) {
	h, _ := setupHandlerWithBoard(t)

	// Missing name
	result, err := h.handleCreateBoard(context.Background(), nil, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	model := result.(BoardListModel)
	if model.Error == "" {
		t.Error("expected error for missing name")
	}

	// Valid creation
	result, err = h.handleCreateBoard(context.Background(), nil, map[string]interface{}{"name": "new-board"})
	if err != nil {
		t.Fatal(err)
	}
	model = result.(BoardListModel)
	if model.Error != "" {
		t.Errorf("unexpected error: %s", model.Error)
	}
	if len(model.Boards) != 2 {
		t.Errorf("boards = %d, want 2", len(model.Boards))
	}
}

func TestHandleDeleteBoard(t *testing.T) {
	h, _ := setupHandlerWithBoard(t)

	// Missing name
	result, err := h.handleDeleteBoard(context.Background(), nil, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	model := result.(BoardListModel)
	if model.Error == "" {
		t.Error("expected error for missing name")
	}

	// Valid deletion
	result, err = h.handleDeleteBoard(context.Background(), nil, map[string]interface{}{"name": "test-board"})
	if err != nil {
		t.Fatal(err)
	}
	model = result.(BoardListModel)
	if model.Error != "" {
		t.Errorf("unexpected error: %s", model.Error)
	}
	if len(model.Boards) != 0 {
		t.Errorf("boards = %d, want 0", len(model.Boards))
	}
}

func TestHandleSetBoardIconList(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	// Missing slug
	result, _ := h.handleSetBoardIconList(context.Background(), nil, map[string]interface{}{})
	model := result.(BoardListModel)
	if model.Error == "" {
		t.Error("expected error for missing slug")
	}

	// Valid
	result, _ = h.handleSetBoardIconList(context.Background(), nil, map[string]interface{}{
		"name": slug, "icon": "🚀",
	})
	model = result.(BoardListModel)
	if model.Error != "" {
		t.Errorf("unexpected error: %s", model.Error)
	}
}

func TestBoardViewModel(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	model, err := h.boardViewModel(slug)
	if err != nil {
		t.Fatal(err)
	}
	if model.Board == nil {
		t.Fatal("board is nil")
	}
	if model.BoardSlug != slug {
		t.Errorf("slug = %q", model.BoardSlug)
	}
	if model.Title == "" {
		t.Error("title is empty")
	}

	// Nonexistent board
	model, err = h.boardViewModel("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if model.Error == "" {
		t.Error("expected error for nonexistent board")
	}
}

func TestHandleCreateCard(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	// Missing column
	result, _ := h.handleCreateCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "title": "Card",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing column")
	}

	// Missing title
	result, _ = h.handleCreateCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "column": "not now",
	})
	model = result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing title")
	}

	// Missing slug
	result, _ = h.handleCreateCard(context.Background(), nil, map[string]interface{}{
		"column": "not now", "title": "Card",
	})
	model = result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing slug")
	}

	// Valid
	result, _ = h.handleCreateCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "column": "not now", "title": "New Card",
	})
	model = result.(BoardViewModel)
	if model.Error != "" {
		t.Errorf("unexpected error: %s", model.Error)
	}
	if len(model.Board.Columns[0].Cards) != 1 {
		t.Errorf("cards = %d, want 1", len(model.Board.Columns[0].Cards))
	}
}

func TestHandleMoveCard(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	// Add a card first
	h.handleCreateCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "column": "not now", "title": "Move me",
	})

	// Missing col_idx
	result, _ := h.handleMoveCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "card_idx": "0", "target_column": "done",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing col_idx")
	}

	// Missing target_column
	result, _ = h.handleMoveCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "col_idx": "0", "card_idx": "0",
	})
	model = result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing target_column")
	}

	// Missing slug
	result, _ = h.handleMoveCard(context.Background(), nil, map[string]interface{}{
		"col_idx": "0", "card_idx": "0", "target_column": "done",
	})
	model = result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing slug")
	}

	// Valid move
	result, _ = h.handleMoveCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "col_idx": "0", "card_idx": "0", "target_column": "done",
	})
	model = result.(BoardViewModel)
	if model.Error != "" {
		t.Errorf("unexpected error: %s", model.Error)
	}
}

func TestHandleReorderCard(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	// Add two cards
	h.handleCreateCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "column": "not now", "title": "A",
	})
	h.handleCreateCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "column": "not now", "title": "B",
	})

	// Missing column
	result, _ := h.handleReorderCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "col_idx": "0", "card_idx": "1",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing column")
	}

	// Missing slug
	result, _ = h.handleReorderCard(context.Background(), nil, map[string]interface{}{
		"col_idx": "0", "card_idx": "0", "column": "not now",
	})
	model = result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing slug")
	}

	// Valid reorder with before_idx
	result, _ = h.handleReorderCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "col_idx": "0", "card_idx": "1", "column": "not now", "before_idx": "0",
	})
	model = result.(BoardViewModel)
	if model.Error != "" {
		t.Errorf("unexpected error: %s", model.Error)
	}
}

func TestHandleDeleteCard(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	h.handleCreateCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "column": "not now", "title": "Delete me",
	})

	// Missing slug
	result, _ := h.handleDeleteCard(context.Background(), nil, map[string]interface{}{
		"col_idx": "0", "card_idx": "0",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing slug")
	}

	// Valid delete
	result, _ = h.handleDeleteCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "col_idx": "0", "card_idx": "0",
	})
	model = result.(BoardViewModel)
	if model.Error != "" {
		t.Errorf("unexpected error: %s", model.Error)
	}
}

func TestHandleToggleComplete(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	h.handleCreateCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "column": "not now", "title": "Complete me",
	})

	// Missing slug
	result, _ := h.handleToggleComplete(context.Background(), nil, map[string]interface{}{
		"col_idx": "0", "card_idx": "0",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing slug")
	}

	// Valid toggle
	result, _ = h.handleToggleComplete(context.Background(), nil, map[string]interface{}{
		"name": slug, "col_idx": "0", "card_idx": "0",
	})
	model = result.(BoardViewModel)
	if model.Error != "" {
		t.Errorf("unexpected error: %s", model.Error)
	}
	if !model.Board.Columns[0].Cards[0].Completed {
		t.Error("card should be completed")
	}
}

func TestHandleEditCard(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	h.handleCreateCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "column": "not now", "title": "Edit me",
	})

	// Missing slug
	result, _ := h.handleEditCard(context.Background(), nil, map[string]interface{}{
		"col_idx": "0", "card_idx": "0", "title": "Edited",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing slug")
	}

	// Valid edit with tags
	result, _ = h.handleEditCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "col_idx": "0", "card_idx": "0",
		"title": "Edited", "body": "body text", "tags": "a, b, ,c",
		"priority": "high", "due": "2025-12-01",
	})
	model = result.(BoardViewModel)
	if model.Error != "" {
		t.Errorf("unexpected error: %s", model.Error)
	}
	card := model.Board.Columns[0].Cards[0]
	if card.Title != "Edited" {
		t.Errorf("title = %q", card.Title)
	}
	if len(card.Tags) != 3 {
		t.Errorf("tags = %v, want [a b c]", card.Tags)
	}
}

func TestHandleCreateColumn(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	// Missing column name
	result, _ := h.handleCreateColumn(context.Background(), nil, map[string]interface{}{
		"name": slug,
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing column name")
	}

	// Missing slug
	result, _ = h.handleCreateColumn(context.Background(), nil, map[string]interface{}{
		"column_name": "QA",
	})
	model = result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing slug")
	}

	// Valid
	result, _ = h.handleCreateColumn(context.Background(), nil, map[string]interface{}{
		"name": slug, "column_name": "QA",
	})
	model = result.(BoardViewModel)
	if model.Error != "" {
		t.Errorf("unexpected error: %s", model.Error)
	}
}

func TestHandleRenameColumn(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	// Missing old_name
	result, _ := h.handleRenameColumn(context.Background(), nil, map[string]interface{}{
		"name": slug, "new_name": "Todo",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing old_name")
	}

	// Missing new_name
	result, _ = h.handleRenameColumn(context.Background(), nil, map[string]interface{}{
		"name": slug, "old_name": "not now",
	})
	model = result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing new_name")
	}

	// Missing slug
	result, _ = h.handleRenameColumn(context.Background(), nil, map[string]interface{}{
		"old_name": "not now", "new_name": "Todo",
	})
	model = result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing slug")
	}

	// Valid
	result, _ = h.handleRenameColumn(context.Background(), nil, map[string]interface{}{
		"name": slug, "old_name": "not now", "new_name": "Todo",
	})
	model = result.(BoardViewModel)
	if model.Error != "" {
		t.Errorf("unexpected error: %s", model.Error)
	}
}

func TestHandleDeleteColumn(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	// Missing column name
	result, _ := h.handleDeleteColumn(context.Background(), nil, map[string]interface{}{
		"name": slug,
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing column name")
	}

	// Missing slug
	result, _ = h.handleDeleteColumn(context.Background(), nil, map[string]interface{}{
		"column_name": "done",
	})
	model = result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing slug")
	}

	// Valid
	result, _ = h.handleDeleteColumn(context.Background(), nil, map[string]interface{}{
		"name": slug, "column_name": "done",
	})
	model = result.(BoardViewModel)
	if model.Error != "" {
		t.Errorf("unexpected error: %s", model.Error)
	}
}

func TestHandleUpdateBoardMeta(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	// Missing slug
	result, _ := h.handleUpdateBoardMeta(context.Background(), nil, map[string]interface{}{
		"board_name": "New Name",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing slug")
	}

	// Valid with tags
	result, _ = h.handleUpdateBoardMeta(context.Background(), nil, map[string]interface{}{
		"name": slug, "board_name": "Renamed", "description": "desc", "tags": "x, y",
	})
	model = result.(BoardViewModel)
	if model.Error != "" {
		t.Errorf("unexpected error: %s", model.Error)
	}
	if model.Board.Name != "Renamed" {
		t.Errorf("name = %q", model.Board.Name)
	}
}

func TestHandleToggleColumnCollapse(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	// Missing slug
	result, _ := h.handleToggleColumnCollapse(context.Background(), nil, map[string]interface{}{
		"col_index": "0",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing slug")
	}

	// Missing col_index
	result, _ = h.handleToggleColumnCollapse(context.Background(), nil, map[string]interface{}{
		"name": slug,
	})
	model = result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing col_index")
	}

	// Valid
	result, _ = h.handleToggleColumnCollapse(context.Background(), nil, map[string]interface{}{
		"name": slug, "col_index": "0",
	})
	model = result.(BoardViewModel)
	if model.Error != "" {
		t.Errorf("unexpected error: %s", model.Error)
	}
}

func TestHandleSortColumn(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	// Add cards to sort
	h.handleCreateCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "column": "not now", "title": "B",
	})
	h.handleCreateCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "column": "not now", "title": "A",
	})

	// Missing sort_by
	result, _ := h.handleSortColumn(context.Background(), nil, map[string]interface{}{
		"name": slug, "col_idx": "0",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing sort_by")
	}

	// Missing slug
	result, _ = h.handleSortColumn(context.Background(), nil, map[string]interface{}{
		"col_idx": "0", "sort_by": "name",
	})
	model = result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing slug")
	}

	// Valid
	result, _ = h.handleSortColumn(context.Background(), nil, map[string]interface{}{
		"name": slug, "col_idx": "0", "sort_by": "name",
	})
	model = result.(BoardViewModel)
	if model.Error != "" {
		t.Errorf("unexpected error: %s", model.Error)
	}
}

func TestHandleUpdateBoardSettings(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	// Missing slug
	result, _ := h.handleUpdateBoardSettings(context.Background(), nil, map[string]interface{}{
		"show_checkbox": "true",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing slug")
	}

	// Valid with all settings
	result, _ = h.handleUpdateBoardSettings(context.Background(), nil, map[string]interface{}{
		"name": slug, "show_checkbox": "true", "card_position": "prepend", "expand_columns": "true",
	})
	model = result.(BoardViewModel)
	if model.Error != "" {
		t.Errorf("unexpected error: %s", model.Error)
	}
}

func TestHandleSetBoardIcon(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	// Missing slug
	result, _ := h.handleSetBoardIcon(context.Background(), nil, map[string]interface{}{
		"icon": "🎯",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing slug")
	}

	// Valid
	result, _ = h.handleSetBoardIcon(context.Background(), nil, map[string]interface{}{
		"name": slug, "icon": "🎯",
	})
	model = result.(BoardViewModel)
	if model.Error != "" {
		t.Errorf("unexpected error: %s", model.Error)
	}
}

func TestHandleBoardUpdate(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	// Valid message
	result, err := h.handleBoardUpdate(context.Background(), nil, slug)
	if err != nil {
		t.Fatal(err)
	}
	model := result.(BoardViewModel)
	if model.Board == nil {
		t.Error("board is nil")
	}

	// Invalid message type
	_, err = h.handleBoardUpdate(context.Background(), nil, 123)
	if err == nil {
		t.Error("expected error for invalid message type")
	}
}

func TestCommitWithHandlingNilGit(t *testing.T) {
	h := &Handler{git: nil}
	// Should be a no-op, no panic.
	h.commitWithHandling("/path", "msg")
}

func TestCommitRemoveWithHandlingNilGit(t *testing.T) {
	h := &Handler{git: nil}
	// Should be a no-op, no panic.
	h.commitRemoveWithHandling("/path", "msg")
}

func TestMountBoardViewNilSocket(t *testing.T) {
	h, _ := setupHandlerWithBoard(t)

	// nil socket + nil request context → should return error model
	result, err := h.mountBoardView(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	model := result.(BoardViewModel)
	// live.Request(ctx) returns nil when no request in context, so error expected
	if model.Error == "" {
		t.Error("expected error for nil request context")
	}
}

func TestToBoardSettingsViewExpandColumnsTrue(t *testing.T) {
	expandTrue := true
	v := toBoardSettingsView(models.BoardSettings{
		ExpandColumns: &expandTrue,
	})
	if v.ExpandColumns != "true" {
		t.Errorf("ExpandColumns = %q, want 'true'", v.ExpandColumns)
	}
}

func TestMutateBoardError(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	// op that returns an error
	result, err := h.mutateBoard(slug, "test", func(_ string) error {
		return os.ErrNotExist
	})
	if err != nil {
		t.Fatal(err)
	}
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error from failed op")
	}
}

func TestMutateBoardRemoveError(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	result, err := h.mutateBoardRemove(slug, "test", func(_ string) error {
		return os.ErrNotExist
	})
	if err != nil {
		t.Fatal(err)
	}
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error from failed op")
	}
}

func TestNewHandler(t *testing.T) {
	dir := t.TempDir()
	ws := workspace.Open(dir)
	eng := board.New()

	h := NewHandler(ws, eng, nil)
	if h == nil {
		t.Fatal("handler is nil")
	}
	if h.ws != ws {
		t.Error("ws not set")
	}
	if h.eng != eng {
		t.Error("eng not set")
	}
	if h.pubsub == nil {
		t.Error("pubsub is nil")
	}
}

func TestPublishBoardEvent(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)
	// Should not panic.
	h.publishBoardEvent(slug)
}

func TestBoardListHandler(t *testing.T) {
	h := NewHandler(workspace.Open(t.TempDir()), board.New(), nil)
	handler := h.BoardListHandler()
	if handler == nil {
		t.Fatal("BoardListHandler returned nil")
	}
}

func TestBoardViewHandler(t *testing.T) {
	h := NewHandler(workspace.Open(t.TempDir()), board.New(), nil)
	handler := h.BoardViewHandler()
	if handler == nil {
		t.Fatal("BoardViewHandler returned nil")
	}
}

func TestHandleCreateBoardDuplicate(t *testing.T) {
	h, _ := setupHandlerWithBoard(t)

	// Creating a duplicate board should return an error model (not panic).
	result, _ := h.handleCreateBoard(context.Background(), nil, map[string]interface{}{"name": "test-board"})
	model := result.(BoardListModel)
	if model.Error == "" {
		t.Error("expected error for duplicate board")
	}
}

func TestHandleDeleteBoardNotFound(t *testing.T) {
	h, _ := setupHandlerWithBoard(t)

	result, _ := h.handleDeleteBoard(context.Background(), nil, map[string]interface{}{"name": "nonexistent"})
	model := result.(BoardListModel)
	if model.Error == "" {
		t.Error("expected error for nonexistent board")
	}
}

func TestHandleSetBoardIconListError(t *testing.T) {
	h, _ := setupHandlerWithBoard(t)

	// Board slug that doesn't exist
	result, _ := h.handleSetBoardIconList(context.Background(), nil, map[string]interface{}{
		"name": "nonexistent", "icon": "🎯",
	})
	model := result.(BoardListModel)
	if model.Error == "" {
		t.Error("expected error for nonexistent board")
	}
}

func TestHandleMoveCardMissingCardIdx(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	result, _ := h.handleMoveCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "col_idx": "0", "target_column": "done",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing card_idx")
	}
}

func TestHandleReorderCardMissingCardIdx(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	result, _ := h.handleReorderCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "col_idx": "0", "column": "not now",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing card_idx")
	}
}

func TestHandleDeleteCardMissingCardIdx(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	result, _ := h.handleDeleteCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "col_idx": "0",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing card_idx")
	}
}

func TestHandleToggleCompleteMissingCardIdx(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	result, _ := h.handleToggleComplete(context.Background(), nil, map[string]interface{}{
		"name": slug, "col_idx": "0",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing card_idx")
	}
}

func TestHandleEditCardMissingCardIdx(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	result, _ := h.handleEditCard(context.Background(), nil, map[string]interface{}{
		"name": slug, "col_idx": "0",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing card_idx")
	}
}

func TestHandleSortColumnMissingColIdx(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	result, _ := h.handleSortColumn(context.Background(), nil, map[string]interface{}{
		"name": slug, "sort_by": "name",
	})
	model := result.(BoardViewModel)
	if model.Error == "" {
		t.Error("expected error for missing col_idx")
	}
}

func TestCommitWithHandlingGitError(t *testing.T) {
	// Use a git repo but commit a nonexistent file to trigger the error log path.
	dir := t.TempDir()
	ws := workspace.Open(dir)

	gitRepo, err := gitpkg.Open(dir, true)
	if err != nil {
		t.Fatal(err)
	}

	h := &Handler{ws: ws, git: gitRepo}
	// Should not panic; logs the error internally.
	h.commitWithHandling("nonexistent-file.md", "test")
}

func TestCommitRemoveWithHandlingGitError(t *testing.T) {
	dir := t.TempDir()
	ws := workspace.Open(dir)

	gitRepo, err := gitpkg.Open(dir, true)
	if err != nil {
		t.Fatal(err)
	}

	h := &Handler{ws: ws, git: gitRepo}
	// Should not panic; logs the error internally.
	h.commitRemoveWithHandling("nonexistent-file.md", "test")
}
