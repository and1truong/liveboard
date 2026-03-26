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
  tags: auth, backend
  priority: high

- [ ] Build mobile layout
  tags: ui

## In Progress

- [ ] Implement billing integration
  tags: payments
  assignee: hong

## Done

- [x] Create landing page
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

func TestParseMinimalBoard(t *testing.T) {
	md := "## Todo\n\n- [ ] First task\n"
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
	if board.Columns[0].Cards[0].Title != "First task" {
		t.Errorf("title = %q", board.Columns[0].Cards[0].Title)
	}
}

func TestParseLegacyIDComments(t *testing.T) {
	// Ensure legacy ID comments are gracefully skipped.
	md := "## Todo\n\n- [ ] Task\n<!-- liveboard:id=old-123 -->\n  tags: a\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Columns[0].Cards) != 1 {
		t.Fatalf("cards = %d", len(board.Columns[0].Cards))
	}
	card := board.Columns[0].Cards[0]
	if card.Title != "Task" {
		t.Errorf("title = %q", card.Title)
	}
	if len(card.Tags) != 1 || card.Tags[0] != "a" {
		t.Errorf("tags = %v", card.Tags)
	}
}

func TestParseInvalidYAML(t *testing.T) {
	md := "---\nname: [invalid yaml\n---\n\n## Col\n"
	_, err := Parse(md)
	if err == nil {
		t.Fatal("expected error for invalid YAML frontmatter")
	}
}

func TestParseInlineHashTags(t *testing.T) {
	md := "## Todo\n\n- [ ] Fix login bug #urgent #backend\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	card := board.Columns[0].Cards[0]
	if card.Title != "Fix login bug" {
		t.Errorf("title = %q, want 'Fix login bug'", card.Title)
	}
	if len(card.Tags) != 2 {
		t.Fatalf("tags = %v, want [urgent backend]", card.Tags)
	}
	if card.Tags[0] != "urgent" || card.Tags[1] != "backend" {
		t.Errorf("tags = %v", card.Tags)
	}
	// InlineTags should match.
	if len(card.InlineTags) != 2 || card.InlineTags[0] != "urgent" || card.InlineTags[1] != "backend" {
		t.Errorf("inline_tags = %v, want [urgent backend]", card.InlineTags)
	}
}

func TestParseDueDate(t *testing.T) {
	md := "## Todo\n\n- [ ] Ship v2\n  due: 2025-06-01\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	card := board.Columns[0].Cards[0]
	if card.Due != "2025-06-01" {
		t.Errorf("due = %q, want '2025-06-01'", card.Due)
	}
}

func TestParseCustomMetadata(t *testing.T) {
	md := "## Todo\n\n- [ ] Task\n  estimate: 3h\n  sprint: 42\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	card := board.Columns[0].Cards[0]
	if card.Metadata == nil {
		t.Fatal("expected metadata map to be initialized")
	}
	if card.Metadata["estimate"] != "3h" {
		t.Errorf("estimate = %q", card.Metadata["estimate"])
	}
	if card.Metadata["sprint"] != "42" {
		t.Errorf("sprint = %q", card.Metadata["sprint"])
	}
}

func TestParseCardBody(t *testing.T) {
	md := "## Todo\n\n- [ ] Task with body\n  This is the first line.\n  This is the second line.\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	card := board.Columns[0].Cards[0]
	if card.Body != "This is the first line.\nThis is the second line." {
		t.Errorf("body = %q", card.Body)
	}
}

func TestParseCardBodySingleLine(t *testing.T) {
	md := "## Todo\n\n- [ ] Task\n  Just one body line.\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	card := board.Columns[0].Cards[0]
	if card.Body != "Just one body line." {
		t.Errorf("body = %q", card.Body)
	}
}

