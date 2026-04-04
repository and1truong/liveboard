package defaults

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// --- helpers for testing Load/Save without polluting real config ---

func backupConfig(t *testing.T) func() {
	t.Helper()
	path := desktopConfigPath()
	if path == "" {
		return func() {}
	}
	orig, err := os.ReadFile(path)
	if err != nil {
		// no existing file — cleanup means removing whatever the test wrote
		return func() {
			_ = os.Remove(path)
		}
	}
	return func() {
		_ = os.WriteFile(path, orig, 0o644)
	}
}

func TestDesktopConfig_LoadSave(t *testing.T) {
	restore := backupConfig(t)
	defer restore()

	want := &DesktopConfig{
		LastWorkspace:    "/tmp/test-workspace",
		RecentWorkspaces: []string{"/tmp/test-workspace", "/tmp/other"},
		WindowWidth:      1200,
		WindowHeight:     800,
	}

	if err := want.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got := LoadDesktopConfig()

	if got.LastWorkspace != want.LastWorkspace {
		t.Errorf("LastWorkspace = %q, want %q", got.LastWorkspace, want.LastWorkspace)
	}
	if got.WindowWidth != want.WindowWidth {
		t.Errorf("WindowWidth = %d, want %d", got.WindowWidth, want.WindowWidth)
	}
	if got.WindowHeight != want.WindowHeight {
		t.Errorf("WindowHeight = %d, want %d", got.WindowHeight, want.WindowHeight)
	}
	if len(got.RecentWorkspaces) != len(want.RecentWorkspaces) {
		t.Fatalf("RecentWorkspaces len = %d, want %d", len(got.RecentWorkspaces), len(want.RecentWorkspaces))
	}
	for i, ws := range want.RecentWorkspaces {
		if got.RecentWorkspaces[i] != ws {
			t.Errorf("RecentWorkspaces[%d] = %q, want %q", i, got.RecentWorkspaces[i], ws)
		}
	}
}

func TestDesktopConfig_AddRecent(t *testing.T) {
	t.Run("most recent first", func(t *testing.T) {
		c := &DesktopConfig{}
		c.AddRecent("/a")
		c.AddRecent("/b")
		c.AddRecent("/c")

		if c.RecentWorkspaces[0] != "/c" {
			t.Errorf("first = %q, want /c", c.RecentWorkspaces[0])
		}
		if c.LastWorkspace != "/c" {
			t.Errorf("LastWorkspace = %q, want /c", c.LastWorkspace)
		}
	})

	t.Run("deduplication", func(t *testing.T) {
		c := &DesktopConfig{}
		c.AddRecent("/a")
		c.AddRecent("/b")
		c.AddRecent("/a") // re-add

		if len(c.RecentWorkspaces) != 2 {
			t.Fatalf("len = %d, want 2", len(c.RecentWorkspaces))
		}
		if c.RecentWorkspaces[0] != "/a" {
			t.Errorf("first = %q, want /a (moved to front)", c.RecentWorkspaces[0])
		}
		if c.RecentWorkspaces[1] != "/b" {
			t.Errorf("second = %q, want /b", c.RecentWorkspaces[1])
		}
	})

	t.Run("cap at 10", func(t *testing.T) {
		c := &DesktopConfig{}
		for i := 0; i < 15; i++ {
			c.AddRecent(filepath.Join("/ws", string(rune('a'+i))))
		}
		if len(c.RecentWorkspaces) != maxRecentWorkspaces {
			t.Errorf("len = %d, want %d", len(c.RecentWorkspaces), maxRecentWorkspaces)
		}
	})
}

func TestDesktopConfig_CleanStale(t *testing.T) {
	// Create real temp dirs
	real1 := t.TempDir()
	real2 := t.TempDir()
	fake1 := "/tmp/liveboard-test-nonexistent-xyz123"
	fake2 := "/tmp/liveboard-test-nonexistent-abc456"

	c := &DesktopConfig{
		LastWorkspace:    fake1,
		RecentWorkspaces: []string{real1, fake1, real2, fake2},
	}

	c.CleanStale()

	if len(c.RecentWorkspaces) != 2 {
		t.Fatalf("len = %d, want 2", len(c.RecentWorkspaces))
	}
	if c.RecentWorkspaces[0] != real1 || c.RecentWorkspaces[1] != real2 {
		t.Errorf("RecentWorkspaces = %v, want [%s %s]", c.RecentWorkspaces, real1, real2)
	}
	if c.LastWorkspace != "" {
		t.Errorf("LastWorkspace = %q, want empty (was stale)", c.LastWorkspace)
	}
}

func TestDesktopConfig_CleanStale_ValidLast(t *testing.T) {
	realDir := t.TempDir()
	c := &DesktopConfig{
		LastWorkspace:    realDir,
		RecentWorkspaces: []string{realDir},
	}
	c.CleanStale()
	if c.LastWorkspace != realDir {
		t.Errorf("LastWorkspace should remain %q, got %q", realDir, c.LastWorkspace)
	}
}

