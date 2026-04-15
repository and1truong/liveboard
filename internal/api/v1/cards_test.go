package v1_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	v1 "github.com/and1truong/liveboard/internal/api/v1"
	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/search"
	"github.com/and1truong/liveboard/internal/workspace"
)

// newDepsWithTwoBoards builds a workspace with "target.md" and "source.md"
// and attaches a search index.
func newDepsWithTwoBoards(t *testing.T) v1.Deps {
	t.Helper()
	dir := t.TempDir()
	seed := "---\nversion: 1\nname: %s\n---\n\n## Todo\n\n"
	for _, name := range []string{"target", "source"} {
		path := filepath.Join(dir, name+".md")
		if err := os.WriteFile(path, []byte(fmt.Sprintf(seed, name)), 0o644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	ws := workspace.Open(dir)
	idx, err := search.New()
	if err != nil {
		t.Fatal(err)
	}
	return v1.Deps{
		Workspace: ws,
		Engine:    board.New(),
		Search:    idx,
	}
}

func addCardToBoard(t *testing.T, r chi.Router, slug, title string) {
	t.Helper()
	body := `{"client_version":-1,"op":{"type":"add_card","column":"Todo","title":"` + title + `"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards/"+slug+"/mutations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("add_card to %s: want 200, got %d: %s", slug, rec.Code, rec.Body.String())
	}
}

func getBoardCardID(t *testing.T, r chi.Router, slug string, colIdx, cardIdx int) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/"+slug, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("getBoard %s: want 200, got %d: %s", slug, rec.Code, rec.Body.String())
	}
	var b struct {
		Columns []struct {
			Cards []struct {
				ID string `json:"id"`
			} `json:"cards"`
		} `json:"columns"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&b); err != nil {
		t.Fatalf("decode board: %v", err)
	}
	if colIdx >= len(b.Columns) {
		t.Fatalf("col %d out of range (got %d columns)", colIdx, len(b.Columns))
	}
	if cardIdx >= len(b.Columns[colIdx].Cards) {
		t.Fatalf("card %d out of range (got %d cards)", cardIdx, len(b.Columns[colIdx].Cards))
	}
	id := b.Columns[colIdx].Cards[cardIdx].ID
	if id == "" {
		t.Fatalf("card at col=%d card=%d has no id", colIdx, cardIdx)
	}
	return id
}

func editCardLinks(t *testing.T, r chi.Router, slug string, colIdx, cardIdx int, links []string) {
	t.Helper()
	linksJSON, _ := json.Marshal(links)
	body := fmt.Sprintf(
		`{"client_version":-1,"op":{"type":"edit_card","col_idx":%d,"card_idx":%d,"title":"","body":"","tags":[],"priority":"","due":"","assignee":"","links":%s}}`,
		colIdx, cardIdx, string(linksJSON),
	)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards/"+slug+"/mutations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("edit_card on %s: want 200, got %d: %s", slug, rec.Code, rec.Body.String())
	}
}

type backlinkHitDTO struct {
	BoardID   string `json:"board_id"`
	BoardName string `json:"board_name"`
	ColIdx    int    `json:"col_idx"`
	CardIdx   int    `json:"card_idx"`
	CardTitle string `json:"card_title"`
}

func TestBacklinks_ReturnsSourceCard(t *testing.T) {
	deps := newDepsWithTwoBoards(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	// Add a card to target board and get its auto-assigned ID.
	addCardToBoard(t, r, "target", "target-card")
	targetID := getBoardCardID(t, r, "target", 0, 0)

	// Add a card to source board.
	addCardToBoard(t, r, "source", "source-card")

	// Edit the source card to link to the target card.
	editCardLinks(t, r, "source", 0, 0, []string{"target:" + targetID})

	// GET backlinks for the target card.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cards/"+targetID+"/backlinks", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("backlinks: want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var hits []backlinkHitDTO
	if err := json.NewDecoder(rec.Body).Decode(&hits); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("want 1 backlink hit, got %d: %v", len(hits), hits)
	}
	if hits[0].BoardID != "source" {
		t.Errorf("board_id = %q, want \"source\"", hits[0].BoardID)
	}
}

func TestBacklinks_NoSearchReturnsEmpty(t *testing.T) {
	deps := newTestDeps(t)
	deps.Search = nil
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cards/some-id/backlinks", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("body = %q, want []", rec.Body.String())
	}
}
