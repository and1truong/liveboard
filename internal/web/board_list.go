package web

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

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

// BoardListPage handles GET / — renders the full board list page.
func (h *Handler) BoardListPage(w http.ResponseWriter, _ *http.Request) {
	model, _ := h.boardListModel()
	renderFullPage(w, h.boardListTpl, model)
}

// HandleCreateBoard handles POST /boards/new — creates a board and returns the boards grid partial.
func (h *Handler) HandleCreateBoard(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if name == "" {
		model, _ := h.boardListModel()
		model.Error = "Board name is required"
		renderPartial(w, h.boardGridTpl, "boards-grid", model)
		return
	}

	if _, err := h.ws.CreateBoard(name); err != nil {
		model, _ := h.boardListModel()
		model.Error = err.Error()
		renderPartial(w, h.boardGridTpl, "boards-grid", model)
		return
	}

	boardPath := h.ws.BoardPath(name)
	h.commitWithHandling(boardPath, fmt.Sprintf("Create board: %s", name))

	model, _ := h.boardListModel()
	renderPartial(w, h.boardGridTpl, "boards-grid", model)
}

// HandleDeleteBoard handles POST /boards/{slug}/delete — deletes a board.
func (h *Handler) HandleDeleteBoard(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if name == "" {
		model, _ := h.boardListModel()
		model.Error = "Board name is required"
		renderPartial(w, h.boardGridTpl, "boards-grid", model)
		return
	}

	if err := h.ws.DeleteBoard(name); err != nil {
		model, _ := h.boardListModel()
		model.Error = err.Error()
		renderPartial(w, h.boardGridTpl, "boards-grid", model)
		return
	}

	boardPath := h.ws.BoardPath(name)
	h.commitRemoveWithHandling(boardPath, fmt.Sprintf("Delete board: %s", name))

	model, _ := h.boardListModel()
	renderPartial(w, h.boardGridTpl, "boards-grid", model)
}

// HandleSetBoardIconList handles POST /boards/{slug}/icon from the board list page.
func (h *Handler) HandleSetBoardIconList(w http.ResponseWriter, r *http.Request) {
	slug := r.FormValue("name")
	if slug == "" {
		model, _ := h.boardListModel()
		model.Error = "Board name is required"
		renderPartial(w, h.boardGridTpl, "boards-grid", model)
		return
	}

	icon := r.FormValue("icon")

	boardPath := h.ws.BoardPath(slug)
	if err := h.eng.UpdateBoardIcon(boardPath, icon); err != nil {
		model, _ := h.boardListModel()
		model.Error = err.Error()
		renderPartial(w, h.boardGridTpl, "boards-grid", model)
		return
	}
	h.commitWithHandling(boardPath, "Set board icon")

	model, _ := h.boardListModel()
	renderPartial(w, h.boardGridTpl, "boards-grid", model)
}
