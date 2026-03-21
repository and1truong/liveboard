package board

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/and1truong/liveboard/pkg/models"
)

const testBoard = `---
name: Test Board
---

## Backlog

- [ ] Task one
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
	card, err := eng.AddCard(path, "Backlog", "New task", false)
	if err != nil {
		t.Fatal(err)
	}
	if card.Title != "New task" {
		t.Errorf("title = %q", card.Title)
	}

	// Verify it's in the file.
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "New task") {
		t.Error("card not found in file")
	}
}

func TestMoveCard(t *testing.T) {
	path, eng := setupTestBoard(t)
	// Move card at col=0, card=0 to "In Progress"
	if err := eng.MoveCard(path, 0, 0, "In Progress"); err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	// Should not be in Backlog.
	if len(board.Columns[0].Cards) != 0 {
		t.Error("card still in Backlog")
	}
	// Should be in In Progress.
	if len(board.Columns[1].Cards) != 1 {
		t.Errorf("In Progress cards = %d, want 1", len(board.Columns[1].Cards))
	}
	if board.Columns[1].Cards[0].Title != "Task one" {
		t.Errorf("moved card title = %q", board.Columns[1].Cards[0].Title)
	}
}

func TestCompleteCard(t *testing.T) {
	path, eng := setupTestBoard(t)
	if err := eng.CompleteCard(path, 0, 0); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "- [x] Task one") {
		t.Error("card not marked as completed")
	}
}

func TestTagCard(t *testing.T) {
	path, eng := setupTestBoard(t)
	if err := eng.TagCard(path, 0, 0, []string{"urgent", "backend"}); err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	card := board.Columns[0].Cards[0]
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
	if err := eng.DeleteCard(path, 0, 0); err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Columns[0].Cards) != 0 {
		t.Error("deleted card still present")
	}
}

func TestAddColumn(t *testing.T) {
	path, eng := setupTestBoard(t)
	if err := eng.AddColumn(path, "Testing"); err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
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

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Columns) != 2 {
		t.Fatalf("columns = %d, want 2", len(board.Columns))
	}
}

func TestShowCard(t *testing.T) {
	path, eng := setupTestBoard(t)
	card, col, err := eng.ShowCard(path, 0, 0)
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

// --- New tests for uncovered functions ---

const testBoardMultiCard = `---
name: Multi Card Board
---

## Backlog

- [ ] Alpha
  priority: high
  due: 2025-03-01
- [ ] Charlie
  priority: low
  due: 2025-01-15
- [ ] Bravo
  priority: critical
  due: 2025-02-10

## In Progress

- [ ] Delta

