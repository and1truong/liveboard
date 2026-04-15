package v1_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	v1 "github.com/and1truong/liveboard/internal/api/v1"
)

// doReq is a small helper for test-driving the v1 router.
func doReq(t *testing.T, deps v1.Deps, method, path, body string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	var req *http.Request
	if reader != nil {
		req = httptest.NewRequest(method, path, reader)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec, rec.Body.String()
}

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

func TestGetBoard(t *testing.T) {
	deps := newTestDeps(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/demo", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var board struct {
		Version int    `json:"version"`
		Name    string `json:"name"`
		Columns []struct {
			Name  string `json:"name"`
			Cards []any  `json:"cards"`
		} `json:"columns"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&board); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if board.Name != "Demo" {
		t.Errorf("want name=Demo, got %q", board.Name)
	}
	if len(board.Columns) != 1 {
		t.Fatalf("want 1 column, got %d", len(board.Columns))
	}
	if len(board.Columns[0].Cards) != 1 {
		t.Errorf("want 1 card, got %d", len(board.Columns[0].Cards))
	}
}

func TestGetBoardNotFound(t *testing.T) {
	deps := newTestDeps(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/does-not-exist", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body.Code != "NOT_FOUND" {
		t.Errorf("want code=NOT_FOUND, got %q", body.Code)
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

func TestCreateBoard(t *testing.T) {
	deps := newTestDepsWithSSE(t)
	rec, body := doReq(t, deps, http.MethodPost, "/api/v1/boards", `{"name":"Foo"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, body)
	}
	var s struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Version int    `json:"version"`
	}
	if err := json.Unmarshal([]byte(body), &s); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if s.ID != "Foo" || s.Name != "Foo" {
		t.Errorf("summary = %+v", s)
	}
}

func TestCreateBoard_collision(t *testing.T) {
	deps := newTestDepsWithSSE(t)
	if rec, body := doReq(t, deps, http.MethodPost, "/api/v1/boards", `{"name":"Foo"}`); rec.Code != http.StatusCreated {
		t.Fatalf("setup: %d %s", rec.Code, body)
	}
	rec, body := doReq(t, deps, http.MethodPost, "/api/v1/boards", `{"name":"Foo"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, body)
	}
	if !strings.Contains(body, "ALREADY_EXISTS") {
		t.Errorf("want ALREADY_EXISTS, body = %s", body)
	}
}

func TestCreateBoard_invalid(t *testing.T) {
	deps := newTestDepsWithSSE(t)
	rec, _ := doReq(t, deps, http.MethodPost, "/api/v1/boards", `{"name":"   "}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestCreateBoard_malformedJSON(t *testing.T) {
	deps := newTestDepsWithSSE(t)
	rec, _ := doReq(t, deps, http.MethodPost, "/api/v1/boards", `not json`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d", rec.Code)
	}
}
