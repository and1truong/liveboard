package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/and1truong/liveboard/internal/workspace"
	"github.com/and1truong/liveboard/pkg/models"
)

func setupTest(t *testing.T) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	ws := workspace.Open(dir)
	srv := NewServer(ws, ws.Engine, nil)
	return httptest.NewServer(srv.Router())
}

func doRawBody(t *testing.T, ts *httptest.Server, method, path, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("unexpected nil response")
	}
	return resp
}

func doJSON(t *testing.T, ts *httptest.Server, method, path string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, ts.URL+path, &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("unexpected nil response")
	}
	return resp
}

func decodeResp(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatal(err)
	}
}

func TestListBoardsEmpty(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	resp := doJSON(t, ts, "GET", "/boards", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var boards []models.Board
	decodeResp(t, resp, &boards)
	if len(boards) != 0 {
		t.Fatalf("expected 0 boards, got %d", len(boards))
	}
}

func TestCreateAndGetBoard(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	// Create
	resp := doJSON(t, ts, "POST", "/boards", map[string]string{"name": "sprint"})
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var board models.Board
	decodeResp(t, resp, &board)
	if board.Name != "sprint" {
		t.Fatalf("expected name 'sprint', got %q", board.Name)
	}
	if len(board.Columns) != 3 {
		t.Fatalf("expected 3 default columns, got %d", len(board.Columns))
	}

	// Get
	resp = doJSON(t, ts, "GET", "/boards/sprint", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var got models.Board
	decodeResp(t, resp, &got)
	if got.Name != "sprint" {
		t.Fatalf("expected name 'sprint', got %q", got.Name)
	}

	// List
	resp = doJSON(t, ts, "GET", "/boards", nil)
	var boards []models.Board
	decodeResp(t, resp, &boards)
	if len(boards) != 1 {
		t.Fatalf("expected 1 board, got %d", len(boards))
	}
}

func TestDeleteBoard(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "temp"})

	resp := doJSON(t, ts, "DELETE", "/boards/temp", nil)
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	resp = doJSON(t, ts, "GET", "/boards/temp", nil)
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestCreateBoardConflict(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "dup"})
	resp := doJSON(t, ts, "POST", "/boards", map[string]string{"name": "dup"})
	if resp.StatusCode != 409 {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestAddAndDeleteColumn(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "proj"})

	// Add column
	resp := doJSON(t, ts, "POST", "/boards/proj/columns", map[string]string{"name": "QA"})
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	// Verify column exists
	resp = doJSON(t, ts, "GET", "/boards/proj", nil)
	var board models.Board
	decodeResp(t, resp, &board)
	found := false
	for _, c := range board.Columns {
		if c.Name == "QA" {
			found = true
		}
	}
	if !found {
		t.Fatal("column QA not found after adding")
	}

	// Delete column
	resp = doJSON(t, ts, "DELETE", "/boards/proj/columns/QA", nil)
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestCardCRUD(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	// Create board with default columns: "not now", "maybe?", "done"
	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "tasks"})

	// Add card to first column ("not now")
	resp := doJSON(t, ts, "POST", "/boards/tasks/columns/not now/cards", map[string]string{"title": "Fix bug"})
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var card models.Card
	decodeResp(t, resp, &card)
	if card.Title != "Fix bug" {
		t.Fatalf("expected title 'Fix bug', got %q", card.Title)
	}

	// Get card via index-based route (col=0, card=0)
	resp = doJSON(t, ts, "GET", "/boards/tasks/cols/0/cards/0", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var cr cardResponse
	decodeResp(t, resp, &cr)
	if cr.Column != "not now" {
		t.Fatalf("expected column 'not now', got %q", cr.Column)
	}

	// Delete card
	resp = doJSON(t, ts, "DELETE", "/boards/tasks/cols/0/cards/0", nil)
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Verify deleted — board should have empty first column
	resp = doJSON(t, ts, "GET", "/boards/tasks", nil)
	var board models.Board
	decodeResp(t, resp, &board)
	if len(board.Columns[0].Cards) != 0 {
		t.Fatalf("expected 0 cards after delete, got %d", len(board.Columns[0].Cards))
	}
}

func TestMoveCard(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "flow"})

	// Add card to "not now" (col 0)
	doJSON(t, ts, "POST", "/boards/flow/columns/not now/cards", map[string]string{"title": "Task A"})

	// Move card from col 0, card 0 to "done"
	resp := doJSON(t, ts, "POST", "/boards/flow/cols/0/cards/0/move", map[string]string{"column": "done"})
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Verify it's in "done" (col 2, card 0)
	resp = doJSON(t, ts, "GET", "/boards/flow/cols/2/cards/0", nil)
	var cr cardResponse
	decodeResp(t, resp, &cr)
	if cr.Column != "done" {
		t.Fatalf("expected column 'done', got %q", cr.Column)
	}
}

