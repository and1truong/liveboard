package web

import (
	"context"

	"github.com/jfyne/live"

	"github.com/and1truong/liveboard/internal/search"
)

// SearchModel is the state for the search page.
type SearchModel struct {
	Title     string                `json:"title"`
	Query     string                `json:"query"`
	Results   []search.SearchResult `json:"results"`
	Boards    []BoardSummary        `json:"boards"`
	BoardSlug string                `json:"board_slug"`
	Error     string                `json:"error,omitempty"`
}

// mountSearch initializes the search model.
func (h *Handler) mountSearch(_ context.Context, _ *live.Socket) (interface{}, error) {
	allBoards, _ := h.ws.ListBoards()
	return SearchModel{
		Title:     "Search — LiveBoard",
		BoardSlug: "__search__",
		Boards:    toBoardSummaries(allBoards),
	}, nil
}

// handleSearch performs a live search query.
func (h *Handler) handleSearch(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
	allBoards, _ := h.ws.ListBoards()
	query, _ := p["query"].(string)

	model := SearchModel{
		Title:     "Search — LiveBoard",
		Query:     query,
		BoardSlug: "__search__",
		Boards:    toBoardSummaries(allBoards),
	}

	if h.search == nil {
		model.Error = "Search index not available"
		return model, nil
	}

	if query == "" {
		return model, nil
	}

	results, err := h.search.Search(query, 50)
	if err != nil {
		model.Error = err.Error()
		return model, nil
	}

	model.Results = results
	return model, nil
}
