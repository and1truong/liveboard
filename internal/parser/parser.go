// Package parser parses Markdown files into Board models.
package parser

import (
	"bufio"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/and1truong/liveboard/pkg/models"
)

var (
	cardRe    = regexp.MustCompile(`^- \[([ xX])\] (.+)$`)
	metaRe    = regexp.MustCompile(`^  (\w+): (.+)$`)
	hashTagRe = regexp.MustCompile(`#(\w[\w-]*)`)
)

// BoardSummaryInfo holds lightweight metadata extracted without full card parsing.
type BoardSummaryInfo struct {
	Board       models.Board
	CardCount   int
	DoneCount   int
	ColumnCount int
}

// ParseSummary reads only the YAML frontmatter and counts columns/cards
// without building full card objects. Much faster than Parse for listings.
func ParseSummary(content string) (*BoardSummaryInfo, error) {
	board := &models.Board{}

	body := content
	if strings.HasPrefix(content, "---\n") {
		parts := strings.SplitN(content[4:], "\n---\n", 2)
		if len(parts) == 2 {
			if err := yaml.Unmarshal([]byte(parts[0]), board); err != nil {
				return nil, err
			}
			body = parts[1]
		}
	}

	info := &BoardSummaryInfo{Board: *board}
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "## ") {
			info.ColumnCount++
		} else if strings.HasPrefix(line, "- [") && len(line) > 5 && line[4] == ']' {
			info.CardCount++
			if line[3] == 'x' || line[3] == 'X' {
				info.DoneCount++
			}
		}
	}
	return info, nil
}

// Parse reads a Markdown board string and returns a Board model.
func Parse(content string) (*models.Board, error) {
	board := &models.Board{}

	// Split frontmatter and body.
	body := content
	if strings.HasPrefix(content, "---\n") {
		parts := strings.SplitN(content[4:], "\n---\n", 2)
		if len(parts) == 2 {
			if err := yaml.Unmarshal([]byte(parts[0]), board); err != nil {
				return nil, err
			}
			body = parts[1]
		}
	}

	// Parse body line by line.
	scanner := bufio.NewScanner(strings.NewReader(body))
	var currentCol *models.Column
	var currentCard *models.Card

	for scanner.Scan() {
		line := scanner.Text()

		// H2 heading → new column.
		if strings.HasPrefix(line, "## ") {
			// Flush current card.
			if currentCard != nil && currentCol != nil {
				currentCol.Cards = append(currentCol.Cards, *currentCard)
				currentCard = nil
			}
			colName := strings.TrimPrefix(line, "## ")
			board.Columns = append(board.Columns, models.Column{Name: colName})
			currentCol = &board.Columns[len(board.Columns)-1]
			continue
		}

		// Card line.
		if m := cardRe.FindStringSubmatch(line); m != nil {
			// Flush previous card.
			if currentCard != nil && currentCol != nil {
				currentCol.Cards = append(currentCol.Cards, *currentCard)
			}
			completed := m[1] == "x" || m[1] == "X"
			title := m[2]

			// Extract inline hash tags from title.
			var inlineTags []string
			if matches := hashTagRe.FindAllStringSubmatch(title, -1); matches != nil {
				for _, match := range matches {
					inlineTags = append(inlineTags, match[1])
				}
				title = strings.TrimSpace(hashTagRe.ReplaceAllString(title, ""))
			}

			currentCard = &models.Card{
				Title:     title,
				Completed: completed,
				Tags:      inlineTags,
			}
			continue
		}

		// Skip HTML comments (e.g. legacy liveboard:id lines).
		if strings.HasPrefix(strings.TrimSpace(line), "<!--") {
			continue
		}

		// Card metadata and body lines.
		if currentCard != nil {
			if m := metaRe.FindStringSubmatch(line); m != nil {
				key := m[1]
				val := strings.TrimSpace(m[2])
				switch key {
				case "tags":
					for _, t := range strings.Split(val, ",") {
						t = strings.TrimSpace(t)
						if t != "" {
							currentCard.Tags = append(currentCard.Tags, t)
						}
					}
				case "assignee":
					currentCard.Assignee = val
				case "priority":
					currentCard.Priority = val
				case "due":
					currentCard.Due = val
				default:
					if currentCard.Metadata == nil {
						currentCard.Metadata = make(map[string]string)
					}
					currentCard.Metadata[key] = val
				}
				continue
			}
			// Indented non-metadata lines are body text.
			if strings.HasPrefix(line, "  ") {
				bodyLine := line[2:]
				if currentCard.Body == "" {
					currentCard.Body = bodyLine
				} else {
					currentCard.Body += "\n" + bodyLine
				}
				continue
			}
		}
	}

	// Flush final card.
	if currentCard != nil && currentCol != nil {
		currentCol.Cards = append(currentCol.Cards, *currentCard)
	}

	return board, nil
}
