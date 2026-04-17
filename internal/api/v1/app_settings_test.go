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

func TestGetAppSettings(t *testing.T) {
	deps := newTestDeps(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var s map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&s); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func TestPutAppSettings(t *testing.T) {
	deps := newTestDeps(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	body := `{"site_name":"My Board","theme":"dark"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d: %s", rec.Code, rec.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	var s struct {
		SiteName string `json:"site_name"`
	}
	_ = json.NewDecoder(rec2.Body).Decode(&s)
	if s.SiteName != "My Board" {
		t.Errorf("want site_name=My Board, got %q", s.SiteName)
	}
}

func TestPutAppSettings_badJSON(t *testing.T) {
	deps := newTestDeps(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestPutAppSettings_tooLarge(t *testing.T) {
	deps := newTestDeps(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", strings.NewReader("{}"))
	req.ContentLength = 1<<20 + 1
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("want 413, got %d", rec.Code)
	}
}
