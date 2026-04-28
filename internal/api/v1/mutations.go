package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/pkg/models"
)

type mutationRequest struct {
	ClientVersion int              `json:"client_version"`
	Op            board.MutationOp `json:"op"`
}

func (d Deps) postMutation(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	slug := chi.URLParam(r, "*")

	boardPath, pathErr := d.Workspace.BoardPath(slug)
	if pathErr != nil {
		writeError(w, pathErr)
		return
	}

	var req mutationRequest
	if decodeErr := decodeJSON(r, &req); decodeErr != nil {
		writeError(w, fmt.Errorf("%w: %v", errInvalid, decodeErr))
		return
	}

	if req.Op.Type == "move_card_to_board" && req.Op.MoveCardToBoard != nil {
		d.handleMoveCardToBoard(w, slug, boardPath, req.ClientVersion, req.Op.MoveCardToBoard)
		return
	}

	var updated *models.Board
	dispatchErr := d.Engine.MutateBoard(boardPath, req.ClientVersion, func(b *models.Board) error {
		if e := board.ApplyMutation(b, req.Op); e != nil {
			return e
		}
		updated = b
		return nil
	})
	if dispatchErr != nil {
		writeError(w, dispatchErr)
		return
	}

	if d.SSE != nil {
		d.SSE.Publish(slug)
	}
	if d.Search != nil && updated != nil {
		_ = d.Search.UpdateBoard(slug, updated)
	}

	_ = json.NewEncoder(w).Encode(updated)
}

func (d Deps) handleMoveCardToBoard(w http.ResponseWriter, slug, boardPath string, clientVersion int, p *board.MoveCardToBoardOp) {
	dstPath, pathErr := d.Workspace.BoardPath(p.DstBoard)
	if pathErr != nil {
		writeError(w, pathErr)
		return
	}
	if moveErr := d.Engine.MoveCardToBoard(boardPath, clientVersion, p.ColIdx, p.CardIdx, dstPath, p.DstColumn); moveErr != nil {
		writeError(w, moveErr)
		return
	}
	updated, loadErr := d.Engine.LoadBoard(boardPath)
	if loadErr != nil {
		writeError(w, loadErr)
		return
	}
	if d.SSE != nil {
		d.SSE.Publish(slug)
		d.SSE.Publish(p.DstBoard)
	}
	if d.Search != nil {
		_ = d.Search.UpdateBoard(slug, updated)
		if dst, err := d.Engine.LoadBoard(dstPath); err == nil {
			_ = d.Search.UpdateBoard(p.DstBoard, dst)
		}
	}
	_ = json.NewEncoder(w).Encode(updated)
}
