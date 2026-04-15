package search_test

import (
	"testing"

	"github.com/and1truong/liveboard/internal/search"
	"github.com/and1truong/liveboard/pkg/models"
)

func newBoard(name string, columns ...models.Column) *models.Board {
	return &models.Board{Name: name, Version: 1, Columns: columns}
}

func col(name string, cards ...models.Card) models.Column {
	return models.Column{Name: name, Cards: cards}
}

func card(title, body string, tags ...string) models.Card {
	return models.Card{Title: title, Body: body, Tags: tags}
}

func mustNew(t *testing.T) *search.Index {
	t.Helper()
	idx, err := search.New()
	if err != nil {
		t.Fatal(err)
	}
	if idx == nil {
		t.Fatal("nil index")
	}
	return idx
}

func TestSearch_BuildAndQuery(t *testing.T) {
	idx := mustNew(t)
	b := newBoard("Welcome", col("Todo", card("Read the docs", "see the wiki", "docs")))
	if err := idx.UpdateBoard("welcome", b); err != nil {
		t.Fatal(err)
	}

	hits, err := idx.Search("docs", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected at least 1 hit")
	}
	h := hits[0]
	if h.BoardID != "welcome" {
		t.Errorf("board_id = %q", h.BoardID)
	}
	if h.CardIdx != 0 || h.ColIdx != 0 {
		t.Errorf("indices = (%d,%d)", h.ColIdx, h.CardIdx)
	}
	if h.CardTitle != "Read the docs" {
		t.Errorf("title = %q", h.CardTitle)
	}
}

func TestSearch_UpdateReplaces(t *testing.T) {
	idx := mustNew(t)
	_ = idx.UpdateBoard("foo", newBoard("Foo", col("Todo", card("alpha", ""))))
	_ = idx.UpdateBoard("foo", newBoard("Foo", col("Todo", card("bravo", ""))))
	if hits, _ := idx.Search("alpha", 10); len(hits) != 0 {
		t.Errorf("expected old text gone, got %d hits", len(hits))
	}
	if hits, _ := idx.Search("bravo", 10); len(hits) == 0 {
		t.Errorf("expected new text indexed")
	}
}

func TestSearch_DeleteBoard(t *testing.T) {
	idx := mustNew(t)
	_ = idx.UpdateBoard("foo", newBoard("Foo", col("Todo", card("unique-token", ""))))
	_ = idx.DeleteBoard("foo")
	if hits, _ := idx.Search("unique-token", 10); len(hits) != 0 {
		t.Errorf("expected 0 hits after delete, got %d", len(hits))
	}
}

func TestSearch_TwoBoardsCorrectAttribution(t *testing.T) {
	idx := mustNew(t)
	_ = idx.UpdateBoard("a", newBoard("A", col("Todo", card("hello world", ""))))
	_ = idx.UpdateBoard("b", newBoard("B", col("Todo", card("hello there", ""))))
	hits, _ := idx.Search("hello", 10)
	if len(hits) < 2 {
		t.Fatalf("expected 2 hits, got %d", len(hits))
	}
	seen := map[string]bool{}
	for _, h := range hits {
		seen[h.BoardID] = true
	}
	if !seen["a"] || !seen["b"] {
		t.Errorf("expected both boards in hits, got %v", seen)
	}
}
