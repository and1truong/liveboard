package workspace

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/and1truong/liveboard/internal/web"
)

type legacyTagFrontmatter struct {
	Tags      []string          `yaml:"tags"`
	TagColors map[string]string `yaml:"tag-colors"`
}

// MigrateBoardTagsToWorkspace is a one-shot, idempotent migration: when a
// workspace predates the move of tags + tag-colors from board frontmatter
// onto AppSettings, scan every board for legacy `tags:` / `tag-colors:`
// frontmatter entries and union them into settings.json.
//
// Only fields that are currently empty on AppSettings are populated, so
// repeated calls are safe. Legacy fields stay in the markdown files until
// the next write to each board, at which point the writer drops them.
func (w *Workspace) MigrateBoardTagsToWorkspace() error {
	current := web.LoadSettingsFromDir(w.Dir)
	needTags := len(current.Tags) == 0
	needColors := len(current.TagColors) == 0
	if !needTags && !needColors {
		return nil
	}

	mergedTags, mergedColors, err := w.collectLegacyTags(needTags, needColors)
	if err != nil {
		return err
	}
	if len(mergedTags) == 0 && len(mergedColors) == 0 {
		return nil
	}

	return web.MutateSettings(w.Dir, func(s *web.AppSettings) {
		if needTags && len(mergedTags) > 0 && len(s.Tags) == 0 {
			s.Tags = mergedTags
		}
		if needColors && len(mergedColors) > 0 && len(s.TagColors) == 0 {
			s.TagColors = mergedColors
		}
	})
}

func (w *Workspace) collectLegacyTags(needTags, needColors bool) ([]string, map[string]string, error) {
	var mergedTags []string
	mergedColors := map[string]string{}
	seenTag := map[string]struct{}{}

	err := w.walkBoards(func(relDir string, entry os.DirEntry) {
		legacy, ok := readLegacyTagFrontmatter(filepath.Join(w.Dir, relDir, entry.Name()))
		if !ok {
			return
		}
		if needTags {
			mergedTags = appendLegacyTags(mergedTags, seenTag, legacy.Tags)
		}
		if needColors {
			mergeLegacyColors(mergedColors, legacy.TagColors)
		}
	})
	return mergedTags, mergedColors, err
}

func readLegacyTagFrontmatter(path string) (legacyTagFrontmatter, bool) {
	var out legacyTagFrontmatter
	data, err := os.ReadFile(path)
	if err != nil {
		return out, false
	}
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return out, false
	}
	parts := strings.SplitN(content[4:], "\n---\n", 2)
	if len(parts) != 2 {
		return out, false
	}
	if yaml.Unmarshal([]byte(parts[0]), &out) != nil {
		return out, false
	}
	return out, true
}

func appendLegacyTags(dst []string, seen map[string]struct{}, tags []string) []string {
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		dst = append(dst, t)
	}
	return dst
}

func mergeLegacyColors(dst, src map[string]string) {
	for k, v := range src {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if _, ok := dst[k]; !ok {
			dst[k] = v
		}
	}
}
