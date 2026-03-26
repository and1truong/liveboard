package writer

import (
	"strings"
	"testing"

	"github.com/and1truong/liveboard/internal/parser"
	"github.com/and1truong/liveboard/pkg/models"
)

func TestRender(t *testing.T) {
	board := &models.Board{
		Name:        "Test",
		Description: "A test board",
		Columns: []models.Column{
			{
				Name: "Backlog",
				Cards: []models.Card{
					{Title: "Task one", Tags: []string{"backend"}},
				},
			},
			{Name: "Done"},
		},
	}

	content, err := Render(board)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(content, "name: Test") {
		t.Error("missing frontmatter name")
	}
	if !strings.Contains(content, "## Backlog") {
		t.Error("missing Backlog column")
	}
	if !strings.Contains(content, "- [ ] Task one") {
		t.Error("missing card")
	}
	if strings.Contains(content, "liveboard:id") {
		t.Error("should not contain ID comments")
	}
	if !strings.Contains(content, "tags: backend") {
		t.Error("missing tags")
	}
	if !strings.Contains(content, "## Done") {
		t.Error("missing Done column")
	}
}

func TestRoundTrip(t *testing.T) {
	original := `---
name: Roundtrip
---

## Backlog

- [ ] First
  tags: a, b
  priority: high

- [x] Second

## Done
`

	board, err := parser.Parse(original)
	if err != nil {
		t.Fatal(err)
	}

	rendered, err := Render(board)
	if err != nil {
		t.Fatal(err)
	}

	// Re-parse the rendered output.
	board2, err := parser.Parse(rendered)
	if err != nil {
		t.Fatal(err)
	}

	if board2.Name != "Roundtrip" {
		t.Errorf("name = %q", board2.Name)
	}
	if len(board2.Columns) != 2 {
		t.Fatalf("columns = %d", len(board2.Columns))
	}
	if len(board2.Columns[0].Cards) != 2 {
		t.Fatalf("cards = %d", len(board2.Columns[0].Cards))
	}
	if board2.Columns[0].Cards[0].Priority != "high" {
		t.Errorf("priority = %q", board2.Columns[0].Cards[0].Priority)
	}
	if !board2.Columns[0].Cards[1].Completed {
		t.Error("second card should be completed")
	}
}

func TestRoundTripHyphenatedMetadata(t *testing.T) {
	original := `---
name: Hyphens
---

## Todo

- [ ] Task with custom fields
  custom-key: some value
  story-points: 5
`

	board, err := parser.Parse(original)
	if err != nil {
		t.Fatal(err)
	}

	card := board.Columns[0].Cards[0]
	if card.Metadata["custom-key"] != "some value" {
		t.Errorf("custom-key = %q, want %q", card.Metadata["custom-key"], "some value")
	}
	if card.Metadata["story-points"] != "5" {
		t.Errorf("story-points = %q, want %q", card.Metadata["story-points"], "5")
	}

	rendered, err := Render(board)
	if err != nil {
		t.Fatal(err)
	}

	board2, err := parser.Parse(rendered)
	if err != nil {
		t.Fatal(err)
	}

	card2 := board2.Columns[0].Cards[0]
	if card2.Metadata["custom-key"] != "some value" {
		t.Errorf("roundtrip custom-key = %q", card2.Metadata["custom-key"])
	}
	if card2.Metadata["story-points"] != "5" {
		t.Errorf("roundtrip story-points = %q", card2.Metadata["story-points"])
	}
}

func TestRoundTripListCollapse(t *testing.T) {
	original := `---
name: Collapse Test
list-collapse:
    - false
    - false
    - true
---

## Todo

## In Progress

## Done
`

	board, err := parser.Parse(original)
	if err != nil {
		t.Fatal(err)
	}

	if len(board.ListCollapse) != 3 {
		t.Fatalf("list-collapse = %v, want 3 elements", board.ListCollapse)
	}
	if board.ListCollapse[2] != true {
		t.Errorf("list-collapse[2] = %v, want true", board.ListCollapse[2])
	}

	rendered, err := Render(board)
	if err != nil {
		t.Fatal(err)
	}

	board2, err := parser.Parse(rendered)
	if err != nil {
		t.Fatal(err)
	}

	if len(board2.ListCollapse) != 3 {
		t.Fatalf("roundtrip list-collapse = %v, want 3 elements", board2.ListCollapse)
	}
	if board2.ListCollapse[0] != false || board2.ListCollapse[1] != false || board2.ListCollapse[2] != true {
		t.Errorf("roundtrip list-collapse = %v", board2.ListCollapse)
	}
}

func TestRoundTripInlineTags(t *testing.T) {
	original := `---
name: Inline Tags
---

## Todo

- [ ] Fix login bug #urgent #backend
`

	board, err := parser.Parse(original)
	if err != nil {
		t.Fatal(err)
	}

	card := board.Columns[0].Cards[0]
	if card.Title != "Fix login bug" {
		t.Errorf("title = %q, want %q", card.Title, "Fix login bug")
	}
	if len(card.InlineTags) != 2 || card.InlineTags[0] != "urgent" || card.InlineTags[1] != "backend" {
		t.Errorf("inline_tags = %v, want [urgent backend]", card.InlineTags)
	}

	rendered, err := Render(board)
	if err != nil {
		t.Fatal(err)
	}

	// Inline tags should be restored in the title line.
	if !strings.Contains(rendered, "- [ ] Fix login bug #urgent #backend") {
		t.Errorf("inline tags not restored in rendered output:\n%s", rendered)
	}
	// Should NOT have a separate tags: metadata line for inline-only tags.
	if strings.Contains(rendered, "  tags:") {
		t.Errorf("should not have metadata tags line for inline-only tags:\n%s", rendered)
	}

	// Verify roundtrip preserves everything.
	board2, err := parser.Parse(rendered)
	if err != nil {
		t.Fatal(err)
	}
	card2 := board2.Columns[0].Cards[0]
	if card2.Title != "Fix login bug" {
		t.Errorf("roundtrip title = %q", card2.Title)
	}
	if len(card2.Tags) != 2 {
		t.Errorf("roundtrip tags = %v", card2.Tags)
	}
}

