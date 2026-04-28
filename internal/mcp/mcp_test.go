package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/workspace"
)

const testBoardMD = `---
name: Test Board
---

## Todo

- [ ] Task one
  priority: high
  tags: backend
  assignee: alice
  due: 2026-01-15

- [ ] Task two

## Done

- [x] Finished task
`

func setup(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	ws := workspace.Open(dir)
	eng := board.New()
	srv := New(ws, eng, "test")
	_ = os.WriteFile(filepath.Join(dir, "test-board.md"), []byte(testBoardMD), 0644)
	return srv, dir
}

// clientSession creates an in-memory MCP client session connected to the server.
func clientSession(t *testing.T, srv *Server) *mcpsdk.ClientSession {
	t.Helper()
	ctx := context.Background()
	st, ct := mcpsdk.NewInMemoryTransports()

	// Connect server first (non-blocking via goroutine).
	go func() {
		_, _ = srv.MCP().Connect(ctx, st, nil)
	}()

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0.1"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func callTool(t *testing.T, srv *Server, name string, args any) *mcpsdk.CallToolResult {
	t.Helper()
	cs := clientSession(t, srv)
	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	return result
}

func resultText(t *testing.T, r *mcpsdk.CallToolResult) string {
	t.Helper()
	for _, c := range r.Content {
		if tc, ok := c.(*mcpsdk.TextContent); ok {
			return tc.Text
		}
	}
	// Fallback: marshal content to string for debugging
	data, _ := json.Marshal(r.Content)
	t.Fatalf("no text content in result: %s", string(data))
	return ""
}

// ---- Tests ----

func TestNew(t *testing.T) {
	srv, _ := setup(t)
	if srv == nil {
		t.Fatal("server is nil")
	}
	if srv.MCP() == nil {
		t.Fatal("MCP() is nil")
	}
}

func TestHelpers(t *testing.T) {
	// errResult
	r, _, err := errResult(errors.New("boom"))
	if err != nil {
		t.Fatal(err)
	}
	if !r.IsError {
		t.Error("expected IsError")
	}
	if tc, ok := r.Content[0].(*mcpsdk.TextContent); ok {
		if tc.Text != "boom" {
			t.Errorf("got %q", tc.Text)
		}
	}

	// textResult
	r, _, err = textResult("hello")
	if err != nil {
		t.Fatal(err)
	}
	if r.IsError {
		t.Error("unexpected IsError")
	}

	// jsonResult
	r, _, err = jsonResult(map[string]string{"k": "v"})
	if err != nil {
		t.Fatal(err)
	}
	if tc, ok := r.Content[0].(*mcpsdk.TextContent); ok {
		if !strings.Contains(tc.Text, `"k"`) {
			t.Errorf("got %q", tc.Text)
		}
	}

	// boolPtr
	b := boolPtr(true)
	if b == nil || !*b {
		t.Error("boolPtr(true) failed")
	}
}

func TestStreamableHTTPHandler(t *testing.T) {
	srv, _ := setup(t)
	h := srv.StreamableHTTPHandler()
	if h == nil {
		t.Fatal("handler is nil")
	}
}

func TestListBoards(t *testing.T) {
	srv, _ := setup(t)
	r := callTool(t, srv, "list_boards", nil)
	text := resultText(t, r)
	if !strings.Contains(text, "Test Board") {
		t.Errorf("expected board name, got %s", text)
	}
	if !strings.Contains(text, `"columns"`) {
		t.Error("expected columns field")
	}
}

func TestGetBoard(t *testing.T) {
	srv, _ := setup(t)
	r := callTool(t, srv, "get_board", map[string]any{"board": "test-board"})
	text := resultText(t, r)
	if !strings.Contains(text, "Todo") {
		t.Error("expected Todo column")
	}
	if !strings.Contains(text, "Task one") {
		t.Error("expected Task one card")
	}
	if !strings.Contains(text, `"index"`) {
		t.Error("expected indexed structure")
	}
	if !strings.Contains(text, `"version"`) {
		t.Error("expected version field")
	}
}

