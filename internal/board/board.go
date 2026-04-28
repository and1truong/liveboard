// Package board implements CRUD operations on Markdown-based kanban boards.
package board

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/and1truong/liveboard/internal/parser"
	"github.com/and1truong/liveboard/internal/util/cardid"
	"github.com/and1truong/liveboard/internal/writer"
	"github.com/and1truong/liveboard/pkg/models"
)

// Sentinel errors for the board engine.
var (
	// ErrVersionConflict is returned when a mutation's client version doesn't match the board's current version.
	ErrVersionConflict = errors.New("board version conflict")
	// ErrNotFound is returned when a board, column, or card cannot be found.
	ErrNotFound = errors.New("not found")
	// ErrOutOfRange is returned when column or card indices are invalid.
	ErrOutOfRange = errors.New("out of range")
	// ErrInvalidInput is returned when a mutation receives semantically invalid input
	// (e.g. source and destination are the same board in MoveCardToBoard).
	ErrInvalidInput = errors.New("invalid input")
	// ErrPartialSourceCleanup is returned by MoveCardToBoard when the destination
	// write succeeded but source removal failed. Callers should distinguish this
	// from ErrVersionConflict because the destination board has already been
	// mutated and the card now exists on both boards.
	ErrPartialSourceCleanup = errors.New("destination written but source removal failed")
)

// Engine provides CRUD operations on boards backed by Markdown files.
type Engine struct {
	locks sync.Map // map[string]*sync.Mutex — per-board locks
}

// New creates a new Engine instance.
func New() *Engine {
	return &Engine{}
}

// ensureCardID assigns a fresh ID to c if it has none.
func ensureCardID(c *models.Card) {
	if c == nil {
		return
	}
	if c.ID == "" {
		c.ID = cardid.NewID()
	}
}

// boardLock returns the per-board mutex, creating one if needed.
func (e *Engine) boardLock(boardPath string) *sync.Mutex {
	val, _ := e.locks.LoadOrStore(boardPath, &sync.Mutex{})
	mu, ok := val.(*sync.Mutex)
	if !ok {
		panic("boardLock: unexpected type in sync.Map")
	}
	return mu
}

// MutateBoard serializes access to a board, checks the client version against the
// on-disk version (skip if clientVersion < 0), applies the mutation, increments the
// version, and writes the result to disk.
func (e *Engine) MutateBoard(boardPath string, clientVersion int, fn func(*models.Board) error) error {
	lock := e.boardLock(boardPath)
	lock.Lock()
	defer lock.Unlock()

	board, err := e.LoadBoard(boardPath)
	if err != nil {
		return err
	}

	if clientVersion >= 0 && board.Version != clientVersion {
		return ErrVersionConflict
	}

	if err := fn(board); err != nil {
		return err
	}

	board.Version++
	if err := renderAndWrite(board, boardPath); err != nil {
		board.Version-- // rollback so in-memory state stays consistent
		return err
	}
	return nil
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

// applyAddCard adds a new card to the specified column of b.
// Returns a pointer to the card as stored in the board slice.
func applyAddCard(b *models.Board, columnName, title string, prepend bool) (*models.Card, error) {
	for i := range b.Columns {
		if b.Columns[i].Name == columnName {
			if prepend {
				b.Columns[i].Cards = append([]models.Card{{Title: title}}, b.Columns[i].Cards...)
				ensureCardID(&b.Columns[i].Cards[0])
				return &b.Columns[i].Cards[0], nil
			}
			b.Columns[i].Cards = append(b.Columns[i].Cards, models.Card{Title: title})
			ensureCardID(&b.Columns[i].Cards[len(b.Columns[i].Cards)-1])
			return &b.Columns[i].Cards[len(b.Columns[i].Cards)-1], nil
		}
	}
	return nil, fmt.Errorf("column %q: %w", columnName, ErrNotFound)
}

// AddCard adds a new card to the specified column.
// If prepend is true, the card is inserted at the beginning; otherwise appended.
func (e *Engine) AddCard(boardPath, columnName, title string, prepend bool) (*models.Card, error) {
	var out *models.Card
	err := e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		c, err := applyAddCard(b, columnName, title, prepend)
		out = c
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// applyMoveCard moves a card to a different column within b.
func applyMoveCard(b *models.Board, colIdx, cardIdx int, targetColumn string) error {
	if err := validateIndices(b, colIdx, cardIdx); err != nil {
		return err
	}
	ensureCardID(&b.Columns[colIdx].Cards[cardIdx])
	card := b.Columns[colIdx].Cards[cardIdx]
	b.Columns[colIdx].Cards = removeCardAt(b.Columns[colIdx].Cards, cardIdx)
	for i := range b.Columns {
		if b.Columns[i].Name == targetColumn {
			b.Columns[i].Cards = append(b.Columns[i].Cards, card)
			return nil
		}
	}
	return fmt.Errorf("target column %q: %w", targetColumn, ErrNotFound)
}

// MoveCard moves a card to a different column.
func (e *Engine) MoveCard(boardPath string, colIdx, cardIdx int, targetColumn string) error {
	return e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		return applyMoveCard(b, colIdx, cardIdx, targetColumn)
	})
}

