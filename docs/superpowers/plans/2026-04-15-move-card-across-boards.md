# Move Card Across Boards Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow users to move a card from one board into a column on a different board, via the card modal and the right-click quick-edit menu.

**Architecture:** A new engine method `Engine.MoveCardToBoard` performs the move as two sequential `MutateBoard` calls — target-first (version bypass), then source (version-checked). No cross-board locking; failure after target-write leaves a duplicate that the user can delete. Auto-merges missing tags/members into the target frontmatter. Exposed via a new web handler and REST endpoint. A new `/api/boards/list-lite` endpoint feeds cascading board/column selects in the card modal and a submenu in the quick-edit context menu.

**Tech Stack:** Go 1.24, chi/v5, HTMX + SSE, Alpine.js, vanilla JS.

---

## File Structure

**Create:**
- (none)

**Modify:**
- `internal/board/board.go` — add `MoveCardToBoard` method
- `internal/board/board_test.go` — engine tests
- `internal/web/board_card_handlers.go` — add `HandleMoveCardToBoard`
- `internal/web/board_list_handler.go` — add `HandleBoardsListLite`
- `internal/web/handler_forwarding.go` — forward to `HandleBoardsListLite`
- `internal/web/web_test.go` — web handler tests
- `internal/api/cards.go` — add `moveCardToBoard` REST handler
- `internal/api/server.go` — register new routes
- `internal/api/server_test.go` — API tests
- `internal/templates/layout.html` — add "Move to board" UI block inside the card modal
- `web/js/liveboard.card-modal.js` — cascading selects + submit
- `web/js/liveboard.drag.js` — "Move to board ▸" submenu in quick-edit

---

## Task 1: Engine — `MoveCardToBoard` happy path

**Files:**
- Modify: `internal/board/board.go`
- Test: `internal/board/board_test.go`

- [ ] **Step 1: Write the failing test**

Add at the bottom of `internal/board/board_test.go`:

```go
func TestMoveCardToBoard_HappyPath(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.md")
	dstPath := filepath.Join(dir, "dst.md")

	srcMD := "---\nversion: 3\nname: Src\ntags: [alpha]\nmembers: [alice]\n---\n\n## Todo\n\n- [ ] Task A\n  tags: alpha\n  assignee: alice\n\n## Done\n"
	dstMD := "---\nversion: 7\nname: Dst\ntags: [beta]\nmembers: [bob]\n---\n\n## Inbox\n\n- [ ] Existing\n"

	if err := os.WriteFile(srcPath, []byte(srcMD), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dstPath, []byte(dstMD), 0644); err != nil {
		t.Fatal(err)
	}

	e := New()
	if err := e.MoveCardToBoard(srcPath, 3, 0, 0, dstPath, "Inbox"); err != nil {
		t.Fatalf("MoveCardToBoard: %v", err)
	}

	src, err := e.LoadBoard(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	dst, err := e.LoadBoard(dstPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(src.Columns[0].Cards) != 0 {
		t.Errorf("source Todo should be empty, got %d cards", len(src.Columns[0].Cards))
	}
	if src.Version != 4 {
		t.Errorf("source version = %d, want 4", src.Version)
	}
	if len(dst.Columns[0].Cards) != 2 || dst.Columns[0].Cards[0].Title != "Task A" {
		t.Errorf("dst Inbox = %#v, want [Task A, Existing]", dst.Columns[0].Cards)
	}
	if dst.Version != 8 {
		t.Errorf("dst version = %d, want 8", dst.Version)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/board/ -run TestMoveCardToBoard_HappyPath -v`
Expected: FAIL with "e.MoveCardToBoard undefined".

- [ ] **Step 3: Add minimal implementation**

Append to `internal/board/board.go` (after `MoveCard`):

