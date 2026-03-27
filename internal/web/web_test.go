package web

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/workspace"
	"github.com/and1truong/liveboard/pkg/models"
)

// setupHandlerWithBoard creates a Handler with a real workspace containing one board.
func setupHandlerWithBoard(t *testing.T) (*Handler, string) {
	t.Helper()
	dir := t.TempDir()
	ws := workspace.Open(dir)
	eng := board.New()

	h := &Handler{
		ws:  ws,
		eng: eng,
		SSE: NewSSEBroker(),
	}

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
	v := toBoardSettingsView(models.BoardSettings{})
	if v.ShowCheckbox != "" || v.CardPosition != "" || v.ExpandColumns != "" {
		t.Errorf("expected empty strings for nil settings, got %+v", v)
	}

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

func TestFormInt(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader("col_idx=5"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	v, err := formInt(req, "col_idx")
	if err != nil {
		t.Fatal(err)
	}
	if v != 5 {
		t.Errorf("got %d, want 5", v)
	}

	// Missing key (need a fresh request since form was already parsed)
	req2 := httptest.NewRequest("POST", "/", strings.NewReader("other=1"))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = formInt(req2, "missing")
	if err == nil {
		t.Error("expected error for missing key")
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
	if s.SiteName != "LiveBoard" {
		t.Errorf("SiteName = %q, want 'LiveBoard'", s.SiteName)
	}
}

func TestSettingsAPIHandler(t *testing.T) {
	dir := t.TempDir()
	h := &Handler{
		ws: &workspace.Workspace{Dir: dir},
	}

	handler := h.SettingsAPIHandler()

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

	body := `{"theme":"dark","color_theme":"gitlab","column_width":300,"sidebar_position":"right","show_checkbox":false,"newline_trigger":"enter","card_position":"prepend"}`
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
	if saved.ColorTheme != "gitlab" {
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
	if s.ColorTheme != "aqua" {
		t.Errorf("color_theme = %q, want 'aqua'", s.ColorTheme)
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

	s := h.loadSettings()
	if s.Theme != "system" {
		t.Errorf("default theme = %q", s.Theme)
	}

	s.Theme = "dark"
	s.ColumnWidth = 400
	if err := h.saveSettings(s); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "settings.json")); err != nil {
		t.Fatalf("settings file not found: %v", err)
	}

	loaded := h.loadSettings()
	if loaded.Theme != "dark" {
		t.Errorf("loaded theme = %q", loaded.Theme)
	}
	if loaded.ColumnWidth != 400 {
		t.Errorf("loaded column_width = %d", loaded.ColumnWidth)
	}
}

func TestValidColorThemes(t *testing.T) {
	expected := []string{"gitlab", "emerald", "rose", "aqua"}
	for _, theme := range expected {
		if !validColorThemes[theme] {
			t.Errorf("expected %q to be valid", theme)
		}
	}
	if validColorThemes["nonexistent"] {
		t.Error("expected 'nonexistent' to be invalid")
	}
}

func TestSanitizeSettingsSiteName(t *testing.T) {
	// Empty → default
	s := AppSettings{}
	sanitizeSettings(&s)
	if s.SiteName != "LiveBoard" {
		t.Errorf("empty site_name = %q, want 'LiveBoard'", s.SiteName)
	}

	// Whitespace-only → default
	s = AppSettings{SiteName: "   "}
	sanitizeSettings(&s)
	if s.SiteName != "LiveBoard" {
		t.Errorf("whitespace site_name = %q, want 'LiveBoard'", s.SiteName)
	}

	// Trimmed
	s = AppSettings{SiteName: "  MyBoard  "}
	sanitizeSettings(&s)
	if s.SiteName != "MyBoard" {
		t.Errorf("trimmed site_name = %q, want 'MyBoard'", s.SiteName)
	}

	// Truncated to 50 runes
	long := strings.Repeat("あ", 60)
	s = AppSettings{SiteName: long}
	sanitizeSettings(&s)
	if len([]rune(s.SiteName)) != 50 {
		t.Errorf("truncated site_name rune len = %d, want 50", len([]rune(s.SiteName)))
	}

	// Exactly 50 runes — no truncation
	exact := strings.Repeat("x", 50)
	s = AppSettings{SiteName: exact}
	sanitizeSettings(&s)
	if s.SiteName != exact {
		t.Errorf("50-char site_name was modified")
	}

	// Valid name passes through
	s = AppSettings{SiteName: "Acme Corp"}
	sanitizeSettings(&s)
	if s.SiteName != "Acme Corp" {
		t.Errorf("valid site_name = %q", s.SiteName)
	}
}

func TestSettingsAPISiteName(t *testing.T) {
	h := &Handler{ws: &workspace.Workspace{Dir: t.TempDir()}}
	handler := h.SettingsAPIHandler()

	// Default GET returns "LiveBoard"
	req := httptest.NewRequest("GET", "/api/settings", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var got AppSettings
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.SiteName != "LiveBoard" {
		t.Errorf("default site_name = %q", got.SiteName)
	}

	// POST custom site name
	body := `{"site_name":"My Team","theme":"dark","color_theme":"aqua","column_width":280,"sidebar_position":"left","newline_trigger":"shift-enter","card_position":"append"}`
	req = httptest.NewRequest("POST", "/api/settings", strings.NewReader(body))
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var saved AppSettings
	if err := json.NewDecoder(w.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.SiteName != "My Team" {
		t.Errorf("saved site_name = %q, want 'My Team'", saved.SiteName)
	}

	// Verify persisted
	req = httptest.NewRequest("GET", "/api/settings", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var reloaded AppSettings
	if err := json.NewDecoder(w.Body).Decode(&reloaded); err != nil {
		t.Fatal(err)
	}
	if reloaded.SiteName != "My Team" {
		t.Errorf("reloaded site_name = %q", reloaded.SiteName)
	}

	// POST empty site name → defaults to "LiveBoard"
	body = `{"site_name":"","theme":"dark","color_theme":"aqua","column_width":280,"sidebar_position":"left","newline_trigger":"shift-enter","card_position":"append"}`
	req = httptest.NewRequest("POST", "/api/settings", strings.NewReader(body))
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if err := json.NewDecoder(w.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.SiteName != "LiveBoard" {
		t.Errorf("empty site_name should default to 'LiveBoard', got %q", saved.SiteName)
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

// --- Model / helper tests ---

func TestBoardListModel(t *testing.T) {
	h, _ := setupHandlerWithBoard(t)

	model, err := h.boardListModel()
	if err != nil {
		t.Fatal(err)
	}
	if model.Title != "LiveBoard" {
		t.Errorf("title = %q", model.Title)
	}
	if model.SiteName != "LiveBoard" {
		t.Errorf("site_name = %q, want 'LiveBoard'", model.SiteName)
	}
	if len(model.Boards) != 1 {
		t.Fatalf("boards = %d, want 1", len(model.Boards))
	}
	if model.Boards[0].Name != "test-board" {
		t.Errorf("board name = %q", model.Boards[0].Name)
	}
}

func TestBoardListModelCustomSiteName(t *testing.T) {
	h, _ := setupHandlerWithBoard(t)

	s := h.loadSettings()
	s.SiteName = "MyKanban"
	if err := h.saveSettings(s); err != nil {
		t.Fatal(err)
	}

	model, err := h.boardListModel()
	if err != nil {
		t.Fatal(err)
	}
	if model.Title != "MyKanban" {
		t.Errorf("title = %q, want 'MyKanban'", model.Title)
	}
	if model.SiteName != "MyKanban" {
		t.Errorf("site_name = %q, want 'MyKanban'", model.SiteName)
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
	if model.SiteName != "LiveBoard" {
		t.Errorf("site_name = %q, want 'LiveBoard'", model.SiteName)
	}
	if !strings.Contains(model.Title, "LiveBoard") {
		t.Errorf("title %q should contain 'LiveBoard'", model.Title)
	}

	// Nonexistent board
	model, err = h.boardViewModel("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent board")
	}
	if model.Error == "" {
		t.Error("expected error message in model for nonexistent board")
	}
}

func TestBoardViewModelCustomSiteName(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	s := h.loadSettings()
	s.SiteName = "TeamBoard"
	if err := h.saveSettings(s); err != nil {
		t.Fatal(err)
	}

	model, err := h.boardViewModel(slug)
	if err != nil {
		t.Fatal(err)
	}
	if model.SiteName != "TeamBoard" {
		t.Errorf("site_name = %q, want 'TeamBoard'", model.SiteName)
	}
	if !strings.Contains(model.Title, "TeamBoard") {
		t.Errorf("title %q should contain 'TeamBoard'", model.Title)
	}
}

func TestMutateBoardError(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	model, err := h.mutateBoard(slug, -1, func(_ *models.Board) error {
		return os.ErrNotExist
	})
	if err != nil {
		t.Fatal(err)
	}
	if model.Error == "" {
		t.Error("expected error from failed op")
	}
}

func TestMutateBoardRemoveError(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	model, err := h.mutateBoard(slug, -1, func(_ *models.Board) error {
		return os.ErrNotExist
	})
	if err != nil {
		t.Fatal(err)
	}
	if model.Error == "" {
		t.Error("expected error from failed op")
	}
}

// --- Optimistic concurrency tests ---

func TestFormVersion(t *testing.T) {
	// Present and valid
	req := httptest.NewRequest("POST", "/", strings.NewReader("version=3"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if v := formVersion(req); v != 3 {
		t.Errorf("formVersion = %d, want 3", v)
	}

	// Missing — returns -1 (skip check)
	req = httptest.NewRequest("POST", "/", strings.NewReader("other=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if v := formVersion(req); v != -1 {
		t.Errorf("formVersion = %d, want -1 for missing", v)
	}

	// Invalid — returns -1
	req = httptest.NewRequest("POST", "/", strings.NewReader("version=abc"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if v := formVersion(req); v != -1 {
		t.Errorf("formVersion = %d, want -1 for invalid", v)
	}
}

func TestMutateBoardVersionConflictReturnsError(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	// First mutation at version 0 — succeeds.
	_, err := h.mutateBoard(slug, 0, func(b *models.Board) error {
		b.Name = "V1"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Second mutation with stale version 0 — should return ErrVersionConflict.
	_, err = h.mutateBoard(slug, 0, func(b *models.Board) error {
		b.Name = "Should Fail"
		return nil
	})
	if err != board.ErrVersionConflict {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
}

func TestMutateBoardNoVersionSkipsCheck(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	// Mutate without version (-1) — should always succeed.
	_, err := h.mutateBoard(slug, -1, func(b *models.Board) error {
		b.Name = "No Version"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Again with -1 — should still succeed even though version changed.
	_, err = h.mutateBoard(slug, -1, func(b *models.Board) error {
		b.Name = "Still No Version"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBoardViewModelIncludesVersion(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	model, err := h.boardViewModel(slug)
	if err != nil {
		t.Fatal(err)
	}
	if model.Version != 0 {
		t.Errorf("initial version = %d, want 0", model.Version)
	}

	// Mutate to increment version.
	_, _ = h.mutateBoard(slug, -1, func(b *models.Board) error {
		b.Name = "V1"
		return nil
	})

	model, err = h.boardViewModel(slug)
	if err != nil {
		t.Fatal(err)
	}
	if model.Version != 1 {
		t.Errorf("version after mutation = %d, want 1", model.Version)
	}
}

func TestMutateBoardRemoveVersionConflict(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)

	// Advance version.
	_, _ = h.mutateBoard(slug, -1, func(b *models.Board) error {
		b.Name = "V1"
		return nil
	})

	// Remove with stale version — should conflict.
	_, err := h.mutateBoard(slug, 0, func(_ *models.Board) error {
		return nil
	})
	if err != board.ErrVersionConflict {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
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

func TestNewHandler(t *testing.T) {
	dir := t.TempDir()
	ws := workspace.Open(dir)
	eng := board.New()

	h := NewHandler(ws, eng, "test", false)
	if h == nil {
		t.Fatal("handler is nil")
	}
	if h.ws != ws {
		t.Error("ws not set")
	}
	if h.eng != eng {
		t.Error("eng not set")
	}
	if h.SSE == nil {
		t.Error("SSE broker is nil")
	}
}

func TestPublishBoardEvent(t *testing.T) {
	h, slug := setupHandlerWithBoard(t)
	// Should not panic.
	h.publishBoardEvent(slug)
}

// --- SSE broker tests ---

func TestSSEBrokerPubSub(t *testing.T) {
	broker := NewSSEBroker()

	ch := broker.Subscribe("test")
	defer broker.Unsubscribe("test", ch)

	broker.Publish("test")

	select {
	case msg := <-ch:
		if msg != "test" {
			t.Errorf("got %q, want 'test'", msg)
		}
	default:
		t.Error("expected message on channel")
	}
}

func TestSSEBrokerUnsubscribe(t *testing.T) {
	broker := NewSSEBroker()
	ch := broker.Subscribe("test")
	broker.Unsubscribe("test", ch)

	broker.Publish("test")

	select {
	case <-ch:
		t.Error("should not receive after unsubscribe")
	default:
		// Expected
	}
}