func TestCreateBoard(t *testing.T) {
	srv, dir := setup(t)
	r := callTool(t, srv, "create_board", map[string]any{"name": "new-board"})
	text := resultText(t, r)
	if !strings.Contains(text, "Created") {
		t.Errorf("unexpected result: %s", text)
	}
	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, "new-board.md")); err != nil {
		t.Error("board file not created")
	}
}

func TestDeleteBoard(t *testing.T) {
	srv, dir := setup(t)
	r := callTool(t, srv, "delete_board", map[string]any{"board": "test-board"})
	text := resultText(t, r)
	if !strings.Contains(text, "Deleted") {
		t.Errorf("unexpected result: %s", text)
	}
	if _, err := os.Stat(filepath.Join(dir, "test-board.md")); !os.IsNotExist(err) {
		t.Error("board file still exists")
	}
}

func TestAddCard(t *testing.T) {
	srv, _ := setup(t)
	r := callTool(t, srv, "add_card", map[string]any{
		"board":  "test-board",
		"column": "Todo",
		"title":  "New card",
	})
	text := resultText(t, r)
	if !strings.Contains(text, "Added card") {
		t.Errorf("unexpected: %s", text)
	}

	// Verify via get_board
	r2 := callTool(t, srv, "get_board", map[string]any{"board": "test-board"})
	if !strings.Contains(resultText(t, r2), "New card") {
		t.Error("card not found after add")
	}
}

func TestAddCardPrepend(t *testing.T) {
	srv, _ := setup(t)
	callTool(t, srv, "add_card", map[string]any{
		"board":   "test-board",
		"column":  "Todo",
		"title":   "Prepended card",
		"prepend": true,
	})
	r := callTool(t, srv, "get_board", map[string]any{"board": "test-board"})
	text := resultText(t, r)
	// Prepended card should appear at index 0
	if !strings.Contains(text, "Prepended card") {
		t.Error("prepended card not found")
	}
}

func TestShowCard(t *testing.T) {
	srv, _ := setup(t)
	r := callTool(t, srv, "show_card", map[string]any{
		"board":        "test-board",
		"column_index": 0,
		"card_index":   0,
	})
	text := resultText(t, r)
	if !strings.Contains(text, "Task one") {
		t.Error("expected Task one")
	}
	if !strings.Contains(text, "Todo") {
		t.Error("expected column name")
	}
	if !strings.Contains(text, "high") {
		t.Error("expected priority")
	}
}

func TestEditCard(t *testing.T) {
	srv, _ := setup(t)
	r := callTool(t, srv, "edit_card", map[string]any{
		"board":        "test-board",
		"column_index": 0,
		"card_index":   0,
		"title":        "Updated task",
		"priority":     "low",
		"tags":         []string{"frontend"},
		"assignee":     "bob",
		"due":          "2026-06-01",
		"body":         "new body text",
	})
	text := resultText(t, r)
	if !strings.Contains(text, "updated") {
		t.Errorf("unexpected: %s", text)
	}

	// Verify changes
	r2 := callTool(t, srv, "show_card", map[string]any{
		"board":        "test-board",
		"column_index": 0,
		"card_index":   0,
	})
	show := resultText(t, r2)
	if !strings.Contains(show, "Updated task") {
		t.Error("title not updated")
	}
	if !strings.Contains(show, "low") {
		t.Error("priority not updated")
	}
}

func TestMoveCard(t *testing.T) {
	srv, _ := setup(t)
	r := callTool(t, srv, "move_card", map[string]any{
		"board":         "test-board",
		"column_index":  0,
		"card_index":    0,
		"target_column": "Done",
	})
	text := resultText(t, r)
	if !strings.Contains(text, "moved") {
		t.Errorf("unexpected: %s", text)
	}

	// Verify card is in Done
	r2 := callTool(t, srv, "get_board", map[string]any{"board": "test-board"})
	var board indexedBoard
	_ = json.Unmarshal([]byte(resultText(t, r2)), &board)
	for _, col := range board.Columns {
		if col.Name == "Done" {
			found := false
			for _, card := range col.Cards {
				if card.Title == "Task one" {
					found = true
				}
			}
			if !found {
				t.Error("card not found in Done column")
			}
		}
	}
}

