package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	if len(board.Columns) != 4 {
		t.Fatalf("expected 4 default columns, got %d", len(board.Columns))
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

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "tasks"})

	// Add card
	resp := doJSON(t, ts, "POST", "/boards/tasks/columns/Backlog/cards", map[string]string{"title": "Fix bug"})
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var card models.Card
	decodeResp(t, resp, &card)
	if card.Title != "Fix bug" {
		t.Fatalf("expected title 'Fix bug', got %q", card.Title)
	}
	if card.ID == "" {
		t.Fatal("expected card ID to be set")
	}

	// Get card
	resp = doJSON(t, ts, "GET", "/cards/"+card.ID, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var cr cardResponse
	decodeResp(t, resp, &cr)
	if cr.Column != "Backlog" {
		t.Fatalf("expected column 'Backlog', got %q", cr.Column)
	}

	// Delete card
	resp = doJSON(t, ts, "DELETE", "/cards/"+card.ID, nil)
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Verify deleted
	resp = doJSON(t, ts, "GET", "/cards/"+card.ID, nil)
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestMoveCard(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "flow"})

	resp := doJSON(t, ts, "POST", "/boards/flow/columns/Backlog/cards", map[string]string{"title": "Task A"})
	var card models.Card
	decodeResp(t, resp, &card)

	resp = doJSON(t, ts, "POST", "/cards/"+card.ID+"/move", map[string]string{"column": "Done"})
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	resp = doJSON(t, ts, "GET", "/cards/"+card.ID, nil)
	var cr cardResponse
	decodeResp(t, resp, &cr)
	if cr.Column != "Done" {
		t.Fatalf("expected column 'Done', got %q", cr.Column)
	}
}

func TestCompleteCard(t *testing.T) {
	ts := setupTest(t)
	defer ts.Close()

	doJSON(t, ts, "POST", "/boards", map[string]string{"name": "done"})

	resp := doJSON(t, ts, "POST", "/boards/done/columns/Backlog/cards", map[string]string{"title": "Finish"})
	var card models.Card
	decodeResp(t, resp, &card)

	resp = doJSON(t, ts, "POST", "/cards/"+card.ID+"/complete", nil)
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	resp = doJSON(t, ts, "GET", "/cards/"+card.ID, nil)
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

	resp := doJSON(t, ts, "POST", "/boards/tags/columns/Backlog/cards", map[string]string{"title": "Label me"})
	var card models.Card
	decodeResp(t, resp, &card)

	resp = doJSON(t, ts, "POST", "/cards/"+card.ID+"/tag", map[string]any{"tags": []string{"urgent", "bug"}})
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	resp = doJSON(t, ts, "GET", "/cards/"+card.ID, nil)
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

	resp = doJSON(t, ts, "GET", "/cards/00000000-0000-0000-0000-000000000000", nil)
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 for missing card, got %d", resp.StatusCode)
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
