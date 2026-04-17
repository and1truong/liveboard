package v1_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	v1 "github.com/and1truong/liveboard/internal/api/v1"
	"github.com/and1truong/liveboard/internal/search"
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

func newDepsWithSearch(t *testing.T) v1.Deps {
	t.Helper()
	d := newTestDeps(t)
	idx, err := search.New()
	if err != nil {
		t.Fatal(err)
	}
	d.Search = idx
	return d
}

func routerFor(t *testing.T, deps v1.Deps) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))
	return r
}

func addCard(t *testing.T, r chi.Router, slug, title string) {
	t.Helper()
	body := `{"client_version":-1,"op":{"type":"add_card","column":"Todo","title":"` + title + `"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards/mutate/"+slug, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("add_card: want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSearch_FindsCard(t *testing.T) {
	deps := newDepsWithSearch(t)
	r := routerFor(t, deps)
	addCard(t, r, "demo", "alpha-token bravo")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=alpha-token", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("search: want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var hits []searchHitDTO
	if err := json.NewDecoder(rec.Body).Decode(&hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatalf("expected hits, got 0")
	}
	if hits[0].BoardID != "demo" {
		t.Errorf("board_id = %q", hits[0].BoardID)
	}
	if hits[0].CardID == "" {
		t.Errorf("card_id is empty")
	}
}

func TestSearch_EmptyQueryReturnsEmpty(t *testing.T) {
	deps := newDepsWithSearch(t)
	r := routerFor(t, deps)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("body = %q", rec.Body.String())
	}
}

func TestSearch_TooLongIsInvalid(t *testing.T) {
	deps := newDepsWithSearch(t)
	r := routerFor(t, deps)
	long := strings.Repeat("x", 257)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q="+long, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestSearch_DeletedBoardGone(t *testing.T) {
	deps := newDepsWithSearch(t)
	r := routerFor(t, deps)
	addCard(t, r, "demo", "unique-token")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/boards/board/demo", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/search?q=unique-token", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("expected [], got %q", rec.Body.String())
	}
}

func TestCreateBoard_withSearch(t *testing.T) {
	deps := newDepsWithSearch(t)
	r := routerFor(t, deps)
	rec, body := doReq(t, deps, http.MethodPost, "/api/v1/boards", `{"name":"SearchBoard"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, body)
	}
	// Index should include the new board (no error path needed — just cover the branch).
	_ = r
}

func TestRenameBoard_withSearch(t *testing.T) {
	deps := newDepsWithSearch(t)
	r := routerFor(t, deps)

	// Index a card so search has something for "demo".
	addCard(t, r, "demo", "findme-token")

	// Rename demo → new-name; slug != b.Name triggers DeleteBoard + UpdateBoard.
	body := `{"new_name":"new-name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/boards/board/demo", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("rename: %d %s", rec.Code, rec.Body.String())
	}
}

func TestPostMutationMoveCardToBoard_withSearch(t *testing.T) {
	deps := newDepsWithSearch(t)
	other := "---\nversion: 1\nname: Other\n---\n\n## Inbox\n"
	if err := os.WriteFile(filepath.Join(deps.Workspace.Dir, "other.md"), []byte(other), 0o644); err != nil {
		t.Fatalf("seed other: %v", err)
	}
	r := routerFor(t, deps)
	addCard(t, r, "demo", "moveable-card")

	body := map[string]any{
		"client_version": -1,
		"op": map[string]any{
			"type": "move_card_to_board", "col_idx": 0, "card_idx": 0,
			"dst_board": "other", "dst_column": "Inbox",
		},
	}
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards/mutate/demo", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("move: %d %s", rec.Code, rec.Body.String())
	}
}
