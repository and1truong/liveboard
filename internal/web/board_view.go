package web

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/jfyne/live"

	"github.com/and1truong/liveboard/pkg/models"
)

// BoardViewModel is the state for the board view page.
type BoardViewModel struct {
	Title       string         `json:"title"`
	Board       *models.Board  `json:"board"`
	BoardName   string         `json:"board_name"`
	BoardSlug   string         `json:"board_slug"` // filename stem for loading
	Boards      []BoardSummary `json:"boards"`
	Error       string         `json:"error,omitempty"`
	SelectedID  string         `json:"selected_id,omitempty"`
	ShowAddCard string         `json:"show_add_card,omitempty"` // Column name
	NeedsReload bool           `json:"needs_reload,omitempty"`
}

// boardViewModel loads a board by slug and returns a populated BoardViewModel.
func (h *Handler) boardViewModel(slug string) (BoardViewModel, error) {
	board, err := h.ws.LoadBoard(slug)
	if err != nil {
		return BoardViewModel{BoardSlug: slug, Error: err.Error()}, nil
	}
	allBoards, _ := h.ws.ListBoards()
	return BoardViewModel{
		Title:     board.Name + " — LiveBoard",
		Board:     board,
		BoardName: board.Name,
		BoardSlug: slug,
		Boards:    toBoardSummaries(allBoards),
	}, nil
}

