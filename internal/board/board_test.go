package board

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
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

func TestMoveColumnPrepend(t *testing.T) {
	path, eng := setupTestBoard(t)

	// Move "Done" to first position (Backlog, In Progress, Done → Done, Backlog, In Progress)
	if err := eng.MoveColumn(path, "Done", ""); err != nil {
		t.Fatal(err)
	}

	board, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(board.Columns))
	}
	if board.Columns[0].Name != "Done" {
		t.Errorf("col 0 = %q, want 'Done'", board.Columns[0].Name)
	}
	if board.Columns[1].Name != "Backlog" {
		t.Errorf("col 1 = %q, want 'Backlog'", board.Columns[1].Name)
	}
	if board.Columns[2].Name != "In Progress" {
		t.Errorf("col 2 = %q, want 'In Progress'", board.Columns[2].Name)
	}
}

func TestMoveColumnNotFound(t *testing.T) {
	path, eng := setupTestBoard(t)

	err := eng.MoveColumn(path, "Nonexistent", "Done")
	if err == nil {
		t.Fatal("expected error for nonexistent column")
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

// --- Optimistic concurrency (MutateBoard) tests ---

func loadBoardOrFail(t *testing.T, eng *Engine, path string) *models.Board {
	t.Helper()
	b, err := eng.LoadBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestMutateBoardVersionIncrement(t *testing.T) {
	path, eng := setupTestBoard(t)

	// Board starts at version 0 (no version field in testBoard).
	b := loadBoardOrFail(t, eng, path)
	if b.Version != 0 {
		t.Fatalf("initial version = %d, want 0", b.Version)
	}

	// Mutate with version check (client knows version 0).
	if err := eng.MutateBoard(path, 0, func(b *models.Board) error {
		b.Name = "Mutated"
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	b = loadBoardOrFail(t, eng, path)
	if b.Version != 1 {
		t.Errorf("version after first mutation = %d, want 1", b.Version)
	}
	if b.Name != "Mutated" {
		t.Errorf("name = %q, want 'Mutated'", b.Name)
	}

	// Second mutation at version 1.
	if err := eng.MutateBoard(path, 1, func(b *models.Board) error {
		b.Name = "Mutated Again"
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	b = loadBoardOrFail(t, eng, path)
	if b.Version != 2 {
		t.Errorf("version after second mutation = %d, want 2", b.Version)
	}
}

func TestMutateBoardVersionConflict(t *testing.T) {
	path, eng := setupTestBoard(t)

	// Advance to version 1.
	if err := eng.MutateBoard(path, 0, func(b *models.Board) error {
		b.Name = "V1"
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Try to mutate with stale version 0 — should get ErrVersionConflict.
	err := eng.MutateBoard(path, 0, func(b *models.Board) error {
		b.Name = "Should Not Apply"
		return nil
	})
	if err != ErrVersionConflict {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}

	// Board should still be at version 1 with name "V1".
	b := loadBoardOrFail(t, eng, path)
	if b.Version != 1 {
		t.Errorf("version = %d, want 1", b.Version)
	}
	if b.Name != "V1" {
		t.Errorf("name = %q, want 'V1'", b.Name)
	}
}

func TestMutateBoardSkipVersionCheck(t *testing.T) {
	path, eng := setupTestBoard(t)

	// Advance to version 1.
	if err := eng.MutateBoard(path, -1, func(b *models.Board) error {
		b.Name = "V1"
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// With clientVersion=-1, version check is skipped even though board is at version 1.
	if err := eng.MutateBoard(path, -1, func(b *models.Board) error {
		b.Name = "Skipped Check"
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	b := loadBoardOrFail(t, eng, path)
	if b.Name != "Skipped Check" {
		t.Errorf("name = %q, want 'Skipped Check'", b.Name)
	}
	if b.Version != 2 {
		t.Errorf("version = %d, want 2", b.Version)
	}
}

func TestMutateBoardBackwardCompatNoVersion(t *testing.T) {
	// Board file without a version field — should default to version 0.
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.md")
	content := `---
name: Legacy Board
---

## Todo

- [ ] Old task
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	eng := New()

	b := loadBoardOrFail(t, eng, path)
	if b.Version != 0 {
		t.Fatalf("legacy board version = %d, want 0", b.Version)
	}

	// Client sends version 0 — should match and succeed.
	if err := eng.MutateBoard(path, 0, func(b *models.Board) error {
		b.Name = "Upgraded"
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	b = loadBoardOrFail(t, eng, path)
	if b.Version != 1 {
		t.Errorf("version = %d, want 1", b.Version)
	}
	if b.Name != "Upgraded" {
		t.Errorf("name = %q", b.Name)
	}

	// Verify version is now in the file.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "version: 1") {
		t.Error("version field not written to file")
	}
}

func TestMutateBoardMutationError(t *testing.T) {
	path, eng := setupTestBoard(t)

	// If the mutation function returns an error, version should not increment.
	err := eng.MutateBoard(path, 0, func(_ *models.Board) error {
		return os.ErrPermission
	})
	if err == nil {
		t.Fatal("expected error")
	}

	b := loadBoardOrFail(t, eng, path)
	if b.Version != 0 {
		t.Errorf("version should remain 0 on failed mutation, got %d", b.Version)
	}
}

func TestMutateBoardWithVersionedFile(t *testing.T) {
	// Board file that already has a version field.
	dir := t.TempDir()
	path := filepath.Join(dir, "versioned.md")
	content := `---
version: 5
name: Versioned Board
---

## Col

- [ ] Item
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	eng := New()

	b := loadBoardOrFail(t, eng, path)
	if b.Version != 5 {
		t.Fatalf("version = %d, want 5", b.Version)
	}

	// Correct version — should succeed.
	if err := eng.MutateBoard(path, 5, func(b *models.Board) error {
		b.Name = "Updated"
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	b = loadBoardOrFail(t, eng, path)
	if b.Version != 6 {
		t.Errorf("version = %d, want 6", b.Version)
	}

	// Wrong version — should conflict.
	err := eng.MutateBoard(path, 5, func(_ *models.Board) error {
		return nil
	})
	if err != ErrVersionConflict {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
}

func TestMoveCardToBoard_HappyPath(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.md")
	dstPath := filepath.Join(dir, "dst.md")

	srcMD := "---\nversion: 3\nname: Src\ntags: [alpha]\nmembers: [alice]\n---\n\n## Todo\n\n- [ ] Task A\n  tags: alpha\n  assignee: alice\n\n## Done\n"
	dstMD := "---\nversion: 7\nname: Dst\ntags: [beta]\nmembers: [bob]\n---\n\n## Inbox\n\n- [ ] Existing\n"

	if err := os.WriteFile(srcPath, []byte(srcMD), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dstPath, []byte(dstMD), 0644); err != nil {
		t.Fatal(err)
	}

	e := New()
	if err := e.MoveCardToBoard(srcPath, 3, 0, 0, dstPath, "Inbox"); err != nil {
		t.Fatalf("MoveCardToBoard: %v", err)
	}

	src, err := e.LoadBoard(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	dst, err := e.LoadBoard(dstPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(src.Columns[0].Cards) != 0 {
		t.Errorf("source Todo should be empty, got %d cards", len(src.Columns[0].Cards))
	}
	if src.Version != 4 {
		t.Errorf("source version = %d, want 4", src.Version)
	}
	if len(dst.Columns[0].Cards) != 2 || dst.Columns[0].Cards[0].Title != "Task A" {
		t.Errorf("dst Inbox = %#v, want [Task A, Existing]", dst.Columns[0].Cards)
	}
	if dst.Version != 8 {
		t.Errorf("dst version = %d, want 8", dst.Version)
	}
}

func TestMoveCardToBoard_MergesTagsAndMembers(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.md")
	dstPath := filepath.Join(dir, "dst.md")
	srcMD := "---\nversion: 1\nname: Src\ntags: [urgent, legal]\nmembers: [carol]\n---\n\n## Todo\n\n- [ ] Task\n  tags: urgent, legal\n  assignee: carol\n"
	dstMD := "---\nversion: 1\nname: Dst\ntags: [urgent]\nmembers: []\n---\n\n## Inbox\n"
	_ = os.WriteFile(srcPath, []byte(srcMD), 0644)
	_ = os.WriteFile(dstPath, []byte(dstMD), 0644)
	e := New()
	if err := e.MoveCardToBoard(srcPath, 1, 0, 0, dstPath, "Inbox"); err != nil {
		t.Fatal(err)
	}
	dst, err := e.LoadBoard(dstPath)
	if err != nil || dst == nil {
		t.Fatalf("LoadBoard: %v", err)
	}
	if !slices.Contains(dst.Tags, "legal") {
		t.Errorf("dst.Tags = %v, want contains legal", dst.Tags)
	}
	if !slices.Contains(dst.Members, "carol") {
		t.Errorf("dst.Members = %v, want contains carol", dst.Members)
	}
}

func TestMoveCardToBoard_TargetColumnNotFound(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.md")
	dstPath := filepath.Join(dir, "dst.md")
	_ = os.WriteFile(srcPath, []byte("---\nversion: 1\nname: S\n---\n\n## Todo\n\n- [ ] T\n"), 0644)
	_ = os.WriteFile(dstPath, []byte("---\nversion: 1\nname: D\n---\n\n## Inbox\n"), 0644)
	e := New()
	err := e.MoveCardToBoard(srcPath, 1, 0, 0, dstPath, "Nope")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	src, err := e.LoadBoard(srcPath)
	if err != nil || src == nil {
		t.Fatalf("LoadBoard: %v", err)
	}
	if len(src.Columns[0].Cards) != 1 {
		t.Error("source should be unchanged after failed target lookup")
	}
}

func TestMoveCardToBoard_SourceVersionConflict(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.md")
	dstPath := filepath.Join(dir, "dst.md")
	_ = os.WriteFile(srcPath, []byte("---\nversion: 5\nname: S\n---\n\n## Todo\n\n- [ ] T\n"), 0644)
	_ = os.WriteFile(dstPath, []byte("---\nversion: 1\nname: D\n---\n\n## Inbox\n"), 0644)
	e := New()
	err := e.MoveCardToBoard(srcPath, 2, 0, 0, dstPath, "Inbox")
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("err = %v, want ErrVersionConflict", err)
	}
	dst, err := e.LoadBoard(dstPath)
	if err != nil || dst == nil {
		t.Fatalf("LoadBoard: %v", err)
	}
	if len(dst.Columns[0].Cards) != 1 {
		t.Error("target should have the card even though source removal failed")
	}
}

func TestMoveCardToBoard_SameBoardRejected(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "b.md")
	_ = os.WriteFile(p, []byte("---\nversion: 1\nname: B\n---\n\n## A\n\n- [ ] T\n\n## B\n"), 0644)
	e := New()
	if err := e.MoveCardToBoard(p, 1, 0, 0, p, "B"); err == nil {
		t.Fatal("expected error for same-board move")
	}
}
