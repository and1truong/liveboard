package defaults

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const maxRecentWorkspaces = 10

// DesktopConfig stores persistent state for the desktop app.
type DesktopConfig struct {
	LastWorkspace    string   `json:"last_workspace"`
	RecentWorkspaces []string `json:"recent_workspaces"`
}

func desktopConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "liveboard", "desktop.json")
}

// LoadDesktopConfig reads the desktop config from ~/.config/liveboard/desktop.json.
func LoadDesktopConfig() *DesktopConfig {
	path := desktopConfigPath()
	if path == "" {
		return &DesktopConfig{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return &DesktopConfig{}
	}
	var cfg DesktopConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &DesktopConfig{}
	}
	return &cfg
}

// Save writes the config to disk.
func (c *DesktopConfig) Save() error {
	path := desktopConfigPath()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// AddRecent adds a workspace path to the front of the recent list,
// removing duplicates and capping at maxRecentWorkspaces.
// It also sets LastWorkspace.
func (c *DesktopConfig) AddRecent(dir string) {
	c.LastWorkspace = dir
	filtered := make([]string, 0, maxRecentWorkspaces)
	filtered = append(filtered, dir)
	for _, d := range c.RecentWorkspaces {
		if d != dir {
			filtered = append(filtered, d)
		}
		if len(filtered) >= maxRecentWorkspaces {
			break
		}
	}
	c.RecentWorkspaces = filtered
}

// CleanStale removes entries from RecentWorkspaces where the directory
// no longer exists on disk.
func (c *DesktopConfig) CleanStale() {
	filtered := make([]string, 0, len(c.RecentWorkspaces))
	for _, d := range c.RecentWorkspaces {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			filtered = append(filtered, d)
		}
	}
	c.RecentWorkspaces = filtered
	if c.LastWorkspace != "" {
		if info, err := os.Stat(c.LastWorkspace); err != nil || !info.IsDir() {
			c.LastWorkspace = ""
		}
	}
}
