package parser

import (
	"reflect"
	"testing"

	"github.com/and1truong/liveboard/internal/writer"
	"github.com/and1truong/liveboard/pkg/models"
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

// --- Edge case tests appended below ---

func TestParseSummaryBasic(t *testing.T) {
	info, err := ParseSummary(sampleBoard)
	if err != nil {
		t.Fatal(err)
	}
	if info.Board.Name != "Product Roadmap" {
		t.Errorf("name = %q", info.Board.Name)
	}
	if info.ColumnCount != 3 {
		t.Errorf("columns = %d, want 3", info.ColumnCount)
	}
	if info.CardCount != 4 {
		t.Errorf("cards = %d, want 4", info.CardCount)
	}
	if info.DoneCount != 1 {
		t.Errorf("done = %d, want 1", info.DoneCount)
	}
}

func TestParseSummaryInvalidYAML(t *testing.T) {
	md := "---\nname: [broken\n---\n\n## Col\n"
	_, err := ParseSummary(md)
	if err == nil {
		t.Fatal("expected error for invalid YAML in ParseSummary")
	}
}

func TestParseSummaryNoFrontmatter(t *testing.T) {
	md := "## Col\n\n- [x] Done\n- [ ] Open\n"
	info, err := ParseSummary(md)
	if err != nil {
		t.Fatal(err)
	}
	if info.ColumnCount != 1 {
		t.Errorf("columns = %d, want 1", info.ColumnCount)
	}
	if info.CardCount != 2 {
		t.Errorf("cards = %d, want 2", info.CardCount)
	}
	if info.DoneCount != 1 {
		t.Errorf("done = %d, want 1", info.DoneCount)
	}
}

func TestParseSummaryUppercaseX(t *testing.T) {
	md := "## Col\n\n- [X] Done with uppercase\n"
	info, err := ParseSummary(md)
	if err != nil {
		t.Fatal(err)
	}
	if info.DoneCount != 1 {
		t.Errorf("done = %d, want 1", info.DoneCount)
	}
}

func TestParseEmptyBoardFrontmatterOnly(t *testing.T) {
	md := "---\nname: Empty Board\ndescription: no columns\n---\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	if board.Name != "Empty Board" {
		t.Errorf("name = %q", board.Name)
	}
	if len(board.Columns) != 0 {
		t.Errorf("columns = %d, want 0", len(board.Columns))
	}
}

func TestParseColumnsWithNoCards(t *testing.T) {
	md := "---\nname: Sparse\n---\n\n## Empty Column\n\n## Another Empty\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Columns) != 2 {
		t.Fatalf("columns = %d, want 2", len(board.Columns))
	}
	if len(board.Columns[0].Cards) != 0 {
		t.Errorf("first col cards = %d, want 0", len(board.Columns[0].Cards))
	}
	if len(board.Columns[1].Cards) != 0 {
		t.Errorf("second col cards = %d, want 0", len(board.Columns[1].Cards))
	}
}

func TestParseUppercaseXCheckbox(t *testing.T) {
	md := "## Done\n\n- [X] Completed with uppercase X\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	card := board.Columns[0].Cards[0]
	if !card.Completed {
		t.Error("card with [X] should be completed")
	}
}

func TestParseUnicodeInTitleAndMetadata(t *testing.T) {
	md := "## 🚀 Sprint\n\n- [ ] Lörem ïpsum 日本語タスク\n  assignee: José García\n  priority: high\n  custom-note: données résumé\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	if board.Columns[0].Name != "🚀 Sprint" {
		t.Errorf("col name = %q", board.Columns[0].Name)
	}
	card := board.Columns[0].Cards[0]
	if card.Title != "Lörem ïpsum 日本語タスク" {
		t.Errorf("title = %q", card.Title)
	}
	if card.Assignee != "José García" {
		t.Errorf("assignee = %q", card.Assignee)
	}
	if card.Metadata["custom-note"] != "données résumé" {
		t.Errorf("custom-note = %q", card.Metadata["custom-note"])
	}
}

func TestParseSpecialCharsInTitle(t *testing.T) {
	md := "## Todo\n\n- [ ] Fix \"quotes\" and [brackets] & <angles>\n- Plain item with (parens) + {braces}\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	cards := board.Columns[0].Cards
	if len(cards) != 2 {
		t.Fatalf("cards = %d, want 2", len(cards))
	}
	if cards[0].Title != `Fix "quotes" and [brackets] & <angles>` {
		t.Errorf("title = %q", cards[0].Title)
	}
	if cards[1].Title != "Plain item with (parens) + {braces}" {
		t.Errorf("title = %q", cards[1].Title)
	}
}

