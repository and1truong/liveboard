// Package api implements the LiveBoard REST API server and handlers.
package api

import (
	"net/http"
)

func (s *Server) listBoards(w http.ResponseWriter, _ *http.Request) {
	boards, err := s.ws.ListBoards()
	if err != nil {
		handleError(w, err)
		return
	}
	// Return empty array instead of null.
	if boards == nil {
		respond(w, http.StatusOK, []struct{}{})
		return
	}
	respond(w, http.StatusOK, boards)
}

func (s *Server) createBoard(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	board, err := s.ws.CreateBoard(body.Name)
	if err != nil {
		handleError(w, err)
		return
	}

	respondCreated(w, board)
}

func (s *Server) getBoard(w http.ResponseWriter, r *http.Request) {
	name := pathParam(r, "board")
	board, err := s.ws.LoadBoard(name)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, board)
}

func (s *Server) deleteBoard(w http.ResponseWriter, r *http.Request) {
	name := pathParam(r, "board")

	if err := s.ws.DeleteBoard(name); err != nil {
		handleError(w, err)
		return
	}

	respondNoContent(w)
}
