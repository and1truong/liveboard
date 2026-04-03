package web

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/pkg/models"
)

// HandleCreateColumn handles POST /board/{slug}/columns.
func (bv *BoardViewHandler) HandleCreateColumn(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colName := r.FormValue("column_name")

	if colName == "" {
		model, _ := bv.boardViewModel(slug)
		model.Error = "Column name is required"
		bv.renderBoardContent(w, model)
		return
	}

	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		bv.renderBoardContent(w, model)
		return
	}

	version := formVersion(r)
	model, mutErr := bv.mutateBoard(slug, version, func(b *models.Board) error {
		b.Columns = append(b.Columns, models.Column{Name: colName})
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		bv.handleConflict(w, slug)
		return
	}
	bv.renderBoardContent(w, model)
}

// HandleRenameColumn handles POST /board/{slug}/columns/rename.
func (bv *BoardViewHandler) HandleRenameColumn(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	oldName := r.FormValue("old_name")
	newName := r.FormValue("new_name")

	if oldName == "" {
		model, _ := bv.boardViewModel(slug)
		model.Error = "Old column name is required"
		bv.renderBoardContent(w, model)
		return
	}

	if newName == "" {
		model, _ := bv.boardViewModel(slug)
		model.Error = "New column name is required"
		bv.renderBoardContent(w, model)
		return
	}

	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		bv.renderBoardContent(w, model)
		return
	}

	version := formVersion(r)
	model, mutErr := bv.mutateBoard(slug, version, func(b *models.Board) error {
		for i := range b.Columns {
			if b.Columns[i].Name == oldName {
				b.Columns[i].Name = newName
				return nil
			}
		}
		return fmt.Errorf("column %q not found", oldName)
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		bv.handleConflict(w, slug)
		return
	}
	bv.renderBoardContent(w, model)
}

// HandleDeleteColumn handles POST /board/{slug}/columns/delete.
func (bv *BoardViewHandler) HandleDeleteColumn(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colName := r.FormValue("column_name")

	if colName == "" {
		model, _ := bv.boardViewModel(slug)
		model.Error = "Column name is required"
		bv.renderBoardContent(w, model)
		return
	}

	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		bv.renderBoardContent(w, model)
		return
	}

	version := formVersion(r)
	model, mutErr := bv.mutateBoard(slug, version, func(b *models.Board) error {
		var cols []models.Column
		found := false
		for j, col := range b.Columns {
			if col.Name == colName {
				found = true
				if j < len(b.ListCollapse) {
					b.ListCollapse = append(b.ListCollapse[:j], b.ListCollapse[j+1:]...)
				}
				continue
			}
			cols = append(cols, col)
		}
		if !found {
			return fmt.Errorf("column %q not found", colName)
		}
		b.Columns = cols
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		bv.handleConflict(w, slug)
		return
	}
	bv.renderBoardContent(w, model)
}

// HandleToggleColumnCollapse handles POST /board/{slug}/columns/collapse.
func (bv *BoardViewHandler) HandleToggleColumnCollapse(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		bv.renderBoardContent(w, model)
		return
	}

	colIndex, err := formInt(r, "col_index")
	if err != nil {
		model, _ := bv.boardViewModel(slug)
		model.Error = err.Error()
		bv.renderBoardContent(w, model)
		return
	}

	version := formVersion(r)
	model, mutErr := bv.mutateBoard(slug, version, func(b *models.Board) error {
		if colIndex < 0 || colIndex >= len(b.Columns) {
			return fmt.Errorf("column index %d out of range", colIndex)
		}
		for len(b.ListCollapse) < len(b.Columns) {
			b.ListCollapse = append(b.ListCollapse, false)
		}
		b.ListCollapse[colIndex] = !b.ListCollapse[colIndex]
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		bv.handleConflict(w, slug)
		return
	}
	bv.renderBoardContent(w, model)
}

