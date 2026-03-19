package web

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jfyne/live"

	"github.com/and1truong/liveboard/pkg/models"
)

// BoardListModel is the state for the board list page.
type BoardListModel struct {
	Title       string         `json:"title"`
	Boards      []BoardSummary `json:"boards"`
	BoardSlug   string         `json:"board_slug"` // always empty; shared with layout template
	Error       string         `json:"error,omitempty"`
	Creating    bool           `json:"creating,omitempty"`
	NeedsReload bool           `json:"needs_reload,omitempty"`
}

// BoardSummary represents a simplified board for list display.
type BoardSummary struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"` // filename stem, used for URLs
	Description string `json:"description,omitempty"`
	CardCount   int    `json:"card_count"`
}

// boardSlug extracts the filename stem from a board's FilePath.
func boardSlug(b models.Board) string {
	return strings.TrimSuffix(filepath.Base(b.FilePath), ".md")
}

// toBoardSummaries converts a slice of boards to BoardSummary.
func toBoardSummaries(boards []models.Board) []BoardSummary {
	summaries := make([]BoardSummary, len(boards))
	for i, b := range boards {
		cardCount := 0
		for _, col := range b.Columns {
			cardCount += len(col.Cards)
		}
		summaries[i] = BoardSummary{
			Name:        b.Name,
			Slug:        boardSlug(b),
			Description: b.Description,
			CardCount:   cardCount,
		}
	}
	return summaries
}

// mountBoardList initializes the board list model.
func (h *Handler) mountBoardList(_ context.Context, _ *live.Socket) (interface{}, error) {
	boards, err := h.ws.ListBoards()
	if err != nil {
		return BoardListModel{Error: err.Error()}, nil
	}

	return BoardListModel{Title: "LiveBoard", Boards: toBoardSummaries(boards)}, nil
}

// handleParams handles parameter changes (e.g., URL params).
func (h *Handler) handleParams(_ context.Context, _ *live.Socket, _ live.Params) (interface{}, error) {
	boards, err := h.ws.ListBoards()
	if err != nil {
		return BoardListModel{Error: err.Error()}, nil
	}

	return BoardListModel{Title: "LiveBoard", Boards: toBoardSummaries(boards)}, nil
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

	return BoardListModel{Boards: toBoardSummaries(boards)}, nil
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

	return BoardListModel{Boards: toBoardSummaries(boards)}, nil
}

// handleShowCreateForm shows the create board form.
func (h *Handler) handleShowCreateForm(_ context.Context, _ *live.Socket, _ live.Params) (interface{}, error) {
	return BoardListModel{Creating: true}, nil
}

// handleCancelCreate cancels the create board form.
func (h *Handler) handleCancelCreate(_ context.Context, _ *live.Socket, _ live.Params) (interface{}, error) {
	return BoardListModel{Creating: false}, nil
}
