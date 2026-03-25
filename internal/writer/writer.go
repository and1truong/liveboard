// Package writer renders Board models to Markdown.
package writer

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/and1truong/liveboard/pkg/models"
)

// Render converts a Board model into a Markdown string.
func Render(board *models.Board) (string, error) {
	var b strings.Builder

	// Write YAML frontmatter.
	fm := struct {
		Version      int                  `yaml:"version"`
		Name         string               `yaml:"name"`
		Description  string               `yaml:"description,omitempty"`
		Icon         string               `yaml:"icon,omitempty"`
		Tags         []string             `yaml:"tags,omitempty"`
		TagColors    map[string]string    `yaml:"tag-colors,omitempty"`
		Members  []string             `yaml:"members,omitempty"`
		Settings models.BoardSettings `yaml:"settings,omitempty"`
	}{
		Version:      board.Version,
		Name:         board.Name,
		Description:  board.Description,
		Icon:         board.Icon,
		Tags:         board.Tags,
		TagColors:    board.TagColors,
		Members:      board.Members,
		Settings: board.Settings,
	}
	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("marshal frontmatter: %w", err)
	}
	b.WriteString("---\n")
	b.Write(fmBytes)
	b.WriteString("---\n")

	// Write columns and cards.
	for _, col := range board.Columns {
		b.WriteString("\n## " + col.Name + "\n")

		for _, card := range col.Cards {
			b.WriteByte('\n')
			writeCard(&b, &card)
		}
	}

	return b.String(), nil
}

func writeCard(b *strings.Builder, card *models.Card) {
	checkbox := " "
	if card.Completed {
		checkbox = "x"
	}
	fmt.Fprintf(b, "- [%s] %s\n", checkbox, card.Title)

	if len(card.Tags) > 0 {
		b.WriteString("  tags: " + strings.Join(card.Tags, ", ") + "\n")
	}
	if card.Assignee != "" {
		b.WriteString("  assignee: " + card.Assignee + "\n")
	}
	if card.Priority != "" {
		b.WriteString("  priority: " + card.Priority + "\n")
	}
	if card.Due != "" {
		b.WriteString("  due: " + card.Due + "\n")
	}
	for k, v := range card.Metadata {
		fmt.Fprintf(b, "  %s: %s\n", k, v)
	}
	if card.Body != "" {
		for _, line := range strings.Split(card.Body, "\n") {
			b.WriteString("  " + line + "\n")
		}
	}
}