// HandleSortColumn handles POST /board/{slug}/columns/sort.
func (bv *BoardViewHandler) HandleSortColumn(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colIdx, err := formInt(r, "col_idx")
	if err != nil {
		model, _ := bv.boardViewModel(slug)
		model.Error = err.Error()
		bv.renderBoardContent(w, model)
		return
	}

	sortBy := r.FormValue("sort_by")
	if sortBy == "" {
		model, _ := bv.boardViewModel(slug)
		model.Error = "sort_by is required"
		bv.renderBoardContent(w, model)
		return
	}

	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		bv.renderBoardContent(w, model)
		return
	}

	version := formVersion(r)
	model, mutErr := bv.mutateBoard(slug, version, func(b *models.Board) error {
		if colIdx < 0 || colIdx >= len(b.Columns) {
			return fmt.Errorf("column index %d out of range", colIdx)
		}
		cards := b.Columns[colIdx].Cards
		switch sortBy {
		case "name":
			sortCardsByName(cards)
		case "priority":
			sortCardsByPriority(cards)
		case "due":
			sortCardsByDue(cards)
		default:
			return fmt.Errorf("unknown sort key %q", sortBy)
		}
		b.Columns[colIdx].Cards = cards
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		bv.handleConflict(w, slug)
		return
	}
	bv.renderBoardContent(w, model)
}

// reorderColumns moves colName after afterCol (or to front if afterCol is empty)
// and rebuilds ListCollapse to match the new order.
func reorderColumns(b *models.Board, colName, afterCol string) error {
	// Align ListCollapse with columns.
	for len(b.ListCollapse) < len(b.Columns) {
		b.ListCollapse = append(b.ListCollapse, false)
	}

	// Build collapse state map.
	collapseByName := make(map[string]bool, len(b.Columns))
	for i, col := range b.Columns {
		collapseByName[col.Name] = b.ListCollapse[i]
	}

	// Find and remove the column being moved.
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
		return fmt.Errorf("column %q not found", colName)
	}

	// Rebuild: if afterCol is empty, prepend; otherwise insert after afterCol.
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

	// Rebuild ListCollapse to match new order.
	b.ListCollapse = make([]bool, len(b.Columns))
	for i, col := range b.Columns {
		b.ListCollapse[i] = collapseByName[col.Name]
	}
	return nil
}

// HandleMoveColumn handles POST /board/{slug}/columns/move.
func (bv *BoardViewHandler) HandleMoveColumn(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colName := r.FormValue("column")
	afterCol := r.FormValue("after_column")

	if colName == "" {
		model, _ := bv.boardViewModel(slug)
		model.Error = "Column name is required"
		bv.renderBoardContent(w, model)
		return
	}

	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		bv.renderBoardContent(w, model)
		return
	}

	version := formVersion(r)
	model, mutErr := bv.mutateBoard(slug, version, func(b *models.Board) error {
		return reorderColumns(b, colName, afterCol)
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		bv.handleConflict(w, slug)
		return
	}
	bv.renderBoardContent(w, model)
}

// Sort helpers

func sortCardsByName(cards []models.Card) {
	sort.SliceStable(cards, func(i, j int) bool {
		return strings.ToLower(cards[i].Title) < strings.ToLower(cards[j].Title)
	})
}

func sortCardsByPriority(cards []models.Card) {
	sort.SliceStable(cards, func(i, j int) bool {
		return priorityRank(cards[i].Priority) > priorityRank(cards[j].Priority)
	})
}

func sortCardsByDue(cards []models.Card) {
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
}

// flattenCards collects all cards from all columns with their position indices.
func flattenCards(b *models.Board) []CardWithPosition {
	n := 0
	for _, col := range b.Columns {
		n += len(col.Cards)
	}
	all := make([]CardWithPosition, 0, n)
	for ci, col := range b.Columns {
		for ci2, card := range col.Cards {
			all = append(all, CardWithPosition{
				Card:       card,
				ColIdx:     ci,
				CardIdx:    ci2,
				ColumnName: col.Name,
			})
		}
	}
	return all
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