// MoveCardToBoard moves a card from srcPath to dstColumn on dstPath.
// The card is inserted at the top of the target column. Missing tags and
// members on the target board's frontmatter are auto-added.
//
// Not atomic across boards: target is written first (version bypass), then
// source (version-checked against srcVersion). If the source write fails
// after the target write succeeded, the card is duplicated and the caller
// receives a wrapped error.
func (e *Engine) MoveCardToBoard(srcPath string, srcVersion, srcColIdx, cardIdx int, dstPath, dstColumn string) error {
	if srcPath == dstPath {
		return fmt.Errorf("%w: source and destination boards must differ", ErrInvalidInput)
	}

	srcSnapshot, err := e.LoadBoard(srcPath)
	if err != nil {
		return err
	}
	if err := validateIndices(srcSnapshot, srcColIdx, cardIdx); err != nil {
		return err
	}
	// Optimistic-lock check before any mutation: if the caller's version does
	// not match what's on disk, bail out before writing to the destination so
	// we don't leave a duplicate card behind.
	if srcVersion >= 0 && srcSnapshot.Version != srcVersion {
		return ErrVersionConflict
	}
	cardCopy := srcSnapshot.Columns[srcColIdx].Cards[cardIdx]

	if err := e.MutateBoard(dstPath, -1, func(b *models.Board) error {
		for i := range b.Columns {
			if b.Columns[i].Name == dstColumn {
				b.Columns[i].Cards = append([]models.Card{cardCopy}, b.Columns[i].Cards...)
				if cardCopy.Assignee != "" {
					mergeMissing(&b.Members, []string{cardCopy.Assignee})
				}
				return nil
			}
		}
		return fmt.Errorf("target column %q: %w", dstColumn, ErrNotFound)
	}); err != nil {
		return err
	}

	if err := e.MutateBoard(srcPath, srcVersion, func(b *models.Board) error {
		if err := validateIndices(b, srcColIdx, cardIdx); err != nil {
			return err
		}
		b.Columns[srcColIdx].Cards = removeCardAt(b.Columns[srcColIdx].Cards, cardIdx)
		return nil
	}); err != nil {
		return fmt.Errorf("%w: card added to %s: %w", ErrPartialSourceCleanup, dstPath, err)
	}
	return nil
}

func mergeMissing(existing *[]string, incoming []string) {
	for _, v := range incoming {
		if v == "" {
			continue
		}
		if !slices.Contains(*existing, v) {
			*existing = append(*existing, v)
		}
	}
}

// applyReorderCard moves a card to a specific position within b.
// beforeIdx is the index to insert before; -1 means append to end.
func applyReorderCard(b *models.Board, colIdx, cardIdx, beforeIdx int, targetColumn string) error {
	if err := validateIndices(b, colIdx, cardIdx); err != nil {
		return err
	}
	ensureCardID(&b.Columns[colIdx].Cards[cardIdx])
	card := b.Columns[colIdx].Cards[cardIdx]
	b.Columns[colIdx].Cards = removeCardAt(b.Columns[colIdx].Cards, cardIdx)

	targetIdx := -1
	for i := range b.Columns {
		if b.Columns[i].Name == targetColumn {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		return fmt.Errorf("target column %q: %w", targetColumn, ErrNotFound)
	}

	cards := b.Columns[targetIdx].Cards
	if beforeIdx < 0 || beforeIdx >= len(cards) {
		cards = append(cards, card)
	} else {
		cards = slices.Insert(cards, beforeIdx, card)
	}
	b.Columns[targetIdx].Cards = cards
	return nil
}

// ReorderCard moves a card to a specific position within a column.
// beforeIdx is the index to insert before; -1 means append to end.
func (e *Engine) ReorderCard(boardPath string, colIdx, cardIdx, beforeIdx int, targetColumn string) error {
	return e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		return applyReorderCard(b, colIdx, cardIdx, beforeIdx, targetColumn)
	})
}

