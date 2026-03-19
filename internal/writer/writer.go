// Package writer renders Board models to Markdown and performs in-place edits.
package writer

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/and1truong/liveboard/pkg/models"
)

// Render converts a Board model into a Markdown string.
func Render(board *models.Board) (string, error) {
	var b strings.Builder

	// Write YAML frontmatter.
	fm := struct {
		Name         string   `yaml:"name"`
		Description  string   `yaml:"description,omitempty"`
		Tags         []string `yaml:"tags,omitempty"`
		ListCollapse []bool   `yaml:"list-collapse,omitempty"`
	}{
		Name:         board.Name,
		Description:  board.Description,
		Tags:         board.Tags,
		ListCollapse: board.ListCollapse,
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

	if card.ID != "" {
		fmt.Fprintf(b, "<!-- liveboard:id=%s -->\n", card.ID)
	}

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
}

// AddCard inserts a card into the raw Markdown content under the specified column.
// Returns the modified content. This operates on raw text to preserve formatting.
func AddCard(content string, columnName string, card *models.Card) string {
	lines := strings.Split(content, "\n")
	var result []string
	colHeader := "## " + columnName
	inserted := false

	for i := 0; i < len(lines); i++ {
		result = append(result, lines[i])

		if !inserted && strings.TrimSpace(lines[i]) == colHeader {
			// Find the end of this column's cards (next H2 or EOF).
			insertIdx := len(result)
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(lines[j], "## ") {
					break
				}
				result = append(result, lines[j])
				insertIdx = len(result)
				i = j
			}
			// Insert card at end of column.
			var cardLines []string
			cardLines = append(cardLines, renderCardLines(card)...)
			// Insert before the next section.
			result = append(result[:insertIdx], append(cardLines, result[insertIdx:]...)...)
			inserted = true
		}
	}

	return strings.Join(result, "\n")
}

// RemoveCard removes a card by ID from the raw Markdown content.
func RemoveCard(content string, cardID string) string {
	lines := strings.Split(content, "\n")
	var result []string
	skip := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if cardRe.MatchString(line) {
			// Check if next line has matching ID.
			if i+1 < len(lines) && extractID(lines[i+1]) == cardID {
				skip = true
				continue
			}
		}

		if skip {
			trimmed := strings.TrimSpace(line)
			// End of card: next card, next column, or empty line followed by non-metadata.
			if strings.HasPrefix(line, "- [") || strings.HasPrefix(line, "## ") {
				skip = false
				result = append(result, line)
			} else if trimmed == "" || isCardMeta(line) || isIDComment(line) {
				continue
			} else {
				skip = false
				result = append(result, line)
			}
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// MoveCard removes a card from its current position and adds it to the target column.
func MoveCard(content string, cardID string, targetColumn string, card *models.Card) string {
	content = RemoveCard(content, cardID)
	return AddCard(content, targetColumn, card)
}

// InsertCardBefore inserts a card immediately before the card with beforeCardID.
// If beforeCardID is empty, it falls back to appending at the end of targetColumn.
func InsertCardBefore(content string, card *models.Card, beforeCardID, targetColumn string) string {
	if beforeCardID == "" {
		return AddCard(content, targetColumn, card)
	}

	lines := strings.Split(content, "\n")
	var result []string
	inserted := false

	for i := 0; i < len(lines); i++ {
		if !inserted && cardRe.MatchString(lines[i]) {
			if i+1 < len(lines) && extractID(lines[i+1]) == beforeCardID {
				result = append(result, renderCardLines(card)...)
				result = append(result, "") // blank line separator
				inserted = true
			}
		}
		result = append(result, lines[i])
	}

	if !inserted {
		return AddCard(content, targetColumn, card)
	}

	return strings.Join(result, "\n")
}

// UpdateCard replaces a card in-place by ID.
func UpdateCard(content string, cardID string, card *models.Card) string {
	lines := strings.Split(content, "\n")
	var result []string
	replaced := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if !replaced && cardRe.MatchString(line) {
			if i+1 < len(lines) && extractID(lines[i+1]) == cardID {
				// Replace this card's lines.
				result = append(result, renderCardLines(card)...)
				replaced = true
				// Skip old card lines.
				i++ // skip ID line
				for i+1 < len(lines) {
					next := lines[i+1]
					if isCardMeta(next) || strings.TrimSpace(next) == "" {
						i++
						if strings.TrimSpace(next) == "" {
							break
						}
					} else {
						break
					}
				}
				continue
			}
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

func renderCardLines(card *models.Card) []string {
	var lines []string
	checkbox := " "
	if card.Completed {
		checkbox = "x"
	}
	lines = append(lines, fmt.Sprintf("- [%s] %s", checkbox, card.Title))
	if card.ID != "" {
		lines = append(lines, fmt.Sprintf("<!-- liveboard:id=%s -->", card.ID))
	}
	if len(card.Tags) > 0 {
		lines = append(lines, "  tags: "+strings.Join(card.Tags, ", "))
	}
	if card.Assignee != "" {
		lines = append(lines, "  assignee: "+card.Assignee)
	}
	if card.Priority != "" {
		lines = append(lines, "  priority: "+card.Priority)
	}
	if card.Due != "" {
		lines = append(lines, "  due: "+card.Due)
	}
	for k, v := range card.Metadata {
		lines = append(lines, fmt.Sprintf("  %s: %s", k, v))
	}
	return lines
}

var cardRe = regexp.MustCompile(`^- \[([ xX])\] `)

func extractID(line string) string {
	m := idRe.FindStringSubmatch(strings.TrimSpace(line))
	if m != nil {
		return m[1]
	}
	return ""
}

func isIDComment(line string) bool {
	return idRe.MatchString(strings.TrimSpace(line))
}

func isCardMeta(line string) bool {
	return metaRe.MatchString(line)
}

var (
	idRe   = regexp.MustCompile(`^<!-- liveboard:id=(\S+) -->$`)
	metaRe = regexp.MustCompile(`^  \w+: .+$`)
)
