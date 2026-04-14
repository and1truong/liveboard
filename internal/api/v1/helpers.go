package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/workspace"
)

type errorResponse struct {
	Error  string `json:"error"`
	Code   string `json:"code"`
	Status int    `json:"status"`
}

// errInvalid is a generic "bad request" sentinel for v1-layer validation.
var errInvalid = errors.New("invalid request")

// writeError maps engine/workspace errors to HTTP status + canonical error code.
// Codes are the protocol-level codes used by the shell adapter and must stay
// consistent with the design spec.
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "INTERNAL"
	switch {
	case errors.Is(err, board.ErrNotFound), errors.Is(err, os.ErrNotExist):
		status, code = http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, board.ErrVersionConflict):
		status, code = http.StatusConflict, "VERSION_CONFLICT"
	case errors.Is(err, workspace.ErrAlreadyExists):
		status, code = http.StatusConflict, "ALREADY_EXISTS"
	case errors.Is(err, board.ErrOutOfRange):
		status, code = http.StatusBadRequest, "OUT_OF_RANGE"
	case errors.Is(err, workspace.ErrInvalidBoardName), errors.Is(err, board.ErrInvalidInput), errors.Is(err, errInvalid):
		status, code = http.StatusBadRequest, "INVALID"
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: err.Error(), Code: code, Status: status})
}

func decodeJSON(r *http.Request, v any) error { //nolint:unused
	return json.NewDecoder(r.Body).Decode(v)
}