func TestParseNoFrontmatter(t *testing.T) {
	md := "## Col A\n\n- [ ] Task one\n\n## Col B\n\n- [ ] Task two\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	if board.Name != "" {
		t.Errorf("name should be empty, got %q", board.Name)
	}
	if len(board.Columns) != 2 {
		t.Fatalf("columns = %d, want 2", len(board.Columns))
	}
	if len(board.Columns[0].Cards) != 1 || len(board.Columns[1].Cards) != 1 {
		t.Errorf("expected 1 card in each column")
	}
}

func TestParseCardBeforeAnyColumn(t *testing.T) {
	// A card defined before any column heading gets attached to the first
	// column encountered (flushed when the next card is parsed).
	md := "- [ ] Orphan card\n\n## Real Column\n\n- [ ] Real card\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Columns) != 1 {
		t.Fatalf("columns = %d, want 1", len(board.Columns))
	}
	// Both cards end up in "Real Column"
	if len(board.Columns[0].Cards) != 2 {
		t.Errorf("cards = %d, want 2", len(board.Columns[0].Cards))
	}
}

func TestParseTagsWithEmptyEntries(t *testing.T) {
	md := "## Todo\n\n- [ ] Task\n  tags: a,,b, ,c\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	card := board.Columns[0].Cards[0]
	// Empty entries should be filtered out
	for _, tag := range card.Tags {
		if tag == "" {
			t.Error("empty tag should not be present")
		}
	}
	if len(card.Tags) != 3 {
		t.Errorf("tags = %v, want [a b c]", card.Tags)
	}
}

func TestParseCollapseFewerThanColumns(t *testing.T) {
	md := "---\nname: Test\nlist-collapse:\n    - true\n---\n\n## A\n\n## B\n\n## C\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Columns) != 3 {
		t.Fatalf("columns = %d", len(board.Columns))
	}
	if !board.Columns[0].Collapsed {
		t.Error("col A should be collapsed")
	}
	if board.Columns[1].Collapsed {
		t.Error("col B should not be collapsed (no collapse entry)")
	}
	if board.Columns[2].Collapsed {
		t.Error("col C should not be collapsed (no collapse entry)")
	}
}

func TestParseNonIndentedLineBreaksCard(t *testing.T) {
	// A non-indented, non-card, non-heading line after card metadata
	md := "## Todo\n\n- [ ] Task\n  priority: high\nsome random line\n- [ ] Next task\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	cards := board.Columns[0].Cards
	if len(cards) != 2 {
		t.Fatalf("cards = %d, want 2", len(cards))
	}
	if cards[0].Priority != "high" {
		t.Errorf("first card priority = %q", cards[0].Priority)
	}
}

func TestParseSummaryFrontmatterOnly(t *testing.T) {
	md := "---\nname: Just Frontmatter\n---\n"
	info, err := ParseSummary(md)
	if err != nil {
		t.Fatal(err)
	}
	if info.Board.Name != "Just Frontmatter" {
		t.Errorf("name = %q", info.Board.Name)
	}
	if info.ColumnCount != 0 || info.CardCount != 0 || info.DoneCount != 0 {
		t.Errorf("counts should all be 0: col=%d card=%d done=%d", info.ColumnCount, info.CardCount, info.DoneCount)
	}
}

func TestParseMultipleCardsFlushBetweenColumns(t *testing.T) {
	// Ensure the last card in col A is flushed when col B starts
	md := "## A\n\n- [ ] Card in A\n  assignee: alice\n\n## B\n\n- [ ] Card in B\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Columns[0].Cards) != 1 {
		t.Fatalf("col A cards = %d, want 1", len(board.Columns[0].Cards))
	}
	if board.Columns[0].Cards[0].Assignee != "alice" {
		t.Errorf("col A card assignee = %q", board.Columns[0].Cards[0].Assignee)
	}
	if len(board.Columns[1].Cards) != 1 {
		t.Fatalf("col B cards = %d, want 1", len(board.Columns[1].Cards))
	}
}

func TestParseInlineTagsMergedWithMetaTags(t *testing.T) {
	md := "## Todo\n\n- [ ] Task #inline-tag\n  tags: meta-tag\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	card := board.Columns[0].Cards[0]
	if len(card.Tags) != 2 {
		t.Fatalf("tags = %v, want [inline-tag meta-tag]", card.Tags)
	}
	if card.Tags[0] != "inline-tag" || card.Tags[1] != "meta-tag" {
		t.Errorf("tags = %v", card.Tags)
	}
	if len(card.InlineTags) != 1 || card.InlineTags[0] != "inline-tag" {
		t.Errorf("inline_tags = %v", card.InlineTags)
	}
}

