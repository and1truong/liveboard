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

// LayoutSettings holds visual settings rendered into layout.html for FOUC prevention.
// Embedded in all page models so the layout template can access them uniformly.
type LayoutSettings struct {
	Theme             string `json:"layout_theme"`
	ColorTheme        string `json:"layout_color_theme"`
	ColumnWidth       int    `json:"layout_column_width"`
	SidebarPosition   string `json:"layout_sidebar_position"`
	FontFamily        string `json:"layout_font_family"`
	KeyboardShortcuts bool   `json:"layout_keyboard_shortcuts"`
	Version           string `json:"layout_version"`
	ReadOnly          bool   `json:"layout_read_only"`
}

// layoutSettings extracts layout-relevant fields from AppSettings and injects the build version.
func (b *Base) layoutSettings(s AppSettings) LayoutSettings {
	return LayoutSettings{
		Theme:             s.Theme,
		ColorTheme:        s.ColorTheme,
		ColumnWidth:       s.ColumnWidth,
		SidebarPosition:   s.SidebarPosition,
		FontFamily:        s.FontFamily,
		KeyboardShortcuts: s.KeyboardShortcuts,
		Version:           b.version,
		ReadOnly:          b.ReadOnly,
	}
}

// AppSettings holds persisted user preferences.
type AppSettings struct {
	SiteName          string   `json:"site_name"`
	Theme             string   `json:"theme"`
	ColorTheme        string   `json:"color_theme"`
	FontFamily        string   `json:"font_family"`
	ColumnWidth       int      `json:"column_width"`
	SidebarPosition   string   `json:"sidebar_position"`
	DefaultColumns    []string `json:"default_columns,omitempty"`
	ShowCheckbox      bool     `json:"show_checkbox"`
	NewLineTrigger    string   `json:"newline_trigger"`
	CardPosition      string   `json:"card_position"`
	CardDisplayMode   string   `json:"card_display_mode"`
	PinnedBoards      []string `json:"pinned_boards,omitempty"`
	KeyboardShortcuts bool     `json:"keyboard_shortcuts"`
	WeekStart         string   `json:"week_start,omitempty"`
	LastBoard         string   `json:"last_board,omitempty"`
	ReminderEnabled   bool     `json:"reminder_enabled,omitempty"`
	ReminderTimezone  string   `json:"reminder_timezone,omitempty"`
}

var validColorThemes = map[string]bool{
	"emerald": true, "rose": true, "aqua": true,
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
		WeekStart:       "sunday",
	}
}

// LoadSettingsFromDir reads settings.json from dir, returning defaults if missing.
func LoadSettingsFromDir(dir string) AppSettings {
	s := defaultSettings()
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	return s
}

// SaveSettingsToDir writes settings.json to dir.
func SaveSettingsToDir(dir string, s AppSettings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "settings.json"), data, 0644)
}

// loadSettings reads settings.json, returning defaults if missing.
func (b *Base) loadSettings() AppSettings {
	return LoadSettingsFromDir(b.ws.Dir)
}

// saveSettings writes settings.json.
func (b *Base) saveSettings(s AppSettings) error {
	return SaveSettingsToDir(b.ws.Dir, s)
}

// SettingsHandler handles settings page and API.
type SettingsHandler struct {
	*Base
}

// SettingsModel is the template data for the settings page.
type SettingsModel struct {
	LayoutSettings
	Title     string
	SiteName  string
	Boards    []BoardSummary
	AllTags   []string
	BoardSlug string
	Version   string
}

// SettingsHandler returns an http.Handler for the settings page.
func (sh *SettingsHandler) SettingsHandler() http.Handler {
	tpl := template.Must(template.New("layout.html").Funcs(funcMap()).ParseFS(tmplfs.FS, "layout.html", "settings.html"))

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		infos, _ := sh.ws.ListBoardSummaries()
		settings := sh.loadSettings()
		summaries := sortBoardsWithPins(toBoardSummariesFast(infos), settings.PinnedBoards)
		model := SettingsModel{
			LayoutSettings: sh.layoutSettings(settings),
			Title:          "Settings — " + settings.SiteName,
			SiteName:       settings.SiteName,
			Boards:         summaries,
			AllTags:        collectAllTags(summaries),
			BoardSlug:      "__settings__",
			Version:        sh.version,
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

// SanitizeSettings clamps and normalizes settings values to valid ranges.
func SanitizeSettings(s *AppSettings) {
	if s.ColumnWidth < 180 || s.ColumnWidth > 600 {
		s.ColumnWidth = 280
	}
	s.Theme = oneOf(s.Theme, "system", "dark", "light")
	if !validColorThemes[s.ColorTheme] {
		s.ColorTheme = "aqua"
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
func (sh *SettingsHandler) SettingsAPIHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			s := sh.loadSettings()
			_ = json.NewEncoder(w).Encode(s)
		case http.MethodPost:
			var s AppSettings
			if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
				http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
				return
			}
			SanitizeSettings(&s)
			if err := sh.saveSettings(s); err != nil {
				http.Error(w, `{"error":"save failed"}`, http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(s)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})
}
