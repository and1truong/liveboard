package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type backlinkHitDTO struct {
	BoardID   string `json:"board_id"`
	BoardName string `json:"board_name"`
	ColIdx    int    `json:"col_idx"`
	CardIdx   int    `json:"card_idx"`
	CardTitle string `json:"card_title"`
}

func (d Deps) getBacklinks(w http.ResponseWriter, r *http.Request) {
	cardID := chi.URLParam(r, "cardId")
	if cardID == "" {
		writeError(w, fmt.Errorf("%w: cardId required", errInvalid))
		return
	}
	if d.Search == nil {
		_ = json.NewEncoder(w).Encode([]backlinkHitDTO{})
		return
	}
	hits, err := d.Search.Backlinks(cardID)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]backlinkHitDTO, 0, len(hits))
	for _, h := range hits {
		out = append(out, backlinkHitDTO{
			BoardID:   h.BoardID,
			BoardName: h.BoardName,
			ColIdx:    h.ColIdx,
			CardIdx:   h.CardIdx,
			CardTitle: h.CardTitle,
		})
	}
	_ = json.NewEncoder(w).Encode(out)
}
