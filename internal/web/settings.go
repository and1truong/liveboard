package web

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	tmplfs "github.com/and1truong/liveboard/internal/templates"
)

// AppSettings holds persisted user preferences.
type AppSettings struct {
	SiteName        string   `json:"site_name"`
	Theme           string   `json:"theme"`
	ColorTheme      string   `json:"color_theme"`
	FontFamily      string   `json:"font_family"`
	ColumnWidth     int      `json:"column_width"`
	SidebarPosition string   `json:"sidebar_position"`
	DefaultColumns  []string `json:"default_columns,omitempty"`
	ShowCheckbox    bool     `json:"show_checkbox"`
	NewLineTrigger  string   `json:"newline_trigger"`
	CardPosition    string   `json:"card_position"`
	CardDisplayMode string   `json:"card_display_mode"`
	PinnedBoards    []string `json:"pinned_boards,omitempty"`
}

var validColorThemes = map[string]bool{
	"default": true, "github": true, "gitlab": true,
	"emerald": true, "rose": true, "sunset": true,
	"aqua": true, "graphite": true, "macos": true,
}

var validFonts = map[string]bool{
	"system": true, "inter": true, "ibm-plex-sans": true,
	"source-sans-3": true, "nunito-sans": true, "dm-sans": true,
	"rubik": true,
}

func defaultSettings() AppSettings {
	return AppSettings{
		SiteName:        "LiveBoard",
		Theme:           "system",
		ColorTheme:      "aqua",
		FontFamily:      "system",
		ColumnWidth:     280,
		SidebarPosition: "left",
		DefaultColumns:  []string{"not now", "maybe?", "done"},
		ShowCheckbox:    true,
		NewLineTrigger:  "shift-enter",
		CardPosition:    "append",
		CardDisplayMode: "full",
	}
}

// settingsPath returns the path to settings.json in the workspace dir.
func (h *Handler) settingsPath() string {
	return filepath.Join(h.ws.Dir, "settings.json")
}

// loadSettings reads settings.json, returning defaults if missing.
func (h *Handler) loadSettings() AppSettings {
	s := defaultSettings()
	data, err := os.ReadFile(h.settingsPath())
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	return s
}

// saveSettings writes settings.json.
func (h *Handler) saveSettings(s AppSettings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(h.settingsPath(), data, 0644)
}

// SettingsModel is the template data for the settings page.
type SettingsModel struct {
	Title     string
	SiteName  string
	Boards    []BoardSummary
	AllTags   []string
	BoardSlug string
	Version   string
}

// SettingsHandler returns an http.Handler for the settings page.
func (h *Handler) SettingsHandler() http.Handler {
	tpl := template.Must(template.New("layout.html").Funcs(funcMap()).ParseFS(tmplfs.FS, "layout.html", "settings.html"))

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		boards, _ := h.ws.ListBoards()
		settings := h.loadSettings()
		summaries := sortBoardsWithPins(toBoardSummaries(boards), settings.PinnedBoards)
		model := SettingsModel{
			Title:     "Settings — " + settings.SiteName,
			SiteName:  settings.SiteName,
			Boards:    summaries,
			AllTags:   collectAllTags(summaries),
			BoardSlug: "__settings__",
			Version:   h.version,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tpl.Execute(w, model); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// oneOf returns val if it matches one of the allowed values, otherwise def.
func oneOf(val, def string, allowed ...string) string {
	for _, a := range allowed {
		if val == a {
			return val
		}
	}
	return def
}

// sanitizeSettings clamps and normalizes settings values to valid ranges.
func sanitizeSettings(s *AppSettings) {
	if s.ColumnWidth < 180 || s.ColumnWidth > 600 {
		s.ColumnWidth = 280
	}
	s.Theme = oneOf(s.Theme, "system", "dark", "light")
	if !validColorThemes[s.ColorTheme] {
		s.ColorTheme = "default"
	}
	if !validFonts[s.FontFamily] {
		s.FontFamily = "system"
	}
	s.SidebarPosition = oneOf(s.SidebarPosition, "left", "left", "right")
	if len(s.DefaultColumns) == 0 {
		s.DefaultColumns = defaultSettings().DefaultColumns
	}
	s.NewLineTrigger = oneOf(s.NewLineTrigger, "shift-enter", "enter", "shift-enter")
	s.CardPosition = oneOf(s.CardPosition, "append", "prepend", "append")
	s.CardDisplayMode = oneOf(s.CardDisplayMode, "full", "full", "hide", "trim")
	s.SiteName = strings.TrimSpace(s.SiteName)
	if s.SiteName == "" {
		s.SiteName = "LiveBoard"
	}
	if len([]rune(s.SiteName)) > 50 {
		s.SiteName = string([]rune(s.SiteName)[:50])
	}
}

// SettingsAPIHandler returns an http.Handler for GET/POST /api/settings.
func (h *Handler) SettingsAPIHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			s := h.loadSettings()
			_ = json.NewEncoder(w).Encode(s)
		case http.MethodPost:
			var s AppSettings
			if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
				http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
				return
			}
			sanitizeSettings(&s)
			if err := h.saveSettings(s); err != nil {
				http.Error(w, `{"error":"save failed"}`, http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(s)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})
}
