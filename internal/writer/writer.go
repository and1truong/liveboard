// Package writer renders Board models to Markdown.
package writer

import (
	"fmt"
	"sort"
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
		Members      []string             `yaml:"members,omitempty"`
		ListCollapse []bool               `yaml:"list-collapse,omitempty"`
		Settings     models.BoardSettings `yaml:"settings,omitempty"`
	}{
		Version:      board.Version,
		Name:         board.Name,
		Description:  board.Description,
		Icon:         board.Icon,
		Tags:         board.Tags,
		TagColors:    board.TagColors,
		Members:      board.Members,
		ListCollapse: board.ListCollapse,
		Settings:     board.Settings,
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
	// Build title with inline tags restored.
	var title string
	if len(card.InlineTags) > 0 {
		var tb strings.Builder
		tb.WriteString(card.Title)
		for _, t := range card.InlineTags {
			tb.WriteString(" #")
			tb.WriteString(t)
		}
		title = tb.String()
	} else {
		title = card.Title
	}

	if card.NoCheckbox {
		fmt.Fprintf(b, "- %s\n", title)
	} else {
		checkbox := " "
		if card.Completed {
			checkbox = "x"
		}
		fmt.Fprintf(b, "- [%s] %s\n", checkbox, title)
	}

	if card.ID != "" {
		fmt.Fprintf(b, "  id: %s\n", card.ID)
	}

	// Write metadata-only tags (exclude inline tags already in title).
	metaTags := metadataOnlyTags(card.Tags, card.InlineTags)
	if len(metaTags) > 0 {
		b.WriteString("  tags: " + strings.Join(metaTags, ", ") + "\n")
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

	// Sort metadata keys for deterministic output.
	if len(card.Metadata) > 0 {
		keys := make([]string, 0, len(card.Metadata))
		for k := range card.Metadata {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(b, "  %s: %s\n", k, card.Metadata[k])
		}
	}

	if card.Body != "" {
		for _, line := range strings.Split(card.Body, "\n") {
			b.WriteString("  " + line + "\n")
		}
	}
}

// metadataOnlyTags returns tags that are NOT in the inline set.
func metadataOnlyTags(all, inline []string) []string {
	if len(inline) == 0 {
		return all
	}
	inlineSet := make(map[string]struct{}, len(inline))
	for _, t := range inline {
		inlineSet[t] = struct{}{}
	}
	var out []string
	for _, t := range all {
		if _, ok := inlineSet[t]; !ok {
			out = append(out, t)
		}
	}
	return out
}
