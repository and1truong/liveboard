package parser

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/and1truong/liveboard/pkg/models"
	"gopkg.in/yaml.v3"
)

var (
	cardRe    = regexp.MustCompile(`^- \[([ xX])\] (.+)$`)
	idRe      = regexp.MustCompile(`^<!-- liveboard:id=(\S+) -->$`)
	metaRe    = regexp.MustCompile(`^  (\w+): (.+)$`)
	hashTagRe = regexp.MustCompile(`#(\w[\w-]*)`)
)

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
	var bodyLines []string
	inBody := false

	flushCard := func() {
		if currentCard != nil && currentCol != nil {
			if len(bodyLines) > 0 {
				currentCard.Body = strings.TrimSpace(strings.Join(bodyLines, "\n"))
			}
			currentCol.Cards = append(currentCol.Cards, *currentCard)
		}
		currentCard = nil
		bodyLines = nil
		inBody = false
	}

	for scanner.Scan() {
		line := scanner.Text()

		// H2 heading → new column.
		if strings.HasPrefix(line, "## ") {
			flushCard()
			colName := strings.TrimPrefix(line, "## ")
			board.Columns = append(board.Columns, models.Column{Name: colName})
			currentCol = &board.Columns[len(board.Columns)-1]
			continue
		}

		// Card line.
		if m := cardRe.FindStringSubmatch(line); m != nil {
			flushCard()
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

		if currentCard == nil {
			continue
		}

		trimmed := strings.TrimSpace(line)

		// Blank line: if we've already seen the ID, enter body mode.
		if trimmed == "" {
			if currentCard.ID != "" {
				inBody = true
			}
			continue
		}

		// Body content: 2-space indented lines after the blank separator.
		if inBody {
			if strings.HasPrefix(line, "  ") {
				bodyLines = append(bodyLines, line[2:])
			}
			continue
		}

		// Card ID comment.
		if m := idRe.FindStringSubmatch(trimmed); m != nil {
			currentCard.ID = m[1]
			continue
		}

		// Card metadata.
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
	}

	// Flush final card.
	flushCard()

	return board, nil
}