func TestCompleteCard(t *testing.T) {
	srv, _ := setup(t)
	// Task one is incomplete
	r := callTool(t, srv, "complete_card", map[string]any{
		"board":        "test-board",
		"column_index": 0,
		"card_index":   0,
	})
	text := resultText(t, r)
	if !strings.Contains(text, "toggled") {
		t.Errorf("unexpected: %s", text)
	}

	// Verify it's completed
	r2 := callTool(t, srv, "show_card", map[string]any{
		"board":        "test-board",
		"column_index": 0,
		"card_index":   0,
	})
	if !strings.Contains(resultText(t, r2), "true") {
		t.Error("card not marked completed")
	}
}

func TestDeleteCard(t *testing.T) {
	srv, _ := setup(t)
	r := callTool(t, srv, "delete_card", map[string]any{
		"board":        "test-board",
		"column_index": 0,
		"card_index":   0,
	})
	text := resultText(t, r)
	if !strings.Contains(text, "deleted") {
		t.Errorf("unexpected: %s", text)
	}

	// Verify card count reduced
	r2 := callTool(t, srv, "get_board", map[string]any{"board": "test-board"})
	var b indexedBoard
	_ = json.Unmarshal([]byte(resultText(t, r2)), &b)
	if len(b.Columns[0].Cards) != 1 {
		t.Errorf("expected 1 card in Todo, got %d", len(b.Columns[0].Cards))
	}
}

func TestAddColumn(t *testing.T) {
	srv, _ := setup(t)
	r := callTool(t, srv, "add_column", map[string]any{
		"board": "test-board",
		"name":  "In Progress",
	})
	text := resultText(t, r)
	if !strings.Contains(text, "Added column") {
		t.Errorf("unexpected: %s", text)
	}

	r2 := callTool(t, srv, "get_board", map[string]any{"board": "test-board"})
	if !strings.Contains(resultText(t, r2), "In Progress") {
		t.Error("column not added")
	}
}

func TestDeleteColumn(t *testing.T) {
	srv, _ := setup(t)
	r := callTool(t, srv, "delete_column", map[string]any{
		"board": "test-board",
		"name":  "Done",
	})
	text := resultText(t, r)
	if !strings.Contains(text, "Deleted column") {
		t.Errorf("unexpected: %s", text)
	}

	r2 := callTool(t, srv, "get_board", map[string]any{"board": "test-board"})
	if strings.Contains(resultText(t, r2), `"name": "Done"`) {
		t.Error("column still exists")
	}
}

func TestRenameColumn(t *testing.T) {
	srv, _ := setup(t)
	r := callTool(t, srv, "rename_column", map[string]any{
		"board":    "test-board",
		"old_name": "Todo",
		"new_name": "Backlog",
	})
	text := resultText(t, r)
	if !strings.Contains(text, "Renamed") {
		t.Errorf("unexpected: %s", text)
	}

	r2 := callTool(t, srv, "get_board", map[string]any{"board": "test-board"})
	got := resultText(t, r2)
	if !strings.Contains(got, "Backlog") {
		t.Error("column not renamed")
	}
}

func TestMoveColumn(t *testing.T) {
	srv, _ := setup(t)
	// Move "Todo" after "Done" (was first, now second)
	r := callTool(t, srv, "move_column", map[string]any{
		"board": "test-board",
		"name":  "Todo",
		"after": "Done",
	})
	text := resultText(t, r)
	if !strings.Contains(text, "Moved") {
		t.Errorf("unexpected: %s", text)
	}

	r2 := callTool(t, srv, "get_board", map[string]any{"board": "test-board"})
	var b indexedBoard
	_ = json.Unmarshal([]byte(resultText(t, r2)), &b)
	if len(b.Columns) < 2 {
		t.Fatal("not enough columns")
	}
	if b.Columns[0].Name != "Done" {
		t.Errorf("expected Done first, got %s", b.Columns[0].Name)
	}
}

