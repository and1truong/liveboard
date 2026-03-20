package search

import (
	"testing"

	"github.com/and1truong/liveboard/pkg/models"
)

func testBoard() *models.Board {
	return &models.Board{
		Name: "Test Board",
		Columns: []models.Column{
			{
				Name: "To Do",
				Cards: []models.Card{
					{Title: "Fix login bug", Body: "Users cannot login with SSO", Tags: []string{"bug", "auth"}, Priority: "high"},
					{Title: "Add dark mode", Body: "Implement theme toggle", Tags: []string{"feature", "ui"}},
					{Title: "Write unit tests", Body: "Increase coverage to 80%", Tags: []string{"test"}, Assignee: "alice"},
				},
			},
			{
				Name: "Done",
				Cards: []models.Card{
					{Title: "Setup CI pipeline", Body: "GitHub Actions workflow", Tags: []string{"devops"}},
				},
			},
		},
	}
}

func TestNewIndex(t *testing.T) {
	idx, err := NewIndex("")
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	defer idx.Close()
}

func TestNewIndexWithLanguage(t *testing.T) {
	for _, lang := range []string{"en", "fr", "de", "es", "cjk", "ar", "ru"} {
		idx, err := NewIndex(lang)
		if err != nil {
			t.Fatalf("NewIndex(%q): %v", lang, err)
		}
		idx.Close()
	}
}

func TestIndexAndSearch(t *testing.T) {
	idx, err := NewIndex("")
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	defer idx.Close()

	board := testBoard()
	if err := idx.IndexBoard("test-board", board); err != nil {
		t.Fatalf("IndexBoard: %v", err)
	}

	results, err := idx.Search("login bug", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result for 'login bug'")
	}

	// Top result should be the login bug card
	if results[0].CardTitle != "Fix login bug" {
		t.Errorf("expected top result 'Fix login bug', got %q", results[0].CardTitle)
	}
	if results[0].BoardSlug != "test-board" {
		t.Errorf("expected board_slug 'test-board', got %q", results[0].BoardSlug)
	}
	if results[0].ColumnName != "To Do" {
		t.Errorf("expected column 'To Do', got %q", results[0].ColumnName)
	}
}

func TestSearchBody(t *testing.T) {
	idx, err := NewIndex("")
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	defer idx.Close()

	board := testBoard()
	if err := idx.IndexBoard("test-board", board); err != nil {
		t.Fatalf("IndexBoard: %v", err)
	}

	results, err := idx.Search("coverage", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected result for 'coverage' (in card body)")
	}
	if results[0].CardTitle != "Write unit tests" {
		t.Errorf("expected 'Write unit tests', got %q", results[0].CardTitle)
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	idx, err := NewIndex("")
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	defer idx.Close()

	results, err := idx.Search("", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if results != nil {
		t.Error("expected nil results for empty query")
	}
}

func TestRemoveBoard(t *testing.T) {
	idx, err := NewIndex("")
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	defer idx.Close()

	board := testBoard()
	if err := idx.IndexBoard("test-board", board); err != nil {
		t.Fatalf("IndexBoard: %v", err)
	}

	// Verify cards are indexed
	results, err := idx.Search("login", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results before removal")
	}

	// Remove and verify
	if err := idx.RemoveBoard("test-board"); err != nil {
		t.Fatalf("RemoveBoard: %v", err)
	}

	results, err = idx.Search("login", 10)
	if err != nil {
		t.Fatalf("Search after remove: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results after removal, got %d", len(results))
	}
}

func TestReindexBoard(t *testing.T) {
	idx, err := NewIndex("")
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	defer idx.Close()

	board := testBoard()
	if err := idx.IndexBoard("test-board", board); err != nil {
		t.Fatalf("IndexBoard: %v", err)
	}

	// Modify board and reindex
	board.Columns[0].Cards[0].Title = "Fix signup bug"
	if err := idx.RemoveBoard("test-board"); err != nil {
		t.Fatalf("RemoveBoard: %v", err)
	}
	if err := idx.IndexBoard("test-board", board); err != nil {
		t.Fatalf("IndexBoard (reindex): %v", err)
	}

	// Old title should not match
	results, err := idx.Search("login", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results for old title 'login', got %d", len(results))
	}

	// New title should match
	results, err = idx.Search("signup", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for new title 'signup'")
	}
}

func TestMultilingualSearch(t *testing.T) {
	// Test with CJK content using default multilingual analyzer
	idx, err := NewIndex("")
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	defer idx.Close()

	board := &models.Board{
		Name: "Multilingual Board",
		Columns: []models.Column{
			{
				Name: "Tasks",
				Cards: []models.Card{
					{Title: "ログインバグを修正", Body: "SSOでログインできない"},
					{Title: "Corriger le bug de connexion", Body: "Les utilisateurs ne peuvent pas se connecter"},
					{Title: "Fehler bei der Anmeldung beheben", Body: "Benutzer können sich nicht anmelden"},
				},
			},
		},
	}

	if err := idx.IndexBoard("multilingual", board); err != nil {
		t.Fatalf("IndexBoard: %v", err)
	}

	// Search for Japanese content
	results, err := idx.Search("ログイン", 10)
	if err != nil {
		t.Fatalf("Search (Japanese): %v", err)
	}
	if len(results) == 0 {
		t.Error("expected results for Japanese query 'ログイン'")
	}

	// Search for French content
	results, err = idx.Search("connexion", 10)
	if err != nil {
		t.Fatalf("Search (French): %v", err)
	}
	if len(results) == 0 {
		t.Error("expected results for French query 'connexion'")
	}

	// Search for German content
	results, err = idx.Search("Anmeldung", 10)
	if err != nil {
		t.Fatalf("Search (German): %v", err)
	}
	if len(results) == 0 {
		t.Error("expected results for German query 'Anmeldung'")
	}
}

func TestSearchResultFields(t *testing.T) {
	idx, err := NewIndex("")
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	defer idx.Close()

	board := testBoard()
	if err := idx.IndexBoard("my-board", board); err != nil {
		t.Fatalf("IndexBoard: %v", err)
	}

	results, err := idx.Search("CI pipeline", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected results for 'CI pipeline'")
	}

	r := results[0]
	if r.BoardSlug != "my-board" {
		t.Errorf("BoardSlug = %q, want 'my-board'", r.BoardSlug)
	}
	if r.BoardName != "Test Board" {
		t.Errorf("BoardName = %q, want 'Test Board'", r.BoardName)
	}
	if r.ColumnName != "Done" {
		t.Errorf("ColumnName = %q, want 'Done'", r.ColumnName)
	}
	if r.Score <= 0 {
		t.Error("expected positive score")
	}
}
