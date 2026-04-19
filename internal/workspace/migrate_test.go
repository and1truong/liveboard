package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/and1truong/liveboard/internal/web"
)

func writeBoard(t *testing.T, dir, name, frontmatter string) {
	t.Helper()
	path := filepath.Join(dir, name)
	content := "---\n" + frontmatter + "---\n\n## Todo\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeSettings(t *testing.T, dir string, s web.AppSettings) {
	t.Helper()
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), data, 0o644); err != nil {
		t.Fatalf("write settings.json: %v", err)
	}
}

func TestMigrateBoardTagsToWorkspace_Unions(t *testing.T) {
	dir := t.TempDir()
	writeBoard(t, dir, "a.md",
		"version: 1\nname: A\ntags: [alpha, shared]\ntag-colors:\n  alpha: \"#ff0000\"\n  shared: \"#00ff00\"\n")
	writeBoard(t, dir, "b.md",
		"version: 1\nname: B\ntags: [beta, shared]\ntag-colors:\n  beta: \"#0000ff\"\n  shared: \"#abcdef\"\n")

	w := Open(dir)
	if err := w.MigrateBoardTagsToWorkspace(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	got := web.LoadSettingsFromDir(dir)
	want := []string{"alpha", "shared", "beta"}
	for _, tag := range want {
		if !slices.Contains(got.Tags, tag) {
			t.Errorf("Tags missing %q: got %v", tag, got.Tags)
		}
	}
	if got.TagColors["alpha"] != "#ff0000" || got.TagColors["beta"] != "#0000ff" {
		t.Errorf("TagColors = %v", got.TagColors)
	}
	// First-seen wins on collision.
	if got.TagColors["shared"] != "#00ff00" {
		t.Errorf("shared color = %q, want #00ff00 (first-seen)", got.TagColors["shared"])
	}
}

func TestMigrateBoardTagsToWorkspace_Idempotent(t *testing.T) {
	dir := t.TempDir()
	writeBoard(t, dir, "a.md",
		"version: 1\nname: A\ntags: [alpha]\ntag-colors:\n  alpha: \"#ff0000\"\n")

	w := Open(dir)
	if err := w.MigrateBoardTagsToWorkspace(); err != nil {
		t.Fatalf("migrate 1: %v", err)
	}
	first := web.LoadSettingsFromDir(dir)

	if err := w.MigrateBoardTagsToWorkspace(); err != nil {
		t.Fatalf("migrate 2: %v", err)
	}
	second := web.LoadSettingsFromDir(dir)

	if !slices.Equal(first.Tags, second.Tags) {
		t.Errorf("Tags differ: %v vs %v", first.Tags, second.Tags)
	}
	if len(first.TagColors) != len(second.TagColors) {
		t.Errorf("TagColors len differ: %d vs %d", len(first.TagColors), len(second.TagColors))
	}
}

func TestMigrateBoardTagsToWorkspace_SkipsWhenAlreadySet(t *testing.T) {
	dir := t.TempDir()
	writeBoard(t, dir, "a.md",
		"version: 1\nname: A\ntags: [alpha]\ntag-colors:\n  alpha: \"#ff0000\"\n")
	writeSettings(t, dir, web.AppSettings{
		Tags:      []string{"preexisting"},
		TagColors: map[string]string{"preexisting": "#123456"},
	})

	w := Open(dir)
	if err := w.MigrateBoardTagsToWorkspace(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	got := web.LoadSettingsFromDir(dir)
	if !slices.Equal(got.Tags, []string{"preexisting"}) {
		t.Errorf("Tags got clobbered: %v", got.Tags)
	}
	if got.TagColors["preexisting"] != "#123456" {
		t.Errorf("TagColors got clobbered: %v", got.TagColors)
	}
	if _, exists := got.TagColors["alpha"]; exists {
		t.Errorf("should not have added alpha when tag_colors was already set")
	}
}

func TestMigrateBoardTagsToWorkspace_NoBoards(t *testing.T) {
	dir := t.TempDir()
	w := Open(dir)
	if err := w.MigrateBoardTagsToWorkspace(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}
