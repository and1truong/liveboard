// Package web holds settings persistence and resolution used by the REST API
// and CLI tooling. The HTMX-era handlers, templates, and SSE broker historically
// lived in this package; after the HTMX removal, only settings and the SSE
// broker remain.
package web

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/and1truong/liveboard/pkg/models"
)

// AppSettings holds persisted user preferences (workspace-global).
type AppSettings struct {
	SiteName          string            `json:"site_name"`
	Theme             string            `json:"theme"`
	ColorTheme        string            `json:"color_theme"`
	FontFamily        string            `json:"font_family"`
	ColumnWidth       int               `json:"column_width"`
	SidebarPosition   string            `json:"sidebar_position"`
	DefaultColumns    []string          `json:"default_columns,omitempty"`
	ShowCheckbox      bool              `json:"show_checkbox"`
	NewLineTrigger    string            `json:"newline_trigger"`
	CardPosition      string            `json:"card_position"`
	CardDisplayMode   string            `json:"card_display_mode"`
	PinnedBoards      []string          `json:"pinned_boards,omitempty"`
	KeyboardShortcuts bool              `json:"keyboard_shortcuts"`
	WeekStart         string            `json:"week_start,omitempty"`
	LastBoard         string            `json:"last_board,omitempty"`
	ReminderEnabled   bool              `json:"reminder_enabled,omitempty"`
	ReminderTimezone  string            `json:"reminder_timezone,omitempty"`
	Tags              []string          `json:"tags,omitempty"`
	TagColors         map[string]string `json:"tag_colors,omitempty"`
	FolderCollapse    map[string]bool   `json:"folder_collapse,omitempty"`
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

// settingsMu serializes read-modify-write cycles on settings.json so concurrent
// mutations do not lose updates. Callers performing load+mutate+save must go
// through MutateSettings.
var settingsMu sync.Mutex

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

// MutateSettings atomically loads settings, applies fn, and saves the result.
// Concurrent callers are serialized so read-modify-write cannot lose updates.
func MutateSettings(dir string, fn func(*AppSettings)) error {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	s := LoadSettingsFromDir(dir)
	fn(&s)
	return SaveSettingsToDir(dir, s)
}

// RewritePinsOnRename updates pinned-board entries in settings.json when a
// board id changes (rename or move across folders).
func RewritePinsOnRename(dir, oldID, newID string) error {
	if oldID == newID {
		return nil
	}
	return MutateSettings(dir, func(s *AppSettings) {
		for i, p := range s.PinnedBoards {
			if p == oldID {
				s.PinnedBoards[i] = newID
			}
		}
	})
}

// RewritePinsOnFolderRename updates pinned-board entries when a folder is
// renamed: every pin starting with "oldFolder/" gets rewritten to "newFolder/".
func RewritePinsOnFolderRename(dir, oldFolder, newFolder string) error {
	if oldFolder == newFolder {
		return nil
	}
	prefix := oldFolder + "/"
	return MutateSettings(dir, func(s *AppSettings) {
		for i, p := range s.PinnedBoards {
			if strings.HasPrefix(p, prefix) {
				s.PinnedBoards[i] = newFolder + "/" + p[len(prefix):]
			}
		}
	})
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
	s.Tags = sanitizeTagList(s.Tags)
	s.TagColors = sanitizeTagColors(s.TagColors)
}

func sanitizeTagList(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func sanitizeTagColors(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || !isHexColor(v) {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isHexColor(s string) bool {
	if len(s) != 4 && len(s) != 7 {
		return false
	}
	if s[0] != '#' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
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

// ResolveSettings merges global defaults with per-board overrides.
func ResolveSettings(global AppSettings, bs models.BoardSettings) ResolvedSettings {
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
