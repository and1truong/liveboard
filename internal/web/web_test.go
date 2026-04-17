package web

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/parser"
	"github.com/and1truong/liveboard/internal/workspace"
	"github.com/and1truong/liveboard/pkg/models"
)

// setupHandlerWithBoard creates a Handler with a real workspace containing one board.
func setupHandlerWithBoard(t *testing.T) (*Handler, string) {
	t.Helper()
	dir := t.TempDir()
	ws := workspace.Open(dir)
	eng := board.New()

	h := NewHandler(ws, eng, "", false, false)

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
	b := &Base{ws: &workspace.Workspace{Dir: dir}}
	h := &Handler{Base: b, Settings: &SettingsHandler{Base: b}}

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

	body := `{"theme":"dark","color_theme":"emerald","column_width":300,"sidebar_position":"right","show_checkbox":false,"newline_trigger":"enter","card_position":"prepend"}`
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
	if saved.ColorTheme != "emerald" {
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
	b := &Base{ws: &workspace.Workspace{Dir: dir}}
	h := &Handler{Base: b, Settings: &SettingsHandler{Base: b}}

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
	h := &Handler{Base: &Base{
		ws: &workspace.Workspace{Dir: dir},
	}}

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
	expected := []string{"emerald", "rose", "aqua"}
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
	SanitizeSettings(&s)
	if s.SiteName != "LiveBoard" {
		t.Errorf("empty site_name = %q, want 'LiveBoard'", s.SiteName)
	}

	// Whitespace-only → default
	s = AppSettings{SiteName: "   "}
	SanitizeSettings(&s)
	if s.SiteName != "LiveBoard" {
		t.Errorf("whitespace site_name = %q, want 'LiveBoard'", s.SiteName)
	}

	// Trimmed
	s = AppSettings{SiteName: "  MyBoard  "}
	SanitizeSettings(&s)
	if s.SiteName != "MyBoard" {
		t.Errorf("trimmed site_name = %q, want 'MyBoard'", s.SiteName)
	}

	// Truncated to 50 runes
	long := strings.Repeat("あ", 60)
	s = AppSettings{SiteName: long}
	SanitizeSettings(&s)
	if len([]rune(s.SiteName)) != 50 {
		t.Errorf("truncated site_name rune len = %d, want 50", len([]rune(s.SiteName)))
	}

	// Exactly 50 runes — no truncation
	exact := strings.Repeat("x", 50)
	s = AppSettings{SiteName: exact}
	SanitizeSettings(&s)
	if s.SiteName != exact {
		t.Errorf("50-char site_name was modified")
	}

	// Valid name passes through
	s = AppSettings{SiteName: "Acme Corp"}
	SanitizeSettings(&s)
	if s.SiteName != "Acme Corp" {
		t.Errorf("valid site_name = %q", s.SiteName)
	}
}

func TestSettingsAPISiteName(t *testing.T) {
	b := &Base{ws: &workspace.Workspace{Dir: t.TempDir()}}
	h := &Handler{Base: b, Settings: &SettingsHandler{Base: b}}
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
	b := &Base{ws: &workspace.Workspace{Dir: t.TempDir()}}
	h := &Handler{Base: b, Settings: &SettingsHandler{Base: b}}
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
	b := &Base{ws: &workspace.Workspace{Dir: t.TempDir()}}
	h := &Handler{Base: b, Settings: &SettingsHandler{Base: b}}
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
	if model.BoardVersion != 0 {
		t.Errorf("initial version = %d, want 0", model.BoardVersion)
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
	if model.BoardVersion != 1 {
		t.Errorf("version after mutation = %d, want 1", model.BoardVersion)
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

	h := NewHandler(ws, eng, "test", false, false)
	if h == nil {
		t.Fatal("handler is nil")
		return
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

// --- Helpers for HTTP handler tests ---

func setupTestHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	dir := t.TempDir()
	ws := workspace.Open(dir)
	eng := board.New()
	h := NewHandler(ws, eng, "test", false, false)
	if _, err := ws.CreateBoard("test-board"); err != nil {
		t.Fatal(err)
	}
	return h, "test-board"
}

func withSlug(r *http.Request, slug string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("slug", slug)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func postForm(url string, values map[string]string) *http.Request {
	form := make(neturl.Values)
	for k, v := range values {
		form.Set(k, v)
	}
	r := httptest.NewRequest("POST", url, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

// setupBoardWithColumn creates a board with one column containing cards for mutation tests.
func setupBoardWithColumn(t *testing.T) (*Handler, string) {
	t.Helper()
	h, slug := setupTestHandler(t)
	_, err := h.mutateBoard(slug, -1, func(b *models.Board) error {
		b.Columns = []models.Column{
			{Name: "Todo", Cards: []models.Card{
				{Title: "Card A"},
				{Title: "Card B", Priority: "high", Due: "2026-03-25"},
			}},
			{Name: "Done", Cards: []models.Card{
				{Title: "Card C", Completed: true},
			}},
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return h, slug
}

// --- Board list handler tests ---

func TestBoardListPage(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	h.BoardListPage(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "test-board") {
		t.Error("response should contain board name")
	}
}

func TestHandleCreateBoard(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := postForm("/boards/new", map[string]string{"name": "new-board"})
	h.HandleCreateBoard(w, r)
	if got := w.Header().Get("HX-Redirect"); got != "/board/new-board" {
		t.Errorf("HX-Redirect = %q, want /board/new-board", got)
	}
}

func TestHandleCreateBoard_EmptyName(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := postForm("/boards/new", map[string]string{})
	h.HandleCreateBoard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty body with error")
	}
}

func TestHandleDeleteBoard(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := postForm("/boards/delete", map[string]string{"name": "test-board"})
	h.HandleDeleteBoard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	// Board should be gone
	model, _ := h.boardListModel()
	if len(model.Boards) != 0 {
		t.Errorf("expected 0 boards, got %d", len(model.Boards))
	}
}

func TestHandleDeleteBoard_EmptyName(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := postForm("/boards/delete", map[string]string{})
	h.HandleDeleteBoard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty body with error")
	}
}

func TestHandleTogglePin(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := postForm("/api/boards/pin", map[string]string{"slug": "test-board"})
	h.HandleTogglePin(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	// Verify pinned
	s := h.loadSettings()
	found := false
	for _, p := range s.PinnedBoards {
		if p == "test-board" {
			found = true
		}
	}
	if !found {
		t.Error("board should be pinned")
	}
}

func TestHandleTogglePin_Unpin(t *testing.T) {
	h, _ := setupTestHandler(t)
	// Pin first
	r := postForm("/api/boards/pin", map[string]string{"slug": "test-board"})
	h.HandleTogglePin(httptest.NewRecorder(), r)
	// Unpin
	w := httptest.NewRecorder()
	r = postForm("/api/boards/pin", map[string]string{"slug": "test-board"})
	h.HandleTogglePin(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	s := h.loadSettings()
	for _, p := range s.PinnedBoards {
		if p == "test-board" {
			t.Error("board should be unpinned")
		}
	}
}

func TestHandleTogglePin_EmptySlug(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := postForm("/api/boards/pin", map[string]string{})
	h.HandleTogglePin(w, r)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestHandleSetBoardIconList(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := postForm("/boards/icon", map[string]string{"name": "test-board", "icon": "🚀"})
	h.HandleSetBoardIconList(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleSidebarBoards(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/boards/sidebar?slug=test-board", nil)
	h.HandleSidebarBoards(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty sidebar")
	}
}

func TestHandleBoardsListLite(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	// Create a second board with its own columns.
	const slug2 = "second-board"
	if _, err := h.ws.CreateBoard(slug2); err != nil {
		t.Fatal(err)
	}
	if _, err := h.mutateBoard(slug2, -1, func(b *models.Board) error {
		b.Name = "Second Board"
		b.Columns = []models.Column{
			{Name: "Backlog"},
			{Name: "In Progress"},
			{Name: "Shipped"},
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/boards/list-lite", nil)
	h.HandleBoardsListLite(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}

	var entries []BoardListLiteEntry
	if err := json.NewDecoder(w.Body).Decode(&entries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len = %d, want 2: %+v", len(entries), entries)
	}

	byslug := map[string]BoardListLiteEntry{}
	for _, e := range entries {
		byslug[e.Slug] = e
	}

	e1, ok := byslug[slug]
	if !ok {
		t.Fatalf("missing %q in %+v", slug, entries)
	}
	if want := []string{"Todo", "Done"}; !equalStrings(e1.Columns, want) {
		t.Errorf("first board columns = %v, want %v", e1.Columns, want)
	}
	if e1.Name == "" {
		t.Errorf("first board name empty")
	}

	e2, ok := byslug[slug2]
	if !ok {
		t.Fatalf("missing %q in %+v", slug2, entries)
	}
	if e2.Name != "Second Board" {
		t.Errorf("second board name = %q, want Second Board", e2.Name)
	}
	if want := []string{"Backlog", "In Progress", "Shipped"}; !equalStrings(e2.Columns, want) {
		t.Errorf("second board columns = %v, want %v", e2.Columns, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// --- Card handler tests ---

func TestHandleCreateCard(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/cards", map[string]string{
		"column": "Todo", "title": "New Card",
	})
	r = withSlug(r, slug)
	h.HandleCreateCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	// Verify card added
	b, err := h.ws.LoadBoard(slug)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range b.Columns[0].Cards {
		if c.Title == "New Card" {
			found = true
		}
	}
	if !found {
		t.Error("card not created")
	}
}

func TestHandleCreateCard_MissingFields(t *testing.T) {
	h, slug := setupBoardWithColumn(t)

	// Missing title
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/cards", map[string]string{"column": "Todo"})
	r = withSlug(r, slug)
	h.HandleCreateCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}

	// Missing column
	w = httptest.NewRecorder()
	r = postForm("/board/"+slug+"/cards", map[string]string{"title": "X"})
	r = withSlug(r, slug)
	h.HandleCreateCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}

	// Missing slug
	w = httptest.NewRecorder()
	r = postForm("/board/x/cards", map[string]string{"column": "Todo", "title": "X"})
	// no withSlug — slug will be empty
	h.HandleCreateCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleMoveCard(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/cards/move", map[string]string{
		"col_idx": "0", "card_idx": "0", "target_column": "Done",
	})
	r = withSlug(r, slug)
	h.HandleMoveCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	b, err := h.ws.LoadBoard(slug)
	if err != nil {
		t.Fatal(err)
	}
	// Card A should now be in Done
	found := false
	for _, c := range b.Columns[1].Cards {
		if c.Title == "Card A" {
			found = true
		}
	}
	if !found {
		t.Error("card not moved to Done")
	}
}

func TestHandleMoveCardToBoard(t *testing.T) {
	h, srcSlug := setupTestHandler(t)
	// Seed src with a Todo column + Task A
	_, err := h.mutateBoard(srcSlug, -1, func(b *models.Board) error {
		b.Columns = []models.Column{
			{Name: "Todo", Cards: []models.Card{{Title: "Task A"}}},
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Create dst board with an Inbox column
	if _, cerr := h.ws.CreateBoard("dst"); cerr != nil {
		t.Fatal(cerr)
	}
	_, err = h.mutateBoard("dst", -1, func(b *models.Board) error {
		b.Columns = []models.Column{{Name: "Inbox"}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Load the real source version for the optimistic-lock check; -1 is
	// rejected by HandleMoveCardToBoard.
	srcPre, err := h.ws.LoadBoard(srcSlug)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := postForm("/board/"+srcSlug+"/cards/move-to-board", map[string]string{
		"col_idx": "0", "card_idx": "0", "dst_board": "dst", "dst_column": "Inbox",
		"version": strconv.Itoa(srcPre.Version),
	})
	r = withSlug(r, srcSlug)
	h.HandleMoveCardToBoard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	dst, err := h.ws.LoadBoard("dst")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range dst.Columns[0].Cards {
		if c.Title == "Task A" {
			found = true
		}
	}
	if !found {
		t.Error("Task A was not moved to dst Inbox")
	}
	// Verify the card was moved (not copied): source Todo must no longer contain it.
	src, err := h.ws.LoadBoard(srcSlug)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range src.Columns[0].Cards {
		if c.Title == "Task A" {
			t.Error("Task A still present in source after move")
		}
	}
}

func TestHandleReorderCard(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/cards/reorder", map[string]string{
		"col_idx": "0", "card_idx": "1", "column": "Todo", "before_idx": "0",
	})
	r = withSlug(r, slug)
	h.HandleReorderCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	b, err := h.ws.LoadBoard(slug)
	if err != nil {
		t.Fatal(err)
	}
	if b.Columns[0].Cards[0].Title != "Card B" {
		t.Errorf("expected Card B first, got %q", b.Columns[0].Cards[0].Title)
	}
}

func TestHandleDeleteCard(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/cards/delete", map[string]string{
		"col_idx": "0", "card_idx": "0",
	})
	r = withSlug(r, slug)
	h.HandleDeleteCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	b, err := h.ws.LoadBoard(slug)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Columns[0].Cards) != 1 {
		t.Errorf("expected 1 card in Todo, got %d", len(b.Columns[0].Cards))
	}
}

func TestHandleToggleComplete(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/cards/complete", map[string]string{
		"col_idx": "0", "card_idx": "0",
	})
	r = withSlug(r, slug)
	h.HandleToggleComplete(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	b, err := h.ws.LoadBoard(slug)
	if err != nil {
		t.Fatal(err)
	}
	if !b.Columns[0].Cards[0].Completed {
		t.Error("card should be completed")
	}
}

func TestHandleEditCard(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/cards/edit", map[string]string{
		"col_idx": "0", "card_idx": "0",
		"title": "Updated", "body": "body text", "tags": "go, web",
		"priority": "critical", "due": "2026-12-01", "assignee": "alice",
	})
	r = withSlug(r, slug)
	h.HandleEditCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	b, err := h.ws.LoadBoard(slug)
	if err != nil {
		t.Fatal(err)
	}
	c := b.Columns[0].Cards[0]
	if c.Title != "Updated" {
		t.Errorf("title = %q", c.Title)
	}
	if c.Body != "body text" {
		t.Errorf("body = %q", c.Body)
	}
	if len(c.Tags) != 2 {
		t.Errorf("tags = %v", c.Tags)
	}
	if c.Priority != "critical" {
		t.Errorf("priority = %q", c.Priority)
	}
	if c.Due != "2026-12-01" {
		t.Errorf("due = %q", c.Due)
	}
	if c.Assignee != "alice" {
		t.Errorf("assignee = %q", c.Assignee)
	}
	// Check assignee added to members
	found := false
	for _, m := range b.Members {
		if m == "alice" {
			found = true
		}
	}
	if !found {
		t.Error("alice should be in members")
	}
}

// --- Column handler tests ---

func TestHandleCreateColumn(t *testing.T) {
	h, slug := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/columns", map[string]string{"column_name": "In Progress"})
	r = withSlug(r, slug)
	h.HandleCreateColumn(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	b, err := h.ws.LoadBoard(slug)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, col := range b.Columns {
		if col.Name == "In Progress" {
			found = true
		}
	}
	if !found {
		t.Error("column not created")
	}
}

func TestHandleCreateColumn_EmptyName(t *testing.T) {
	h, slug := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/columns", map[string]string{})
	r = withSlug(r, slug)
	h.HandleCreateColumn(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleRenameColumn(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/columns/rename", map[string]string{
		"old_name": "Todo", "new_name": "Backlog",
	})
	r = withSlug(r, slug)
	h.HandleRenameColumn(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	b, err := h.ws.LoadBoard(slug)
	if err != nil {
		t.Fatal(err)
	}
	if b.Columns[0].Name != "Backlog" {
		t.Errorf("name = %q, want Backlog", b.Columns[0].Name)
	}
}

func TestHandleRenameColumn_MissingNames(t *testing.T) {
	h, slug := setupBoardWithColumn(t)

	// Missing old_name
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/columns/rename", map[string]string{"new_name": "X"})
	r = withSlug(r, slug)
	h.HandleRenameColumn(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}

	// Missing new_name
	w = httptest.NewRecorder()
	r = postForm("/board/"+slug+"/columns/rename", map[string]string{"old_name": "Todo"})
	r = withSlug(r, slug)
	h.HandleRenameColumn(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleDeleteColumn(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/columns/delete", map[string]string{"column_name": "Done"})
	r = withSlug(r, slug)
	h.HandleDeleteColumn(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	b, err := h.ws.LoadBoard(slug)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Columns) != 1 {
		t.Errorf("expected 1 column, got %d", len(b.Columns))
	}
}

func TestHandleDeleteColumn_EmptyName(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/columns/delete", map[string]string{})
	r = withSlug(r, slug)
	h.HandleDeleteColumn(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleToggleColumnCollapse(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/columns/collapse", map[string]string{"col_index": "0"})
	r = withSlug(r, slug)
	h.HandleToggleColumnCollapse(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	b, err := h.ws.LoadBoard(slug)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.ListCollapse) == 0 || !b.ListCollapse[0] {
		t.Error("column 0 should be collapsed")
	}
}

func TestHandleToggleColumnCollapse_EmptySlug(t *testing.T) {
	h, _ := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/x/columns/collapse", map[string]string{"col_index": "0"})
	// no withSlug
	h.HandleToggleColumnCollapse(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleSortColumn(t *testing.T) {
	h, slug := setupBoardWithColumn(t)

	for _, sortBy := range []string{"name", "priority", "due"} {
		w := httptest.NewRecorder()
		r := postForm("/board/"+slug+"/columns/sort", map[string]string{
			"col_idx": "0", "sort_by": sortBy,
		})
		r = withSlug(r, slug)
		h.HandleSortColumn(w, r)
		if w.Code != 200 {
			t.Fatalf("sort_by=%s: status = %d", sortBy, w.Code)
		}
	}
}

func TestHandleSortColumn_MissingSortBy(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/columns/sort", map[string]string{"col_idx": "0"})
	r = withSlug(r, slug)
	h.HandleSortColumn(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleMoveColumn(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/columns/move", map[string]string{
		"column": "Done", "after_column": "",
	})
	r = withSlug(r, slug)
	h.HandleMoveColumn(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	b, err := h.ws.LoadBoard(slug)
	if err != nil {
		t.Fatal(err)
	}
	if b.Columns[0].Name != "Done" {
		t.Errorf("first column = %q, want Done", b.Columns[0].Name)
	}
}

func TestHandleMoveColumn_EmptyColumn(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/columns/move", map[string]string{})
	r = withSlug(r, slug)
	h.HandleMoveColumn(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

// --- Board meta/settings handler tests ---

func TestHandleUpdateBoardMeta(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/meta", map[string]string{
		"board_name": "Renamed", "description": "A board", "tags": "go, web",
	})
	r = withSlug(r, slug)
	h.HandleUpdateBoardMeta(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	b, err := h.ws.LoadBoard(slug)
	if err != nil {
		t.Fatal(err)
	}
	if b.Name != "Renamed" {
		t.Errorf("name = %q", b.Name)
	}
	if b.Description != "A board" {
		t.Errorf("description = %q", b.Description)
	}
	if len(b.Tags) != 2 {
		t.Errorf("tags = %v", b.Tags)
	}
}

func TestHandleUpdateBoardMeta_EmptySlug(t *testing.T) {
	h, _ := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/x/meta", map[string]string{"board_name": "X"})
	// no withSlug
	h.HandleUpdateBoardMeta(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleUpdateBoardSettings(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/settings", map[string]string{
		"show_checkbox": "false", "view_mode": "list",
		"card_position": "prepend", "expand_columns": "true",
		"card_display_mode": "trim", "week_start": "monday",
	})
	r = withSlug(r, slug)
	h.HandleUpdateBoardSettings(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	b, err := h.ws.LoadBoard(slug)
	if err != nil {
		t.Fatal(err)
	}
	if b.Settings.ShowCheckbox == nil || *b.Settings.ShowCheckbox {
		t.Error("show_checkbox should be false")
	}
	if b.Settings.ViewMode == nil || *b.Settings.ViewMode != "list" {
		t.Error("view_mode should be list")
	}
}

func TestHandleUpdateBoardSettings_EmptySlug(t *testing.T) {
	h, _ := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/x/settings", map[string]string{"show_checkbox": "true"})
	h.HandleUpdateBoardSettings(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleSetBoardIcon(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/icon", map[string]string{"icon": "🎯"})
	r = withSlug(r, slug)
	h.HandleSetBoardIcon(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	b, err := h.ws.LoadBoard(slug)
	if err != nil {
		t.Fatal(err)
	}
	if b.Icon != "🎯" {
		t.Errorf("icon = %q", b.Icon)
	}
}

func TestHandleSetBoardIcon_EmptySlug(t *testing.T) {
	h, _ := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/x/icon", map[string]string{"icon": "🎯"})
	h.HandleSetBoardIcon(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

// --- Page handler tests ---

func TestBoardViewPage(t *testing.T) {
	h, slug := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/board/"+slug, nil)
	r = withSlug(r, slug)
	h.BoardViewPage(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty HTML")
	}
}

func TestBoardViewPage_EmptySlug(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/board/", nil)
	h.BoardViewPage(w, r)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestBoardViewPage_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/board/nonexistent", nil)
	r = withSlug(r, "nonexistent")
	h.BoardViewPage(w, r)
	if w.Code != 303 {
		t.Fatalf("status = %d, want 303 redirect", w.Code)
	}
}

func TestBoardViewPage_DesktopLastBoard(t *testing.T) {
	h, slug := setupTestHandler(t)
	h.IsDesktop = true
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/board/"+slug, nil)
	r = withSlug(r, slug)
	h.BoardViewPage(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	s := h.loadSettings()
	if s.LastBoard != slug {
		t.Errorf("last_board = %q, want %q", s.LastBoard, slug)
	}
}

func TestBoardContent(t *testing.T) {
	h, slug := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/board/"+slug+"/content", nil)
	r = withSlug(r, slug)
	h.BoardContent(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty content")
	}
}

func TestBoardContent_EmptySlug(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/board//content", nil)
	h.BoardContent(w, r)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestBoardContent_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/board/nonexistent/content", nil)
	r = withSlug(r, "nonexistent")
	h.BoardContent(w, r)
	if w.Code != 404 {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestExportHandler(t *testing.T) {
	h, _ := setupTestHandler(t)
	handler := h.ExportHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/export", nil)
	handler.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/zip" {
		t.Errorf("content-type = %q, want application/zip", ct)
	}
}

func TestSettingsHandler(t *testing.T) {
	h, _ := setupTestHandler(t)
	handler := h.SettingsHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/settings", nil)
	handler.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty settings page")
	}
}

// --- Helper/utility function tests ---

func TestSplitTags(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"go, web, api", 3},
		{"", 0},
		{",,,", 0},
		{"single", 1},
		{" spaced , tags ", 2},
	}
	for _, tc := range cases {
		got := splitTags(tc.in)
		if len(got) != tc.want {
			t.Errorf("splitTags(%q) = %d tags, want %d", tc.in, len(got), tc.want)
		}
	}
}

func TestEnsureMember(t *testing.T) {
	b := &models.Board{Members: []string{"alice"}}
	ensureMember(b, "alice")
	if len(b.Members) != 1 {
		t.Error("duplicate added")
	}
	ensureMember(b, "bob")
	if len(b.Members) != 2 {
		t.Error("new member not added")
	}
}

func TestValidateIndices(t *testing.T) {
	b := &models.Board{
		Columns: []models.Column{
			{Name: "A", Cards: []models.Card{{Title: "1"}, {Title: "2"}}},
		},
	}
	if err := validateIndices(b, 0, 0); err != nil {
		t.Errorf("valid: %v", err)
	}
	if err := validateIndices(b, 0, 1); err != nil {
		t.Errorf("valid: %v", err)
	}
	if err := validateIndices(b, -1, 0); err == nil {
		t.Error("expected error for negative col")
	}
	if err := validateIndices(b, 1, 0); err == nil {
		t.Error("expected error for out-of-range col")
	}
	if err := validateIndices(b, 0, 5); err == nil {
		t.Error("expected error for out-of-range card")
	}
	if err := validateIndices(b, 0, -1); err == nil {
		t.Error("expected error for negative card")
	}
}

func TestRemoveCardAt(t *testing.T) {
	cards := []models.Card{{Title: "A"}, {Title: "B"}, {Title: "C"}}
	result := removeCardAt(cards, 1)
	if len(result) != 2 {
		t.Fatalf("len = %d", len(result))
	}
	if result[0].Title != "A" || result[1].Title != "C" {
		t.Errorf("got %v", result)
	}
}

func TestSortCardsByName(t *testing.T) {
	cards := []models.Card{{Title: "Banana"}, {Title: "apple"}, {Title: "cherry"}}
	sortCardsByName(cards)
	if cards[0].Title != "apple" {
		t.Errorf("first = %q", cards[0].Title)
	}
}

func TestSortCardsByPriority(t *testing.T) {
	cards := []models.Card{
		{Title: "Low", Priority: "low"},
		{Title: "Crit", Priority: "critical"},
		{Title: "Med", Priority: "medium"},
	}
	sortCardsByPriority(cards)
	if cards[0].Title != "Crit" {
		t.Errorf("first = %q", cards[0].Title)
	}
}

func TestSortCardsByDue(t *testing.T) {
	cards := []models.Card{
		{Title: "No due"},
		{Title: "Later", Due: "2026-12-01"},
		{Title: "Sooner", Due: "2026-01-01"},
	}
	sortCardsByDue(cards)
	if cards[0].Title != "Sooner" {
		t.Errorf("first = %q", cards[0].Title)
	}
	if cards[2].Title != "No due" {
		t.Errorf("last = %q", cards[2].Title)
	}
}

func TestPriorityRank(t *testing.T) {
	cases := map[string]int{
		"critical": 4, "high": 3, "medium": 2, "low": 1, "": 0, "unknown": 0,
	}
	for p, want := range cases {
		if got := priorityRank(p); got != want {
			t.Errorf("priorityRank(%q) = %d, want %d", p, got, want)
		}
	}
}

func TestFlattenCards(t *testing.T) {
	b := &models.Board{
		Columns: []models.Column{
			{Name: "A", Cards: []models.Card{{Title: "1"}, {Title: "2"}}},
			{Name: "B", Cards: []models.Card{{Title: "3"}}},
		},
	}
	all := flattenCards(b)
	if len(all) != 3 {
		t.Fatalf("len = %d", len(all))
	}
	if all[0].ColIdx != 0 || all[0].CardIdx != 0 || all[0].ColumnName != "A" {
		t.Errorf("first = %+v", all[0])
	}
	if all[2].ColIdx != 1 || all[2].CardIdx != 0 || all[2].ColumnName != "B" {
		t.Errorf("last = %+v", all[2])
	}
}

func TestReorderColumns(t *testing.T) {
	b := &models.Board{
		Columns: []models.Column{
			{Name: "A"}, {Name: "B"}, {Name: "C"},
		},
	}

	// Move C to front
	if err := reorderColumns(b, "C", ""); err != nil {
		t.Fatal(err)
	}
	if b.Columns[0].Name != "C" {
		t.Errorf("first = %q", b.Columns[0].Name)
	}

	// Move A after C
	if err := reorderColumns(b, "A", "C"); err != nil {
		t.Fatal(err)
	}
	if b.Columns[0].Name != "C" || b.Columns[1].Name != "A" {
		t.Errorf("order = %v %v %v", b.Columns[0].Name, b.Columns[1].Name, b.Columns[2].Name)
	}

	// Nonexistent column
	if err := reorderColumns(b, "Z", ""); err == nil {
		t.Error("expected error for nonexistent column")
	}
}

func TestRelativeTime(t *testing.T) {
	if relativeTime(time.Time{}) != "" {
		t.Error("zero time should return empty")
	}
	if got := relativeTime(time.Now().Add(-30 * time.Second)); got != "just now" {
		t.Errorf("30s ago = %q", got)
	}
	if got := relativeTime(time.Now().Add(-1 * time.Minute)); got != "1m ago" {
		t.Errorf("1m ago = %q", got)
	}
	if got := relativeTime(time.Now().Add(-5 * time.Minute)); got != "5m ago" {
		t.Errorf("5m ago = %q", got)
	}
	if got := relativeTime(time.Now().Add(-1 * time.Hour)); got != "1h ago" {
		t.Errorf("1h ago = %q", got)
	}
	if got := relativeTime(time.Now().Add(-3 * time.Hour)); got != "3h ago" {
		t.Errorf("3h ago = %q", got)
	}
	if got := relativeTime(time.Now().Add(-24 * time.Hour)); got != "1d ago" {
		t.Errorf("1d ago = %q", got)
	}
	if got := relativeTime(time.Now().Add(-5 * 24 * time.Hour)); got != "5d ago" {
		t.Errorf("5d ago = %q", got)
	}
	if got := relativeTime(time.Now().Add(-60 * 24 * time.Hour)); !strings.Contains(got, "200") {
		// Should be formatted as a date
		if len(got) < 5 {
			t.Errorf("60d ago = %q, expected date format", got)
		}
	}
}

func TestCollectAllTags(t *testing.T) {
	boards := []BoardSummary{
		{Tags: []string{"go", "web"}},
		{Tags: []string{"web", "api"}},
		{Tags: nil},
	}
	tags := collectAllTags(boards)
	if len(tags) != 3 {
		t.Fatalf("len = %d, want 3", len(tags))
	}
	// Should be sorted
	if tags[0] != "api" || tags[1] != "go" || tags[2] != "web" {
		t.Errorf("tags = %v", tags)
	}
}

func TestSortBoardsWithPins(t *testing.T) {
	boards := []BoardSummary{
		{Name: "C", Slug: "c"},
		{Name: "A", Slug: "a"},
		{Name: "B", Slug: "b"},
	}
	result := sortBoardsWithPins(boards, []string{"b"})
	if result[0].Slug != "b" {
		t.Errorf("first = %q, want pinned 'b'", result[0].Slug)
	}
	if !result[0].Pinned {
		t.Error("b should be pinned")
	}
	if result[1].Slug != "a" {
		t.Errorf("second = %q, want 'a'", result[1].Slug)
	}
}

func TestToBoardSummariesFast(t *testing.T) {
	infos := []parser.BoardSummaryInfo{
		{
			Board:       models.Board{Name: "Test", FilePath: "/test.md", Icon: "🚀"},
			CardCount:   5,
			DoneCount:   2,
			ColumnCount: 3,
		},
	}
	summaries := toBoardSummariesFast(infos)
	if len(summaries) != 1 {
		t.Fatalf("len = %d", len(summaries))
	}
	if summaries[0].Name != "Test" {
		t.Errorf("name = %q", summaries[0].Name)
	}
	if summaries[0].CardCount != 5 {
		t.Errorf("card_count = %d", summaries[0].CardCount)
	}
	if summaries[0].DoneCount != 2 {
		t.Errorf("done_count = %d", summaries[0].DoneCount)
	}
	if summaries[0].ColumnCount != 3 {
		t.Errorf("column_count = %d", summaries[0].ColumnCount)
	}
	if summaries[0].Icon != "🚀" {
		t.Errorf("icon = %q", summaries[0].Icon)
	}
}

// --- Additional edge-case handler tests ---

func TestHandleMoveCard_MissingFields(t *testing.T) {
	h, slug := setupBoardWithColumn(t)

	// Missing col_idx
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/cards/move", map[string]string{
		"card_idx": "0", "target_column": "Done",
	})
	r = withSlug(r, slug)
	h.HandleMoveCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}

	// Missing card_idx
	w = httptest.NewRecorder()
	r = postForm("/board/"+slug+"/cards/move", map[string]string{
		"col_idx": "0", "target_column": "Done",
	})
	r = withSlug(r, slug)
	h.HandleMoveCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}

	// Missing target_column
	w = httptest.NewRecorder()
	r = postForm("/board/"+slug+"/cards/move", map[string]string{
		"col_idx": "0", "card_idx": "0",
	})
	r = withSlug(r, slug)
	h.HandleMoveCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}

	// Missing slug
	w = httptest.NewRecorder()
	r = postForm("/board/x/cards/move", map[string]string{
		"col_idx": "0", "card_idx": "0", "target_column": "Done",
	})
	h.HandleMoveCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleReorderCard_MissingFields(t *testing.T) {
	h, slug := setupBoardWithColumn(t)

	// Missing col_idx
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/cards/reorder", map[string]string{
		"card_idx": "0", "column": "Todo",
	})
	r = withSlug(r, slug)
	h.HandleReorderCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}

	// Missing card_idx
	w = httptest.NewRecorder()
	r = postForm("/board/"+slug+"/cards/reorder", map[string]string{
		"col_idx": "0", "column": "Todo",
	})
	r = withSlug(r, slug)
	h.HandleReorderCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}

	// Missing column
	w = httptest.NewRecorder()
	r = postForm("/board/"+slug+"/cards/reorder", map[string]string{
		"col_idx": "0", "card_idx": "0",
	})
	r = withSlug(r, slug)
	h.HandleReorderCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}

	// Missing slug
	w = httptest.NewRecorder()
	r = postForm("/board/x/cards/reorder", map[string]string{
		"col_idx": "0", "card_idx": "0", "column": "Todo",
	})
	h.HandleReorderCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleDeleteCard_MissingFields(t *testing.T) {
	h, _ := setupBoardWithColumn(t)

	// Missing slug
	w := httptest.NewRecorder()
	r := postForm("/board/x/cards/delete", map[string]string{
		"col_idx": "0", "card_idx": "0",
	})
	h.HandleDeleteCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleToggleComplete_MissingFields(t *testing.T) {
	h, slug := setupBoardWithColumn(t)

	// Missing slug
	w := httptest.NewRecorder()
	r := postForm("/board/x/cards/complete", map[string]string{
		"col_idx": "0", "card_idx": "0",
	})
	h.HandleToggleComplete(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}

	// Missing col_idx
	w = httptest.NewRecorder()
	r = postForm("/board/"+slug+"/cards/complete", map[string]string{"card_idx": "0"})
	r = withSlug(r, slug)
	h.HandleToggleComplete(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleEditCard_MissingFields(t *testing.T) {
	h, _ := setupBoardWithColumn(t)

	// Missing slug
	w := httptest.NewRecorder()
	r := postForm("/board/x/cards/edit", map[string]string{
		"col_idx": "0", "card_idx": "0", "title": "X",
	})
	h.HandleEditCard(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleCreateColumn_MissingSlug(t *testing.T) {
	h, _ := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/x/columns", map[string]string{"column_name": "New"})
	h.HandleCreateColumn(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleRenameColumn_MissingSlug(t *testing.T) {
	h, _ := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/x/columns/rename", map[string]string{
		"old_name": "Todo", "new_name": "New",
	})
	h.HandleRenameColumn(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleDeleteColumn_MissingSlug(t *testing.T) {
	h, _ := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/x/columns/delete", map[string]string{"column_name": "Todo"})
	h.HandleDeleteColumn(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleSortColumn_MissingSlug(t *testing.T) {
	h, _ := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/x/columns/sort", map[string]string{
		"col_idx": "0", "sort_by": "name",
	})
	h.HandleSortColumn(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleMoveColumn_MissingSlug(t *testing.T) {
	h, _ := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/x/columns/move", map[string]string{"column": "Todo"})
	h.HandleMoveColumn(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestOneOf(t *testing.T) {
	if oneOf("dark", "system", "dark", "light") != "dark" {
		t.Error("valid value not returned")
	}
	if oneOf("invalid", "system", "dark", "light") != "system" {
		t.Error("default not returned for invalid")
	}
}

func TestLayoutSettings(t *testing.T) {
	h, _ := setupTestHandler(t)
	s := defaultSettings()
	ls := h.layoutSettings(s)
	if ls.Theme != "system" {
		t.Errorf("theme = %q", ls.Theme)
	}
	if ls.Version != "test" {
		t.Errorf("version = %q", ls.Version)
	}
}

func TestHandleUpdateBoardMeta_TagColors(t *testing.T) {
	h, slug := setupBoardWithColumn(t)
	w := httptest.NewRecorder()
	r := postForm("/board/"+slug+"/meta", map[string]string{
		"board_name": "Test", "tag_colors": `{"go":"#00ff00"}`,
	})
	r = withSlug(r, slug)
	h.HandleUpdateBoardMeta(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	b, err := h.ws.LoadBoard(slug)
	if err != nil {
		t.Fatal(err)
	}
	if b.TagColors == nil || b.TagColors["go"] != "#00ff00" {
		t.Errorf("tag_colors = %v", b.TagColors)
	}
}

func TestHandleConflict(t *testing.T) {
	h, slug := setupTestHandler(t)
	w := httptest.NewRecorder()
	h.handleConflict(w, slug)
	if w.Code != 409 {
		t.Fatalf("status = %d, want 409", w.Code)
	}
}

func TestRenderFullPageAndPartial(t *testing.T) {
	h, _ := setupTestHandler(t)
	model, _ := h.boardListModel()

	// renderFullPage
	w := httptest.NewRecorder()
	renderFullPage(w, h.BoardList.boardListTpl, model)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("content-type = %q", ct)
	}
}

func TestLoadSettingsFromDir(t *testing.T) {
	// Nonexistent dir returns defaults
	s := LoadSettingsFromDir("/nonexistent/path/xyz")
	if s.Theme != "system" {
		t.Errorf("theme = %q", s.Theme)
	}

	// Valid dir with settings
	dir := t.TempDir()
	data := []byte(`{"theme":"dark","site_name":"Test"}`)
	_ = os.WriteFile(filepath.Join(dir, "settings.json"), data, 0644)
	s = LoadSettingsFromDir(dir)
	if s.Theme != "dark" {
		t.Errorf("theme = %q", s.Theme)
	}
	if s.SiteName != "Test" {
		t.Errorf("site_name = %q", s.SiteName)
	}
}

func TestHandleSetBoardIconList_EmptyName(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := postForm("/boards/icon", map[string]string{"icon": "🚀"})
	h.HandleSetBoardIconList(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleSetBoardIconList_NonexistentBoard(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := postForm("/boards/icon", map[string]string{"name": "nonexistent", "icon": "🚀"})
	h.HandleSetBoardIconList(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandleSidebarBoards_NoSlug(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/boards/sidebar", nil)
	h.HandleSidebarBoards(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestResolveSettings_ViewModeTable(t *testing.T) {
	global := defaultSettings()
	vm := "table"
	bs := models.BoardSettings{ViewMode: &vm}
	rs := resolveSettings(global, bs)
	if rs.ViewMode != "list" {
		t.Errorf("table should map to list, got %q", rs.ViewMode)
	}
}

func TestResolveSettings_WeekStart(t *testing.T) {
	global := defaultSettings()
	ws := "monday"
	bs := models.BoardSettings{WeekStart: &ws}
	rs := resolveSettings(global, bs)
	if rs.WeekStart != "monday" {
		t.Errorf("week_start = %q", rs.WeekStart)
	}
}

func TestResolveSettings_CardDisplayMode(t *testing.T) {
	global := defaultSettings()
	cdm := "trim"
	bs := models.BoardSettings{CardDisplayMode: &cdm}
	rs := resolveSettings(global, bs)
	if rs.CardDisplayMode != "trim" {
		t.Errorf("card_display_mode = %q", rs.CardDisplayMode)
	}
}

func TestToBoardSettingsView_AllFields(t *testing.T) {
	vm := "list"
	cdm := "hide"
	ws := "monday"
	v := toBoardSettingsView(models.BoardSettings{
		ViewMode:        &vm,
		CardDisplayMode: &cdm,
		WeekStart:       &ws,
	})
	if v.ViewMode != "list" {
		t.Errorf("view_mode = %q", v.ViewMode)
	}
	if v.CardDisplayMode != "hide" {
		t.Errorf("card_display_mode = %q", v.CardDisplayMode)
	}
	if v.WeekStart != "monday" {
		t.Errorf("week_start = %q", v.WeekStart)
	}
}

func TestHandleCreateBoard_Duplicate(t *testing.T) {
	h, _ := setupTestHandler(t)
	w := httptest.NewRecorder()
	r := postForm("/boards/new", map[string]string{"name": "test-board"})
	h.HandleCreateBoard(w, r)
	// Should get an error partial since test-board already exists
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestFuncMap(t *testing.T) {
	fm := funcMap()
	mdFunc, ok := fm["md"]
	if !ok {
		t.Fatal("md function not found")
	}
	fn, ok2 := mdFunc.(func(string) template.HTML)
	if !ok2 {
		t.Fatal("md function has wrong type")
	}
	result := fn("**bold**")
	if !strings.Contains(string(result), "<strong>") {
		t.Errorf("md output = %q", result)
	}
	// Test with link
	result = fn("https://example.com")
	if !strings.Contains(string(result), "target=\"_blank\"") {
		t.Errorf("link output = %q", result)
	}
}

func TestSortCardsByDue_BothEmpty(t *testing.T) {
	cards := []models.Card{
		{Title: "A"},
		{Title: "B"},
	}
	sortCardsByDue(cards)
	// Order should remain stable
	if cards[0].Title != "A" {
		t.Errorf("first = %q", cards[0].Title)
	}
}

func TestReorderColumns_WithCollapse(t *testing.T) {
	b := &models.Board{
		Columns:      []models.Column{{Name: "A"}, {Name: "B"}, {Name: "C"}},
		ListCollapse: []bool{true, false, true},
	}
	if err := reorderColumns(b, "C", "A"); err != nil {
		t.Fatal(err)
	}
	// C should be after A: A, C, B
	if b.Columns[0].Name != "A" || b.Columns[1].Name != "C" || b.Columns[2].Name != "B" {
		t.Errorf("order = %s %s %s", b.Columns[0].Name, b.Columns[1].Name, b.Columns[2].Name)
	}
	// Collapse should follow: A=true, C=true, B=false
	if !b.ListCollapse[0] || !b.ListCollapse[1] || b.ListCollapse[2] {
		t.Errorf("collapse = %v", b.ListCollapse)
	}
}
