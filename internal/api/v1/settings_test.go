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

	// Body should contain resolved fields — exact contents depend on defaults.
	if rec.Body.Len() < 3 {
		t.Errorf("body too small: %q", rec.Body.String())
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

	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Fatalf("want 200/204, got %d: %s", rec.Code, rec.Body.String())
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
