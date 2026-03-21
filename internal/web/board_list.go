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
	Title     string         `json:"title"`
	SiteName  string         `json:"site_name"`
	Boards    []BoardSummary `json:"boards"`
	BoardSlug string         `json:"board_slug"` // always empty; shared with layout template
	Error     string         `json:"error,omitempty"`
}

// BoardSummary represents a simplified board for list display.
type BoardSummary struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"` // filename stem, used for URLs
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
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
			Icon:        b.Icon,
			CardCount:   cardCount,
		}
	}
	return summaries
}

// boardListModel loads the board list and returns a populated BoardListModel.
func (h *Handler) boardListModel() (BoardListModel, error) {
	boards, err := h.ws.ListBoards()
	if err != nil {
		return BoardListModel{Error: err.Error()}, nil
	}
	siteName := h.loadSettings().SiteName
	return BoardListModel{Title: siteName, SiteName: siteName, Boards: toBoardSummaries(boards)}, nil
}

// mountBoardList initializes the board list model.
func (h *Handler) mountBoardList(_ context.Context, _ *live.Socket) (interface{}, error) {
	return h.boardListModel()
}

// handleParams handles parameter changes (e.g., URL params).
func (h *Handler) handleParams(_ context.Context, _ *live.Socket, _ live.Params) (interface{}, error) {
	return h.boardListModel()
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

	boardPath := h.ws.BoardPath(name)
	h.commitWithHandling(boardPath, fmt.Sprintf("Create board: %s", name))

	return h.boardListModel()
}

// handleSetBoardIconList sets the emoji icon for a board (from the board list page).
func (h *Handler) handleSetBoardIconList(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	slug, ok := slugFromParams(p)
	if !ok || slug == "" {
		return BoardListModel{Error: "Board name is required"}, nil
	}

	icon, _ := p["icon"].(string)

	boardPath := h.ws.BoardPath(slug)
	if err := h.eng.UpdateBoardIcon(boardPath, icon); err != nil {
		return BoardListModel{Error: err.Error()}, nil
	}
	h.commitWithHandling(boardPath, "Set board icon")

	return h.boardListModel()
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

	boardPath := h.ws.BoardPath(name)
	h.commitRemoveWithHandling(boardPath, fmt.Sprintf("Delete board: %s", name))

	return h.boardListModel()
}
