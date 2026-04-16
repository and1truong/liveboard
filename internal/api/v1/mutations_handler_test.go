package v1_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	v1 "github.com/and1truong/liveboard/internal/api/v1"
)

func TestPostMutationAddCard(t *testing.T) {
	deps := newTestDeps(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	body := map[string]any{
		"client_version": -1,
		"op": map[string]any{
			"type":   "add_card",
			"column": "Todo",
			"title":  "via-rest",
		},
	}
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards/demo/mutations", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Version int `json:"version"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Version != 2 {
		t.Errorf("want version == 2, got %d", resp.Version)
	}
}

func TestPostMutationVersionConflict(t *testing.T) {
	deps := newTestDeps(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	body := map[string]any{
		"client_version": 0, // stale
		"op":             map[string]any{"type": "add_card", "column": "Todo", "title": "stale"},
	}
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards/demo/mutations", bytes.NewReader(buf))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", rec.Code)
	}

	var body2 struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&body2)
	if body2.Code != "VERSION_CONFLICT" {
		t.Errorf("want code=VERSION_CONFLICT, got %q", body2.Code)
	}
}

func TestPostMutationMoveCardToBoard(t *testing.T) {
	deps := newTestDeps(t)
	// Seed a second board "other" with a target column.
	other := "---\nversion: 1\nname: Other\n---\n\n## Inbox\n"
	if err := os.WriteFile(filepath.Join(deps.Workspace.Dir, "other.md"), []byte(other), 0o644); err != nil {
		t.Fatalf("seed other: %v", err)
	}

	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	body := map[string]any{
		"client_version": 1,
		"op": map[string]any{
			"type":       "move_card_to_board",
			"col_idx":    0,
			"card_idx":   0,
			"dst_board":  "other",
			"dst_column": "Inbox",
		},
	}
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards/demo/mutations", bytes.NewReader(buf))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Source should now have no cards in col 0.
	var src struct {
		Version int `json:"version"`
		Columns []struct {
			Cards []struct {
				Title string `json:"title"`
			} `json:"cards"`
		} `json:"columns"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&src); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(src.Columns) == 0 || len(src.Columns[0].Cards) != 0 {
		t.Errorf("expected src col 0 empty, got %+v", src.Columns)
	}

	// Destination should now contain the moved card.
	dst, err := deps.Workspace.LoadBoard("other")
	if err != nil {
		t.Fatalf("load dst: %v", err)
	}
	if len(dst.Columns) == 0 || len(dst.Columns[0].Cards) != 1 || dst.Columns[0].Cards[0].Title != "Seed" {
		t.Errorf("expected 'Seed' moved into other.Inbox, got %+v", dst.Columns)
	}
}

func TestListBoardsLite(t *testing.T) {
	deps := newTestDeps(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/list-lite", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var entries []struct {
		Slug    string   `json:"slug"`
		Name    string   `json:"name"`
		Columns []string `json:"columns"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&entries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	if entries[0].Slug == "" || len(entries[0].Columns) == 0 {
		t.Errorf("entry missing slug/columns: %+v", entries[0])
	}
}
