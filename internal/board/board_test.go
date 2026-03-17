package board

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testBoard = `---
name: Test Board
---

## Backlog

- [ ] Task one
<!-- liveboard:id=id-001 -->
  tags: backend

## In Progress

## Done
`

func setupTestBoard(t *testing.T) (string, *Engine) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte(testBoard), 0644); err != nil {
		t.Fatal(err)
	}
	return path, New()
}

func TestLoadBoard(t *testing.T) {
	path, eng := setupTestBoard(t)
	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	if board.Name != "Test Board" {
		t.Errorf("name = %q", board.Name)
	}
	if len(board.Columns) != 3 {
		t.Fatalf("columns = %d", len(board.Columns))
	}
}

func TestAddCard(t *testing.T) {
	path, eng := setupTestBoard(t)
	card, err := eng.AddCard(path, "Backlog", "New task")
	if err != nil {
		t.Fatal(err)
	}
	if card.ID == "" {
		t.Error("card ID should be assigned")
	}

	// Verify it's in the file.
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "New task") {
		t.Error("card not found in file")
	}
}

func TestMoveCard(t *testing.T) {
	path, eng := setupTestBoard(t)
	if err := eng.MoveCard(path, "id-001", "In Progress"); err != nil {
		t.Fatal(err)
	}

	board, _ := eng.LoadBoard(path)
	// Should not be in Backlog.
	for _, card := range board.Columns[0].Cards {
		if card.ID == "id-001" {
			t.Error("card still in Backlog")
		}
	}
	// Should be in In Progress.
	found := false
	for _, card := range board.Columns[1].Cards {
		if card.ID == "id-001" {
			found = true
		}
	}
	if !found {
		t.Error("card not found in In Progress")
	}
}

func TestCompleteCard(t *testing.T) {
	path, eng := setupTestBoard(t)
	if err := eng.CompleteCard(path, "id-001"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "- [x] Task one") {
		t.Error("card not marked as completed")
	}
}

func TestTagCard(t *testing.T) {
	path, eng := setupTestBoard(t)
	if err := eng.TagCard(path, "id-001", []string{"urgent", "backend"}); err != nil {
		t.Fatal(err)
	}

	board, _ := eng.LoadBoard(path)
	card := findCard(board, "id-001")
	if card == nil {
		t.Fatal("card not found")
	}
	// Should have backend + urgent (backend already existed, no duplicates).
	hasUrgent := false
	for _, tag := range card.Tags {
		if tag == "urgent" {
			hasUrgent = true
		}
	}
	if !hasUrgent {
		t.Errorf("tags = %v, missing urgent", card.Tags)
	}
}

func TestDeleteCard(t *testing.T) {
	path, eng := setupTestBoard(t)
	if err := eng.DeleteCard(path, "id-001"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "id-001") {
		t.Error("deleted card still present")
	}
}

func TestAddColumn(t *testing.T) {
	path, eng := setupTestBoard(t)
	if err := eng.AddColumn(path, "Testing"); err != nil {
		t.Fatal(err)
	}

	board, _ := eng.LoadBoard(path)
	if len(board.Columns) != 4 {
		t.Fatalf("columns = %d, want 4", len(board.Columns))
	}
	if board.Columns[3].Name != "Testing" {
		t.Errorf("new column = %q", board.Columns[3].Name)
	}
}

func TestDeleteColumn(t *testing.T) {
	path, eng := setupTestBoard(t)
	if err := eng.DeleteColumn(path, "Done"); err != nil {
		t.Fatal(err)
	}

	board, _ := eng.LoadBoard(path)
	if len(board.Columns) != 2 {
		t.Fatalf("columns = %d, want 2", len(board.Columns))
	}
}

func TestShowCard(t *testing.T) {
	path, eng := setupTestBoard(t)
	card, col, err := eng.ShowCard(path, "id-001")
	if err != nil {
		t.Fatal(err)
	}
	if card.Title != "Task one" {
		t.Errorf("title = %q", card.Title)
	}
	if col != "Backlog" {
		t.Errorf("column = %q", col)
	}
}
