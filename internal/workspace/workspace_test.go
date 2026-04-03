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
	got, err := ws.BoardPath("roadmap")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(ws.Dir, "roadmap.md")
	if got != want {
		t.Errorf("BoardPath = %q, want %q", got, want)
	}
}

func TestBoardPath_Traversal(t *testing.T) {
	ws := setupWorkspace(t)
	cases := []string{
		"../../../etc/passwd",
		"../../escape",
		"/etc/passwd",
		"board/../../escape",
		"valid/../../../escape",
		"board\x00evil",
	}
	for _, name := range cases {
		_, err := ws.BoardPath(name)
		if err == nil {
			t.Errorf("BoardPath(%q) should fail, but got nil error", name)
		}
	}
}

func TestValidateBoardName(t *testing.T) {
	valid := []string{"roadmap", "my-board", "My Board", "board_v2", "日本語"}
	for _, name := range valid {
		if err := ValidateBoardName(name); err != nil {
			t.Errorf("ValidateBoardName(%q) = %v, want nil", name, err)
		}
	}

	invalid := []string{"", "../escape", "/absolute", "has/slash", "bad\x00null"}
	for _, name := range invalid {
		if err := ValidateBoardName(name); err == nil {
			t.Errorf("ValidateBoardName(%q) = nil, want error", name)
		}
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
	if p, e := ws.BoardPath("roadmap"); e != nil {
		t.Fatal(e)
	} else if _, err := os.Stat(p); os.IsNotExist(err) {
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
	if p, e := ws.BoardPath("roadmap"); e != nil {
		t.Fatal(e)
	} else if _, err := os.Stat(p); !os.IsNotExist(err) {
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

func TestListBoardSummaries(t *testing.T) {
	ws := setupWorkspace(t)
	createBoardFile(t, ws.Dir, "roadmap")
	createBoardFile(t, ws.Dir, "sprints")

	summaries, err := ws.ListBoardSummaries()
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}
	for _, s := range summaries {
		if s.Board.FilePath == "" {
			t.Error("expected FilePath to be set")
		}
		if s.ColumnCount != 2 {
			t.Errorf("expected 2 columns, got %d", s.ColumnCount)
		}
		if s.CardCount != 1 {
			t.Errorf("expected 1 card, got %d", s.CardCount)
		}
		if s.Board.UpdatedAt.IsZero() {
			t.Error("expected UpdatedAt to be set")
		}
	}
}

func TestListBoardSummaries_Empty(t *testing.T) {
	ws := setupWorkspace(t)
	summaries, err := ws.ListBoardSummaries()
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 0 {
		t.Errorf("expected 0 summaries, got %d", len(summaries))
	}
}

func TestListBoardSummaries_SkipsReadmeAndDirs(t *testing.T) {
	ws := setupWorkspace(t)
	createBoardFile(t, ws.Dir, "roadmap")
	if err := os.WriteFile(filepath.Join(ws.Dir, "README.md"), []byte("# Readme\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(ws.Dir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}

	summaries, err := ws.ListBoardSummaries()
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 {
		t.Errorf("expected 1 summary (README and dirs excluded), got %d", len(summaries))
	}
}

func TestListBoardSummaries_SkipsUnparseable(t *testing.T) {
	ws := setupWorkspace(t)
	createBoardFile(t, ws.Dir, "good")
	// Write a file with broken frontmatter that parser.ParseSummary should still handle
	if err := os.WriteFile(filepath.Join(ws.Dir, "bad.md"), []byte("not valid board content"), 0644); err != nil {
		t.Fatal(err)
	}

	summaries, err := ws.ListBoardSummaries()
	if err != nil {
		t.Fatal(err)
	}
	// ParseSummary may or may not error on content without frontmatter;
	// either way it should not crash and we get at least the good board
	if len(summaries) < 1 {
		t.Errorf("expected at least 1 summary, got %d", len(summaries))
	}
}

func TestListBoardSummaries_BadDir(t *testing.T) {
	ws := Open("/nonexistent/path")
	_, err := ws.ListBoardSummaries()
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestListBoards_BadDir(t *testing.T) {
	ws := Open("/nonexistent/path")
	_, err := ws.ListBoards()
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestCreateBoard_WithSettingsDefaultColumns(t *testing.T) {
	ws := setupWorkspace(t)
	settingsContent := `{"default_columns": ["Inbox", "In Progress", "Review", "Shipped"]}`
	if err := os.WriteFile(filepath.Join(ws.Dir, "settings.json"), []byte(settingsContent), 0644); err != nil {
		t.Fatal(err)
	}

	b, err := ws.CreateBoard("roadmap")
	if err != nil {
		t.Fatal(err)
	}

	wantCols := []string{"Inbox", "In Progress", "Review", "Shipped"}
	if len(b.Columns) != len(wantCols) {
		t.Fatalf("expected %d columns, got %d", len(wantCols), len(b.Columns))
	}
	for i, want := range wantCols {
		if b.Columns[i].Name != want {
			t.Errorf("column[%d] = %q, want %q", i, b.Columns[i].Name, want)
		}
	}
}

func TestGetDefaultColumns_SettingsJsonTakesPrecedence(t *testing.T) {
	ws := setupWorkspace(t)

	// Create both settings.json and .liveboard/config.yaml
	settingsContent := `{"default_columns": ["A", "B"]}`
	if err := os.WriteFile(filepath.Join(ws.Dir, "settings.json"), []byte(settingsContent), 0644); err != nil {
		t.Fatal(err)
	}
	configDir := filepath.Join(ws.Dir, ".liveboard")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configContent := "board:\n  default_columns:\n    - X\n    - Y\n    - Z\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cols := ws.getDefaultColumns()
	// settings.json should win
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns from settings.json, got %d", len(cols))
	}
	if cols[0] != "A" || cols[1] != "B" {
		t.Errorf("cols = %v, want [A B]", cols)
	}
}

func TestGetDefaultColumns_InvalidSettingsJson(t *testing.T) {
	ws := setupWorkspace(t)
	// Write invalid JSON — should fall through to defaults
	if err := os.WriteFile(filepath.Join(ws.Dir, "settings.json"), []byte("{bad json}"), 0644); err != nil {
		t.Fatal(err)
	}

	cols := ws.getDefaultColumns()
	if len(cols) != len(defaultColumns) {
		t.Fatalf("expected fallback to default columns, got %d", len(cols))
	}
}

func TestGetDefaultColumns_EmptySettingsJsonColumns(t *testing.T) {
	ws := setupWorkspace(t)
	// settings.json with empty default_columns — should fall through
	if err := os.WriteFile(filepath.Join(ws.Dir, "settings.json"), []byte(`{"default_columns": []}`), 0644); err != nil {
		t.Fatal(err)
	}

	cols := ws.getDefaultColumns()
	if len(cols) != len(defaultColumns) {
		t.Fatalf("expected fallback to default columns when settings has empty array, got %d", len(cols))
	}
}

func TestCreateBoard_InvalidName(t *testing.T) {
	ws := setupWorkspace(t)
	_, err := ws.CreateBoard("../escape")
	if err == nil {
		t.Error("expected error for invalid board name")
	}
}

func TestDeleteBoard_InvalidName(t *testing.T) {
	ws := setupWorkspace(t)
	err := ws.DeleteBoard("../escape")
	if err == nil {
		t.Error("expected error for invalid board name")
	}
}

func TestLoadBoard_NotFound(t *testing.T) {
	ws := setupWorkspace(t)
	_, err := ws.LoadBoard("nonexistent")
	if err == nil {
		t.Error("expected error for missing board")
	}
}

func TestLoadBoard_InvalidName(t *testing.T) {
	ws := setupWorkspace(t)
	_, err := ws.LoadBoard("../escape")
	if err == nil {
		t.Error("expected error for invalid board name")
	}
}

func TestListBoards_SkipsUnparseableFiles(t *testing.T) {
	ws := setupWorkspace(t)
	createBoardFile(t, ws.Dir, "good")
	// Write a non-board .md file that the parser will reject
	if err := os.WriteFile(filepath.Join(ws.Dir, "broken.md"), []byte("not a board"), 0644); err != nil {
		t.Fatal(err)
	}

	boards, err := ws.ListBoards()
	if err != nil {
		t.Fatal(err)
	}
	// "broken.md" may or may not parse, but we should get at least the good board and no crash
	if len(boards) < 1 {
		t.Errorf("expected at least 1 board, got %d", len(boards))
	}
}

func TestListBoards_SkipsNonMdFiles(t *testing.T) {
	ws := setupWorkspace(t)
	createBoardFile(t, ws.Dir, "roadmap")
	if err := os.WriteFile(filepath.Join(ws.Dir, "notes.txt"), []byte("text file"), 0644); err != nil {
		t.Fatal(err)
	}

	boards, err := ws.ListBoards()
	if err != nil {
		t.Fatal(err)
	}
	if len(boards) != 1 {
		t.Errorf("expected 1 board (txt excluded), got %d", len(boards))
	}
}
