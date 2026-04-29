package attachments

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// bodyAttachmentRe matches attachment:<hash>[.ext] inside body markdown.
// Matches the inert-URL scheme used by the renderer's attachmentScheme rewrite.
var bodyAttachmentRe = regexp.MustCompile(`attachment:([a-f0-9]{64}(?:\.[a-z0-9]{1,16})?)`)

// metaAttachmentsLine matches "  attachments: <json>".
var metaAttachmentsLine = regexp.MustCompile(`^\s{2}attachments:\s*(.+)$`)

// CollectReferenced walks workspaceDir, scans every .md file for attachment
// references (both the card-level attachments: field and the body
// attachment:<hash> URL scheme), and returns the union as a set keyed by hash
// (e.g. "a3f9....pdf").
//
// This is a textual scan, not a full parse — it must stay cheap because GC
// and export both call it across every board.
func CollectReferenced(workspaceDir string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	err := filepath.WalkDir(workspaceDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			// Skip the pool dir and any hidden dirs.
			name := d.Name()
			if path != workspaceDir && (name == PoolDir || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		scanRefs(string(data), out)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func scanRefs(content string, out map[string]struct{}) {
	for _, line := range strings.Split(content, "\n") {
		if m := metaAttachmentsLine.FindStringSubmatch(line); m != nil {
			var atts []struct {
				H string `json:"h"`
			}
			if err := json.Unmarshal([]byte(m[1]), &atts); err == nil {
				for _, a := range atts {
					if a.H != "" {
						out[a.H] = struct{}{}
					}
				}
			}
		}
		for _, m := range bodyAttachmentRe.FindAllStringSubmatch(line, -1) {
			out[m[1]] = struct{}{}
		}
	}
}