func TestSortColumn(t *testing.T) {
	srv, _ := setup(t)
	r := callTool(t, srv, "sort_column", map[string]any{
		"board":        "test-board",
		"column_index": 0,
		"sort_by":      "name",
	})
	text := resultText(t, r)
	if !strings.Contains(text, "sorted") {
		t.Errorf("unexpected: %s", text)
	}
}

func TestSortColumnByPriority(t *testing.T) {
	srv, _ := setup(t)
	r := callTool(t, srv, "sort_column", map[string]any{
		"board":        "test-board",
		"column_index": 0,
		"sort_by":      "priority",
	})
	text := resultText(t, r)
	if !strings.Contains(text, "sorted by priority") {
		t.Errorf("unexpected: %s", text)
	}
}

func TestSortColumnByDue(t *testing.T) {
	srv, _ := setup(t)
	r := callTool(t, srv, "sort_column", map[string]any{
		"board":        "test-board",
		"column_index": 0,
		"sort_by":      "due",
	})
	text := resultText(t, r)
	if !strings.Contains(text, "sorted by due") {
		t.Errorf("unexpected: %s", text)
	}
}

// ---- Error cases ----

func TestErrorInvalidBoard(t *testing.T) {
	srv, _ := setup(t)

	// get_board with nonexistent board
	r := callTool(t, srv, "get_board", map[string]any{"board": "nonexistent"})
	if !r.IsError {
		t.Error("expected error for nonexistent board")
	}

	// delete_board with nonexistent board
	r = callTool(t, srv, "delete_board", map[string]any{"board": "nonexistent"})
	if !r.IsError {
		t.Error("expected error for nonexistent board delete")
	}
}

func TestErrorInvalidColumn(t *testing.T) {
	srv, _ := setup(t)

	// add_card to nonexistent column
	r := callTool(t, srv, "add_card", map[string]any{
		"board":  "test-board",
		"column": "Nonexistent",
		"title":  "fail",
	})
	if !r.IsError {
		t.Error("expected error for nonexistent column")
	}

	// delete nonexistent column — engine does not error, just no-ops
	_ = callTool(t, srv, "delete_column", map[string]any{
		"board": "test-board",
		"name":  "Nonexistent",
	})

	// rename nonexistent column
	r = callTool(t, srv, "rename_column", map[string]any{
		"board":    "test-board",
		"old_name": "Nonexistent",
		"new_name": "Foo",
	})
	if !r.IsError {
		t.Error("expected error for nonexistent column rename")
	}

	// move nonexistent column
	r = callTool(t, srv, "move_column", map[string]any{
		"board": "test-board",
		"name":  "Nonexistent",
		"after": "Todo",
	})
	if !r.IsError {
		t.Error("expected error for nonexistent column move")
	}
}

func TestErrorInvalidCardIndex(t *testing.T) {
	srv, _ := setup(t)

	// show_card out of range
	r := callTool(t, srv, "show_card", map[string]any{
		"board":        "test-board",
		"column_index": 0,
		"card_index":   99,
	})
	if !r.IsError {
		t.Error("expected error for out-of-range card")
	}

	// edit_card out of range
	r = callTool(t, srv, "edit_card", map[string]any{
		"board":        "test-board",
		"column_index": 0,
		"card_index":   99,
		"title":        "fail",
	})
	if !r.IsError {
		t.Error("expected error for out-of-range edit")
	}

	// delete_card out of range
	r = callTool(t, srv, "delete_card", map[string]any{
		"board":        "test-board",
		"column_index": 0,
		"card_index":   99,
	})
	if !r.IsError {
		t.Error("expected error for out-of-range delete")
	}

	// move_card out of range
	r = callTool(t, srv, "move_card", map[string]any{
		"board":         "test-board",
		"column_index":  0,
		"card_index":    99,
		"target_column": "Done",
	})
	if !r.IsError {
		t.Error("expected error for out-of-range move")
	}

	// complete_card out of range
	r = callTool(t, srv, "complete_card", map[string]any{
		"board":        "test-board",
		"column_index": 0,
		"card_index":   99,
	})
	if !r.IsError {
		t.Error("expected error for out-of-range complete")
	}

	// out of range column index
	r = callTool(t, srv, "show_card", map[string]any{
		"board":        "test-board",
		"column_index": 99,
		"card_index":   0,
	})
	if !r.IsError {
		t.Error("expected error for out-of-range column index")
	}

	// sort out of range column
	r = callTool(t, srv, "sort_column", map[string]any{
		"board":        "test-board",
		"column_index": 99,
		"sort_by":      "name",
	})
	if !r.IsError {
		t.Error("expected error for out-of-range sort column")
	}
}

