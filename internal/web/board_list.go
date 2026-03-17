package web

import (
	"context"
	"fmt"

	"github.com/jfyne/live"
)

// BoardListModel is the state for the board list page.
type BoardListModel struct {
	Boards      []BoardSummary `json:"boards"`
	Error       string         `json:"error,omitempty"`
	Creating    bool           `json:"creating,omitempty"`
	NeedsReload bool           `json:"needs_reload,omitempty"`
}

// BoardSummary represents a simplified board for list display.
type BoardSummary struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CardCount   int    `json:"card_count"`
}

// mountBoardList initializes the board list model.
func (h *Handler) mountBoardList(_ context.Context, _ *live.Socket) (interface{}, error) {
	boards, err := h.ws.ListBoards()
	if err != nil {
		return BoardListModel{Error: err.Error()}, nil
	}

	summaries := make([]BoardSummary, len(boards))
	for i, b := range boards {
		cardCount := 0
		for _, col := range b.Columns {
			cardCount += len(col.Cards)
		}
		summaries[i] = BoardSummary{
			Name:        b.Name,
			Description: b.Description,
			CardCount:   cardCount,
		}
	}

	return BoardListModel{Boards: summaries}, nil
}

// handleParams handles parameter changes (e.g., URL params).
func (h *Handler) handleParams(_ context.Context, _ *live.Socket, _ live.Params) (interface{}, error) {
	// Reload boards when params change
	boards, err := h.ws.ListBoards()
	if err != nil {
		return BoardListModel{Error: err.Error()}, nil
	}

	summaries := make([]BoardSummary, len(boards))
	for i, b := range boards {
		cardCount := 0
		for _, col := range b.Columns {
			cardCount += len(col.Cards)
		}
		summaries[i] = BoardSummary{
			Name:        b.Name,
			Description: b.Description,
			CardCount:   cardCount,
		}
	}

	return BoardListModel{Boards: summaries}, nil
}

// handleCreateBoard creates a new board.
func (h *Handler) handleCreateBoard(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	name, ok := p["name"].(string)
	if !ok || name == "" {
		return BoardListModel{Error: "Board name is required"}, nil
	}

	_, err := h.ws.CreateBoard(name)
	if err != nil {
		return BoardListModel{Error: err.Error()}, nil
	}

	// Git commit for board creation
	if h.git != nil {
		boardPath := h.ws.BoardPath(name)
		_ = h.git.Commit(boardPath, fmt.Sprintf("Create board: %s", name))
	}

	// Reload boards
	boards, err := h.ws.ListBoards()
	if err != nil {
		return BoardListModel{Error: err.Error()}, nil
	}

	summaries := make([]BoardSummary, len(boards))
	for i, b := range boards {
		cardCount := 0
		for _, col := range b.Columns {
			cardCount += len(col.Cards)
		}
		summaries[i] = BoardSummary{
			Name:        b.Name,
			Description: b.Description,
			CardCount:   cardCount,
		}
	}

	return BoardListModel{Boards: summaries}, nil
}

// handleDeleteBoard deletes a board.
func (h *Handler) handleDeleteBoard(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	name, ok := p["name"].(string)
	if !ok || name == "" {
		return BoardListModel{Error: "Board name is required"}, nil
	}

	err := h.ws.DeleteBoard(name)
	if err != nil {
		return BoardListModel{Error: err.Error()}, nil
	}

	// Git commit for board deletion
	if h.git != nil {
		boardPath := h.ws.BoardPath(name)
		_ = h.git.CommitRemove(boardPath, fmt.Sprintf("Delete board: %s", name))
	}

	// Reload boards
	boards, err := h.ws.ListBoards()
	if err != nil {
		return BoardListModel{Error: err.Error()}, nil
	}

	summaries := make([]BoardSummary, len(boards))
	for i, b := range boards {
		cardCount := 0
		for _, col := range b.Columns {
			cardCount += len(col.Cards)
		}
		summaries[i] = BoardSummary{
			Name:        b.Name,
			Description: b.Description,
			CardCount:   cardCount,
		}
	}

	return BoardListModel{Boards: summaries}, nil
}

// handleShowCreateForm shows the create board form.
func (h *Handler) handleShowCreateForm(_ context.Context, _ *live.Socket, _ live.Params) (interface{}, error) {
	return BoardListModel{Creating: true}, nil
}

// handleCancelCreate cancels the create board form.
func (h *Handler) handleCancelCreate(_ context.Context, _ *live.Socket, _ live.Params) (interface{}, error) {
	return BoardListModel{Creating: false}, nil
}