func TestCompleteCard(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "comp"})

	doJSON(t, ts, "POST", "/boards/comp/columns/not now/cards", map[string]string{"title": "Finish"})

	resp := doJSON(t, ts, "POST", "/boards/comp/cols/0/cards/0/complete", nil)
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	resp = doJSON(t, ts, "GET", "/boards/comp/cols/0/cards/0", nil)
	var cr cardResponse
	decodeResp(t, resp, &cr)
	if !cr.Completed {
		t.Fatal("expected card to be completed")
	}
}

func TestTagCard(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "tags"})

	doJSON(t, ts, "POST", "/boards/tags/columns/not now/cards", map[string]string{"title": "Label me"})

	resp := doJSON(t, ts, "POST", "/boards/tags/cols/0/cards/0/tag", map[string]any{"tags": []string{"urgent", "bug"}})
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	resp = doJSON(t, ts, "GET", "/boards/tags/cols/0/cards/0", nil)
	var cr cardResponse
	decodeResp(t, resp, &cr)
	if len(cr.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(cr.Tags))
	}
}

func TestNotFoundErrors(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	resp := doJSON(t, ts, "GET", "/boards/nonexistent", nil)
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 for missing board, got %d", resp.StatusCode)
	}
}

func TestStubEndpoints(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	for _, path := range []string{"/search", "/events", "/events/ws"} {
		resp := doJSON(t, ts, "GET", path, nil)
		if resp.StatusCode != 501 {
			t.Fatalf("expected 501 for %s, got %d", path, resp.StatusCode)
		}
		_ = resp.Body.Close()
	}
}

// --- New tests for error cases and uncovered endpoints ---

func TestCreateBoardMissingName(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	resp := doJSON(t, ts, "POST", "/boards", map[string]string{})
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestCreateBoardInvalidJSON(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	resp := doRawBody(t, ts, "POST", "/boards", "{invalid")
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAddCardMissingTitle(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "b1"})

	resp := doJSON(t, ts, "POST", "/boards/b1/columns/not now/cards", map[string]string{})
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAddCardInvalidJSON(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "b2"})

	resp := doRawBody(t, ts, "POST", "/boards/b2/columns/not now/cards", "{bad")
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestGetCardInvalidIndices(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "idx"})

	// Non-numeric index
	resp := doJSON(t, ts, "GET", "/boards/idx/cols/abc/cards/0", nil)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for non-numeric col index, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Out of range
	resp = doJSON(t, ts, "GET", "/boards/idx/cols/99/cards/0", nil)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for out-of-range, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestMoveCardMissingColumn(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "mv"})
	doJSON(t, ts, "POST", "/boards/mv/columns/not now/cards", map[string]string{"title": "T"})

	resp := doJSON(t, ts, "POST", "/boards/mv/cols/0/cards/0/move", map[string]string{})
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestMoveCardInvalidJSON(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "mvj"})
	doJSON(t, ts, "POST", "/boards/mvj/columns/not now/cards", map[string]string{"title": "T"})

	resp := doRawBody(t, ts, "POST", "/boards/mvj/cols/0/cards/0/move", "{bad")
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestTagCardMissingTags(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "tg"})
	doJSON(t, ts, "POST", "/boards/tg/columns/not now/cards", map[string]string{"title": "T"})

	resp := doJSON(t, ts, "POST", "/boards/tg/cols/0/cards/0/tag", map[string]any{"tags": []string{}})
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestTagCardInvalidJSON(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "tgj"})
	doJSON(t, ts, "POST", "/boards/tgj/columns/not now/cards", map[string]string{"title": "T"})

	resp := doRawBody(t, ts, "POST", "/boards/tgj/cols/0/cards/0/tag", "{bad")
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAddColumnMissingName(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "col"})

	resp := doJSON(t, ts, "POST", "/boards/col/columns", map[string]string{})
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAddColumnInvalidJSON(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "colj"})

	resp := doRawBody(t, ts, "POST", "/boards/colj/columns", "{bad")
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestMoveColumn(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "mcol"})

	// Move "not now" after "done"
	resp := doJSON(t, ts, "POST", "/boards/mcol/columns/not now/move", map[string]string{"after": "done"})
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Verify order changed
	resp = doJSON(t, ts, "GET", "/boards/mcol", nil)
	var board models.Board
	decodeResp(t, resp, &board)
	if board.Columns[0].Name != "maybe?" {
		t.Errorf("expected 'maybe?' first, got %q", board.Columns[0].Name)
	}
}

