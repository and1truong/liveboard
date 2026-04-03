package web

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/reminder"
	"github.com/and1truong/liveboard/pkg/models"
)

// HandleCreateCard handles POST /board/{slug}/cards.
func (bv *BoardViewHandler) HandleCreateCard(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	column := r.FormValue("column")
	title := r.FormValue("title")

	if column == "" || title == "" || slug == "" {
		model, _ := bv.boardViewModel(slug)
		if slug == "" {
			model.Error = "Board name is required"
		} else if column == "" {
			model.Error = "Column name is required"
		} else {
			model.Error = "Card title is required"
		}
		bv.renderBoardContent(w, model)
		return
	}

	// Resolve card position setting to determine prepend vs append.
	loadedBoard, loadErr := bv.ws.LoadBoard(slug)
	prepend := false
	if loadErr == nil {
		global := bv.loadSettings()
		rs := resolveSettings(global, loadedBoard.Settings)
		prepend = rs.CardPosition == "prepend"
	}

	version := formVersion(r)
	model, err := bv.mutateBoard(slug, version, func(b *models.Board) error {
		card := models.Card{
			Title:    title,
			Metadata: map[string]string{"id": reminder.GenerateID()},
		}
		for i := range b.Columns {
			if b.Columns[i].Name == column {
				if prepend {
					b.Columns[i].Cards = append([]models.Card{card}, b.Columns[i].Cards...)
				} else {
					b.Columns[i].Cards = append(b.Columns[i].Cards, card)
				}
				return nil
			}
		}
		return fmt.Errorf("column %q not found", column)
	})
	if errors.Is(err, board.ErrVersionConflict) {
		bv.handleConflict(w, slug)
		return
	}
	bv.renderBoardContent(w, model)
}

// HandleMoveCard handles POST /board/{slug}/cards/move.
func (bv *BoardViewHandler) HandleMoveCard(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colIdx, err := formInt(r, "col_idx")
	if err != nil {
		model, _ := bv.boardViewModel(slug)
		model.Error = err.Error()
		bv.renderBoardContent(w, model)
		return
	}

	cardIdx, err := formInt(r, "card_idx")
	if err != nil {
		model, _ := bv.boardViewModel(slug)
		model.Error = err.Error()
		bv.renderBoardContent(w, model)
		return
	}

	targetColumn := r.FormValue("target_column")
	if targetColumn == "" {
		model, _ := bv.boardViewModel(slug)
		model.Error = "Target column is required"
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
		if err := validateIndices(b, colIdx, cardIdx); err != nil {
			return err
		}
		card := b.Columns[colIdx].Cards[cardIdx]
		b.Columns[colIdx].Cards = removeCardAt(b.Columns[colIdx].Cards, cardIdx)
		for i := range b.Columns {
			if b.Columns[i].Name == targetColumn {
				b.Columns[i].Cards = append(b.Columns[i].Cards, card)
				return nil
			}
		}
		return fmt.Errorf("target column %q not found", targetColumn)
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		bv.handleConflict(w, slug)
		return
	}
	bv.renderBoardContent(w, model)
}

// HandleReorderCard handles POST /board/{slug}/cards/reorder.
func (bv *BoardViewHandler) HandleReorderCard(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colIdx, err := formInt(r, "col_idx")
	if err != nil {
		model, _ := bv.boardViewModel(slug)
		model.Error = err.Error()
		bv.renderBoardContent(w, model)
		return
	}

	cardIdx, err := formInt(r, "card_idx")
	if err != nil {
		model, _ := bv.boardViewModel(slug)
		model.Error = err.Error()
		bv.renderBoardContent(w, model)
		return
	}

	column := r.FormValue("column")
	if column == "" {
		model, _ := bv.boardViewModel(slug)
		model.Error = "Column is required"
		bv.renderBoardContent(w, model)
		return
	}

	beforeIdx := -1
	if s := r.FormValue("before_idx"); s != "" {
		beforeIdx, _ = strconv.Atoi(s)
	}

	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		bv.renderBoardContent(w, model)
		return
	}

	version := formVersion(r)
	model, mutErr := bv.mutateBoard(slug, version, func(b *models.Board) error {
		if err := validateIndices(b, colIdx, cardIdx); err != nil {
			return err
		}
		card := b.Columns[colIdx].Cards[cardIdx]
		b.Columns[colIdx].Cards = removeCardAt(b.Columns[colIdx].Cards, cardIdx)

		targetIdx := -1
		for i := range b.Columns {
			if b.Columns[i].Name == column {
				targetIdx = i
				break
			}
		}
		if targetIdx < 0 {
			return fmt.Errorf("target column %q not found", column)
		}

		cards := b.Columns[targetIdx].Cards
		if beforeIdx < 0 || beforeIdx >= len(cards) {
			cards = append(cards, card)
		} else {
			cards = slices.Insert(cards, beforeIdx, card)
		}
		b.Columns[targetIdx].Cards = cards
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		bv.handleConflict(w, slug)
		return
	}
	bv.renderBoardContent(w, model)
}

