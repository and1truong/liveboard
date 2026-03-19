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
