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

func TestGetBoardSettings(t *testing.T) {
	deps := newTestDeps(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/demo/settings", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resolved struct {
		ViewMode string `json:"view_mode"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resolved); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resolved.ViewMode == "" {
		t.Errorf("want non-empty view_mode in resolved settings, got empty string")
	}
}

func TestPutBoardSettings(t *testing.T) {
	deps := newTestDeps(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	patch := `{"view_mode":"calendar"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/boards/demo/settings",
		bytes.NewReader([]byte(patch)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify persistence: GET the board and check settings.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/boards/demo", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	var b struct {
		Settings struct {
			ViewMode *string `json:"view_mode"`
		} `json:"settings"`
	}
	_ = json.NewDecoder(rec2.Body).Decode(&b)
	if b.Settings.ViewMode == nil || *b.Settings.ViewMode != "calendar" {
		t.Errorf("view_mode not persisted: %+v", b.Settings.ViewMode)
	}
}

func TestGetBoardSettings_notFound(t *testing.T) {
	deps := newTestDeps(t)
	rec, body := doReq(t, deps, http.MethodGet, "/api/v1/boards/nope/settings", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d: %s", rec.Code, body)
	}
}

func TestPutBoardSettings_notFound(t *testing.T) {
	deps := newTestDeps(t)
	rec, body := doReq(t, deps, http.MethodPut, "/api/v1/boards/nope/settings", `{"view_mode":"board"}`)
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d: %s", rec.Code, body)
	}
}

func TestPutBoardSettings_badJSON(t *testing.T) {
	deps := newTestDeps(t)
	rec, body := doReq(t, deps, http.MethodPut, "/api/v1/boards/demo/settings", "not json")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d: %s", rec.Code, body)
	}
}

func TestPutBoardSettings_withSSE(t *testing.T) {
	deps := newTestDepsWithSSE(t)
	rec, body := doReq(t, deps, http.MethodPut, "/api/v1/boards/demo/settings", `{"view_mode":"list"}`)
	if rec.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d: %s", rec.Code, body)
	}
}
