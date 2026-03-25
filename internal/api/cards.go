package api

import (
	"fmt"
	"net/http"
	"strconv"
)

func (s *Server) addCard(w http.ResponseWriter, r *http.Request) {
	boardName := pathParam(r, "board")
	colName := pathParam(r, "column")
	var body struct {
		Title string `json:"title"`
	}
	if err := decodeJSON(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Title == "" {
		respondError(w, http.StatusBadRequest, "title is required")
		return
	}

	boardPath, err := s.ws.BoardPath(boardName)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	card, err := s.ws.Engine.AddCard(boardPath, colName, body.Title, false)
	if err != nil {
		handleError(w, err)
		return
	}

	respondCreated(w, card)
}

type cardResponse struct {
	Title     string            `json:"title"`
	Completed bool              `json:"completed"`
	Tags      []string          `json:"tags,omitempty"`
	Assignee  string            `json:"assignee,omitempty"`
	Priority  string            `json:"priority,omitempty"`
	Due       string            `json:"due,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Body      string            `json:"body,omitempty"`
	Column    string            `json:"column"`
}

func (s *Server) getCard(w http.ResponseWriter, r *http.Request) {
	boardName := pathParam(r, "board")
	colIdx, cardIdx, err := cardIndicesFromRequest(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	boardPath, err := s.ws.BoardPath(boardName)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	card, colName, err := s.ws.Engine.ShowCard(boardPath, colIdx, cardIdx)
	if err != nil {
		handleError(w, err)
		return
	}

	respond(w, http.StatusOK, cardResponse{
		Title:     card.Title,
		Completed: card.Completed,
		Tags:      card.Tags,
		Assignee:  card.Assignee,
		Priority:  card.Priority,
		Due:       card.Due,
		Metadata:  card.Metadata,
		Body:      card.Body,
		Column:    colName,
	})
}

func (s *Server) deleteCard(w http.ResponseWriter, r *http.Request) {
	boardName := pathParam(r, "board")
	colIdx, cardIdx, err := cardIndicesFromRequest(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	boardPath, err := s.ws.BoardPath(boardName)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.ws.Engine.DeleteCard(boardPath, colIdx, cardIdx); err != nil {
		handleError(w, err)
		return
	}

	respondNoContent(w)
}

func (s *Server) moveCard(w http.ResponseWriter, r *http.Request) {
	boardName := pathParam(r, "board")
	colIdx, cardIdx, err := cardIndicesFromRequest(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	var body struct {
		Column string `json:"column"`
	}
	if decErr := decodeJSON(r, &body); decErr != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+decErr.Error())
		return
	}
	if body.Column == "" {
		respondError(w, http.StatusBadRequest, "column is required")
		return
	}

	boardPath, err := s.ws.BoardPath(boardName)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if moveErr := s.ws.Engine.MoveCard(boardPath, colIdx, cardIdx, body.Column); moveErr != nil {
		handleError(w, moveErr)
		return
	}

	respondNoContent(w)
}

func (s *Server) completeCard(w http.ResponseWriter, r *http.Request) {
	boardName := pathParam(r, "board")
	colIdx, cardIdx, err := cardIndicesFromRequest(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	boardPath, err := s.ws.BoardPath(boardName)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.ws.Engine.CompleteCard(boardPath, colIdx, cardIdx); err != nil {
		handleError(w, err)
		return
	}

	respondNoContent(w)
}

func (s *Server) tagCard(w http.ResponseWriter, r *http.Request) {
	boardName := pathParam(r, "board")
	colIdx, cardIdx, err := cardIndicesFromRequest(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	var body struct {
		Tags []string `json:"tags"`
	}
	if decErr := decodeJSON(r, &body); decErr != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+decErr.Error())
		return
	}
	if len(body.Tags) == 0 {
		respondError(w, http.StatusBadRequest, "tags is required")
		return
	}

	boardPath, err := s.ws.BoardPath(boardName)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.ws.Engine.TagCard(boardPath, colIdx, cardIdx, body.Tags); err != nil {
		handleError(w, err)
		return
	}

	respondNoContent(w)
}

func cardIndicesFromRequest(r *http.Request) (int, int, error) {
	colStr := pathParam(r, "colIdx")
	cardStr := pathParam(r, "cardIdx")

	colIdx, err := strconv.Atoi(colStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid column index: %s", colStr)
	}
	cardIdx, err := strconv.Atoi(cardStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid card index: %s", cardStr)
	}
	return colIdx, cardIdx, nil
}
