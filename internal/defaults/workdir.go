package defaults

import (
	"os"
	"path/filepath"
)

// WorkDir returns the default workspace directory (current working directory).
// The second return value is always false (retained for signature compatibility).
func WorkDir() (string, bool) {
	dir, _ := os.Getwd()
	return dir, false
}

// DesktopWorkDir checks saved desktop config first,
// then falls back to ~/LiveBoard instead of cwd when launched from Finder.
func DesktopWorkDir() (string, bool) {
	cfg := LoadDesktopConfig()
	if cfg.LastWorkspace != "" {
		if info, err := os.Stat(cfg.LastWorkspace); err == nil && info.IsDir() {
			return cfg.LastWorkspace, false
		}
	}

	dir, _ := WorkDir()
	// When launched from Finder, cwd is "/" — use ~/LiveBoard instead
	if dir == "/" {
		home, err := os.UserHomeDir()
		if err == nil {
			fallback := filepath.Join(home, "LiveBoard")
			if err := os.MkdirAll(fallback, 0o755); err != nil {
				return "", false
			}
			return fallback, false
		}
	}
	return dir, false
}