func TestErrorBoardPathTraversal(t *testing.T) {
	srv, _ := setup(t)

	r := callTool(t, srv, "add_card", map[string]any{
		"board":  "../../../etc/passwd",
		"column": "Todo",
		"title":  "hack",
	})
	if !r.IsError {
		t.Error("expected error for path traversal")
	}
}

// mcpHiddenMutations lists mutation variants from board.MutationVariantNames()
// that are deliberately NOT exposed as MCP tools, with the reason. Adding a
// new mutation must either:
//   - register an MCP tool with the same name, or
//   - add an entry here.
//
// This forces the decision at the time of writing rather than letting AI
// surfaces silently lag behind the wire-level mutation registry. Reasons are
// kept terse so future maintainers can re-evaluate without spelunking.
var mcpHiddenMutations = map[string]string{
	"reorder_card":           "drag-drop visual ordering; LLMs use move_card for column changes",
	"tag_card":               "tags edited via edit_card; redundant surface",
	"toggle_column_collapse": "renderer UI state, not workflow",
	"update_board_meta":      "board name/description rename — exposing risks AI-driven renames mid-conversation",
	"update_board_members":   "workspace-admin concern, not LLM-driven",
	"update_board_icon":      "cosmetic; no LLM use case",
	"update_board_settings":  "per-board UI preferences (view_mode, etc.) — not LLM-driven",
	"move_card_to_board":     "two-phase cross-board write; needs special handling beyond engine.Apply*",
}

// TestMCPMutationCoverage asserts every mutation registered in
// internal/board is either exposed as an identically-named MCP tool or
// explicitly listed in mcpHiddenMutations. Without this guard, adding a
// mutation to internal/board/mutation.go silently leaves the MCP surface
// out of sync — and the divergence is invisible until a human tries to
// drive the new operation through an LLM.
func TestMCPMutationCoverage(t *testing.T) {
	srv, _ := setup(t)
	cs := clientSession(t, srv)

	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	exposed := make(map[string]struct{}, len(res.Tools))
	for _, tool := range res.Tools {
		exposed[tool.Name] = struct{}{}
	}

	for _, name := range board.MutationVariantNames() {
		_, isExposed := exposed[name]
		reason, isHidden := mcpHiddenMutations[name]
		switch {
		case isExposed && isHidden:
			t.Errorf("mutation %q is both an MCP tool and listed in mcpHiddenMutations (reason=%q) — pick one", name, reason)
		case !isExposed && !isHidden:
			t.Errorf("mutation %q has no MCP tool and no entry in mcpHiddenMutations — either register a tool or add a hide-reason", name)
		}
	}

	// Catch entries that no longer correspond to a real mutation (e.g. a
	// variant was removed from the registry but the hide-reason was left
	// stale). Without this, the allowlist accumulates dead names.
	live := make(map[string]struct{}, 32)
	for _, name := range board.MutationVariantNames() {
		live[name] = struct{}{}
	}
	for name := range mcpHiddenMutations {
		if _, ok := live[name]; !ok {
			t.Errorf("mcpHiddenMutations lists %q but no such mutation exists in the registry", name)
		}
	}
}
