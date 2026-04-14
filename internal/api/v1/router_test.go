package v1_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	v1 "github.com/and1truong/liveboard/internal/api/v1"
)

func TestVersionsMountedAtApiVersions(t *testing.T) {
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(v1.Deps{}))
	r.Method(http.MethodGet, "/api/versions", v1.VersionsHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/versions", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
}

func TestRouterMountsV1Prefix(t *testing.T) {
	r := chi.NewRouter()
	deps := v1.Deps{}
	r.Mount("/api/v1", v1.Router(deps))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/__does_not_exist__", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rec.Code)
	}
}