func TestMoveColumnMissingAfter(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "mcol2"})

	resp := doJSON(t, ts, "POST", "/boards/mcol2/columns/not now/move", map[string]string{})
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestMoveColumnInvalidJSON(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "mcol3"})

	resp := doRawBody(t, ts, "POST", "/boards/mcol3/columns/not now/move", "{bad")
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestDeleteCardOutOfRange(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "del"})

	resp := doJSON(t, ts, "DELETE", "/boards/del/cols/0/cards/99", nil)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for out-of-range card, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestCompleteCardOutOfRange(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "cmp"})

	resp := doJSON(t, ts, "POST", "/boards/cmp/cols/0/cards/99/complete", nil)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for out-of-range card, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestDeleteBoardNotFound(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	resp := doJSON(t, ts, "DELETE", "/boards/nonexistent", nil)
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// --- PATCH column stub ---

func TestPatchColumnStub(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "pboard"})

	resp := doJSON(t, ts, "PATCH", "/boards/pboard/columns/not now", map[string]string{"name": "later"})
	if resp.StatusCode != 501 {
		t.Fatalf("expected 501 for PATCH column stub, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// --- Not-found error paths ---

func TestAddColumnBoardNotFound(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	resp := doJSON(t, ts, "POST", "/boards/ghost/columns", map[string]string{"name": "Col"})
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 for missing board, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestDeleteColumnNotFound(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "dcol"})

	// Deleting a nonexistent column is idempotent (no-op returns 204)
	resp := doJSON(t, ts, "DELETE", "/boards/dcol/columns/nonexistent", nil)
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204 for idempotent delete, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestMoveColumnBoardNotFound(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	resp := doJSON(t, ts, "POST", "/boards/ghost/columns/x/move", map[string]string{"after": "y"})
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 for missing board, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAddCardBoardNotFound(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	resp := doJSON(t, ts, "POST", "/boards/ghost/columns/todo/cards", map[string]string{"title": "T"})
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 for missing board, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAddCardColumnNotFound(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "acnf"})

	resp := doJSON(t, ts, "POST", "/boards/acnf/columns/nonexistent/cards", map[string]string{"title": "T"})
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 for missing column, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestGetCardBoardNotFound(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	resp := doJSON(t, ts, "GET", "/boards/ghost/cols/0/cards/0", nil)
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 for missing board, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestDeleteCardBoardNotFound(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	resp := doJSON(t, ts, "DELETE", "/boards/ghost/cols/0/cards/0", nil)
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 for missing board, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestMoveCardBoardNotFound(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	resp := doJSON(t, ts, "POST", "/boards/ghost/cols/0/cards/0/move", map[string]string{"column": "done"})
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 for missing board, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestCompleteCardBoardNotFound(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	resp := doJSON(t, ts, "POST", "/boards/ghost/cols/0/cards/0/complete", nil)
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 for missing board, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestTagCardBoardNotFound(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	resp := doJSON(t, ts, "POST", "/boards/ghost/cols/0/cards/0/tag", map[string]any{"tags": []string{"x"}})
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 for missing board, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// --- Additional invalid-index tests ---

func TestGetCardInvalidCardIndex(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "cidx"})

	resp := doJSON(t, ts, "GET", "/boards/cidx/cols/0/cards/abc", nil)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for non-numeric card index, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestDeleteCardInvalidIndices(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "didx"})

	resp := doJSON(t, ts, "DELETE", "/boards/didx/cols/abc/cards/0", nil)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for non-numeric col index, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestMoveCardInvalidIndices(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "midx"})

	resp := doJSON(t, ts, "POST", "/boards/midx/cols/abc/cards/0/move", map[string]string{"column": "done"})
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for non-numeric col index, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestCompleteCardInvalidIndices(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "cpidx"})

	resp := doJSON(t, ts, "POST", "/boards/cpidx/cols/0/cards/abc/complete", nil)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for non-numeric card index, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestTagCardInvalidIndices(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "tgidx"})

	resp := doJSON(t, ts, "POST", "/boards/tgidx/cols/abc/cards/0/tag", map[string]any{"tags": []string{"x"}})
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for non-numeric col index, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestTagCardOutOfRange(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "tgor"})

	resp := doJSON(t, ts, "POST", "/boards/tgor/cols/0/cards/99/tag", map[string]any{"tags": []string{"x"}})
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for out-of-range card, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestMoveCardOutOfRange(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "mvor"})

	resp := doJSON(t, ts, "POST", "/boards/mvor/cols/0/cards/99/move", map[string]string{"column": "done"})
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for out-of-range card, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// --- Behavioral / integration tests ---

func TestCompleteCardToggleBack(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "toggle"})
	doJSON(t, ts, "POST", "/boards/toggle/columns/not now/cards", map[string]string{"title": "Task"})

	// Complete
	resp := doJSON(t, ts, "POST", "/boards/toggle/cols/0/cards/0/complete", nil)
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	resp = doJSON(t, ts, "GET", "/boards/toggle/cols/0/cards/0", nil)
	var cr cardResponse
	decodeResp(t, resp, &cr)
	if !cr.Completed {
		t.Fatal("expected card to be completed")
	}

	// Toggle back to incomplete
	resp = doJSON(t, ts, "POST", "/boards/toggle/cols/0/cards/0/complete", nil)
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	resp = doJSON(t, ts, "GET", "/boards/toggle/cols/0/cards/0", nil)
	decodeResp(t, resp, &cr)
	if cr.Completed {
		t.Fatal("expected card to be uncompleted after second toggle")
	}
}

