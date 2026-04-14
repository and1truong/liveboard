package v1_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