func TestParseEmptyContent(t *testing.T) {
	board, err := Parse("")
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Columns) != 0 {
		t.Errorf("columns = %d, want 0", len(board.Columns))
	}
}

func TestParseSummaryEmptyContent(t *testing.T) {
	info, err := ParseSummary("")
	if err != nil {
		t.Fatal(err)
	}
	if info.ColumnCount != 0 || info.CardCount != 0 {
		t.Errorf("expected all zeros")
	}
}

func TestParseHTMLCommentIndented(t *testing.T) {
	// HTML comment with leading whitespace should still be skipped
	md := "## Todo\n\n- [ ] Task\n  <!-- some comment -->\n  priority: high\n"
	board, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	card := board.Columns[0].Cards[0]
	if card.Priority != "high" {
		t.Errorf("priority = %q, want high", card.Priority)
	}
}

func TestParseCardID(t *testing.T) {
	md := "---\nversion: 1\nname: B\n---\n\n## Todo\n\n- [ ] Card\n  id: aBc1234XyZ\n"
	b, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Columns) != 1 || len(b.Columns[0].Cards) != 1 {
		t.Fatalf("unexpected structure: %+v", b)
	}
	if got := b.Columns[0].Cards[0].ID; got != "aBc1234XyZ" {
		t.Fatalf("want id %q, got %q", "aBc1234XyZ", got)
	}
}

func TestParseCardIDAbsent(t *testing.T) {
	md := "---\nversion: 1\nname: B\n---\n\n## Todo\n\n- [ ] Card\n"
	b, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	if got := b.Columns[0].Cards[0].ID; got != "" {
		t.Fatalf("want empty id, got %q", got)
	}
}

func TestParseCard_Links(t *testing.T) {
	md := "## Todo\n\n- [ ] Card title\n  links: foo:aBc1234XyZ, bar:Q9rT5pZ2nM\n"
	boards, err := Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(boards.Columns) != 1 || len(boards.Columns[0].Cards) != 1 {
		t.Fatal("unexpected structure")
	}
	got := boards.Columns[0].Cards[0].Links
	want := []string{"foo:aBc1234XyZ", "bar:Q9rT5pZ2nM"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("links = %v, want %v", got, want)
	}
}

func TestParseCardAttachments(t *testing.T) {
	md := `---
version: 1
name: T
---

## Col

- [ ] Card
  attachments: [{"h":"a3f9.pdf","n":"Plan.pdf","s":12,"m":"application/pdf"}]
`
	b, err := Parse(md)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(b.Columns) != 1 || len(b.Columns[0].Cards) != 1 {
		t.Fatalf("shape: %+v", b)
	}
	c := b.Columns[0].Cards[0]
	if len(c.Attachments) != 1 {
		t.Fatalf("attachments len: %d", len(c.Attachments))
	}
	got := c.Attachments[0]
	want := models.Attachment{Hash: "a3f9.pdf", Name: "Plan.pdf", Size: 12, Mime: "application/pdf"}
	if got != want {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestParseCardAttachmentsMalformed(t *testing.T) {
	md := `---
version: 1
name: T
---

## Col

- [ ] Card
  attachments: not-json
`
	b, err := Parse(md)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(b.Columns[0].Cards[0].Attachments) != 0 {
		t.Errorf("expected empty attachments on malformed input")
	}
}

func TestRoundtripCardAttachments(t *testing.T) {
	original := &models.Board{
		Version: 1,
		Name:    "T",
		Columns: []models.Column{{
			Name: "Col",
			Cards: []models.Card{{
				Title: "Card",
				Attachments: []models.Attachment{
					{Hash: "a3f9.pdf", Name: "Plan.pdf", Size: 12, Mime: "application/pdf"},
					{Hash: "7c2b.png", Name: "shot.png", Size: 88, Mime: "image/png"},
				},
			}},
		}},
	}
	rendered, err := writer.Render(original)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	parsed, err := Parse(rendered)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got := parsed.Columns[0].Cards[0].Attachments
	if !reflect.DeepEqual(got, original.Columns[0].Cards[0].Attachments) {
		t.Errorf("roundtrip mismatch:\n got:  %+v\n want: %+v\n rendered:\n%s",
			got, original.Columns[0].Cards[0].Attachments, rendered)
	}
}