// mountBoardView initializes the board view model.
func (h *Handler) mountBoardView(ctx context.Context, s *live.Socket) (interface{}, error) {
	var slug string

	// On websocket reconnect, reuse slug from existing assigns
	if s != nil {
		if m, ok := s.Assigns().(BoardViewModel); ok && m.BoardSlug != "" {
			slug = m.BoardSlug
		}
	}

	// Initial HTTP mount: extract board slug from URL path
	if slug == "" {
		req := live.Request(ctx)
		if req == nil || req.URL == nil {
			return BoardViewModel{Error: "Invalid request"}, nil
		}
		slug = strings.TrimPrefix(req.URL.Path, "/board/")
		slug, _ = url.PathUnescape(slug)
	}

	if slug == "" {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	return h.boardViewModel(slug)
}

// slugFromParams extracts the board slug from event params.
func slugFromParams(p live.Params) (string, bool) {
	slug, ok := p["name"].(string)
	return slug, ok && slug != ""
}

// handleCreateCard creates a new card in a column.
func (h *Handler) handleCreateCard(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	column, ok := p["column"].(string)
	if !ok || column == "" {
		return BoardViewModel{Error: "Column name is required"}, nil
	}

	title, ok := p["title"].(string)
	if !ok || title == "" {
		return BoardViewModel{Error: "Card title is required"}, nil
	}

	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	boardPath := h.ws.BoardPath(slug)
	_, err := h.eng.AddCard(boardPath, column, title)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	h.commitWithHandling(boardPath, fmt.Sprintf("Add card \"%s\" to %s", title, column))
	h.publishBoardEvent(slug, "card_created")

	return h.boardViewModel(slug)
}

// handleMoveCard moves a card to a different column.
func (h *Handler) handleMoveCard(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	cardID, ok := p["card_id"].(string)
	if !ok || cardID == "" {
		return BoardViewModel{Error: "Card ID is required"}, nil
	}

	targetColumn, ok := p["target_column"].(string)
	if !ok || targetColumn == "" {
		return BoardViewModel{Error: "Target column is required"}, nil
	}

	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	boardPath := h.ws.BoardPath(slug)
	err := h.eng.MoveCard(boardPath, cardID, targetColumn)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	h.commitWithHandling(boardPath, fmt.Sprintf("Move card %s to %s", cardID, targetColumn))
	h.publishBoardEvent(slug, "card_moved")

	return h.boardViewModel(slug)
}

// handleReorderCard moves a card to a specific position within a column.
func (h *Handler) handleReorderCard(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	cardID, ok := p["card_id"].(string)
	if !ok || cardID == "" {
		return BoardViewModel{Error: "Card ID is required"}, nil
	}

	column, ok := p["column"].(string)
	if !ok || column == "" {
		return BoardViewModel{Error: "Column is required"}, nil
	}

	beforeCardID, _ := p["before_card_id"].(string)

	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	boardPath := h.ws.BoardPath(slug)
	err := h.eng.ReorderCard(boardPath, cardID, column, beforeCardID)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	h.commitWithHandling(boardPath, fmt.Sprintf("Reorder card %s in %s", cardID, column))
	h.publishBoardEvent(slug, "card_reordered")

	return h.boardViewModel(slug)
}

// handleDeleteCard deletes a card.
func (h *Handler) handleDeleteCard(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	cardID, ok := p["card_id"].(string)
	if !ok || cardID == "" {
		return BoardViewModel{Error: "Card ID is required"}, nil
	}

	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	boardPath := h.ws.BoardPath(slug)
	err := h.eng.DeleteCard(boardPath, cardID)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	h.commitRemoveWithHandling(boardPath, fmt.Sprintf("Delete card %s", cardID))
	h.publishBoardEvent(slug, "card_deleted")

	return h.boardViewModel(slug)
}

// handleToggleComplete marks a card as completed.
func (h *Handler) handleToggleComplete(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	cardID, ok := p["card_id"].(string)
	if !ok || cardID == "" {
		return BoardViewModel{Error: "Card ID is required"}, nil
	}

	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	boardPath := h.ws.BoardPath(slug)
	err := h.eng.CompleteCard(boardPath, cardID)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	h.commitWithHandling(boardPath, fmt.Sprintf("Complete card %s", cardID))
	h.publishBoardEvent(slug, "card_completed")

	return h.boardViewModel(slug)
}

// handleEditCard updates a card's title, body, and tags.
func (h *Handler) handleEditCard(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	cardID, ok := p["card_id"].(string)
	if !ok || cardID == "" {
		return BoardViewModel{Error: "Card ID is required"}, nil
	}

	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	title, _ := p["title"].(string)
	body, _ := p["body"].(string)
	tagsRaw, _ := p["tags"].(string)

	var tags []string
	for _, t := range strings.Split(tagsRaw, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}

	boardPath := h.ws.BoardPath(slug)
	if err := h.eng.EditCard(boardPath, cardID, title, body, tags); err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	h.commitWithHandling(boardPath, fmt.Sprintf("Edit card %s", cardID))
	h.publishBoardEvent(slug, "card_updated")

	return h.boardViewModel(slug)
}

// handleCreateColumn creates a new column.
func (h *Handler) handleCreateColumn(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	colName, ok := p["column_name"].(string)
	if !ok || colName == "" {
		return BoardViewModel{Error: "Column name is required"}, nil
	}

	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	boardPath := h.ws.BoardPath(slug)
	err := h.eng.AddColumn(boardPath, colName)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	h.commitWithHandling(boardPath, fmt.Sprintf("Add column: %s", colName))
	h.publishBoardEvent(slug, "column_created")

	return h.boardViewModel(slug)
}

// handleShowAddCard shows the add card form for a column.
func (h *Handler) handleShowAddCard(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	column, _ := p["column"].(string)

	m, err := h.boardViewModel(slug)
	if err != nil {
		return m, err
	}
	if column == "" {
		m.Error = "Column is required"
	} else {
		m.ShowAddCard = column
	}
	return m, nil
}

// handleCancelAddCard cancels the add card form.
func (h *Handler) handleCancelAddCard(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	return h.boardViewModel(slug)
}

// handleRenameColumn renames a column.
func (h *Handler) handleRenameColumn(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	oldName, ok := p["old_name"].(string)
	if !ok || oldName == "" {
		return BoardViewModel{Error: "Old column name is required"}, nil
	}

	newName, ok := p["new_name"].(string)
	if !ok || newName == "" {
		return BoardViewModel{Error: "New column name is required"}, nil
	}

	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	boardPath := h.ws.BoardPath(slug)
	if err := h.eng.RenameColumn(boardPath, oldName, newName); err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	h.commitWithHandling(boardPath, fmt.Sprintf("Rename column %q to %q", oldName, newName))
	h.publishBoardEvent(slug, "column_renamed")

	return h.boardViewModel(slug)
}

// handleDeleteColumn deletes a column and all its cards.
func (h *Handler) handleDeleteColumn(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	colName, ok := p["column_name"].(string)
	if !ok || colName == "" {
		return BoardViewModel{Error: "Column name is required"}, nil
	}

	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	boardPath := h.ws.BoardPath(slug)
	if err := h.eng.DeleteColumn(boardPath, colName); err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	h.commitRemoveWithHandling(boardPath, fmt.Sprintf("Delete column: %s", colName))
	h.publishBoardEvent(slug, "column_deleted")

	return h.boardViewModel(slug)
}

// handleUpdateBoardMeta updates a board's name and description.
func (h *Handler) handleUpdateBoardMeta(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	name, _ := p["board_name"].(string)
	description, _ := p["description"].(string)

	boardPath := h.ws.BoardPath(slug)
	if err := h.eng.UpdateBoardMeta(boardPath, name, description); err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	h.commitWithHandling(boardPath, fmt.Sprintf("Update board meta: %s", name))
	h.publishBoardEvent(slug, "board_meta_updated")

	return h.boardViewModel(slug)
}

// handleBoardUpdate handles PubSub messages for real-time updates.
func (h *Handler) handleBoardUpdate(_ context.Context, _ *live.Socket, msg any) (interface{}, error) {
	slug, ok := msg.(string)
	if !ok {
		return nil, fmt.Errorf("invalid message type")
	}

	return h.boardViewModel(slug)
}
