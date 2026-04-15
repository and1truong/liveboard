package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/and1truong/liveboard/internal/workspace"
)

func setupShellTest(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	ws := workspace.Open(dir)
	return NewServer(ws, ws.Engine, false, false, false, "test", "", "")
}

func TestShellRoute_Disabled(t *testing.T) {
	t.Setenv("LIVEBOARD_APP_SHELL", "")
	s := setupShellTest(t)
	req := httptest.NewRequest(http.MethodGet, "/app/", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("shell route should be 404 when flag disabled; got %d", rec.Code)
	}
}

func TestShellRoute_Enabled(t *testing.T) {
	t.Setenv("LIVEBOARD_APP_SHELL", "1")
	s := setupShellTest(t)
	req := httptest.NewRequest(http.MethodGet, "/app/", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "LiveBoard Shell") {
		t.Fatalf("response did not contain expected shell HTML")
	}
}

func TestShellRoute_Renderer(t *testing.T) {
	t.Setenv("LIVEBOARD_APP_SHELL", "1")
	s := setupShellTest(t)

	req := httptest.NewRequest(http.MethodGet, "/app/renderer/default/", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<div id=\"root\">") {
		t.Fatalf("response did not contain renderer root div")
	}
}