func TestDesktopConfig_MissingFile(t *testing.T) {
	// Write a config pointing to a nonexistent path, then load.
	// But LoadDesktopConfig uses the hardcoded path.
	// Instead, test that loading when the file has invalid JSON returns zero-value.

	restore := backupConfig(t)
	defer restore()

	path := desktopConfigPath()
	if path == "" {
		t.Skip("cannot determine config path")
	}

	// Remove the file so LoadDesktopConfig hits the read-error branch
	_ = os.Remove(path)

	cfg := LoadDesktopConfig()
	if cfg.LastWorkspace != "" {
		t.Errorf("LastWorkspace = %q, want empty", cfg.LastWorkspace)
	}
	if len(cfg.RecentWorkspaces) != 0 {
		t.Errorf("RecentWorkspaces = %v, want empty", cfg.RecentWorkspaces)
	}

	// Also test invalid JSON
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte("{invalid json}"), 0o644)
	cfg2 := LoadDesktopConfig()
	if cfg2.LastWorkspace != "" {
		t.Errorf("invalid JSON: LastWorkspace = %q, want empty", cfg2.LastWorkspace)
	}
}

func TestDesktopConfigPath(t *testing.T) {
	path := desktopConfigPath()
	if path == "" {
		t.Skip("UserHomeDir failed")
	}
	if filepath.Base(path) != "desktop.json" {
		t.Errorf("config filename = %q, want desktop.json", filepath.Base(path))
	}
}

func TestDesktopConfig_JSON_Roundtrip(t *testing.T) {
	// Test marshal/unmarshal directly (doesn't touch filesystem)
	want := DesktopConfig{
		LastWorkspace:    "/test",
		RecentWorkspaces: []string{"/test", "/other"},
		WindowWidth:      800,
		WindowHeight:     600,
	}
	data, err := json.Marshal(&want)
	if err != nil {
		t.Fatal(err)
	}
	var got DesktopConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.LastWorkspace != want.LastWorkspace || got.WindowWidth != want.WindowWidth {
		t.Errorf("roundtrip mismatch: got %+v", got)
	}
}

func TestWorkDir(t *testing.T) {
	dir, isCloud := WorkDir()
	if dir == "" {
		t.Error("WorkDir returned empty string")
	}
	if isCloud {
		t.Error("isCloud should always be false")
	}
	cwd, _ := os.Getwd()
	if dir != cwd {
		t.Errorf("WorkDir = %q, want cwd %q", dir, cwd)
	}
}

func TestWorkDir_Fallback(t *testing.T) {
	// Verify WorkDir returns something valid regardless of environment
	dir, _ := WorkDir()
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("WorkDir returned non-existent path: %v", err)
	}
	if !info.IsDir() {
		t.Error("WorkDir returned a non-directory path")
	}
}

func TestDesktopWorkDir(t *testing.T) {
	restore := backupConfig(t)
	defer restore()

	// Clear any saved config so it falls through to WorkDir logic
	path := desktopConfigPath()
	if path != "" {
		_ = os.Remove(path)
	}

	dir, _ := DesktopWorkDir()
	if dir == "" {
		t.Error("DesktopWorkDir returned empty string")
	}
	// Should be a real directory
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("DesktopWorkDir returned non-existent path %q: %v", dir, err)
	}
	if !info.IsDir() {
		t.Error("DesktopWorkDir returned a non-directory")
	}
}

func TestDesktopWorkDir_SavedConfig(t *testing.T) {
	restore := backupConfig(t)
	defer restore()

	// Create a temp dir and save it as LastWorkspace
	tmpDir := t.TempDir()
	cfg := &DesktopConfig{LastWorkspace: tmpDir}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	dir, isCloud := DesktopWorkDir()
	if dir != tmpDir {
		t.Errorf("DesktopWorkDir = %q, want saved workspace %q", dir, tmpDir)
	}
	if isCloud {
		t.Error("isCloud should be false for saved workspace")
	}
}