func TestRoundTripInlineAndMetadataTags(t *testing.T) {
	original := `---
name: Mixed Tags
---

## Todo

- [ ] Fix login bug #urgent
  tags: backend, api
`

	board, err := parser.Parse(original)
	if err != nil {
		t.Fatal(err)
	}

	card := board.Columns[0].Cards[0]
	// Tags should contain all: inline + metadata
	if len(card.Tags) != 3 {
		t.Fatalf("tags = %v, want 3 tags", card.Tags)
	}
	if len(card.InlineTags) != 1 || card.InlineTags[0] != "urgent" {
		t.Errorf("inline_tags = %v, want [urgent]", card.InlineTags)
	}

	rendered, err := Render(board)
	if err != nil {
		t.Fatal(err)
	}

	// Inline tag in title, metadata tags in tags: line
	if !strings.Contains(rendered, "#urgent") {
		t.Errorf("inline tag not in title:\n%s", rendered)
	}
	if !strings.Contains(rendered, "  tags: backend, api") {
		t.Errorf("metadata tags missing:\n%s", rendered)
	}
}

func TestRoundTripPlainListItem(t *testing.T) {
	original := `---
name: Plain Items
---

## Todo

- Plain task without checkbox
  priority: low
`

	board, err := parser.Parse(original)
	if err != nil {
		t.Fatal(err)
	}

	card := board.Columns[0].Cards[0]
	if card.Title != "Plain task without checkbox" {
		t.Errorf("title = %q", card.Title)
	}
	if !card.NoCheckbox {
		t.Error("expected NoCheckbox = true for plain list item")
	}
	if card.Priority != "low" {
		t.Errorf("priority = %q", card.Priority)
	}

	rendered, err := Render(board)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(rendered, "- Plain task without checkbox\n") {
		t.Errorf("plain item not rendered correctly:\n%s", rendered)
	}
	if strings.Contains(rendered, "- [ ]") || strings.Contains(rendered, "- [x]") {
		t.Errorf("plain item should not have checkbox:\n%s", rendered)
	}

	// Roundtrip
	board2, err := parser.Parse(rendered)
	if err != nil {
		t.Fatal(err)
	}
	card2 := board2.Columns[0].Cards[0]
	if card2.Title != "Plain task without checkbox" {
		t.Errorf("roundtrip title = %q", card2.Title)
	}
	if !card2.NoCheckbox {
		t.Error("roundtrip NoCheckbox should be true")
	}
	if card2.Priority != "low" {
		t.Errorf("roundtrip priority = %q", card2.Priority)
	}
}

func TestRoundTripEmptyMetadataValue(t *testing.T) {
	original := `---
name: Empty Meta
---

## Todo

- [ ] Task
  note:
`

	board, err := parser.Parse(original)
	if err != nil {
		t.Fatal(err)
	}

	card := board.Columns[0].Cards[0]
	if card.Metadata == nil || card.Metadata["note"] != "" {
		t.Errorf("metadata[note] = %q, want empty string", card.Metadata["note"])
	}

	rendered, err := Render(board)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(rendered, "  note: \n") && !strings.Contains(rendered, "  note:\n") {
		t.Errorf("empty metadata not rendered:\n%s", rendered)
	}
}

func TestMetadataOrderDeterministic(t *testing.T) {
	board := &models.Board{
		Name: "Deterministic",
		Columns: []models.Column{
			{
				Name: "Todo",
				Cards: []models.Card{
					{
						Title: "Task",
						Metadata: map[string]string{
							"zebra":    "1",
							"alpha":    "2",
							"middle":   "3",
							"beta":     "4",
							"xenon":    "5",
							"category": "6",
						},
					},
				},
			},
		},
	}

	// Render multiple times and ensure output is identical.
	var first string
	for i := 0; i < 20; i++ {
		rendered, err := Render(board)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = rendered
		} else if rendered != first {
			t.Fatalf("non-deterministic output on iteration %d", i)
		}
	}

	// Verify alphabetical order.
	idx := func(key string) int { return strings.Index(first, "  "+key+":") }
	if idx("alpha") > idx("beta") || idx("beta") > idx("category") || idx("category") > idx("middle") {
		t.Error("metadata keys not in alphabetical order")
	}
}

func TestPlainItemDefaultCheckbox(t *testing.T) {
	// Programmatically created cards (NoCheckbox=false) should still render with checkbox.
	board := &models.Board{
		Name: "Default",
		Columns: []models.Column{
			{
				Name: "Todo",
				Cards: []models.Card{
					{Title: "Normal card"},
				},
			},
		},
	}

	rendered, err := Render(board)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(rendered, "- [ ] Normal card") {
		t.Errorf("default card should have checkbox:\n%s", rendered)
	}
}
