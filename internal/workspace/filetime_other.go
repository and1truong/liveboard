//go:build !darwin

package workspace

import (
	"os"
	"time"
)

// fileBirthTime falls back to modification time on non-macOS platforms.
func fileBirthTime(fi os.FileInfo) time.Time {
	return fi.ModTime()
}
