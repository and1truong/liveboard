package attachments

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GC removes blobs from the pool that are not referenced by any board.
// Returns the sorted list of removed hashes. Missing pool dir → no-op.
//
// Manual-only by design: callers (CLI command, future admin endpoints)
// drive cadence. There is no time-based grace window because there is no
// background sweep racing with uploads.
func GC(workspaceDir string) ([]string, error) {
	refs, err := CollectReferenced(workspaceDir)
	if err != nil {
		return nil, err
	}
	poolDir := filepath.Join(workspaceDir, PoolDir)
	entries, err := os.ReadDir(poolDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var deleted []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Ignore in-flight uploads and thumbnails (handled separately).
		if strings.HasPrefix(name, ".upload-") || strings.HasSuffix(name, ".thumb.jpg") {
			continue
		}
		if _, ok := refs[name]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(poolDir, name)); err != nil {
			return deleted, err
		}
		deleted = append(deleted, name)
	}
	sort.Strings(deleted)
	return deleted, nil
}