// HandleDeleteCard handles POST /board/{slug}/cards/delete.
func (bv *BoardViewHandler) HandleDeleteCard(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colIdx, err := formInt(r, "col_idx")
	if err != nil {
		model, _ := bv.boardViewModel(slug)
		model.Error = err.Error()
		bv.renderBoardContent(w, model)
		return
	}

	cardIdx, err := formInt(r, "card_idx")
	if err != nil {
		model, _ := bv.boardViewModel(slug)
		model.Error = err.Error()
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
		if err := validateIndices(b, colIdx, cardIdx); err != nil {
			return err
		}
		b.Columns[colIdx].Cards = removeCardAt(b.Columns[colIdx].Cards, cardIdx)
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		bv.handleConflict(w, slug)
		return
	}
	bv.renderBoardContent(w, model)
}

// HandleToggleComplete handles POST /board/{slug}/cards/complete.
func (bv *BoardViewHandler) HandleToggleComplete(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colIdx, err := formInt(r, "col_idx")
	if err != nil {
		model, _ := bv.boardViewModel(slug)
		model.Error = err.Error()
		bv.renderBoardContent(w, model)
		return
	}

	cardIdx, err := formInt(r, "card_idx")
	if err != nil {
		model, _ := bv.boardViewModel(slug)
		model.Error = err.Error()
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
		if err := validateIndices(b, colIdx, cardIdx); err != nil {
			return err
		}
		card := &b.Columns[colIdx].Cards[cardIdx]
		card.Completed = !card.Completed
		// Auto-dismiss reminder when card is completed (via callback)
		if card.Completed && bv.onCardCompleted != nil {
			if cardID := reminder.GetCardID(card); cardID != "" {
				bv.onCardCompleted(slug, cardID)
			}
		}
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		bv.handleConflict(w, slug)
		return
	}
	bv.renderBoardContent(w, model)
}

// splitTags splits a comma-separated string into trimmed, non-empty tags.
func splitTags(raw string) []string {
	var tags []string
	for _, t := range strings.Split(raw, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// ensureMember adds member to the board's member list if not already present.
func ensureMember(b *models.Board, member string) {
	for _, m := range b.Members {
		if m == member {
			return
		}
	}
	b.Members = append(b.Members, member)
}

// HandleEditCard handles POST /board/{slug}/cards/edit.
func (bv *BoardViewHandler) HandleEditCard(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colIdx, err := formInt(r, "col_idx")
	if err != nil {
		model, _ := bv.boardViewModel(slug)
		model.Error = err.Error()
		bv.renderBoardContent(w, model)
		return
	}

	cardIdx, err := formInt(r, "card_idx")
	if err != nil {
		model, _ := bv.boardViewModel(slug)
		model.Error = err.Error()
		bv.renderBoardContent(w, model)
		return
	}

	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		bv.renderBoardContent(w, model)
		return
	}

	title := r.FormValue("title")
	body := r.FormValue("body")
	tags := splitTags(r.FormValue("tags"))
	priority := r.FormValue("priority")
	due := r.FormValue("due")
	assignee := r.FormValue("assignee")

	version := formVersion(r)
	model, mutErr := bv.mutateBoard(slug, version, func(b *models.Board) error {
		if err := validateIndices(b, colIdx, cardIdx); err != nil {
			return err
		}
		card := &b.Columns[colIdx].Cards[cardIdx]

		// Ensure card has an ID
		if card.Metadata == nil {
			card.Metadata = map[string]string{}
		}
		if card.Metadata["id"] == "" {
			card.Metadata["id"] = reminder.GenerateID()
		}

		oldDue := card.Due

		if title != "" {
			card.Title = title
		}
		card.Body = body
		card.Tags = tags
		card.Priority = priority
		card.Due = due
		card.Assignee = assignee
		if assignee != "" {
			ensureMember(b, assignee)
		}

		// Auto-recalculate relative reminder if due date changed (via callback)
		if oldDue != due && due != "" && bv.onDueDateChanged != nil {
			bv.onDueDateChanged(slug, card.Metadata["id"], due)
		}

		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		bv.handleConflict(w, slug)
		return
	}
	bv.renderBoardContent(w, model)
}

// validateIndices checks that column and card indices are within bounds.
func validateIndices(b *models.Board, colIdx, cardIdx int) error {
	if colIdx < 0 || colIdx >= len(b.Columns) {
		return fmt.Errorf("column index %d out of range", colIdx)
	}
	if cardIdx < 0 || cardIdx >= len(b.Columns[colIdx].Cards) {
		return fmt.Errorf("card index %d out of range in column %q", cardIdx, b.Columns[colIdx].Name)
	}
	return nil
}

// removeCardAt removes a card at the given index.
func removeCardAt(cards []models.Card, idx int) []models.Card {
	return append(cards[:idx], cards[idx+1:]...)
}
