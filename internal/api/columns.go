package api

import (
	"net/http"
)

func (s *Server) addColumn(w http.ResponseWriter, r *http.Request) {
	boardName := pathParam(r, "board")
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

	boardPath, err := s.ws.BoardPath(boardName)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.ws.Engine.AddColumn(boardPath, body.Name); err != nil {
		handleError(w, err)
		return
	}

	respondCreated(w, struct {
		Name string `json:"name"`
	}{Name: body.Name})
}

func (s *Server) deleteColumn(w http.ResponseWriter, r *http.Request) {
	boardName := pathParam(r, "board")
	colName := pathParam(r, "column")
	boardPath, err := s.ws.BoardPath(boardName)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.ws.Engine.DeleteColumn(boardPath, colName); err != nil {
		handleError(w, err)
		return
	}

	respondNoContent(w)
}

func (s *Server) moveColumn(w http.ResponseWriter, r *http.Request) {
	boardName := pathParam(r, "board")
	colName := pathParam(r, "column")
	var body struct {
		After string `json:"after"`
	}
	if err := decodeJSON(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.After == "" {
		respondError(w, http.StatusBadRequest, "after is required")
		return
	}

	boardPath, err := s.ws.BoardPath(boardName)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.ws.Engine.MoveColumn(boardPath, colName, body.After); err != nil {
		handleError(w, err)
		return
	}

	respondNoContent(w)
}
