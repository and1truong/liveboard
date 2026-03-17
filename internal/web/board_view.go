package web

import (
	"context"
	"fmt"

	"github.com/jfyne/live"

	"github.com/and1truong/liveboard/pkg/models"
)

// BoardViewModel is the state for the board view page.
type BoardViewModel struct {
	Board       *models.Board `json:"board"`
	BoardName   string        `json:"board_name"`
	Error       string        `json:"error,omitempty"`
	SelectedID  string        `json:"selected_id,omitempty"`
	ShowAddCard string        `json:"show_add_card,omitempty"` // Column name
	NeedsReload bool          `json:"needs_reload,omitempty"`
}

// mountBoardView initializes the board view model.
func (h *Handler) mountBoardView(ctx context.Context, s *live.Socket) (interface{}, error) {
	// Get board name from URL
	boardName := live.Request(ctx).URL.Query().Get("name")
	if boardName == "" {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	board, err := h.ws.LoadBoard(boardName)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	return BoardViewModel{
		Board:     board,
		BoardName: boardName,
	}, nil
}

// handleCreateCard creates a new card in a column.
func (h *Handler) handleCreateCard(ctx context.Context, s *live.Socket, p live.Params) (interface{}, error) {
	column, ok := p["column"].(string)
	if !ok || column == "" {
		return BoardViewModel{Error: "Column name is required"}, nil
	}

	title, ok := p["title"].(string)
	if !ok || title == "" {
		return BoardViewModel{Error: "Card title is required"}, nil
	}

	boardName, ok := p["name"].(string)
	if !ok || boardName == "" {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	boardPath := h.ws.BoardPath(boardName)
	_, err := h.eng.AddCard(boardPath, column, title)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	// Git commit for card creation
	if h.git != nil {
		_ = h.git.Commit(boardPath, fmt.Sprintf("Add card \"%s\" to %s", title, column))
	}

	// Publish update to all subscribers
	h.publishBoardEvent(boardName, "card_created")

	// Reload board
	board, err := h.ws.LoadBoard(boardName)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	return BoardViewModel{
		Board:     board,
		BoardName: boardName,
	}, nil
}

// handleMoveCard moves a card to a different column.
func (h *Handler) handleMoveCard(ctx context.Context, s *live.Socket, p live.Params) (interface{}, error) {
	cardID, ok := p["card_id"].(string)
	if !ok || cardID == "" {
		return BoardViewModel{Error: "Card ID is required"}, nil
	}

	targetColumn, ok := p["target_column"].(string)
	if !ok || targetColumn == "" {
		return BoardViewModel{Error: "Target column is required"}, nil
	}

	boardName, ok := p["name"].(string)
	if !ok || boardName == "" {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	boardPath := h.ws.BoardPath(boardName)
	err := h.eng.MoveCard(boardPath, cardID, targetColumn)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	// Git commit for card move
	if h.git != nil {
		_ = h.git.Commit(boardPath, fmt.Sprintf("Move card %s to %s", cardID, targetColumn))
	}

	// Publish update to all subscribers
	h.publishBoardEvent(boardName, "card_moved")

	// Reload board
	board, err := h.ws.LoadBoard(boardName)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	return BoardViewModel{
		Board:     board,
		BoardName: boardName,
	}, nil
}

// handleDeleteCard deletes a card.
func (h *Handler) handleDeleteCard(ctx context.Context, s *live.Socket, p live.Params) (interface{}, error) {
	cardID, ok := p["card_id"].(string)
	if !ok || cardID == "" {
		return BoardViewModel{Error: "Card ID is required"}, nil
	}

	boardName, ok := p["name"].(string)
	if !ok || boardName == "" {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	boardPath := h.ws.BoardPath(boardName)
	err := h.eng.DeleteCard(boardPath, cardID)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	// Git commit for card deletion
	if h.git != nil {
		_ = h.git.CommitRemove(boardPath, fmt.Sprintf("Delete card %s", cardID))
	}

	// Publish update to all subscribers
	h.publishBoardEvent(boardName, "card_deleted")

	// Reload board
	board, err := h.ws.LoadBoard(boardName)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	return BoardViewModel{
		Board:     board,
		BoardName: boardName,
	}, nil
}

// handleToggleComplete marks a card as completed.
func (h *Handler) handleToggleComplete(ctx context.Context, s *live.Socket, p live.Params) (interface{}, error) {
	cardID, ok := p["card_id"].(string)
	if !ok || cardID == "" {
		return BoardViewModel{Error: "Card ID is required"}, nil
	}

	boardName, ok := p["name"].(string)
	if !ok || boardName == "" {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	boardPath := h.ws.BoardPath(boardName)
	err := h.eng.CompleteCard(boardPath, cardID)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	// Git commit for card completion
	if h.git != nil {
		_ = h.git.Commit(boardPath, fmt.Sprintf("Complete card %s", cardID))
	}

	// Publish update to all subscribers
	h.publishBoardEvent(boardName, "card_completed")

	// Reload board
	board, err := h.ws.LoadBoard(boardName)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	return BoardViewModel{
		Board:     board,
		BoardName: boardName,
	}, nil
}

// handleCreateColumn creates a new column.
func (h *Handler) handleCreateColumn(ctx context.Context, s *live.Socket, p live.Params) (interface{}, error) {
	colName, ok := p["column_name"].(string)
	if !ok || colName == "" {
		return BoardViewModel{Error: "Column name is required"}, nil
	}

	boardName, ok := p["name"].(string)
	if !ok || boardName == "" {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	boardPath := h.ws.BoardPath(boardName)
	err := h.eng.AddColumn(boardPath, colName)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	// Git commit for column creation
	if h.git != nil {
		_ = h.git.Commit(boardPath, fmt.Sprintf("Add column: %s", colName))
	}

	// Publish update to all subscribers
	h.publishBoardEvent(boardName, "column_created")

	// Reload board
	board, err := h.ws.LoadBoard(boardName)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	return BoardViewModel{
		Board:     board,
		BoardName: boardName,
	}, nil
}

// handleShowAddCard shows the add card form for a column.
func (h *Handler) handleShowAddCard(ctx context.Context, s *live.Socket, p live.Params) (interface{}, error) {
	boardName, ok := p["name"].(string)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	column, ok := p["column"].(string)
	if ok {
		board, err := h.ws.LoadBoard(boardName)
		if err != nil {
			return BoardViewModel{Error: err.Error()}, nil
		}
		return BoardViewModel{
			Board:     board,
			BoardName: boardName,
			ShowAddCard: column,
		}, nil
	}

	return BoardViewModel{}, nil
}

// handleCancelAddCard cancels the add card form.
func (h *Handler) handleCancelAddCard(ctx context.Context, s *live.Socket, p live.Params) (interface{}, error) {
	boardName, ok := p["name"].(string)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	board, err := h.ws.LoadBoard(boardName)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	return BoardViewModel{
		Board:     board,
		BoardName: boardName,
		ShowAddCard: "",
	}, nil
}

// handleBoardUpdate handles PubSub messages for real-time updates.
func (h *Handler) handleBoardUpdate(ctx context.Context, s *live.Socket, msg any) (interface{}, error) {
	// Reload board on any PubSub message
	boardName, ok := msg.(string)
	if !ok {
		return nil, fmt.Errorf("invalid message type")
	}

	board, err := h.ws.LoadBoard(boardName)
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	return BoardViewModel{
		Board:     board,
		BoardName: boardName,
	}, nil
}