// applyCompleteCard toggles the completed state of a card within b.
func applyCompleteCard(b *models.Board, colIdx, cardIdx int) error {
	if err := validateIndices(b, colIdx, cardIdx); err != nil {
		return err
	}
	ensureCardID(&b.Columns[colIdx].Cards[cardIdx])
	b.Columns[colIdx].Cards[cardIdx].Completed = !b.Columns[colIdx].Cards[cardIdx].Completed
	return nil
}

// CompleteCard toggles the completed state of a card.
func (e *Engine) CompleteCard(boardPath string, colIdx, cardIdx int) error {
	return e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		return applyCompleteCard(b, colIdx, cardIdx)
	})
}

// applyTagCard adds tags to a card within b.
func applyTagCard(b *models.Board, colIdx, cardIdx int, tags []string) error {
	if err := validateIndices(b, colIdx, cardIdx); err != nil {
		return err
	}
	card := &b.Columns[colIdx].Cards[cardIdx]
	ensureCardID(card)
	existing := make(map[string]bool)
	for _, t := range card.Tags {
		existing[t] = true
	}
	for _, t := range tags {
		if !existing[t] {
			card.Tags = append(card.Tags, t)
		}
	}
	return nil
}

// TagCard adds tags to a card.
func (e *Engine) TagCard(boardPath string, colIdx, cardIdx int, tags []string) error {
	return e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		return applyTagCard(b, colIdx, cardIdx, tags)
	})
}

// applyEditCard updates a card's fields within b.
func applyEditCard(b *models.Board, colIdx, cardIdx int, title, body string, tags []string, links []string, priority, due, assignee string) error {
	if err := validateIndices(b, colIdx, cardIdx); err != nil {
		return err
	}
	card := &b.Columns[colIdx].Cards[cardIdx]
	ensureCardID(card)
	if title != "" {
		card.Title = title
	}
	card.Body = body
	card.Tags = tags
	card.Links = links
	card.Priority = priority
	card.Due = due
	card.Assignee = assignee
	return nil
}

// EditCard updates a card's title, body, tags, links, priority, due, and assignee in-place.
func (e *Engine) EditCard(boardPath string, colIdx, cardIdx int, title, body string, tags []string, links []string, priority, due, assignee string) error {
	return e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		return applyEditCard(b, colIdx, cardIdx, title, body, tags, links, priority, due, assignee)
	})
}

// applyDeleteCard removes a card from b by column and card index.
func applyDeleteCard(b *models.Board, colIdx, cardIdx int) error {
	if err := validateIndices(b, colIdx, cardIdx); err != nil {
		return err
	}
	b.Columns[colIdx].Cards = removeCardAt(b.Columns[colIdx].Cards, cardIdx)
	return nil
}

// DeleteCard removes a card by column and card index.
func (e *Engine) DeleteCard(boardPath string, colIdx, cardIdx int) error {
	return e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		return applyDeleteCard(b, colIdx, cardIdx)
	})
}

// ShowCard returns a card by column and card index.
func (e *Engine) ShowCard(boardPath string, colIdx, cardIdx int) (*models.Card, string, error) {
	lock := e.boardLock(boardPath)
	lock.Lock()
	defer lock.Unlock()

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

// applyAddColumn adds a new column to b.
func applyAddColumn(b *models.Board, colName string) error {
	b.Columns = append(b.Columns, models.Column{Name: colName})
	return nil
}

// AddColumn adds a new column to the board.
func (e *Engine) AddColumn(boardPath, colName string) error {
	return e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		return applyAddColumn(b, colName)
	})
}