```go
// MoveCardToBoard moves a card from srcPath to dstColumn on dstPath.
// The card is inserted at the top of the target column. Missing tags and
// members on the target board's frontmatter are auto-added.
//
// Not atomic across boards: target is written first (version bypass), then
// source (version-checked against srcVersion). If the source write fails
// after the target write succeeded, the card is duplicated and the caller
// receives a wrapped error.
func (e *Engine) MoveCardToBoard(srcPath string, srcVersion, srcColIdx, cardIdx int, dstPath, dstColumn string) error {
	if srcPath == dstPath {
		return fmt.Errorf("source and destination boards must differ: %w", ErrNotFound)
	}

	// Snapshot the card from source without holding any lock. The real
	// version check happens in the source MutateBoard below.
	srcSnapshot, err := e.LoadBoard(srcPath)
	if err != nil {
		return err
	}
	if err := validateIndices(srcSnapshot, srcColIdx, cardIdx); err != nil {
		return err
	}
	cardCopy := srcSnapshot.Columns[srcColIdx].Cards[cardIdx]

	// Step 1: insert into target (version bypass — client never saw it).
	if err := e.MutateBoard(dstPath, -1, func(b *models.Board) error {
		for i := range b.Columns {
			if b.Columns[i].Name == dstColumn {
				b.Columns[i].Cards = append([]models.Card{cardCopy}, b.Columns[i].Cards...)
				mergeMissing(&b.Tags, cardCopy.Tags)
				if cardCopy.Assignee != "" {
					mergeMissing(&b.Members, []string{cardCopy.Assignee})
				}
				return nil
			}
		}
		return fmt.Errorf("target column %q: %w", dstColumn, ErrNotFound)
	}); err != nil {
		return err
	}

	// Step 2: remove from source (version-checked).
	if err := e.MutateBoard(srcPath, srcVersion, func(b *models.Board) error {
		if err := validateIndices(b, srcColIdx, cardIdx); err != nil {
			return err
		}
		b.Columns[srcColIdx].Cards = removeCardAt(b.Columns[srcColIdx].Cards, cardIdx)
		return nil
	}); err != nil {
		return fmt.Errorf("card added to %s but source removal failed: %w", dstPath, err)
	}
	return nil
}

func mergeMissing(existing *[]string, incoming []string) {
	for _, v := range incoming {
		if v == "" {
			continue
		}
		if !slices.Contains(*existing, v) {
			*existing = append(*existing, v)
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/board/ -run TestMoveCardToBoard_HappyPath -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/board/board.go internal/board/board_test.go
git commit -m "feat(board): add MoveCardToBoard engine method"
```

---

## Task 2: Engine — tag/member merge, error cases

**Files:**
- Test: `internal/board/board_test.go`

- [ ] **Step 1: Write failing tests**

Append to `internal/board/board_test.go`:

