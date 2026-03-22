package defaults

import (
	"os"
	"path/filepath"
)

// WorkDir returns the default workspace directory, preferring the iCloud
// liveboard folder if it exists, otherwise falling back to cwd.
// The second return value is true when the iCloud path was selected.
func WorkDir() (string, bool) {
	home, err := os.UserHomeDir()
	if err == nil {
		icloud := filepath.Join(home, "Library", "Mobile Documents", "com~apple~CloudDocs", "liveboard")
		if info, err := os.Stat(icloud); err == nil && info.IsDir() {
			return icloud, true
		}
	}
	dir, _ := os.Getwd()
	return dir, false
}

// DesktopWorkDir is like WorkDir but falls back to ~/LiveBoard instead of cwd,
// which is appropriate when launched from Finder where cwd is "/".
func DesktopWorkDir() (string, bool) {
	dir, cloud := WorkDir()
	if cloud {
		return dir, true
	}
	// When launched from Finder, cwd is "/" — use ~/LiveBoard instead
	if dir == "/" {
		home, err := os.UserHomeDir()
		if err == nil {
			fallback := filepath.Join(home, "LiveBoard")
			os.MkdirAll(fallback, 0o755)
			return fallback, false
		}
	}
	return dir, false
}
