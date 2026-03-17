package parser

import (
	"testing"
)

const sampleBoard = `---
name: Product Roadmap
description: Planning upcoming features
tags: [product, roadmap]
---

## Backlog

- [ ] Add OAuth login
<!-- liveboard:id=id-001 -->
  tags: auth, backend
  priority: high

- [ ] Build mobile layout
<!-- liveboard:id=id-002 -->
  tags: ui

## In Progress

- [ ] Implement billing integration
<!-- liveboard:id=id-003 -->
  tags: payments
  assignee: hong

## Done

- [x] Create landing page
<!-- liveboard:id=id-004 -->
`

func TestParseFrontmatter(t *testing.T) {
	board, err := Parse(sampleBoard)
	if err != nil {
		t.Fatal(err)
	}
	if board.Name != "Product Roadmap" {
		t.Errorf("name = %q, want %q", board.Name, "Product Roadmap")
	}
	if board.Description != "Planning upcoming features" {
		t.Errorf("description = %q", board.Description)
	}
	if len(board.Tags) != 2 || board.Tags[0] != "product" {
		t.Errorf("tags = %v", board.Tags)
	}
}

func TestParseColumns(t *testing.T) {
	board, err := Parse(sampleBoard)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Columns) != 3 {
		t.Fatalf("columns = %d, want 3", len(board.Columns))
	}
	names := []string{"Backlog", "In Progress", "Done"}
	for i, col := range board.Columns {
		if col.Name != names[i] {
			t.Errorf("column[%d] = %q, want %q", i, col.Name, names[i])
		}
	}
}

func TestParseCards(t *testing.T) {
	board, err := Parse(sampleBoard)
	if err != nil {
		t.Fatal(err)
	}

	// Backlog should have 2 cards.
	backlog := board.Columns[0]
	if len(backlog.Cards) != 2 {
		t.Fatalf("backlog cards = %d, want 2", len(backlog.Cards))
	}

	card := backlog.Cards[0]
	if card.ID != "id-001" {
		t.Errorf("card id = %q, want %q", card.ID, "id-001")
	}
	if card.Title != "Add OAuth login" {
		t.Errorf("card title = %q", card.Title)
	}
	if card.Completed {
		t.Error("card should not be completed")
	}
	if card.Priority != "high" {
		t.Errorf("priority = %q", card.Priority)
	}
	if len(card.Tags) != 2 {
		t.Errorf("tags = %v, want [auth backend]", card.Tags)
	}

	// In Progress should have 1 card with assignee.
	inprog := board.Columns[1]
	if len(inprog.Cards) != 1 {
		t.Fatalf("in_progress cards = %d", len(inprog.Cards))
	}
	if inprog.Cards[0].Assignee != "hong" {
		t.Errorf("assignee = %q", inprog.Cards[0].Assignee)
	}

	// Done should have 1 completed card.
	done := board.Columns[2]
	if len(done.Cards) != 1 {
		t.Fatalf("done cards = %d", len(done.Cards))
	}
	if !done.Cards[0].Completed {
		t.Error("done card should be completed")
	}
}

func TestParseCardBody(t *testing.T) {
	md := `## Backlog

- [ ] Task with body
<!-- liveboard:id=id-body -->
  tags: important

  First line of body.
  Second line of body.

- [ ] Task without body
<!-- liveboard:id=id-nobody -->
`
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Columns[0].Cards) != 2 {
		t.Fatalf("cards = %d, want 2", len(board.Columns[0].Cards))
	}
	withBody := board.Columns[0].Cards[0]
	if withBody.Body != "First line of body.\nSecond line of body." {
		t.Errorf("body = %q", withBody.Body)
	}
	withoutBody := board.Columns[0].Cards[1]
	if withoutBody.Body != "" {
		t.Errorf("expected empty body, got %q", withoutBody.Body)
	}
}

func TestParseMinimalBoard(t *testing.T) {
	md := "## Todo\n\n- [ ] First task\n<!-- liveboard:id=abc -->\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Columns) != 1 {
		t.Fatalf("columns = %d", len(board.Columns))
	}
	if len(board.Columns[0].Cards) != 1 {
		t.Fatalf("cards = %d", len(board.Columns[0].Cards))
	}
	if board.Columns[0].Cards[0].ID != "abc" {
		t.Errorf("id = %q", board.Columns[0].Cards[0].ID)
	}
}
