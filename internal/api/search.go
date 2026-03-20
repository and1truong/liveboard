package api

import (
	"net/http"
	"strconv"

	"github.com/and1truong/liveboard/internal/search"
)

// searchHandler handles GET /search?q=term&limit=20
func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		respondError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	if s.search == nil {
		respondError(w, http.StatusServiceUnavailable, "search index not available")
		return
	}

	results, err := s.search.Search(q, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if results == nil {
		results = []search.SearchResult{}
	}

	w.Header().Set("Content-Type", "application/json")
	respond(w, http.StatusOK, results)
}