func TestParseCardBodyWithMetadata(t *testing.T) {
	// Body lines should be distinguished from metadata lines
	md := "## Todo\n\n- [ ] Task\n  priority: high\n  Some body text here.\n  More body text.\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	card := board.Columns[0].Cards[0]
	if card.Priority != "high" {
		t.Errorf("priority = %q", card.Priority)
	}
	if card.Body != "Some body text here.\nMore body text." {
		t.Errorf("body = %q", card.Body)
	}
}

func TestParseHyphenatedMetadataKeys(t *testing.T) {
	md := "## Todo\n\n- [ ] Task\n  custom-key: some value\n  story-points: 5\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	card := board.Columns[0].Cards[0]
	if card.Metadata == nil {
		t.Fatal("expected metadata map")
	}
	if card.Metadata["custom-key"] != "some value" {
		t.Errorf("custom-key = %q, want %q", card.Metadata["custom-key"], "some value")
	}
	if card.Metadata["story-points"] != "5" {
		t.Errorf("story-points = %q, want %q", card.Metadata["story-points"], "5")
	}
	// Should NOT be in body.
	if card.Body != "" {
		t.Errorf("body should be empty, got %q", card.Body)
	}
}

func TestParseEmptyMetadataValue(t *testing.T) {
	md := "## Todo\n\n- [ ] Task\n  note:\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	card := board.Columns[0].Cards[0]
	if card.Metadata == nil {
		t.Fatal("expected metadata map")
	}
	if v, ok := card.Metadata["note"]; !ok {
		t.Error("note key should exist in metadata")
	} else if v != "" {
		t.Errorf("note = %q, want empty", v)
	}
}

func TestParsePlainListItem(t *testing.T) {
	md := "## Todo\n\n- Plain task\n  priority: low\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Columns[0].Cards) != 1 {
		t.Fatalf("cards = %d, want 1", len(board.Columns[0].Cards))
	}
	card := board.Columns[0].Cards[0]
	if card.Title != "Plain task" {
		t.Errorf("title = %q", card.Title)
	}
	if !card.NoCheckbox {
		t.Error("expected NoCheckbox = true")
	}
	if card.Completed {
		t.Error("plain item should not be completed")
	}
	if card.Priority != "low" {
		t.Errorf("priority = %q", card.Priority)
	}
}

func TestParsePlainListItemWithInlineTags(t *testing.T) {
	md := "## Todo\n\n- Deploy service #ops #infra\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	card := board.Columns[0].Cards[0]
	if card.Title != "Deploy service" {
		t.Errorf("title = %q", card.Title)
	}
	if !card.NoCheckbox {
		t.Error("expected NoCheckbox = true")
	}
	if len(card.Tags) != 2 || card.Tags[0] != "ops" || card.Tags[1] != "infra" {
		t.Errorf("tags = %v", card.Tags)
	}
	if len(card.InlineTags) != 2 {
		t.Errorf("inline_tags = %v", card.InlineTags)
	}
}

func TestParseListCollapse(t *testing.T) {
	md := "---\nname: Test\nlist-collapse:\n    - false\n    - true\n---\n\n## A\n\n## B\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.ListCollapse) != 2 {
		t.Fatalf("list-collapse = %v", board.ListCollapse)
	}
	if board.ListCollapse[0] != false || board.ListCollapse[1] != true {
		t.Errorf("list-collapse = %v", board.ListCollapse)
	}
}

func TestParseMixedCheckboxAndPlain(t *testing.T) {
	md := "## Todo\n\n- [ ] Checkbox task\n\n- Plain task\n\n- [x] Done task\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	cards := board.Columns[0].Cards
	if len(cards) != 3 {
		t.Fatalf("cards = %d, want 3", len(cards))
	}
	if cards[0].NoCheckbox || cards[0].Completed {
		t.Error("first card: expected checkbox, uncompleted")
	}
	if !cards[1].NoCheckbox {
		t.Error("second card: expected plain item")
	}
	if cards[2].NoCheckbox || !cards[2].Completed {
		t.Error("third card: expected checkbox, completed")
	}
}
