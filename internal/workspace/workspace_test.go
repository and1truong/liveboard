package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

const testBoardContent = `---
name: My Board
---

## Backlog

- [ ] Task one

## Done
`

func setupWorkspace(t *testing.T) *Workspace {
	t.Helper()
	return Open(t.TempDir())
}

func createBoardFile(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name+".md")
	if err := os.WriteFile(path, []byte(testBoardContent), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestOpen(t *testing.T) {
	dir := t.TempDir()
	ws := Open(dir)
	if ws.Dir != dir {
		t.Errorf("Dir = %q, want %q", ws.Dir, dir)
	}
	if ws.Engine == nil {
		t.Error("Engine should not be nil")
	}
}

func TestBoardPath(t *testing.T) {
	ws := setupWorkspace(t)
	got := ws.BoardPath("roadmap")
	want := filepath.Join(ws.Dir, "roadmap.md")
	if got != want {
		t.Errorf("BoardPath = %q, want %q", got, want)
	}
}

func TestListBoards_Empty(t *testing.T) {
	ws := setupWorkspace(t)
	boards, err := ws.ListBoards()
	if err != nil {
		t.Fatal(err)
	}
	if len(boards) != 0 {
		t.Errorf("expected 0 boards, got %d", len(boards))
	}
}

func TestListBoards_WithBoards(t *testing.T) {
	ws := setupWorkspace(t)
	createBoardFile(t, ws.Dir, "roadmap")
	createBoardFile(t, ws.Dir, "sprints")

	boards, err := ws.ListBoards()
	if err != nil {
		t.Fatal(err)
	}
	if len(boards) != 2 {
		t.Errorf("expected 2 boards, got %d", len(boards))
	}
}

func TestListBoards_SkipsReadme(t *testing.T) {
	ws := setupWorkspace(t)
	if err := os.WriteFile(filepath.Join(ws.Dir, "README.md"), []byte("# Readme\n"), 0644); err != nil {
		t.Fatal(err)
	}
	createBoardFile(t, ws.Dir, "roadmap")

	boards, err := ws.ListBoards()
	if err != nil {
		t.Fatal(err)
	}
	if len(boards) != 1 {
		t.Errorf("expected 1 board (README excluded), got %d", len(boards))
	}
}

func TestListBoards_SkipsDirectories(t *testing.T) {
	ws := setupWorkspace(t)
	if err := os.Mkdir(filepath.Join(ws.Dir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	createBoardFile(t, ws.Dir, "roadmap")

	boards, err := ws.ListBoards()
	if err != nil {
		t.Fatal(err)
	}
	if len(boards) != 1 {
		t.Errorf("expected 1 board, got %d", len(boards))
	}
}

func TestCreateBoard(t *testing.T) {
	ws := setupWorkspace(t)
	b, err := ws.CreateBoard("roadmap")
	if err != nil {
		t.Fatal(err)
	}
	if b.Name != "roadmap" {
		t.Errorf("name = %q, want %q", b.Name, "roadmap")
	}
	if len(b.Columns) == 0 {
		t.Error("expected default columns")
	}
	if _, err := os.Stat(ws.BoardPath("roadmap")); os.IsNotExist(err) {
		t.Error("board file not created on disk")
	}
}

func TestCreateBoard_DefaultColumns(t *testing.T) {
	ws := setupWorkspace(t)
	b, err := ws.CreateBoard("roadmap")
	if err != nil {
		t.Fatal(err)
	}

	wantCols := defaultColumns
	if len(b.Columns) != len(wantCols) {
		t.Fatalf("columns = %d, want %d", len(b.Columns), len(wantCols))
	}
	for i, col := range b.Columns {
		if col.Name != wantCols[i] {
			t.Errorf("column[%d] = %q, want %q", i, col.Name, wantCols[i])
		}
	}
}

func TestCreateBoard_AlreadyExists(t *testing.T) {
	ws := setupWorkspace(t)
	createBoardFile(t, ws.Dir, "roadmap")

	_, err := ws.CreateBoard("roadmap")
	if err == nil {
		t.Error("expected error when board already exists")
	}
}

func TestDeleteBoard(t *testing.T) {
	ws := setupWorkspace(t)
	createBoardFile(t, ws.Dir, "roadmap")

	if err := ws.DeleteBoard("roadmap"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ws.BoardPath("roadmap")); !os.IsNotExist(err) {
		t.Error("expected board file to be deleted")
	}
}

func TestDeleteBoard_NotFound(t *testing.T) {
	ws := setupWorkspace(t)
	if err := ws.DeleteBoard("nonexistent"); err == nil {
		t.Error("expected error for missing board")
	}
}

func TestLoadBoard(t *testing.T) {
	ws := setupWorkspace(t)
	createBoardFile(t, ws.Dir, "roadmap")

	b, err := ws.LoadBoard("roadmap")
	if err != nil {
		t.Fatal(err)
	}
	if b.Name != "My Board" {
		t.Errorf("name = %q, want %q", b.Name, "My Board")
	}
}

func TestGetDefaultColumns_FallsBackToDefaults(t *testing.T) {
	ws := setupWorkspace(t)
	cols := ws.getDefaultColumns()
	if len(cols) != len(defaultColumns) {
		t.Fatalf("expected %d columns, got %d", len(defaultColumns), len(cols))
	}
	for i, want := range defaultColumns {
		if cols[i] != want {
			t.Errorf("column[%d] = %q, want %q", i, cols[i], want)
		}
	}
}

func TestGetDefaultColumns_FromConfig(t *testing.T) {
	ws := setupWorkspace(t)

	configDir := filepath.Join(ws.Dir, ".liveboard")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configContent := "board:\n  default_columns:\n    - Todo\n    - Doing\n    - Done\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cols := ws.getDefaultColumns()
	if len(cols) != 3 {
		t.Fatalf("expected 3 columns from config, got %d", len(cols))
	}
	wantCols := []string{"Todo", "Doing", "Done"}
	for i, want := range wantCols {
		if cols[i] != want {
			t.Errorf("column[%d] = %q, want %q", i, cols[i], want)
		}
	}
}
