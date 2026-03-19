package web

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
)

// AppSettings holds persisted user preferences.
type AppSettings struct {
	Theme          string   `json:"theme"`
	ColumnWidth    int      `json:"column_width"`
	DefaultColumns []string `json:"default_columns,omitempty"`
}

func defaultSettings() AppSettings {
	return AppSettings{
		Theme:          "system",
		ColumnWidth:    280,
		DefaultColumns: []string{"not now", "maybe?", "done"},
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
	Boards    []BoardSummary
	BoardSlug string
}

// SettingsHandler returns an http.Handler for the settings page.
func (h *Handler) SettingsHandler() http.Handler {
	var tpl *template.Template
	if h.tmplDir != "" {
		tpl = template.Must(template.ParseFiles(
			filepath.Join(h.tmplDir, "layout.html"),
			filepath.Join(h.tmplDir, "settings.html"),
		))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		boards, _ := h.ws.ListBoards()
		model := SettingsModel{
			Title:     "Settings — LiveBoard",
			Boards:    toBoardSummaries(boards),
			BoardSlug: "__settings__",
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if tpl == nil {
			http.Error(w, "template not found", http.StatusInternalServerError)
			return
		}
		if err := tpl.Execute(w, model); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// SettingsAPIHandler returns an http.Handler for GET/POST /api/settings.
func (h *Handler) SettingsAPIHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			s := h.loadSettings()
			json.NewEncoder(w).Encode(s)
		case http.MethodPost:
			var s AppSettings
			if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
				http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
				return
			}
			if s.ColumnWidth < 180 || s.ColumnWidth > 600 {
				s.ColumnWidth = 280
			}
			if s.Theme != "dark" && s.Theme != "light" {
				s.Theme = "system"
			}
			if len(s.DefaultColumns) == 0 {
				s.DefaultColumns = defaultSettings().DefaultColumns
			}
			if err := h.saveSettings(s); err != nil {
				http.Error(w, `{"error":"save failed"}`, http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(s)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})
}
