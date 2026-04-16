package v1_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	v1 "github.com/and1truong/liveboard/internal/api/v1"
	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/web"
	"github.com/and1truong/liveboard/internal/workspace"
)

// newTestDeps builds a workspace with one seeded board for use by v1 tests.
// Exposed to other tests in this package.
func newTestDeps(t *testing.T) v1.Deps {
	t.Helper()
	dir := t.TempDir()
	seed := "---\nversion: 1\nname: Demo\n---\n\n## Todo\n\n- [ ] Seed\n"
	if err := os.WriteFile(filepath.Join(dir, "demo.md"), []byte(seed), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	ws := workspace.Open(dir)
	return v1.Deps{
		Workspace: ws,
		Engine:    board.New(),
	}
}

func newTestDepsWithSSE(t *testing.T) v1.Deps {
	t.Helper()
	d := newTestDeps(t)
	d.SSE = web.NewSSEBroker()
	return d
}

func newTestDepsEmpty(t *testing.T) v1.Deps { //nolint:unused
	t.Helper()
	dir := t.TempDir()
	ws := workspace.Open(dir)
	return v1.Deps{Workspace: ws, Engine: board.New()}
}

func TestGetWorkspace(t *testing.T) {
	deps := newTestDeps(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("want Content-Type=application/json, got %q", ct)
	}

	var body struct {
		Name       string `json:"name"`
		Dir        string `json:"dir"`
		BoardCount int    `json:"board_count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Name != "LiveBoard" {
		t.Errorf("want name=%q, got %q", "LiveBoard", body.Name)
	}
	if body.Dir == "" {
		t.Error("dir should not be empty")
	}
	if body.BoardCount != 1 {
		t.Errorf("want board_count=1, got %d", body.BoardCount)
	}
}
