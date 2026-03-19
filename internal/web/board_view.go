package web

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/jfyne/live"

	"github.com/and1truong/liveboard/pkg/models"
)

// BoardViewModel is the state for the board view page.
type BoardViewModel struct {
	Title     string         `json:"title"`
	Board     *models.Board  `json:"board"`
	BoardName string         `json:"board_name"`
	BoardSlug string         `json:"board_slug"` // filename stem for loading
	Boards    []BoardSummary `json:"boards"`
	Error     string         `json:"error,omitempty"`
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

// mutateBoard runs op, commits with msg, publishes, and returns the view model.
func (h *Handler) mutateBoard(slug, msg string, op func(string) error) (interface{}, error) {
	boardPath := h.ws.BoardPath(slug)
	if err := op(boardPath); err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}
	h.commitWithHandling(boardPath, msg)
	h.publishBoardEvent(slug)
	return h.boardViewModel(slug)
}

// mutateBoardRemove runs op, commits a removal, publishes, and returns the view model.
func (h *Handler) mutateBoardRemove(slug, msg string, op func(string) error) (interface{}, error) {
	boardPath := h.ws.BoardPath(slug)
	if err := op(boardPath); err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}
	h.commitRemoveWithHandling(boardPath, msg)
	h.publishBoardEvent(slug)
	return h.boardViewModel(slug)
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

// intParam extracts an integer parameter from event params.
func intParam(p live.Params, key string) (int, error) {
	s, ok := p[key].(string)
	if !ok || s == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	return strconv.Atoi(s)
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

	return h.mutateBoard(slug, fmt.Sprintf("Add card \"%s\" to %s", title, column), func(boardPath string) error {
		_, err := h.eng.AddCard(boardPath, column, title)
		return err
	})
}

// handleMoveCard moves a card to a different column.
func (h *Handler) handleMoveCard(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	colIdx, err := intParam(p, "col_idx")
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	cardIdx, err := intParam(p, "card_idx")
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	targetColumn, ok := p["target_column"].(string)
	if !ok || targetColumn == "" {
		return BoardViewModel{Error: "Target column is required"}, nil
	}

	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	return h.mutateBoard(slug, fmt.Sprintf("Move card to %s", targetColumn), func(boardPath string) error {
		return h.eng.MoveCard(boardPath, colIdx, cardIdx, targetColumn)
	})
}

// handleReorderCard moves a card to a specific position within a column.
func (h *Handler) handleReorderCard(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	colIdx, err := intParam(p, "col_idx")
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	cardIdx, err := intParam(p, "card_idx")
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	column, ok := p["column"].(string)
	if !ok || column == "" {
		return BoardViewModel{Error: "Column is required"}, nil
	}

	beforeIdx := -1
	if s, ok := p["before_idx"].(string); ok && s != "" {
		beforeIdx, _ = strconv.Atoi(s)
	}

	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	return h.mutateBoard(slug, fmt.Sprintf("Reorder card in %s", column), func(boardPath string) error {
		return h.eng.ReorderCard(boardPath, colIdx, cardIdx, beforeIdx, column)
	})
}

// handleDeleteCard deletes a card.
func (h *Handler) handleDeleteCard(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	colIdx, err := intParam(p, "col_idx")
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	cardIdx, err := intParam(p, "card_idx")
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	return h.mutateBoardRemove(slug, "Delete card", func(boardPath string) error {
		return h.eng.DeleteCard(boardPath, colIdx, cardIdx)
	})
}

// handleToggleComplete marks a card as completed.
func (h *Handler) handleToggleComplete(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	colIdx, err := intParam(p, "col_idx")
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	cardIdx, err := intParam(p, "card_idx")
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	return h.mutateBoard(slug, "Toggle card complete", func(boardPath string) error {
		return h.eng.CompleteCard(boardPath, colIdx, cardIdx)
	})
}

// handleEditCard updates a card's title, body, and tags.
func (h *Handler) handleEditCard(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	colIdx, err := intParam(p, "col_idx")
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	cardIdx, err := intParam(p, "card_idx")
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	title, _ := p["title"].(string)
	body, _ := p["body"].(string)
	tagsRaw, _ := p["tags"].(string)
	priority, _ := p["priority"].(string)

	var tags []string
	for _, t := range strings.Split(tagsRaw, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}

	return h.mutateBoard(slug, "Edit card", func(boardPath string) error {
		return h.eng.EditCard(boardPath, colIdx, cardIdx, title, body, tags, priority)
	})
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

	return h.mutateBoard(slug, fmt.Sprintf("Add column: %s", colName), func(boardPath string) error {
		return h.eng.AddColumn(boardPath, colName)
	})
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

	return h.mutateBoard(slug, fmt.Sprintf("Rename column %q to %q", oldName, newName), func(boardPath string) error {
		return h.eng.RenameColumn(boardPath, oldName, newName)
	})
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

	return h.mutateBoardRemove(slug, fmt.Sprintf("Delete column: %s", colName), func(boardPath string) error {
		return h.eng.DeleteColumn(boardPath, colName)
	})
}

// handleUpdateBoardMeta updates a board's name and description.
func (h *Handler) handleUpdateBoardMeta(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	name, _ := p["board_name"].(string)
	description, _ := p["description"].(string)

	return h.mutateBoard(slug, fmt.Sprintf("Update board meta: %s", name), func(boardPath string) error {
		return h.eng.UpdateBoardMeta(boardPath, name, description)
	})
}

// handleToggleColumnCollapse toggles the collapsed state of a column.
func (h *Handler) handleToggleColumnCollapse(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	slug, ok := slugFromParams(p)
	if !ok {
		return BoardViewModel{Error: "Board name is required"}, nil
	}

	colIndex, err := intParam(p, "col_index")
	if err != nil {
		return BoardViewModel{Error: err.Error()}, nil
	}

	return h.mutateBoard(slug, "Toggle column collapse", func(boardPath string) error {
		return h.eng.ToggleColumnCollapse(boardPath, colIndex)
	})
}

// handleBoardUpdate handles PubSub messages for real-time updates.
func (h *Handler) handleBoardUpdate(_ context.Context, _ *live.Socket, msg any) (interface{}, error) {
	slug, ok := msg.(string)
	if !ok {
		return nil, fmt.Errorf("invalid message type")
	}

	return h.boardViewModel(slug)
}