func TestMultipleCardsOrdering(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "order"})

	doJSON(t, ts, "POST", "/boards/order/columns/not now/cards", map[string]string{"title": "First"})
	doJSON(t, ts, "POST", "/boards/order/columns/not now/cards", map[string]string{"title": "Second"})
	doJSON(t, ts, "POST", "/boards/order/columns/not now/cards", map[string]string{"title": "Third"})

	// Verify card ordering by index
	for i, expected := range []string{"First", "Second", "Third"} {
		resp := doJSON(t, ts, "GET", fmt.Sprintf("/boards/order/cols/0/cards/%d", i), nil)
		if resp.StatusCode != 200 {
			t.Fatalf("expected 200 for card %d, got %d", i, resp.StatusCode)
		}
		var cr cardResponse
		decodeResp(t, resp, &cr)
		if cr.Title != expected {
			t.Fatalf("card %d: expected %q, got %q", i, expected, cr.Title)
		}
	}
}

func TestDeleteCardShiftsIndices(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "shift"})

	doJSON(t, ts, "POST", "/boards/shift/columns/not now/cards", map[string]string{"title": "A"})
	doJSON(t, ts, "POST", "/boards/shift/columns/not now/cards", map[string]string{"title": "B"})
	doJSON(t, ts, "POST", "/boards/shift/columns/not now/cards", map[string]string{"title": "C"})

	// Delete middle card (index 1 = "B")
	resp := doJSON(t, ts, "DELETE", "/boards/shift/cols/0/cards/1", nil)
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Card at index 1 should now be "C"
	resp = doJSON(t, ts, "GET", "/boards/shift/cols/0/cards/1", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var cr cardResponse
	decodeResp(t, resp, &cr)
	if cr.Title != "C" {
		t.Fatalf("expected card at index 1 to be 'C' after delete, got %q", cr.Title)
	}

	// Only 2 cards remain
	resp = doJSON(t, ts, "GET", "/boards/shift", nil)
	var board models.Board
	decodeResp(t, resp, &board)
	if len(board.Columns[0].Cards) != 2 {
		t.Fatalf("expected 2 cards after delete, got %d", len(board.Columns[0].Cards))
	}
}

func TestMoveCardToNonexistentColumn(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "mvnc"})
	doJSON(t, ts, "POST", "/boards/mvnc/columns/not now/cards", map[string]string{"title": "T"})

	resp := doJSON(t, ts, "POST", "/boards/mvnc/cols/0/cards/0/move", map[string]string{"column": "nonexistent"})
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 for nonexistent target column, got %d", resp.StatusCode)
	}
	var errResp ErrorResponse
	decodeResp(t, resp, &errResp)
	if errResp.Status != 404 {
		t.Fatalf("expected error status 404, got %d", errResp.Status)
	}
	if errResp.Error == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestMoveColumnNonexistentColumn(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "mvcolnf"})

	resp := doJSON(t, ts, "POST", "/boards/mvcolnf/columns/nonexistent/move", map[string]string{"after": "done"})
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 for nonexistent column, got %d", resp.StatusCode)
	}
	var errResp ErrorResponse
	decodeResp(t, resp, &errResp)
	if errResp.Status != 404 {
		t.Fatalf("expected error status 404, got %d", errResp.Status)
	}
	if errResp.Error == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestErrorResponseFormat(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	resp := doJSON(t, ts, "GET", "/boards/nonexistent", nil)
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var errResp ErrorResponse
	decodeResp(t, resp, &errResp)
	if errResp.Status != 404 {
		t.Fatalf("expected status field 404, got %d", errResp.Status)
	}
	if errResp.Error == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestJSONContentTypeHeader(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	resp := doJSON(t, ts, "GET", "/boards", nil)
	ct := resp.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		t.Fatalf("failed to parse Content-Type %q: %v", ct, err)
	}
	if mediaType != "application/json" {
		t.Fatalf("expected media type 'application/json', got %q", mediaType)
	}
	_ = resp.Body.Close()
}
