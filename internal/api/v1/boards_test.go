package v1_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	v1 "github.com/and1truong/liveboard/internal/api/v1"
)

func TestListBoards(t *testing.T) {
	deps := newTestDeps(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("want Content-Type=application/json, got %q", ct)
	}

	var body []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 1 {
		t.Fatalf("want 1 board, got %d", len(body))
	}
	// Board is []models.Board; the identifier field is "name" (json:"name").
	if s, _ := body[0]["name"].(string); s == "" {
		t.Errorf("expected non-empty name, got %v", body[0])
	}
}

func TestListBoardsEmptyReturnsArray(t *testing.T) {
	deps := newTestDepsEmpty(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	body := rec.Body.String()
	// Trim trailing newline from json.Encoder.
	if body != "[]\n" && body != "[]" {
		t.Errorf("want empty JSON array, got %q", body)
	}
}
