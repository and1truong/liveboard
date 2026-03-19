// Package board implements CRUD operations on Markdown-based kanban boards.
package board

import (
	"fmt"
	"os"
	"sort"
	"strings"

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

// renderAndWrite is the common save path: render board to markdown and write to disk.
func renderAndWrite(board *models.Board, path string) error {
	content, err := writer.Render(board)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// AddCard adds a new card to the specified column.
// If prepend is true, the card is inserted at the beginning; otherwise appended.
func (e *Engine) AddCard(boardPath, columnName, title string, prepend bool) (*models.Card, error) {
	board, err := e.LoadBoard(boardPath)
	if err != nil {
		return nil, err
	}

	card := &models.Card{Title: title}

	for i := range board.Columns {
		if board.Columns[i].Name == columnName {
			if prepend {
				board.Columns[i].Cards = append([]models.Card{*card}, board.Columns[i].Cards...)
			} else {
				board.Columns[i].Cards = append(board.Columns[i].Cards, *card)
			}
			if err := renderAndWrite(board, boardPath); err != nil {
				return nil, err
			}
			return card, nil
		}
	}
	return nil, fmt.Errorf("column %q not found", columnName)
}

// MoveCard moves a card to a different column.
func (e *Engine) MoveCard(boardPath string, colIdx, cardIdx int, targetColumn string) error {
	board, err := e.LoadBoard(boardPath)
	if err != nil {
		return err
	}

	if err := validateIndices(board, colIdx, cardIdx); err != nil {
		return err
	}

	card := board.Columns[colIdx].Cards[cardIdx]
	board.Columns[colIdx].Cards = removeCardAt(board.Columns[colIdx].Cards, cardIdx)

	// Find target column and append.
	for i := range board.Columns {
		if board.Columns[i].Name == targetColumn {
			board.Columns[i].Cards = append(board.Columns[i].Cards, card)
			return renderAndWrite(board, boardPath)
		}
	}
	return fmt.Errorf("target column %q not found", targetColumn)
}

// ReorderCard moves a card to a specific position within a column.
// beforeIdx is the index to insert before; -1 means append to end.
func (e *Engine) ReorderCard(boardPath string, colIdx, cardIdx, beforeIdx int, targetColumn string) error {
	board, err := e.LoadBoard(boardPath)
	if err != nil {
		return err
	}

	if err := validateIndices(board, colIdx, cardIdx); err != nil {
		return err
	}

	card := board.Columns[colIdx].Cards[cardIdx]
	board.Columns[colIdx].Cards = removeCardAt(board.Columns[colIdx].Cards, cardIdx)

	// Find target column.
	targetIdx := -1
	for i := range board.Columns {
		if board.Columns[i].Name == targetColumn {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		return fmt.Errorf("target column %q not found", targetColumn)
	}

	cards := board.Columns[targetIdx].Cards
	if beforeIdx < 0 || beforeIdx >= len(cards) {
		cards = append(cards, card)
	} else {
		cards = append(cards[:beforeIdx], append([]models.Card{card}, cards[beforeIdx:]...)...)
	}
	board.Columns[targetIdx].Cards = cards

	return renderAndWrite(board, boardPath)
}

// CompleteCard toggles the completed state of a card.
func (e *Engine) CompleteCard(boardPath string, colIdx, cardIdx int) error {
	board, err := e.LoadBoard(boardPath)
	if err != nil {
		return err
	}

	if err := validateIndices(board, colIdx, cardIdx); err != nil {
		return err
	}

	board.Columns[colIdx].Cards[cardIdx].Completed = !board.Columns[colIdx].Cards[cardIdx].Completed
	return renderAndWrite(board, boardPath)
}

// TagCard adds tags to a card.
func (e *Engine) TagCard(boardPath string, colIdx, cardIdx int, tags []string) error {
	board, err := e.LoadBoard(boardPath)
	if err != nil {
		return err
	}

	if err := validateIndices(board, colIdx, cardIdx); err != nil {
		return err
	}

	card := &board.Columns[colIdx].Cards[cardIdx]
	existing := make(map[string]bool)
	for _, t := range card.Tags {
		existing[t] = true
	}
	for _, t := range tags {
		if !existing[t] {
			card.Tags = append(card.Tags, t)
		}
	}

	return renderAndWrite(board, boardPath)
}

// EditCard updates a card's title, body, tags, and priority in-place.
func (e *Engine) EditCard(boardPath string, colIdx, cardIdx int, title, body string, tags []string, priority, due string) error {
	board, err := e.LoadBoard(boardPath)
	if err != nil {
		return err
	}

	if err := validateIndices(board, colIdx, cardIdx); err != nil {
		return err
	}

	card := &board.Columns[colIdx].Cards[cardIdx]
	if title != "" {
		card.Title = title
	}
	card.Body = body
	card.Tags = tags
	card.Priority = priority
	card.Due = due

	return renderAndWrite(board, boardPath)
}

// DeleteCard removes a card by column and card index.
func (e *Engine) DeleteCard(boardPath string, colIdx, cardIdx int) error {
	board, err := e.LoadBoard(boardPath)
	if err != nil {
		return err
	}

	if err := validateIndices(board, colIdx, cardIdx); err != nil {
		return err
	}

	board.Columns[colIdx].Cards = removeCardAt(board.Columns[colIdx].Cards, cardIdx)
	return renderAndWrite(board, boardPath)
}

// ShowCard returns a card by column and card index.
func (e *Engine) ShowCard(boardPath string, colIdx, cardIdx int) (*models.Card, string, error) {
	board, err := e.LoadBoard(boardPath)
	if err != nil {
		return nil, "", err
	}

	if err := validateIndices(board, colIdx, cardIdx); err != nil {
		return nil, "", err
	}

	card := board.Columns[colIdx].Cards[cardIdx]
	return &card, board.Columns[colIdx].Name, nil
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

// RenameColumn renames a column in-place.
func (e *Engine) RenameColumn(boardPath, oldName, newName string) error {
	content, err := os.ReadFile(boardPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	target := "## " + oldName
	for i, line := range lines {
		if strings.TrimSpace(line) == target {
			lines[i] = "## " + newName
		}
	}

	return os.WriteFile(boardPath, []byte(strings.Join(lines, "\n")), 0644)
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

	// Ensure ListCollapse is aligned with columns.
	for len(board.ListCollapse) < len(board.Columns) {
		board.ListCollapse = append(board.ListCollapse, false)
	}

	// Build index map for collapse state.
	collapseByName := make(map[string]bool, len(board.Columns))
	for i, col := range board.Columns {
		collapseByName[col.Name] = board.ListCollapse[i]
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

	// Rebuild ListCollapse to match new column order.
	board.ListCollapse = make([]bool, len(board.Columns))
	for i, col := range board.Columns {
		board.ListCollapse[i] = collapseByName[col.Name]
	}
	return renderAndWrite(board, boardPath)
}

// ToggleColumnCollapse toggles the collapsed state of a column by index.
func (e *Engine) ToggleColumnCollapse(boardPath string, colIndex int) error {
	board, err := e.LoadBoard(boardPath)
	if err != nil {
		return err
	}

	if colIndex < 0 || colIndex >= len(board.Columns) {
		return fmt.Errorf("column index %d out of range", colIndex)
	}

	// Grow ListCollapse to match number of columns if needed.
	for len(board.ListCollapse) < len(board.Columns) {
		board.ListCollapse = append(board.ListCollapse, false)
	}

	board.ListCollapse[colIndex] = !board.ListCollapse[colIndex]

	return renderAndWrite(board, boardPath)
}

// UpdateBoardMeta updates a board's name, description, and tags.
func (e *Engine) UpdateBoardMeta(boardPath, name, description string, tags []string) error {
	board, err := e.LoadBoard(boardPath)
	if err != nil {
		return err
	}
	if name != "" {
		board.Name = name
	}
	board.Description = description
	board.Tags = tags
	return renderAndWrite(board, boardPath)
}

func validateIndices(board *models.Board, colIdx, cardIdx int) error {
	if colIdx < 0 || colIdx >= len(board.Columns) {
		return fmt.Errorf("column index %d out of range", colIdx)
	}
	if cardIdx < 0 || cardIdx >= len(board.Columns[colIdx].Cards) {
		return fmt.Errorf("card index %d out of range in column %q", cardIdx, board.Columns[colIdx].Name)
	}
	return nil
}

func removeCardAt(cards []models.Card, idx int) []models.Card {
	return append(cards[:idx], cards[idx+1:]...)
}

// SortColumn sorts the cards in a column by the given key.
// Supported keys: "name", "priority", "due".
func (e *Engine) SortColumn(boardPath string, colIdx int, sortBy string) error {
	board, err := e.LoadBoard(boardPath)
	if err != nil {
		return err
	}

	if colIdx < 0 || colIdx >= len(board.Columns) {
		return fmt.Errorf("column index %d out of range", colIdx)
	}

	cards := board.Columns[colIdx].Cards

	switch sortBy {
	case "name":
		sort.SliceStable(cards, func(i, j int) bool {
			return strings.ToLower(cards[i].Title) < strings.ToLower(cards[j].Title)
		})
	case "priority":
		sort.SliceStable(cards, func(i, j int) bool {
			return priorityRank(cards[i].Priority) > priorityRank(cards[j].Priority)
		})
	case "due":
		sort.SliceStable(cards, func(i, j int) bool {
			a, b := cards[i].Due, cards[j].Due
			if a == "" && b == "" {
				return false
			}
			if a == "" {
				return false
			}
			if b == "" {
				return true
			}
			return a < b
		})
	default:
		return fmt.Errorf("unknown sort key %q", sortBy)
	}

	board.Columns[colIdx].Cards = cards
	return renderAndWrite(board, boardPath)
}

func priorityRank(p string) int {
	switch strings.ToLower(p) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

// UpdateBoardSettings replaces a board's per-board settings overrides.
func (e *Engine) UpdateBoardSettings(boardPath string, settings models.BoardSettings) error {
	board, err := e.LoadBoard(boardPath)
	if err != nil {
		return err
	}
	board.Settings = settings
	return renderAndWrite(board, boardPath)
}