// applyDeleteColumn removes a column and all its cards from b.
func applyDeleteColumn(b *models.Board, colName string) error {
	found := false
	var cols []models.Column
	for i, col := range b.Columns {
		if col.Name == colName {
			found = true
			// Remove corresponding collapse state if present.
			if i < len(b.ListCollapse) {
				b.ListCollapse = append(b.ListCollapse[:i], b.ListCollapse[i+1:]...)
			}
			continue
		}
		cols = append(cols, col)
	}
	if !found {
		return nil // idempotent: no-op if column doesn't exist
	}
	b.Columns = cols
	return nil
}

// DeleteColumn removes a column and all its cards.
func (e *Engine) DeleteColumn(boardPath, colName string) error {
	return e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		return applyDeleteColumn(b, colName)
	})
}

// applyRenameColumn renames a column within b.
func applyRenameColumn(b *models.Board, oldName, newName string) error {
	found := false
	for i := range b.Columns {
		if b.Columns[i].Name == oldName {
			b.Columns[i].Name = newName
			found = true
		}
	}
	if !found {
		return fmt.Errorf("column %q: %w", oldName, ErrNotFound)
	}
	return nil
}

// RenameColumn renames a column in-place.
func (e *Engine) RenameColumn(boardPath, oldName, newName string) error {
	return e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		return applyRenameColumn(b, oldName, newName)
	})
}

// applyMoveColumn reorders a column within b to be after afterCol.
// Empty afterCol means prepend to front.
func applyMoveColumn(b *models.Board, colName, afterCol string) error {
	// Ensure ListCollapse is aligned with columns.
	for len(b.ListCollapse) < len(b.Columns) {
		b.ListCollapse = append(b.ListCollapse, false)
	}

	// Build index map for collapse state.
	collapseByName := make(map[string]bool, len(b.Columns))
	for i, col := range b.Columns {
		collapseByName[col.Name] = b.ListCollapse[i]
	}

	// Find and remove the target column.
	var movingCol *models.Column
	var remaining []models.Column
	for _, col := range b.Columns {
		if col.Name == colName {
			c := col
			movingCol = &c
		} else {
			remaining = append(remaining, col)
		}
	}
	if movingCol == nil {
		return fmt.Errorf("column %q: %w", colName, ErrNotFound)
	}

	// Insert after the specified column (empty afterCol = prepend).
	var reordered []models.Column
	if afterCol == "" {
		reordered = append([]models.Column{*movingCol}, remaining...)
	} else {
		for _, col := range remaining {
			reordered = append(reordered, col)
			if col.Name == afterCol {
				reordered = append(reordered, *movingCol)
			}
		}
	}

	b.Columns = reordered

	// Rebuild ListCollapse to match new column order.
	b.ListCollapse = make([]bool, len(b.Columns))
	for i, col := range b.Columns {
		b.ListCollapse[i] = collapseByName[col.Name]
	}
	return nil
}

// MoveColumn reorders a column to be after another column.
func (e *Engine) MoveColumn(boardPath, colName, afterCol string) error {
	return e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		return applyMoveColumn(b, colName, afterCol)
	})
}

// applyToggleColumnCollapse toggles the collapsed state of a column within b.
func applyToggleColumnCollapse(b *models.Board, colIndex int) error {
	if colIndex < 0 || colIndex >= len(b.Columns) {
		return fmt.Errorf("column index %d: %w", colIndex, ErrOutOfRange)
	}
	// Grow ListCollapse to match number of columns if needed.
	for len(b.ListCollapse) < len(b.Columns) {
		b.ListCollapse = append(b.ListCollapse, false)
	}
	b.ListCollapse[colIndex] = !b.ListCollapse[colIndex]
	return nil
}

// ToggleColumnCollapse toggles the collapsed state of a column by index.
func (e *Engine) ToggleColumnCollapse(boardPath string, colIndex int) error {
	return e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		return applyToggleColumnCollapse(b, colIndex)
	})
}

// applyUpdateBoardMeta updates a board's name and description within b.
func applyUpdateBoardMeta(b *models.Board, name, description string) error {
	if name != "" {
		b.Name = name
	}
	b.Description = description
	return nil
}

// UpdateBoardMeta updates a board's name and description.
func (e *Engine) UpdateBoardMeta(boardPath, name, description string) error {
	return e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		return applyUpdateBoardMeta(b, name, description)
	})
}

// applyUpdateBoardMembers sets the member list within b.
func applyUpdateBoardMembers(b *models.Board, members []string) error {
	b.Members = members
	return nil
}

