package api

import (
	"fmt"
	"net/http"
	"path/filepath"
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

	boardPath := s.ws.BoardPath(boardName)
	card, err := s.ws.Engine.AddCard(boardPath, colName, body.Title)
	if err != nil {
		handleError(w, err)
		return
	}

	s.gitCommit(filepath.Base(boardPath), fmt.Sprintf("card: add %q to %s/%s", body.Title, boardName, colName))
	respondCreated(w, card)
}

type cardResponse struct {
	ID        string            `json:"id"`
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
	cardID := pathParam(r, "id")
	board, err := s.ws.FindBoardByCardID(cardID)
	if err != nil {
		handleError(w, err)
		return
	}

	card, colName, err := s.ws.Engine.ShowCard(board.FilePath, cardID)
	if err != nil {
		handleError(w, err)
		return
	}

	respond(w, http.StatusOK, cardResponse{
		ID:        card.ID,
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
	cardID := pathParam(r, "id")
	board, err := s.ws.FindBoardByCardID(cardID)
	if err != nil {
		handleError(w, err)
		return
	}

	if err := s.ws.Engine.DeleteCard(board.FilePath, cardID); err != nil {
		handleError(w, err)
		return
	}

	s.gitCommit(filepath.Base(board.FilePath), fmt.Sprintf("card: delete %s", shortID(cardID)))
	respondNoContent(w)
}

func (s *Server) moveCard(w http.ResponseWriter, r *http.Request) {
	cardID := pathParam(r, "id")
	var body struct {
		Column string `json:"column"`
	}
	if err := decodeJSON(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Column == "" {
		respondError(w, http.StatusBadRequest, "column is required")
		return
	}

	board, err := s.ws.FindBoardByCardID(cardID)
	if err != nil {
		handleError(w, err)
		return
	}

	if err := s.ws.Engine.MoveCard(board.FilePath, cardID, body.Column); err != nil {
		handleError(w, err)
		return
	}

	s.gitCommit(filepath.Base(board.FilePath), fmt.Sprintf("card: move %s → %s", shortID(cardID), body.Column))
	respondNoContent(w)
}

func (s *Server) completeCard(w http.ResponseWriter, r *http.Request) {
	cardID := pathParam(r, "id")
	board, err := s.ws.FindBoardByCardID(cardID)
	if err != nil {
		handleError(w, err)
		return
	}

	if err := s.ws.Engine.CompleteCard(board.FilePath, cardID); err != nil {
		handleError(w, err)
		return
	}

	s.gitCommit(filepath.Base(board.FilePath), fmt.Sprintf("card: complete %s", shortID(cardID)))
	respondNoContent(w)
}

func (s *Server) tagCard(w http.ResponseWriter, r *http.Request) {
	cardID := pathParam(r, "id")
	var body struct {
		Tags []string `json:"tags"`
	}
	if err := decodeJSON(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if len(body.Tags) == 0 {
		respondError(w, http.StatusBadRequest, "tags is required")
		return
	}

	board, err := s.ws.FindBoardByCardID(cardID)
	if err != nil {
		handleError(w, err)
		return
	}

	if err := s.ws.Engine.TagCard(board.FilePath, cardID, body.Tags); err != nil {
		handleError(w, err)
		return
	}

	s.gitCommit(filepath.Base(board.FilePath), fmt.Sprintf("card: tag %s", shortID(cardID)))
	respondNoContent(w)
}

func (s *Server) patchCard(w http.ResponseWriter, r *http.Request) {
	cardID := pathParam(r, "id")
	var req struct {
		Body *string `json:"body"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	board, err := s.ws.FindBoardByCardID(cardID)
	if err != nil {
		handleError(w, err)
		return
	}

	if req.Body != nil {
		if err := s.ws.Engine.UpdateCardBody(board.FilePath, cardID, *req.Body); err != nil {
			handleError(w, err)
			return
		}
		s.gitCommit(filepath.Base(board.FilePath), fmt.Sprintf("card: update body %s", shortID(cardID)))
	}

	respondNoContent(w)
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
