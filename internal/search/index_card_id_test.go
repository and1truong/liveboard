package search

import (
	"testing"

	"github.com/and1truong/liveboard/pkg/models"
	"github.com/blevesearch/bleve/v2"
)

func TestUpdateBoardIndexesCardID(t *testing.T) {
	idx, err := New()
	if err != nil {
		t.Fatal(err)
	}
	b := &models.Board{Name: "B", Columns: []models.Column{{Name: "Todo", Cards: []models.Card{
		{ID: "ABCDE12345", Title: "hello world"},
	}}}}
	if err := idx.UpdateBoard("b", b); err != nil {
		t.Fatal(err)
	}
	q := bleve.NewTermQuery("hello")
	q.SetField("title")
	sr := bleve.NewSearchRequestOptions(q, 10, 0, false)
	sr.Fields = []string{"card_id"}
	res, err := idx.idx.Search(sr)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("nil search result")
	}
	if len(res.Hits) != 1 {
		t.Fatalf("want 1 hit, got %d", len(res.Hits))
	}
	if got, _ := res.Hits[0].Fields["card_id"].(string); got != "ABCDE12345" {
		t.Fatalf("want card_id ABCDE12345, got %v", res.Hits[0].Fields["card_id"])
	}
}