// UpdateBoardMembers sets the member list for a board.
func (e *Engine) UpdateBoardMembers(boardPath string, members []string) error {
	return e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		return applyUpdateBoardMembers(b, members)
	})
}

// applyUpdateBoardIcon sets the icon slug/emoji and/or icon background color within b.
// nil args leave the corresponding field untouched; empty strings clear it.
func applyUpdateBoardIcon(b *models.Board, icon, iconColor *string) error {
	if icon != nil {
		b.Icon = *icon
	}
	if iconColor != nil {
		b.IconColor = *iconColor
	}
	return nil
}

// UpdateBoardIcon sets the icon slug/emoji for a board. Does not touch IconColor.
func (e *Engine) UpdateBoardIcon(boardPath, icon string) error {
	return e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		return applyUpdateBoardIcon(b, &icon, nil)
	})
}

func validateIndices(board *models.Board, colIdx, cardIdx int) error {
	if colIdx < 0 || colIdx >= len(board.Columns) {
		return fmt.Errorf("column index %d: %w", colIdx, ErrOutOfRange)
	}
	if cardIdx < 0 || cardIdx >= len(board.Columns[colIdx].Cards) {
		return fmt.Errorf("card index %d in column %q: %w", cardIdx, board.Columns[colIdx].Name, ErrOutOfRange)
	}
	return nil
}

func removeCardAt(cards []models.Card, idx int) []models.Card {
	return append(cards[:idx], cards[idx+1:]...)
}

// applySortColumn sorts the cards in a column of b by the given key.
// Supported keys: "name", "priority", "due".
func applySortColumn(b *models.Board, colIdx int, sortBy string) error {
	if colIdx < 0 || colIdx >= len(b.Columns) {
		return fmt.Errorf("column index %d: %w", colIdx, ErrOutOfRange)
	}
	cards := b.Columns[colIdx].Cards
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
			a, bv := cards[i].Due, cards[j].Due
			if a == "" && bv == "" {
				return false
			}
			if a == "" {
				return false
			}
			if bv == "" {
				return true
			}
			return a < bv
		})
	default:
		return fmt.Errorf("unknown sort key %q", sortBy)
	}
	b.Columns[colIdx].Cards = cards
	return nil
}

// SortColumn sorts the cards in a column by the given key.
// Supported keys: "name", "priority", "due".
func (e *Engine) SortColumn(boardPath string, colIdx int, sortBy string) error {
	return e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		return applySortColumn(b, colIdx, sortBy)
	})
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

// applyUpdateBoardSettings replaces the per-board settings overrides within b.
func applyUpdateBoardSettings(b *models.Board, settings models.BoardSettings) error {
	b.Settings = settings
	return nil
}

// UpdateBoardSettings replaces a board's per-board settings overrides.
func (e *Engine) UpdateBoardSettings(boardPath string, settings models.BoardSettings) error {
	return e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		return applyUpdateBoardSettings(b, settings)
	})
}

// applyPatchBoardSettings merges non-nil pointer fields from patch into
// b.Settings. Nil fields in patch leave the corresponding override on b
// unchanged, matching the LocalAdapter's `{ ...existing, ...patch }` semantics.
func applyPatchBoardSettings(b *models.Board, patch models.BoardSettings) error {
	if patch.ShowCheckbox != nil {
		b.Settings.ShowCheckbox = patch.ShowCheckbox
	}
	if patch.CardPosition != nil {
		b.Settings.CardPosition = patch.CardPosition
	}
	if patch.ExpandColumns != nil {
		b.Settings.ExpandColumns = patch.ExpandColumns
	}
	if patch.ViewMode != nil {
		b.Settings.ViewMode = patch.ViewMode
	}
	if patch.CardDisplayMode != nil {
		b.Settings.CardDisplayMode = patch.CardDisplayMode
	}
	if patch.WeekStart != nil {
		b.Settings.WeekStart = patch.WeekStart
	}
	return nil
}

// PatchBoardSettings applies a partial update to a board's per-board
// settings overrides under the per-board lock. Non-nil fields in patch
// replace; nil fields preserve existing overrides.
func (e *Engine) PatchBoardSettings(boardPath string, patch models.BoardSettings) error {
	return e.MutateBoard(boardPath, -1, func(b *models.Board) error {
		return applyPatchBoardSettings(b, patch)
	})
}
