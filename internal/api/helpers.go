package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
)

// ErrorResponse is the standard error payload.
type ErrorResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

// respond writes a JSON response with the given status code.
func respond(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// respondCreated writes a 201 JSON response.
func respondCreated(w http.ResponseWriter, v any) {
	respond(w, http.StatusCreated, v)
}

// respondNoContent writes a 204 response with no body.
func respondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// respondError writes a JSON error response.
func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, ErrorResponse{Error: msg, Status: status})
}

// handleError classifies an engine/workspace error and writes the appropriate response.
func handleError(w http.ResponseWriter, err error) {
	msg := err.Error()
	status := http.StatusInternalServerError
	if strings.Contains(msg, "not found") || strings.Contains(msg, "no such file") {
		status = http.StatusNotFound
	} else if strings.Contains(msg, "already exists") {
		status = http.StatusConflict
	} else if strings.Contains(msg, "out of range") {
		status = http.StatusBadRequest
	}
	respondError(w, status, msg)
}

// decodeJSON reads and decodes JSON from the request body into v.
func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	return dec.Decode(v)
}

// pathParam extracts and URL-decodes a chi URL parameter.
func pathParam(r *http.Request, name string) string {
	v := chi.URLParam(r, name)
	decoded, err := url.PathUnescape(v)
	if err != nil {
		return v
	}
	return decoded
}