func TestDesktopWorkDir_StaleConfig(t *testing.T) {
	restore := backupConfig(t)
	defer restore()

	// Save config pointing to nonexistent dir — should fall through
	cfg := &DesktopConfig{LastWorkspace: "/tmp/liveboard-nonexistent-xyz789"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	dir, _ := DesktopWorkDir()
	if dir == "/tmp/liveboard-nonexistent-xyz789" {
		t.Error("DesktopWorkDir should not return stale path")
	}
	if dir == "" {
		t.Error("DesktopWorkDir returned empty string")
	}
}

func TestDesktopConfig_SaveCreatesDir(t *testing.T) {
	// Verify that Save creates the parent directory if needed.
	// This is implicitly tested by LoadSave, but let's verify the file exists.
	restore := backupConfig(t)
	defer restore()

	cfg := &DesktopConfig{LastWorkspace: "/test"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	path := desktopConfigPath()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestDesktopWorkDir_RootCwd(t *testing.T) {
	restore := backupConfig(t)
	defer restore()

	// Clear saved config
	path := desktopConfigPath()
	if path != "" {
		_ = os.Remove(path)
	}

	// Save cwd and change to /
	origDir, getErr := os.Getwd()
	if getErr != nil {
		t.Fatal(getErr)
	}
	defer os.Chdir(origDir) //nolint:errcheck // best-effort restore

	if chdirErr := os.Chdir("/"); chdirErr != nil {
		t.Fatal(chdirErr)
	}

	dir, isCloud := DesktopWorkDir()
	if isCloud {
		t.Error("expected isCloud=false")
	}
	// Should fall back to ~/LiveBoard (created if needed)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(home, "LiveBoard")
	if dir != expected {
		t.Errorf("DesktopWorkDir from / = %q, want %q", dir, expected)
	}
	// Verify it was actually created
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("fallback dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("fallback is not a directory")
	}
}

func TestDesktopConfig_EmptyHome(t *testing.T) {
	// Unset HOME to trigger desktopConfigPath() returning ""
	origHome := os.Getenv("HOME")
	_ = os.Unsetenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", origHome) })

	// desktopConfigPath should return ""
	p := desktopConfigPath()
	if p != "" {
		t.Errorf("desktopConfigPath = %q with no HOME, want empty", p)
	}

	// LoadDesktopConfig with no HOME => zero-value
	cfg := LoadDesktopConfig()
	if cfg.LastWorkspace != "" || len(cfg.RecentWorkspaces) != 0 {
		t.Error("expected zero-value config with no HOME")
	}

	// Save with no HOME => nil error (no-op)
	err := cfg.Save()
	if err != nil {
		t.Errorf("Save with no HOME: %v", err)
	}
}

func TestWorkDir_NoHome(t *testing.T) {
	origHome := os.Getenv("HOME")
	_ = os.Unsetenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", origHome) })

	dir, isCloud := WorkDir()
	if isCloud {
		t.Error("isCloud=true with no HOME")
	}
	// Should fall back to cwd
	cwd, _ := os.Getwd()
	if dir != cwd {
		t.Errorf("WorkDir = %q, want cwd %q", dir, cwd)
	}
}

func TestDesktopWorkDir_NoHome(t *testing.T) {
	origHome := os.Getenv("HOME")
	_ = os.Unsetenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", origHome) })

	dir, isCloud := DesktopWorkDir()
	if isCloud {
		t.Error("isCloud=true with no HOME")
	}
	// LoadDesktopConfig returns zero (no HOME), WorkDir falls back to cwd
	cwd, _ := os.Getwd()
	if dir != cwd {
		t.Errorf("DesktopWorkDir = %q, want cwd %q", dir, cwd)
	}
}

// --- CLIConfig tests ---

func backupCLIConfig(t *testing.T) func() {
	t.Helper()
	path := cliConfigPath()
	if path == "" {
		return func() {}
	}
	orig, err := os.ReadFile(path)
	if err != nil {
		return func() { _ = os.Remove(path) }
	}
	return func() { _ = os.WriteFile(path, orig, 0o644) }
}

func TestLoadCLIConfig_Missing(t *testing.T) {
	restore := backupCLIConfig(t)
	defer restore()

	path := cliConfigPath()
	_ = os.Remove(path)

	cfg := LoadCLIConfig()
	if cfg.Workspace != "" || cfg.Host != "" || cfg.Port != 0 || cfg.ReadOnly {
		t.Errorf("expected zero-value config, got %+v", cfg)
	}
}

func TestLoadCLIConfig_Valid(t *testing.T) {
	restore := backupCLIConfig(t)
	defer restore()

	path := cliConfigPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	data := []byte(`{"workspace":"/tmp/boards","host":"0.0.0.0","port":8080,"readonly":true}`)
	_ = os.WriteFile(path, data, 0o644)

	cfg := LoadCLIConfig()
	if cfg.Workspace != "/tmp/boards" {
		t.Errorf("Workspace = %q, want /tmp/boards", cfg.Workspace)
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want 0.0.0.0", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if !cfg.ReadOnly {
		t.Error("ReadOnly = false, want true")
	}
}

func TestLoadCLIConfig_InvalidJSON(t *testing.T) {
	restore := backupCLIConfig(t)
	defer restore()

	path := cliConfigPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(`{bad json`), 0o644)

	cfg := LoadCLIConfig()
	if cfg.Workspace != "" {
		t.Errorf("expected zero-value on invalid JSON, got %+v", cfg)
	}
}

func TestDesktopConfig_CleanStale_FileNotDir(t *testing.T) {
	// A file (not directory) in RecentWorkspaces should be cleaned
	tmpFile := filepath.Join(t.TempDir(), "afile")
	_ = os.WriteFile(tmpFile, []byte("hi"), 0o644)

	c := &DesktopConfig{
		LastWorkspace:    tmpFile,
		RecentWorkspaces: []string{tmpFile},
	}
	c.CleanStale()

	if len(c.RecentWorkspaces) != 0 {
		t.Errorf("file entry not cleaned: %v", c.RecentWorkspaces)
	}
	if c.LastWorkspace != "" {
		t.Errorf("LastWorkspace not cleared for file: %q", c.LastWorkspace)
	}
}
