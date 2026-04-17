package v1_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	v1 "github.com/and1truong/liveboard/internal/api/v1"
	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/workspace"
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
	// Response must be the BoardSummary DTO the renderer expects, not a raw
	// models.Board. Missing `id` → every row's active check collapses to
	// undefined===undefined, highlighting all rows and blocking load.
	row := body[0]
	id, _ := row["id"].(string)
	name, _ := row["name"].(string)
	// id is the filename stem (canonical for getBoard / LoadBoard).
	// name is the frontmatter display name. Seed is demo.md / name: Demo.
	if id != "demo" {
		t.Errorf("want id=demo (file slug), got %q", id)
	}
	if name != "Demo" {
		t.Errorf("want name=Demo (frontmatter), got %q", name)
	}
	if _, ok := row["version"]; !ok {
		t.Errorf("expected version field, got %v", row)
	}
	// Raw models.Board exposes columns; the DTO must not.
	if _, leaked := row["columns"]; leaked {
		t.Errorf("listBoards leaked raw Board (columns present): %v", row)
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

// Regression: when a board's filename differs from its frontmatter name
// (e.g. demo file oss-project.md with `name: OSS Tracker`), listBoards must
// expose `id` as the filename stem so getBoard can round-trip it. Otherwise
// clicking the row issues GET /api/v1/boards/OSS Tracker → 404.
func TestListBoards_idIsFileSlugNotName(t *testing.T) {
	dir := t.TempDir()
	seed := "---\nversion: 1\nname: OSS Tracker\n---\n\n## Todo\n\n- [ ] Seed\n"
	if err := os.WriteFile(filepath.Join(dir, "oss-project.md"), []byte(seed), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	deps := v1.Deps{Workspace: workspace.Open(dir), Engine: board.New()}

	rec, body := doReq(t, deps, http.MethodGet, "/api/v1/boards", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, body)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(body), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	id, _ := rows[0]["id"].(string)
	if id != "oss-project" {
		t.Fatalf("want id=oss-project (file slug), got %q", id)
	}

	// Round-trip: getBoard must accept the id from listBoards.
	rec2, body2 := doReq(t, deps, http.MethodGet, "/api/v1/boards/"+id, "")
	if rec2.Code != http.StatusOK {
		t.Fatalf("getBoard(%q): %d %s", id, rec2.Code, body2)
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

func TestRenameBoard(t *testing.T) {
	deps := newTestDepsWithSSE(t)
	if rec, body := doReq(t, deps, http.MethodPost, "/api/v1/boards", `{"name":"Foo"}`); rec.Code != http.StatusCreated {
		t.Fatalf("setup: %d %s", rec.Code, body)
	}
	rec, body := doReq(t, deps, http.MethodPatch, "/api/v1/boards/Foo", `{"new_name":"Bar"}`)
	if rec.Code != http.StatusOK {
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
	if s.ID != "Bar" || s.Name != "Bar" {
		t.Errorf("summary = %+v", s)
	}

	// Old slug 404.
	rec2, _ := doReq(t, deps, http.MethodGet, "/api/v1/boards/Foo", "")
	if rec2.Code != http.StatusNotFound {
		t.Errorf("old slug: want 404, got %d", rec2.Code)
	}
	// New slug 200.
	rec3, _ := doReq(t, deps, http.MethodGet, "/api/v1/boards/Bar", "")
	if rec3.Code != http.StatusOK {
		t.Errorf("new slug: want 200, got %d", rec3.Code)
	}
}

func TestRenameBoard_collision(t *testing.T) {
	deps := newTestDepsWithSSE(t)
	doReq(t, deps, http.MethodPost, "/api/v1/boards", `{"name":"Foo"}`)
	doReq(t, deps, http.MethodPost, "/api/v1/boards", `{"name":"Bar"}`)
	rec, body := doReq(t, deps, http.MethodPatch, "/api/v1/boards/Foo", `{"new_name":"Bar"}`)
	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, body = %s", rec.Code, body)
	}
}

func TestRenameBoard_notFound(t *testing.T) {
	deps := newTestDepsWithSSE(t)
	rec, _ := doReq(t, deps, http.MethodPatch, "/api/v1/boards/nope", `{"new_name":"X"}`)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestRenameBoard_invalidName(t *testing.T) {
	deps := newTestDepsWithSSE(t)
	doReq(t, deps, http.MethodPost, "/api/v1/boards", `{"name":"Foo"}`)
	rec, _ := doReq(t, deps, http.MethodPatch, "/api/v1/boards/Foo", `{"new_name":""}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestDeleteBoard(t *testing.T) {
	deps := newTestDepsWithSSE(t)
	if rec, body := doReq(t, deps, http.MethodPost, "/api/v1/boards", `{"name":"Foo"}`); rec.Code != http.StatusCreated {
		t.Fatalf("setup: %d %s", rec.Code, body)
	}
	rec, body := doReq(t, deps, http.MethodDelete, "/api/v1/boards/Foo", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rec.Code, body)
	}
	rec2, _ := doReq(t, deps, http.MethodGet, "/api/v1/boards/Foo", "")
	if rec2.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", rec2.Code)
	}
}

func TestDeleteBoard_notFound(t *testing.T) {
	deps := newTestDepsWithSSE(t)
	rec, _ := doReq(t, deps, http.MethodDelete, "/api/v1/boards/nope", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestToggleBoardPin(t *testing.T) {
	deps := newTestDepsWithSSE(t)

	// Pin demo.
	rec, body := doReq(t, deps, http.MethodPost, "/api/v1/boards/demo/pin", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("pin: want 204, got %d: %s", rec.Code, body)
	}

	// listBoards should show demo as pinned.
	rec2, body2 := doReq(t, deps, http.MethodGet, "/api/v1/boards", "")
	if rec2.Code != http.StatusOK {
		t.Fatalf("list after pin: %d %s", rec2.Code, body2)
	}
	var rows []struct {
		ID     string `json:"id"`
		Pinned bool   `json:"pinned"`
	}
	if err := json.Unmarshal([]byte(body2), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 1 || !rows[0].Pinned {
		t.Errorf("expected demo pinned, got %+v", rows)
	}

	// Unpin demo.
	rec3, _ := doReq(t, deps, http.MethodPost, "/api/v1/boards/demo/pin", "")
	if rec3.Code != http.StatusNoContent {
		t.Fatalf("unpin: want 204, got %d", rec3.Code)
	}

	// listBoards should show demo as unpinned.
	rec4, body4 := doReq(t, deps, http.MethodGet, "/api/v1/boards", "")
	var rows2 []struct {
		Pinned bool `json:"pinned"`
	}
	_ = json.Unmarshal([]byte(body4), &rows2)
	if len(rows2) != 1 || rows2[0].Pinned {
		t.Errorf("expected demo unpinned after toggle, code=%d body=%s", rec4.Code, body4)
	}
}

func TestRenameBoard_malformedJSON(t *testing.T) {
	deps := newTestDepsWithSSE(t)
	rec, _ := doReq(t, deps, http.MethodPatch, "/api/v1/boards/demo", "not json")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestListBoards_nameSort(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"zebra", "apple", "mango"} {
		seed := "---\nversion: 1\nname: " + name + "\n---\n\n## Todo\n\n"
		if err := os.WriteFile(filepath.Join(dir, name+".md"), []byte(seed), 0o644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	deps := v1.Deps{Dir: dir, Workspace: workspace.Open(dir), Engine: board.New()}
	rec, body := doReq(t, deps, http.MethodGet, "/api/v1/boards", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, body)
	}
	var rows []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(body), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d", len(rows))
	}
	if rows[0].ID != "apple" || rows[1].ID != "mango" || rows[2].ID != "zebra" {
		t.Errorf("want alpha sort [apple, mango, zebra], got %v", func() []string {
			ids := make([]string, len(rows))
			for i, r := range rows {
				ids[i] = r.ID
			}
			return ids
		}())
	}
}

func TestListBoards_withCompletedCard(t *testing.T) {
	dir := t.TempDir()
	seed := "---\nversion: 1\nname: Demo\n---\n\n## Todo\n\n- [x] Done\n- [ ] Still todo\n"
	if err := os.WriteFile(filepath.Join(dir, "demo.md"), []byte(seed), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	deps := v1.Deps{Dir: dir, Workspace: workspace.Open(dir), Engine: board.New()}
	_, body := doReq(t, deps, http.MethodGet, "/api/v1/boards", "")
	var rows []struct {
		CardCount int `json:"cardCount"`
		DoneCount int `json:"doneCount"`
	}
	if err := json.Unmarshal([]byte(body), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 1 || rows[0].CardCount != 2 || rows[0].DoneCount != 1 {
		t.Errorf("want cardCount=2 doneCount=1, got %+v", rows)
	}
}

func TestListBoards_pinSorting(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"alpha", "beta", "gamma"} {
		seed := "---\nversion: 1\nname: " + name + "\n---\n\n## Todo\n\n"
		if err := os.WriteFile(filepath.Join(dir, name+".md"), []byte(seed), 0o644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	deps := newTestDepsEmpty(t)
	// Replace the empty deps with multi-board ones sharing Dir.
	deps.Workspace = workspace.Open(dir)
	deps.Dir = dir

	// Pin gamma first, then alpha.
	doReq(t, deps, http.MethodPost, "/api/v1/boards/gamma/pin", "")
	doReq(t, deps, http.MethodPost, "/api/v1/boards/alpha/pin", "")

	rec, body := doReq(t, deps, http.MethodGet, "/api/v1/boards", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, body)
	}
	var rows []struct {
		ID     string `json:"id"`
		Pinned bool   `json:"pinned"`
	}
	if err := json.Unmarshal([]byte(body), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("want 3 boards, got %d", len(rows))
	}
	// Pinned boards come first in pin order, then alpha-sorted unpinned.
	if rows[0].ID != "gamma" {
		t.Errorf("want gamma first (pinned first), got %q", rows[0].ID)
	}
	if rows[1].ID != "alpha" {
		t.Errorf("want alpha second (pinned second), got %q", rows[1].ID)
	}
	if rows[2].ID != "beta" {
		t.Errorf("want beta last (unpinned, alpha-sorted), got %q", rows[2].ID)
	}
	if !rows[0].Pinned || !rows[1].Pinned {
		t.Errorf("expected gamma and alpha marked pinned")
	}
	if rows[2].Pinned {
		t.Errorf("expected beta not pinned")
	}
}
