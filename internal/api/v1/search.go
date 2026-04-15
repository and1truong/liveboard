package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type searchHitDTO struct {
	BoardID   string `json:"board_id"`
	BoardName string `json:"board_name"`
	ColIdx    int    `json:"col_idx"`
	CardIdx   int    `json:"card_idx"`
	CardID    string `json:"card_id"`
	CardTitle string `json:"card_title"`
	Snippet   string `json:"snippet"`
}

func (d Deps) getSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if len(q) > 256 {
		writeError(w, fmt.Errorf("%w: query too long", errInvalid))
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if q == "" || d.Search == nil {
		_ = json.NewEncoder(w).Encode([]searchHitDTO{})
		return
	}
	hits, err := d.Search.Search(q, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]searchHitDTO, 0, len(hits))
	for _, h := range hits {
		out = append(out, searchHitDTO{
			BoardID:   h.BoardID,
			BoardName: h.BoardName,
			ColIdx:    h.ColIdx,
			CardIdx:   h.CardIdx,
			CardID:    h.CardID,
			CardTitle: h.CardTitle,
			Snippet:   h.Snippet,
		})
	}
	_ = json.NewEncoder(w).Encode(out)
}
