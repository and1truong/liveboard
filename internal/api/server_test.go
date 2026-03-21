package api

import (
	"bytes"
	"encoding/json"
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
	srv := NewServer(ws, ws.Engine, nil, false)
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
	if resp.StatusCode != 500 {
		t.Fatalf("expected 500 for out-of-range, got %d", resp.StatusCode)
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
	if resp.StatusCode != 500 {
		t.Fatalf("expected 500 for out-of-range card, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestCompleteCardOutOfRange(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "cmp"})

	resp := doJSON(t, ts, "POST", "/boards/cmp/cols/0/cards/99/complete", nil)
	if resp.StatusCode != 500 {
		t.Fatalf("expected 500 for out-of-range card, got %d", resp.StatusCode)
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