```go
func TestMoveCardToBoard_MergesTagsAndMembers(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.md")
	dstPath := filepath.Join(dir, "dst.md")

	srcMD := "---\nversion: 1\nname: Src\ntags: [urgent, legal]\nmembers: [carol]\n---\n\n## Todo\n\n- [ ] Task\n  tags: urgent, legal\n  assignee: carol\n"
	dstMD := "---\nversion: 1\nname: Dst\ntags: [urgent]\nmembers: []\n---\n\n## Inbox\n"

	_ = os.WriteFile(srcPath, []byte(srcMD), 0644)
	_ = os.WriteFile(dstPath, []byte(dstMD), 0644)

	e := New()
	if err := e.MoveCardToBoard(srcPath, 1, 0, 0, dstPath, "Inbox"); err != nil {
		t.Fatal(err)
	}
	dst, _ := e.LoadBoard(dstPath)
	if !slices.Contains(dst.Tags, "legal") {
		t.Errorf("dst.Tags = %v, want contains legal", dst.Tags)
	}
	if !slices.Contains(dst.Members, "carol") {
		t.Errorf("dst.Members = %v, want contains carol", dst.Members)
	}
}

func TestMoveCardToBoard_TargetColumnNotFound(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.md")
	dstPath := filepath.Join(dir, "dst.md")
	_ = os.WriteFile(srcPath, []byte("---\nversion: 1\nname: S\n---\n\n## Todo\n\n- [ ] T\n"), 0644)
	_ = os.WriteFile(dstPath, []byte("---\nversion: 1\nname: D\n---\n\n## Inbox\n"), 0644)

	e := New()
	err := e.MoveCardToBoard(srcPath, 1, 0, 0, dstPath, "Nope")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	src, _ := e.LoadBoard(srcPath)
	if len(src.Columns[0].Cards) != 1 {
		t.Error("source should be unchanged after failed target lookup")
	}
}

func TestMoveCardToBoard_SourceVersionConflict(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.md")
	dstPath := filepath.Join(dir, "dst.md")
	_ = os.WriteFile(srcPath, []byte("---\nversion: 5\nname: S\n---\n\n## Todo\n\n- [ ] T\n"), 0644)
	_ = os.WriteFile(dstPath, []byte("---\nversion: 1\nname: D\n---\n\n## Inbox\n"), 0644)

	e := New()
	err := e.MoveCardToBoard(srcPath, 2 /* stale */, 0, 0, dstPath, "Inbox")
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("err = %v, want ErrVersionConflict", err)
	}
	// Target was already written — documents the duplicate-on-failure contract.
	dst, _ := e.LoadBoard(dstPath)
	if len(dst.Columns[0].Cards) != 1 {
		t.Error("target should have the card even though source removal failed")
	}
}

func TestMoveCardToBoard_SameBoardRejected(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "b.md")
	_ = os.WriteFile(p, []byte("---\nversion: 1\nname: B\n---\n\n## A\n\n- [ ] T\n\n## B\n"), 0644)
	e := New()
	if err := e.MoveCardToBoard(p, 1, 0, 0, p, "B"); err == nil {
		t.Fatal("expected error for same-board move")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail, then pass**

Run: `go test ./internal/board/ -run TestMoveCardToBoard -v`
Expected: all pass (implementation from Task 1 already handles these cases).

- [ ] **Step 3: Commit**

```bash
git add internal/board/board_test.go
git commit -m "test(board): cover MoveCardToBoard edge cases"
```

---

## Task 3: Web handler — `HandleMoveCardToBoard`

**Files:**
- Modify: `internal/web/board_card_handlers.go`
- Modify: `internal/api/server.go`
- Test: `internal/web/web_test.go`

- [ ] **Step 1: Write failing test**

Find an existing `HandleMoveCard` test in `internal/web/web_test.go` and add a parallel test beside it:

```go
func TestHandleMoveCardToBoard(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.md")
	dstPath := filepath.Join(dir, "dst.md")
	_ = os.WriteFile(srcPath, []byte("---\nversion: 0\nname: Src\n---\n\n## Todo\n\n- [ ] Task A\n"), 0644)
	_ = os.WriteFile(dstPath, []byte("---\nversion: 0\nname: Dst\n---\n\n## Inbox\n"), 0644)

	h := newTestHandler(t, dir)

	form := url.Values{}
	form.Set("col_idx", "0")
	form.Set("card_idx", "0")
	form.Set("dst_board", "dst")
	form.Set("dst_column", "Inbox")
	form.Set("version", "0")

	req := httptest.NewRequest(http.MethodPost, "/board/src/cards/move-to-board", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = withURLParam(req, "slug", "src")
	w := httptest.NewRecorder()

	h.BoardView.HandleMoveCardToBoard(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	dst, _ := h.ws.Engine.LoadBoard(dstPath)
	if len(dst.Columns[0].Cards) != 1 || dst.Columns[0].Cards[0].Title != "Task A" {
		t.Errorf("dst cards = %#v", dst.Columns[0].Cards)
	}
}
```

(Reuse whatever `newTestHandler` / `withURLParam` helpers the surrounding tests use — copy the pattern from `TestHandleMoveCard` in the same file.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/web/ -run TestHandleMoveCardToBoard -v`
Expected: FAIL — method undefined.

- [ ] **Step 3: Add the handler**

Append to `internal/web/board_card_handlers.go`:

```go
// HandleMoveCardToBoard handles POST /board/{slug}/cards/move-to-board.
func (bv *BoardViewHandler) HandleMoveCardToBoard(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	colIdx, err := formInt(r, "col_idx")
	if err != nil {
		model, _ := bv.boardViewModel(slug)
		model.Error = err.Error()
		bv.renderBoardContent(w, model)
		return
	}
	cardIdx, err := formInt(r, "card_idx")
	if err != nil {
		model, _ := bv.boardViewModel(slug)
		model.Error = err.Error()
		bv.renderBoardContent(w, model)
		return
	}

	dstBoardSlug := r.FormValue("dst_board")
	dstColumn := r.FormValue("dst_column")
	if slug == "" || dstBoardSlug == "" || dstColumn == "" {
		model, _ := bv.boardViewModel(slug)
		model.Error = "source slug, destination board, and destination column are required"
		bv.renderBoardContent(w, model)
		return
	}

	srcPath, err := bv.ws.BoardPath(slug)
	if err != nil {
		model, _ := bv.boardViewModel(slug)
		model.Error = err.Error()
		bv.renderBoardContent(w, model)
		return
	}
	dstPath, err := bv.ws.BoardPath(dstBoardSlug)
	if err != nil {
		model, _ := bv.boardViewModel(slug)
		model.Error = err.Error()
		bv.renderBoardContent(w, model)
		return
	}

	version := formVersion(r)
	moveErr := bv.ws.Engine.MoveCardToBoard(srcPath, version, colIdx, cardIdx, dstPath, dstColumn)
	if errors.Is(moveErr, board.ErrVersionConflict) {
		bv.handleConflict(w, slug)
		return
	}

	model, _ := bv.boardViewModel(slug)
	if moveErr != nil {
		model.Error = moveErr.Error()
	} else {
		// Notify target-board viewers via SSE. Source board is refreshed by the swap we return.
		bv.sse.Publish(dstBoardSlug)
	}
	bv.renderBoardContent(w, model)
}
```

(If `bv.sse.Publish` isn't the exact symbol, match whatever the existing handlers call — search for `Publish(` nearby and mirror.)

- [ ] **Step 4: Register the route**

In `internal/api/server.go`, after `r.Post("/board/{slug}/cards/move", ...)` (line 249):

```go
r.Post("/board/{slug}/cards/move-to-board", h.BoardView.HandleMoveCardToBoard)
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/web/ -run TestHandleMoveCardToBoard -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/web/board_card_handlers.go internal/web/web_test.go internal/api/server.go
git commit -m "feat(web): add HandleMoveCardToBoard handler and route"
```

---

## Task 4: Boards-lite endpoint

**Files:**
- Modify: `internal/web/board_list_handler.go`
- Modify: `internal/web/handler_forwarding.go`
- Modify: `internal/api/server.go`
- Test: `internal/web/web_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/web/web_test.go`:

```go
func TestHandleBoardsListLite(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "one.md"), []byte("---\nversion: 0\nname: One\n---\n\n## A\n\n## B\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "two.md"), []byte("---\nversion: 0\nname: Two\n---\n\n## Inbox\n"), 0644)

	h := newTestHandler(t, dir)
	req := httptest.NewRequest(http.MethodGet, "/api/boards/list-lite", nil)
	w := httptest.NewRecorder()
	h.BoardList.HandleBoardsListLite(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp []struct {
		Slug    string   `json:"slug"`
		Name    string   `json:"name"`
		Columns []string `json:"columns"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp) != 2 {
		t.Fatalf("got %d boards, want 2", len(resp))
	}
	bySlug := map[string][]string{}
	for _, b := range resp {
		bySlug[b.Slug] = b.Columns
	}
	if !slices.Equal(bySlug["one"], []string{"A", "B"}) {
		t.Errorf("one.columns = %v", bySlug["one"])
	}
	if !slices.Equal(bySlug["two"], []string{"Inbox"}) {
		t.Errorf("two.columns = %v", bySlug["two"])
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/web/ -run TestHandleBoardsListLite -v`
Expected: FAIL — method undefined.

- [ ] **Step 3: Add the handler**

Append to `internal/web/board_list_handler.go`:

```go
// HandleBoardsListLite handles GET /api/boards/list-lite — returns a slim
// list of boards with their column names for populating cross-board
// destination pickers.
func (bl *BoardListHandler) HandleBoardsListLite(w http.ResponseWriter, r *http.Request) {
	boards, err := bl.ws.ListBoards()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type lite struct {
		Slug    string   `json:"slug"`
		Name    string   `json:"name"`
		Columns []string `json:"columns"`
	}
	out := make([]lite, 0, len(boards))
	for _, b := range boards {
		loaded, err := bl.ws.LoadBoard(b.Slug)
		if err != nil {
			continue
		}
		cols := make([]string, 0, len(loaded.Columns))
		for _, c := range loaded.Columns {
			cols = append(cols, c.Name)
		}
		out = append(out, lite{Slug: b.Slug, Name: loaded.Name, Columns: cols})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
```

Add `"encoding/json"` to the imports in that file if not already present.

(If the iteration shape for `ListBoards` differs — inspect the file and mirror the existing loop pattern used by `HandleSidebarBoards`.)

- [ ] **Step 4: Add forwarding shim**

In `internal/web/handler_forwarding.go`, after the `HandleSidebarBoards` forwarder:

```go
// HandleBoardsListLite forwards to BoardList.
func (h *Handler) HandleBoardsListLite(w http.ResponseWriter, r *http.Request) {
	h.BoardList.HandleBoardsListLite(w, r)
}
```

- [ ] **Step 5: Register the route**

In `internal/api/server.go`, after `r.Get("/api/boards/sidebar", h.BoardList.HandleSidebarBoards)` (line 266):

```go
r.Get("/api/boards/list-lite", h.BoardList.HandleBoardsListLite)
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/web/ -run TestHandleBoardsListLite -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/web/board_list_handler.go internal/web/handler_forwarding.go internal/api/server.go internal/web/web_test.go
git commit -m "feat(web): add /api/boards/list-lite endpoint"
```

---

## Task 5: REST API — `moveCardToBoard`

**Files:**
- Modify: `internal/api/cards.go`
- Modify: `internal/api/server.go`
- Test: `internal/api/server_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/api/server_test.go`, near the existing `moveCard` test (search for `/move` POST tests):

```go
func TestAPIMoveCardToBoard(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "src.md"), []byte("---\nversion: 0\nname: Src\n---\n\n## Todo\n\n- [ ] Task A\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "dst.md"), []byte("---\nversion: 0\nname: Dst\n---\n\n## Inbox\n"), 0644)

	srv := newTestServer(t, dir)
	body := `{"src_col_idx":0,"card_idx":0,"dst_board":"dst","dst_column":"Inbox"}`
	req := httptest.NewRequest(http.MethodPost, "/api/boards/src/cards/move-to-board", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	dst, _ := srv.ws.Engine.LoadBoard(filepath.Join(dir, "dst.md"))
	if len(dst.Columns[0].Cards) != 1 {
		t.Errorf("dst cards = %d, want 1", len(dst.Columns[0].Cards))
	}
}
```

(Use whatever `newTestServer` helper `server_test.go` already provides — match existing conventions.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/api/ -run TestAPIMoveCardToBoard -v`
Expected: FAIL — 404.

- [ ] **Step 3: Add handler**

Append to `internal/api/cards.go`:

```go
func (s *Server) moveCardToBoard(w http.ResponseWriter, r *http.Request) {
	boardName := pathParam(r, "board")
	var body struct {
		SrcColIdx int    `json:"src_col_idx"`
		CardIdx   int    `json:"card_idx"`
		DstBoard  string `json:"dst_board"`
		DstColumn string `json:"dst_column"`
	}
	if err := decodeJSON(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.DstBoard == "" || body.DstColumn == "" {
		respondError(w, http.StatusBadRequest, "dst_board and dst_column are required")
		return
	}

	srcPath, err := s.ws.BoardPath(boardName)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dstPath, err := s.ws.BoardPath(body.DstBoard)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.ws.Engine.MoveCardToBoard(srcPath, -1, body.SrcColIdx, body.CardIdx, dstPath, body.DstColumn); err != nil {
		handleError(w, err)
		return
	}
	respondNoContent(w)
}
```

Note `-1` for version: the REST API is idempotent-friendly and doesn't carry a client version here, consistent with the existing `moveCard` REST handler (see `internal/api/cards.go:103`).

- [ ] **Step 4: Register route**

In `internal/api/server.go`, inside the `/api/boards/{board}/cards` route group (near line 312 `r.Post("/move", s.moveCard)`), add:

```go
r.Post("/move-to-board", s.moveCardToBoard)
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/api/ -run TestAPIMoveCardToBoard -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/api/cards.go internal/api/server.go internal/api/server_test.go
git commit -m "feat(api): add POST /api/boards/{board}/cards/move-to-board"
```

---

## Task 6: Card modal UI — cascading selects

**Files:**
- Modify: `internal/templates/layout.html`
- Modify: `web/js/liveboard.card-modal.js`

- [ ] **Step 1: Locate the modal's move control**

Search for the existing "move" / target-column control in `internal/templates/layout.html`:

Run: `grep -n "target_column\|move-card\|Move to" internal/templates/layout.html`

Expected: identifies the card-modal move section. Read ±30 lines around the match.

- [ ] **Step 2: Add the "Move to board" markup**

Immediately after the existing move-to-column control inside the card modal, add:

```html
<div class="card-modal-move-to-board" x-show="card">
  <label>Move to board</label>
  <select x-model="moveToBoardSlug" @change="moveToBoardColumn = ''">
    <option value="">— select board —</option>
    <template x-for="b in boardsLite.filter(b => b.slug !== currentBoardSlug)" :key="b.slug">
      <option :value="b.slug" x-text="b.name"></option>
    </template>
  </select>
  <select x-model="moveToBoardColumn" :disabled="!moveToBoardSlug">
    <option value="">— select column —</option>
    <template x-for="c in (boardsLite.find(b => b.slug === moveToBoardSlug)?.columns || [])" :key="c">
      <option :value="c" x-text="c"></option>
    </template>
  </select>
  <button type="button"
          :disabled="!moveToBoardSlug || !moveToBoardColumn"
          @click="submitMoveToBoard()">Move</button>
</div>
```

Match surrounding indentation and class-naming style of the file.

- [ ] **Step 3: Extend the Alpine component**

In `web/js/liveboard.card-modal.js`, add these fields to the component's returned object:

```js
boardsLite: [],
moveToBoardSlug: '',
moveToBoardColumn: '',
currentBoardSlug: (document.getElementById('board-version')?.dataset.slug) || '',
```

And add these methods to the same component:

```js
async loadBoardsLite() {
  if (this.boardsLite.length) return;
  try {
    const res = await fetch('/api/boards/list-lite');
    if (res.ok) this.boardsLite = await res.json();
  } catch (e) {
    console.error('boards-lite fetch failed', e);
  }
},

async submitMoveToBoard() {
  if (!this.moveToBoardSlug || !this.moveToBoardColumn) return;
  const form = new FormData();
  form.set('col_idx', this.card.colIdx);
  form.set('card_idx', this.card.cardIdx);
  form.set('dst_board', this.moveToBoardSlug);
  form.set('dst_column', this.moveToBoardColumn);
  form.set('version', document.getElementById('board-version')?.value || '0');
  const res = await fetch(`/board/${this.currentBoardSlug}/cards/move-to-board`, {
    method: 'POST',
    body: form,
    headers: { 'HX-Request': 'true' },
  });
  if (res.ok) {
    const html = await res.text();
    document.getElementById('board-content').innerHTML = html;
    if (window.htmx) window.htmx.process(document.getElementById('board-content'));
    this.close();
    this.moveToBoardSlug = '';
    this.moveToBoardColumn = '';
  }
},
```

Then call `this.loadBoardsLite()` from wherever the modal's existing `open(card)` / init method runs (match the existing open flow — read the file first to find it).

- [ ] **Step 4: Manual smoke test**

Run: `make dev`
Open a board, click a card, verify the "Move to board" selects populate, choose a destination, click Move.
Expected: the card disappears from the source board; opening the destination board shows the card at the top of the chosen column.

- [ ] **Step 5: Commit**

```bash
git add internal/templates/layout.html web/js/liveboard.card-modal.js
git commit -m "feat(ui): add Move to board cascading selects in card modal"
```

---

## Task 7: Quick-edit submenu

**Files:**
- Modify: `web/js/liveboard.drag.js`

- [ ] **Step 1: Locate the existing move-to-column entry**

Run: `grep -n "move-to\|target_column\|Move to" web/js/liveboard.drag.js`

Expected: identifies the quick-edit context menu's move-to-column handler. Read ±40 lines.

- [ ] **Step 2: Add "Move to board ▸" submenu**

Immediately after the existing move-to-column entry, add a submenu item. The existing code will show you the pattern for menu entries, submenus, and Alpine stores used in this file — match it. Concretely:

- Reuse the same Alpine store used by the card modal for `boardsLite` (initialize it at module scope the first time the submenu opens: `fetch('/api/boards/list-lite')`, cache the result on `window.__boardsLiteCache`).
- Submenu level 1: entries for each board where `board.slug !== currentSlug`.
- Submenu level 2 (on hover/click of a board entry): entries for each column of that board.
- On column click, call the same endpoint:

```js
async function moveCardToBoard(currentSlug, colIdx, cardIdx, dstSlug, dstColumn) {
  const form = new FormData();
  form.set('col_idx', colIdx);
  form.set('card_idx', cardIdx);
  form.set('dst_board', dstSlug);
  form.set('dst_column', dstColumn);
  form.set('version', document.getElementById('board-version')?.value || '0');
  const res = await fetch(`/board/${currentSlug}/cards/move-to-board`, {
    method: 'POST',
    body: form,
    headers: { 'HX-Request': 'true' },
  });
  if (res.ok) {
    const html = await res.text();
    document.getElementById('board-content').innerHTML = html;
    if (window.htmx) window.htmx.process(document.getElementById('board-content'));
  }
}
```

Add this function at module scope near the top of `liveboard.drag.js` and call it from the submenu click handler.

- [ ] **Step 3: Manual smoke test**

Run: `make dev`
Right-click a card → "Move to board ▸" → pick a board → pick a column.
Expected: the card disappears from source; destination board (opened in another tab) updates via SSE.

- [ ] **Step 4: Commit**

```bash
git add web/js/liveboard.drag.js
git commit -m "feat(ui): add Move to board submenu in quick-edit"
```

---

## Task 8: Lint and full test run

- [ ] **Step 1: Run linter**

Run: `make lint`
Expected: clean.

- [ ] **Step 2: Run full test suite**

Run: `go test ./...`
Expected: all pass.

- [ ] **Step 3: Commit any lint fixes**

If `make lint` modified files:

```bash
git add -A
git commit -m "chore: lint fixes"
```

---

## Out of Scope

- Command palette entry for cross-board move.
- Bulk move (multiple cards).
- Undo.
- Cross-workspace moves.
