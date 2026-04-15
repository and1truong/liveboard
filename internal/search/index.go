// Package search wraps bleve to provide per-card full-text indexing.
package search

import (
	"fmt"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"

	"github.com/and1truong/liveboard/pkg/models"
)

// Hit is a single search result pointing at a card by (board, col, card) index.
type Hit struct {
	BoardID   string
	BoardName string
	ColIdx    int
	CardIdx   int
	CardTitle string
	Snippet   string
}

type doc struct {
	BoardID   string   `json:"board_id"`
	BoardName string   `json:"board_name"`
	ColIdx    int      `json:"col_idx"`
	CardIdx   int      `json:"card_idx"`
	CardID    string   `json:"card_id"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Tags      []string `json:"tags"`
}

// Index is an in-memory bleve index of cards across all boards.
type Index struct {
	idx bleve.Index
}

// New creates an empty in-memory search index.
func New() (*Index, error) {
	mapping := bleve.NewIndexMapping()
	idx, err := bleve.NewMemOnly(mapping)
	if err != nil {
		return nil, err
	}
	return &Index{idx: idx}, nil
}

// UpdateBoard purges any existing docs for slug and re-indexes the board's cards.
func (i *Index) UpdateBoard(slug string, b *models.Board) error {
	if err := i.DeleteBoard(slug); err != nil {
		return err
	}
	boardName := b.Name
	if boardName == "" {
		boardName = slug
	}
	for cIdx, col := range b.Columns {
		for kIdx, c := range col.Cards {
			d := doc{
				BoardID:   slug,
				BoardName: boardName,
				ColIdx:    cIdx,
				CardIdx:   kIdx,
				CardID:    c.ID,
				Title:     c.Title,
				Body:      c.Body,
				Tags:      c.Tags,
			}
			id := fmt.Sprintf("%s:%d:%d", slug, cIdx, kIdx)
			if err := i.idx.Index(id, d); err != nil {
				return err
			}
		}
	}
	return nil
}

// DeleteBoard removes every doc whose ID has the slug prefix.
func (i *Index) DeleteBoard(slug string) error {
	prefix := slug + ":"
	q := bleve.NewTermQuery(slug)
	q.SetField("board_id")
	sr := bleve.NewSearchRequestOptions(q, 1000, 0, false)
	sr.Fields = []string{"board_id"}
	res, err := i.idx.Search(sr)
	if err != nil {
		return err
	}
	if res == nil {
		return nil
	}
	for _, h := range res.Hits {
		if h == nil {
			continue
		}
		if strings.HasPrefix(h.ID, prefix) {
			if err := i.idx.Delete(h.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// Search runs a query string against the index and returns hits with snippets.
func (i *Index) Search(query string, limit int) ([]Hit, error) {
	if limit <= 0 {
		limit = 20
	}
	q := bleve.NewQueryStringQuery(query)
	sr := bleve.NewSearchRequestOptions(q, limit, 0, false)
	sr.Highlight = bleve.NewHighlight()
	sr.Highlight.AddField("title")
	sr.Highlight.AddField("body")
	sr.Fields = []string{"board_id", "board_name", "col_idx", "card_idx", "title"}
	res, err := i.idx.Search(sr)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return []Hit{}, nil
	}
	hits := make([]Hit, 0, len(res.Hits))
	for _, h := range res.Hits {
		if h == nil {
			continue
		}
		hits = append(hits, Hit{
			BoardID:   getString(h.Fields, "board_id"),
			BoardName: getString(h.Fields, "board_name"),
			ColIdx:    getInt(h.Fields, "col_idx"),
			CardIdx:   getInt(h.Fields, "card_idx"),
			CardTitle: getString(h.Fields, "title"),
			Snippet:   firstSnippet(h.Fragments),
		})
	}
	return hits, nil
}

func getString(m map[string]interface{}, k string) string {
	if v, ok := m[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]interface{}, k string) int {
	if v, ok := m[k]; ok {
		if f, ok := v.(float64); ok {
			return int(f)
		}
	}
	return 0
}

func firstSnippet(frags search.FieldFragmentMap) string {
	for _, field := range []string{"body", "title"} {
		if list, ok := frags[field]; ok && len(list) > 0 {
			return list[0]
		}
	}
	return ""
}
