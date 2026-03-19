// Package board implements CRUD operations on Markdown-based kanban boards.
package board

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"

	"github.com/and1truong/liveboard/internal/parser"
	"github.com/and1truong/liveboard/internal/writer"
	"github.com/and1truong/liveboard/pkg/models"
)

// Engine provides CRUD operations on boards backed by Markdown files.
type Engine struct{}

// New creates a new Engine instance.
func New() *Engine {
	return &Engine{}
}

// LoadBoard reads and parses a board file.
func (e *Engine) LoadBoard(path string) (*models.Board, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read board: %w", err)
	}
	board, err := parser.Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse board: %w", err)
	}
	board.FilePath = path
	return board, nil
}

// AddCard adds a new card to the specified column. Returns the new card with assigned ID.
func (e *Engine) AddCard(boardPath, columnName, title string) (*models.Card, error) {
	content, err := os.ReadFile(boardPath)
	if err != nil {
		return nil, err
	}

	card := &models.Card{
		ID:    generateID(),
		Title: title,
	}

	newContent := writer.AddCard(string(content), columnName, card)
	if err := os.WriteFile(boardPath, []byte(newContent), 0644); err != nil {
		return nil, err
	}
	return card, nil
}

// MoveCard moves a card to a different column.
func (e *Engine) MoveCard(boardPath, cardID, targetColumn string) error {
	content, err := os.ReadFile(boardPath)
	if err != nil {
		return err
	}

	board, err := parser.Parse(string(content))
	if err != nil {
		return err
	}

	card := findCard(board, cardID)
	if card == nil {
		return fmt.Errorf("card %s not found", cardID)
	}

	newContent := writer.MoveCard(string(content), cardID, targetColumn, card)
	return os.WriteFile(boardPath, []byte(newContent), 0644)
}

// ReorderCard moves a card to a specific position within a column.
// If beforeCardID is empty, the card is appended to the end of the column.
func (e *Engine) ReorderCard(boardPath, cardID, column, beforeCardID string) error {
	content, err := os.ReadFile(boardPath)
	if err != nil {
		return err
	}

	board, err := parser.Parse(string(content))
	if err != nil {
		return err
	}

	card := findCard(board, cardID)
	if card == nil {
		return fmt.Errorf("card %s not found", cardID)
	}

	newContent := writer.RemoveCard(string(content), cardID)
	newContent = writer.InsertCardBefore(newContent, card, beforeCardID, column)
	return os.WriteFile(boardPath, []byte(newContent), 0644)
}

// CompleteCard marks a card as completed.
func (e *Engine) CompleteCard(boardPath, cardID string) error {
	content, err := os.ReadFile(boardPath)
	if err != nil {
		return err
	}

	board, err := parser.Parse(string(content))
	if err != nil {
		return err
	}

	card := findCard(board, cardID)
	if card == nil {
		return fmt.Errorf("card %s not found", cardID)
	}

	card.Completed = !card.Completed
	newContent := writer.UpdateCard(string(content), cardID, card)
	return os.WriteFile(boardPath, []byte(newContent), 0644)
}

// TagCard adds tags to a card.
func (e *Engine) TagCard(boardPath, cardID string, tags []string) error {
	content, err := os.ReadFile(boardPath)
	if err != nil {
		return err
	}

	board, err := parser.Parse(string(content))
	if err != nil {
		return err
	}

	card := findCard(board, cardID)
	if card == nil {
		return fmt.Errorf("card %s not found", cardID)
	}

	// Merge tags (no duplicates).
	existing := make(map[string]bool)
	for _, t := range card.Tags {
		existing[t] = true
	}
	for _, t := range tags {
		if !existing[t] {
			card.Tags = append(card.Tags, t)
		}
	}

	newContent := writer.UpdateCard(string(content), cardID, card)
	return os.WriteFile(boardPath, []byte(newContent), 0644)
}

// EditCard updates a card's title, body, and tags in-place.
func (e *Engine) EditCard(boardPath, cardID, title, body string, tags []string) error {
	content, err := os.ReadFile(boardPath)
	if err != nil {
		return err
	}

	board, err := parser.Parse(string(content))
	if err != nil {
		return err
	}

	card := findCard(board, cardID)
	if card == nil {
		return fmt.Errorf("card %s not found", cardID)
	}

	if title != "" {
		card.Title = title
	}
	card.Body = body
	card.Tags = tags

	newContent := writer.UpdateCard(string(content), cardID, card)
	return os.WriteFile(boardPath, []byte(newContent), 0644)
}

// DeleteCard removes a card by ID.
func (e *Engine) DeleteCard(boardPath, cardID string) error {
	content, err := os.ReadFile(boardPath)
	if err != nil {
		return err
	}

	newContent := writer.RemoveCard(string(content), cardID)
	return os.WriteFile(boardPath, []byte(newContent), 0644)
}

// ShowCard returns a card by ID.
func (e *Engine) ShowCard(boardPath, cardID string) (*models.Card, string, error) {
	board, err := e.LoadBoard(boardPath)
	if err != nil {
		return nil, "", err
	}
	for _, col := range board.Columns {
		for _, card := range col.Cards {
			if card.ID == cardID {
				return &card, col.Name, nil
			}
		}
	}
	return nil, "", fmt.Errorf("card %s not found", cardID)
}

// AddColumn adds a new column to the board.
func (e *Engine) AddColumn(boardPath, colName string) error {
	content, err := os.ReadFile(boardPath)
	if err != nil {
		return err
	}

	s := string(content)
	// Append column at end.
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	s += "\n## " + colName + "\n"

	return os.WriteFile(boardPath, []byte(s), 0644)
}

// DeleteColumn removes a column and all its cards.
func (e *Engine) DeleteColumn(boardPath, colName string) error {
	content, err := os.ReadFile(boardPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var result []string
	skip := false
	target := "## " + colName

	for _, line := range lines {
		if strings.TrimSpace(line) == target {
			skip = true
			continue
		}
		if skip && strings.HasPrefix(line, "## ") {
			skip = false
		}
		if !skip {
			result = append(result, line)
		}
	}

	return os.WriteFile(boardPath, []byte(strings.Join(result, "\n")), 0644)
}

// MoveColumn reorders a column to be after another column.
func (e *Engine) MoveColumn(boardPath, colName, afterCol string) error {
	content, err := os.ReadFile(boardPath)
	if err != nil {
		return err
	}

	board, err := parser.Parse(string(content))
	if err != nil {
		return err
	}

	// Find and remove the target column.
	var movingCol *models.Column
	var remaining []models.Column
	for _, col := range board.Columns {
		if col.Name == colName {
			c := col
			movingCol = &c
		} else {
			remaining = append(remaining, col)
		}
	}
	if movingCol == nil {
		return fmt.Errorf("column %q not found", colName)
	}

	// Insert after the specified column.
	var reordered []models.Column
	for _, col := range remaining {
		reordered = append(reordered, col)
		if col.Name == afterCol {
			reordered = append(reordered, *movingCol)
		}
	}

	board.Columns = reordered
	newContent, err := writer.Render(board)
	if err != nil {
		return err
	}
	return os.WriteFile(boardPath, []byte(newContent), 0644)
}

func findCard(board *models.Board, id string) *models.Card {
	for _, col := range board.Columns {
		for i := range col.Cards {
			if col.Cards[i].ID == id {
				return &col.Cards[i]
			}
		}
	}
	return nil
}

func generateID() string {
	id, err := uuid.NewV7()
	if err != nil {
		// Fallback to v4 if v7 fails.
		return uuid.New().String()
	}
	return id.String()
}
