package workspace

import (
	"os"
	"syscall"
	"time"
)

// fileBirthTime returns the file's creation (birth) time on macOS.
func fileBirthTime(fi os.FileInfo) time.Time {
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		return time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec)
	}
	return fi.ModTime()
}