## Done
`

func setupMultiCardBoard(t *testing.T) (string, *Engine) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "multi.md")
	if err := os.WriteFile(path, []byte(testBoardMultiCard), 0644); err != nil {
		t.Fatal(err)
	}
	return path, New()
}

func TestReorderCard(t *testing.T) {
	path, eng := setupMultiCardBoard(t)

	// Reorder card at index 2 (Bravo) to before index 0 (Alpha) in same column
	if err := eng.ReorderCard(path, 0, 2, 0, "Backlog"); err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	if board.Columns[0].Cards[0].Title != "Bravo" {
		t.Errorf("expected Bravo at index 0, got %q", board.Columns[0].Cards[0].Title)
	}
	if len(board.Columns[0].Cards) != 3 {
		t.Errorf("expected 3 cards, got %d", len(board.Columns[0].Cards))
	}
}

func TestReorderCardAppend(t *testing.T) {
	path, eng := setupMultiCardBoard(t)

	// Reorder card from Backlog to In Progress, append (beforeIdx=-1)
	if err := eng.ReorderCard(path, 0, 0, -1, "In Progress"); err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Columns[1].Cards) != 2 {
		t.Fatalf("expected 2 cards in In Progress, got %d", len(board.Columns[1].Cards))
	}
	if board.Columns[1].Cards[1].Title != "Alpha" {
		t.Errorf("expected Alpha appended, got %q", board.Columns[1].Cards[1].Title)
	}
}

func TestReorderCardInvalidTarget(t *testing.T) {
	path, eng := setupMultiCardBoard(t)

	err := eng.ReorderCard(path, 0, 0, 0, "Nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent target column")
	}
}

func TestReorderCardInvalidIndices(t *testing.T) {
	path, eng := setupMultiCardBoard(t)

	err := eng.ReorderCard(path, 99, 0, 0, "Backlog")
	if err == nil {
		t.Fatal("expected error for invalid column index")
	}
}

func TestEditCard(t *testing.T) {
	path, eng := setupMultiCardBoard(t)

	err := eng.EditCard(path, 0, 0, "Updated Title", "Some body text", []string{"tag1", "tag2"}, "medium", "2025-06-01", "alice")
	if err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	card := board.Columns[0].Cards[0]
	if card.Title != "Updated Title" {
		t.Errorf("title = %q, want 'Updated Title'", card.Title)
	}
	if card.Body != "Some body text" {
		t.Errorf("body = %q", card.Body)
	}
	if card.Priority != "medium" {
		t.Errorf("priority = %q", card.Priority)
	}
	if card.Due != "2025-06-01" {
		t.Errorf("due = %q", card.Due)
	}
	if len(card.Tags) != 2 {
		t.Errorf("tags = %v", card.Tags)
	}
}

func TestEditCardPartialUpdate(t *testing.T) {
	path, eng := setupMultiCardBoard(t)

	// Empty title means keep existing
	err := eng.EditCard(path, 0, 0, "", "", nil, "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	if board.Columns[0].Cards[0].Title != "Alpha" {
		t.Errorf("title should be preserved, got %q", board.Columns[0].Cards[0].Title)
	}
}

func TestEditCardInvalidIndex(t *testing.T) {
	path, eng := setupMultiCardBoard(t)

	err := eng.EditCard(path, 0, 99, "X", "", nil, "", "", "")
	if err == nil {
		t.Fatal("expected error for invalid card index")
	}
}

func TestRenameColumn(t *testing.T) {
	path, eng := setupTestBoard(t)

	if err := eng.RenameColumn(path, "Backlog", "Todo"); err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	if board.Columns[0].Name != "Todo" {
		t.Errorf("column name = %q, want 'Todo'", board.Columns[0].Name)
	}
}

func TestMoveColumn(t *testing.T) {
	path, eng := setupTestBoard(t)

	// Move "Backlog" after "Done" (Backlog, In Progress, Done → In Progress, Done, Backlog)
	if err := eng.MoveColumn(path, "Backlog", "Done"); err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(board.Columns))
	}
	if board.Columns[0].Name != "In Progress" {
		t.Errorf("col 0 = %q, want 'In Progress'", board.Columns[0].Name)
	}
	if board.Columns[2].Name != "Backlog" {
		t.Errorf("col 2 = %q, want 'Backlog'", board.Columns[2].Name)
	}
}

func TestMoveColumnNotFound(t *testing.T) {
	path, eng := setupTestBoard(t)

	err := eng.MoveColumn(path, "Nonexistent", "Done")
	if err == nil {
		t.Fatal("expected error for nonexistent column")
	}
}

func TestToggleColumnCollapse(t *testing.T) {
	path, eng := setupTestBoard(t)

	// Toggle on
	if err := eng.ToggleColumnCollapse(path, 0); err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.ListCollapse) < 1 || !board.ListCollapse[0] {
		t.Error("expected column 0 to be collapsed")
	}

	// Toggle off
	if err := eng.ToggleColumnCollapse(path, 0); err != nil {
		t.Fatal(err)
	}
	board, err = eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	if board.ListCollapse[0] {
		t.Error("expected column 0 to be uncollapsed")
	}
}

func TestToggleColumnCollapseInvalidIndex(t *testing.T) {
	path, eng := setupTestBoard(t)

	err := eng.ToggleColumnCollapse(path, 99)
	if err == nil {
		t.Fatal("expected error for invalid index")
	}
}

func TestUpdateBoardMeta(t *testing.T) {
	path, eng := setupTestBoard(t)

	err := eng.UpdateBoardMeta(path, "New Name", "A description", []string{"tag1", "tag2"})
	if err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	if board.Name != "New Name" {
		t.Errorf("name = %q", board.Name)
	}
	if board.Description != "A description" {
		t.Errorf("description = %q", board.Description)
	}
	if len(board.Tags) != 2 {
		t.Errorf("tags = %v", board.Tags)
	}
}

func TestUpdateBoardMetaEmptyName(t *testing.T) {
	path, eng := setupTestBoard(t)

	// Empty name means keep existing
	err := eng.UpdateBoardMeta(path, "", "desc", nil)
	if err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	if board.Name != "Test Board" {
		t.Errorf("name should be preserved, got %q", board.Name)
	}
}

func TestUpdateBoardIcon(t *testing.T) {
	path, eng := setupTestBoard(t)

	if err := eng.UpdateBoardIcon(path, "🚀"); err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	if board.Icon != "🚀" {
		t.Errorf("icon = %q", board.Icon)
	}
}

func TestSortColumnByName(t *testing.T) {
	path, eng := setupMultiCardBoard(t)

	if err := eng.SortColumn(path, 0, "name"); err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	cards := board.Columns[0].Cards
	if cards[0].Title != "Alpha" || cards[1].Title != "Bravo" || cards[2].Title != "Charlie" {
		t.Errorf("sort by name: got %q, %q, %q", cards[0].Title, cards[1].Title, cards[2].Title)
	}
}

func TestSortColumnByPriority(t *testing.T) {
	path, eng := setupMultiCardBoard(t)

	if err := eng.SortColumn(path, 0, "priority"); err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	cards := board.Columns[0].Cards
	// critical > high > low
	if cards[0].Title != "Bravo" {
		t.Errorf("expected Bravo (critical) first, got %q", cards[0].Title)
	}
	if cards[1].Title != "Alpha" {
		t.Errorf("expected Alpha (high) second, got %q", cards[1].Title)
	}
}

func TestSortColumnByDue(t *testing.T) {
	path, eng := setupMultiCardBoard(t)

	if err := eng.SortColumn(path, 0, "due"); err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	cards := board.Columns[0].Cards
	// 2025-01-15, 2025-02-10, 2025-03-01
	if cards[0].Title != "Charlie" {
		t.Errorf("expected Charlie (earliest due) first, got %q", cards[0].Title)
	}
}

func TestSortColumnInvalidKey(t *testing.T) {
	path, eng := setupMultiCardBoard(t)

	err := eng.SortColumn(path, 0, "invalid")
	if err == nil {
		t.Fatal("expected error for unknown sort key")
	}
}

func TestSortColumnInvalidIndex(t *testing.T) {
	path, eng := setupMultiCardBoard(t)

	err := eng.SortColumn(path, 99, "name")
	if err == nil {
		t.Fatal("expected error for invalid column index")
	}
}

func TestUpdateBoardSettings(t *testing.T) {
	path, eng := setupTestBoard(t)

	showCheckbox := true
	cardPos := "prepend"
	expandCols := true
	settings := models.BoardSettings{
		ShowCheckbox:  &showCheckbox,
		CardPosition:  &cardPos,
		ExpandColumns: &expandCols,
	}

	if err := eng.UpdateBoardSettings(path, settings); err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	if board.Settings.ShowCheckbox == nil || !*board.Settings.ShowCheckbox {
		t.Error("ShowCheckbox should be true")
	}
	if board.Settings.CardPosition == nil || *board.Settings.CardPosition != "prepend" {
		t.Error("CardPosition should be 'prepend'")
	}
	if board.Settings.ExpandColumns == nil || !*board.Settings.ExpandColumns {
		t.Error("ExpandColumns should be true")
	}
}

func TestPriorityRank(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"critical", 4},
		{"Critical", 4},
		{"high", 3},
		{"medium", 2},
		{"low", 1},
		{"unknown", 0},
		{"", 0},
	}
	for _, tc := range cases {
		got := priorityRank(tc.input)
		if got != tc.want {
			t.Errorf("priorityRank(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestValidateIndices(t *testing.T) {
	path, eng := setupTestBoard(t)
	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}

	// Valid indices
	if err := validateIndices(board, 0, 0); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Invalid column
	if err := validateIndices(board, -1, 0); err == nil {
		t.Error("expected error for negative column index")
	}
	if err := validateIndices(board, 99, 0); err == nil {
		t.Error("expected error for out-of-range column index")
	}

	// Invalid card (In Progress has 0 cards)
	if err := validateIndices(board, 1, 0); err == nil {
		t.Error("expected error for out-of-range card index in empty column")
	}
}

func TestAddCardPrepend(t *testing.T) {
	path, eng := setupTestBoard(t)

	_, err := eng.AddCard(path, "Backlog", "Prepended task", true)
	if err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	if board.Columns[0].Cards[0].Title != "Prepended task" {
		t.Errorf("expected prepended task first, got %q", board.Columns[0].Cards[0].Title)
	}
}

func TestAddCardInvalidColumn(t *testing.T) {
	path, eng := setupTestBoard(t)

	_, err := eng.AddCard(path, "Nonexistent", "Fail", false)
	if err == nil {
		t.Fatal("expected error for nonexistent column")
	}
}

func TestMoveCardInvalidTarget(t *testing.T) {
	path, eng := setupTestBoard(t)

	err := eng.MoveCard(path, 0, 0, "Nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent target column")
	}
}

func TestSortColumnDueWithEmptyDates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "due.md")
	content := `---
name: Due Test
---

## Col

- [ ] No Due
- [ ] Has Due
  due: 2025-01-01
- [ ] Also No Due
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	eng := New()

	if err := eng.SortColumn(path, 0, "due"); err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	cards := board.Columns[0].Cards
	// Card with due date should come first, then cards without
	if cards[0].Title != "Has Due" {
		t.Errorf("expected 'Has Due' first, got %q", cards[0].Title)
	}
}
