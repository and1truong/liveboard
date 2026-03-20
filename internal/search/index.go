// Package search provides full-text search over board cards using Bleve.
// It supports multilingual content with BM25 ranking.
package search

import (
	"fmt"
	"strings"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/lang/cjk"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/token/unicodenorm"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/highlight/highlighter/ansi"

	"github.com/and1truong/liveboard/pkg/models"
)

// SearchResult represents a single search hit.
type SearchResult struct {
	BoardSlug  string            `json:"board_slug"`
	BoardName  string            `json:"board_name"`
	ColumnName string            `json:"column_name"`
	ColIdx     int               `json:"col_idx"`
	CardIdx    int               `json:"card_idx"`
	CardTitle  string            `json:"card_title"`
	Score      float64           `json:"score"`
	Fragments  map[string]string `json:"fragments,omitempty"`
}

// cardDocument is the Bleve-indexable representation of a card.
type cardDocument struct {
	BoardSlug  string `json:"board_slug"`
	BoardName  string `json:"board_name"`
	ColumnName string `json:"column_name"`
	ColIdx     int    `json:"col_idx"`
	CardIdx    int    `json:"card_idx"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	Tags       string `json:"tags"`
	Assignee   string `json:"assignee"`
	Priority   string `json:"priority"`
}

// Index wraps a Bleve index for card search.
type Index struct {
	idx bleve.Index
	mu  sync.RWMutex
}

// supportedLanguages maps language codes to Bleve analyzer names.
var supportedLanguages = map[string]string{
	"ar": "ar", "bg": "bg", "ca": "ca", "cjk": "cjk",
	"ckb": "ckb", "cs": "cs", "da": "da", "de": "de",
	"el": "el", "en": "en", "es": "es", "eu": "eu",
	"fa": "fa", "fi": "fi", "fr": "fr", "ga": "ga",
	"gl": "gl", "hi": "hi", "hu": "hu", "hy": "hy",
	"id": "id", "in": "in", "it": "it", "nl": "nl",
	"no": "no", "pt": "pt", "ro": "ro", "ru": "ru",
	"sv": "sv", "tr": "tr",
}

// NewIndex creates an in-memory Bleve index with a multilingual-aware mapping.
// The lang parameter selects a language-specific analyzer (e.g. "en", "fr", "cjk").
// If empty or unsupported, a unicode-based default analyzer is used.
func NewIndex(lang string) (*Index, error) {
	indexMapping, err := buildMapping(lang)
	if err != nil {
		return nil, fmt.Errorf("build mapping: %w", err)
	}

	idx, err := bleve.NewMemOnly(indexMapping)
	if err != nil {
		return nil, fmt.Errorf("create index: %w", err)
	}

	return &Index{idx: idx}, nil
}

// buildMapping creates the Bleve index mapping with a custom or language analyzer.
func buildMapping(lang string) (mapping.IndexMapping, error) {
	indexMapping := bleve.NewIndexMapping()

	analyzerName := "standard"
	if lang != "" {
		if _, ok := supportedLanguages[lang]; ok {
			analyzerName = lang
		}
	}

	// Register a custom multilingual analyzer that handles unicode normalization
	// and CJK bigrams as a fallback when no specific language is set.
	if analyzerName == "standard" {
		err := indexMapping.AddCustomAnalyzer("multilingual", map[string]interface{}{
			"type":          custom.Name,
			"tokenizer":     unicode.Name,
			"token_filters": []string{unicodenorm.Name, lowercase.Name, cjk.BigramName},
		})
		if err != nil {
			return nil, err
		}
		analyzerName = "multilingual"
	}

	// Card document mapping
	cardMapping := bleve.NewDocumentMapping()

	titleField := bleve.NewTextFieldMapping()
	titleField.Analyzer = analyzerName
	titleField.Store = true
	cardMapping.AddFieldMappingsAt("title", titleField)

	bodyField := bleve.NewTextFieldMapping()
	bodyField.Analyzer = analyzerName
	bodyField.Store = true
	cardMapping.AddFieldMappingsAt("body", bodyField)

	tagsField := bleve.NewTextFieldMapping()
	tagsField.Analyzer = "keyword"
	tagsField.Store = true
	cardMapping.AddFieldMappingsAt("tags", tagsField)

	assigneeField := bleve.NewTextFieldMapping()
	assigneeField.Analyzer = "keyword"
	assigneeField.Store = true
	cardMapping.AddFieldMappingsAt("assignee", assigneeField)

	priorityField := bleve.NewTextFieldMapping()
	priorityField.Analyzer = "keyword"
	priorityField.Store = true
	cardMapping.AddFieldMappingsAt("priority", priorityField)

	// Stored-only fields (not searchable, used for result display)
	for _, f := range []string{"board_slug", "board_name", "column_name"} {
		fm := bleve.NewTextFieldMapping()
		fm.Index = false
		fm.Store = true
		cardMapping.AddFieldMappingsAt(f, fm)
	}

	for _, f := range []string{"col_idx", "card_idx"} {
		fm := bleve.NewNumericFieldMapping()
		fm.Index = false
		fm.Store = true
		cardMapping.AddFieldMappingsAt(f, fm)
	}

	indexMapping.AddDocumentMapping("card", cardMapping)
	indexMapping.DefaultMapping = cardMapping

	return indexMapping, nil
}

// docID generates a unique document ID for a card.
func docID(boardSlug string, colIdx, cardIdx int) string {
	return fmt.Sprintf("%s:%d:%d", boardSlug, colIdx, cardIdx)
}

// IndexBoard indexes all cards in a board.
func (idx *Index) IndexBoard(boardSlug string, board *models.Board) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	batch := idx.idx.NewBatch()
	for ci, col := range board.Columns {
		for cardi, card := range col.Cards {
			doc := cardDocument{
				BoardSlug:  boardSlug,
				BoardName:  board.Name,
				ColumnName: col.Name,
				ColIdx:     ci,
				CardIdx:    cardi,
				Title:      card.Title,
				Body:       card.Body,
				Tags:       strings.Join(card.Tags, " "),
				Assignee:   card.Assignee,
				Priority:   card.Priority,
			}
			if err := batch.Index(docID(boardSlug, ci, cardi), doc); err != nil {
				return fmt.Errorf("index card: %w", err)
			}
		}
	}
	return idx.idx.Batch(batch)
}

// RemoveBoard removes all indexed cards for a board.
func (idx *Index) RemoveBoard(boardSlug string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Search for all docs with this board slug and delete them.
	q := bleve.NewTermQuery(boardSlug)
	q.SetField("board_slug")
	req := bleve.NewSearchRequest(q)
	req.Size = 10000

	res, err := idx.idx.Search(req)
	if err != nil {
		return err
	}

	batch := idx.idx.NewBatch()
	for _, hit := range res.Hits {
		batch.Delete(hit.ID)
	}
	return idx.idx.Batch(batch)
}

// Search queries the index and returns ranked results.
func (idx *Index) Search(query string, limit int) ([]SearchResult, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}

	// Use a match query which applies the same analyzer to the search terms
	q := bleve.NewMatchQuery(query)
	req := bleve.NewSearchRequest(q)
	req.Size = limit
	req.Fields = []string{"board_slug", "board_name", "column_name", "col_idx", "card_idx", "title"}
	req.Highlight = bleve.NewHighlightWithStyle(ansi.Name)
	req.Highlight.AddField("title")
	req.Highlight.AddField("body")

	res, err := idx.idx.Search(req)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(res.Hits))
	for _, hit := range res.Hits {
		r := SearchResult{
			Score:     hit.Score,
			Fragments: make(map[string]string),
		}

		// Extract stored fields
		if v, ok := hit.Fields["board_slug"].(string); ok {
			r.BoardSlug = v
		}
		if v, ok := hit.Fields["board_name"].(string); ok {
			r.BoardName = v
		}
		if v, ok := hit.Fields["column_name"].(string); ok {
			r.ColumnName = v
		}
		if v, ok := hit.Fields["col_idx"].(float64); ok {
			r.ColIdx = int(v)
		}
		if v, ok := hit.Fields["card_idx"].(float64); ok {
			r.CardIdx = int(v)
		}
		if v, ok := hit.Fields["title"].(string); ok {
			r.CardTitle = v
		}

		// Extract highlight fragments
		for field, frags := range hit.Fragments {
			if len(frags) > 0 {
				r.Fragments[field] = frags[0]
			}
		}

		results = append(results, r)
	}

	return results, nil
}

// Close closes the underlying Bleve index.
func (idx *Index) Close() error {
	return idx.idx.Close()
}
