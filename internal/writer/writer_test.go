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
					{ID: "id-1", Title: "Task one", Tags: []string{"backend"}},
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
	if !strings.Contains(content, "<!-- liveboard:id=id-1 -->") {
		t.Error("missing card ID")
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
<!-- liveboard:id=id-1 -->
  tags: a, b
  priority: high

- [x] Second
<!-- liveboard:id=id-2 -->

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
	if board2.Columns[0].Cards[0].ID != "id-1" {
		t.Errorf("card id = %q", board2.Columns[0].Cards[0].ID)
	}
	if board2.Columns[0].Cards[0].Priority != "high" {
		t.Errorf("priority = %q", board2.Columns[0].Cards[0].Priority)
	}
	if !board2.Columns[0].Cards[1].Completed {
		t.Error("second card should be completed")
	}
}

func TestAddCard(t *testing.T) {
	content := "---\nname: Test\n---\n\n## Backlog\n\n- [ ] Existing\n<!-- liveboard:id=id-1 -->\n\n## Done\n"

	card := &models.Card{ID: "id-new", Title: "New task", Tags: []string{"urgent"}}
	result := AddCard(content, "Backlog", card)

	if !strings.Contains(result, "- [ ] New task") {
		t.Error("new card not added")
	}
	if !strings.Contains(result, "<!-- liveboard:id=id-new -->") {
		t.Error("new card ID not added")
	}
	// Existing card should still be there.
	if !strings.Contains(result, "- [ ] Existing") {
		t.Error("existing card missing")
	}
}

func TestRemoveCard(t *testing.T) {
	content := "## Backlog\n\n- [ ] Keep\n<!-- liveboard:id=id-1 -->\n\n- [ ] Remove\n<!-- liveboard:id=id-2 -->\n  tags: x\n\n## Done\n"

	result := RemoveCard(content, "id-2")
	if strings.Contains(result, "Remove") {
		t.Error("removed card still present")
	}
	if !strings.Contains(result, "Keep") {
		t.Error("kept card missing")
	}
}
