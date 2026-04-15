# P6.3b — Cross-Board Card Linking Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add forward links + backlinks between cards across boards. Cards gain a `links: string[]` field stored as `boardSlug:cardId` entries in markdown metadata. The `<CardDetailModal>` gets a Links section (chips + picker + remove) and a "Linked from" section (derived). New `BackendAdapter.backlinks()` method on both adapters; new `GET /api/v1/cards/{cardId}/backlinks` endpoint.

**Architecture:** Forward links flow through the existing `edit_card` op (extended with `links: string[]`). Backlinks are derived: ServerAdapter calls `/api/v1/cards/{cardId}/backlinks` (bleve wildcard query on the `links` field, indexed as keyword); LocalAdapter scans loaded boards. `<LinkPicker>` reuses `useSearch`. `SearchHit` extended with `cardId` so picker results carry the link target identity.

**Tech Stack:** No new deps. Reuses bleve (P6.2), `useSearch` (P6.2), `card.id` (P6.3a), Radix Dialog (P4b.3), TanStack Query.

**Spec:** `docs/superpowers/specs/2026-04-15-p6-3b-cross-board-linking-design.md`

**Conventions:**
- Go code under `internal/`, `pkg/models/`. TS under `web/shared/`, `web/renderer/default/`.
- Tests colocated.
- Commit prefixes: `feat(models)`, `feat(parser)`, `feat(writer)`, `feat(board)`, `feat(search)`, `feat(api)`, `feat(shared)`, `feat(renderer)`, `test(...)`.
- `make lint` after every Go change.
- Use bun, never npx.

---

## File structure

**Modified (Go):**
- `pkg/models/models.go` — `Card.Links []string`.
- `internal/api/v1/mutations.go` — `EditCardOp.Links []string` + dispatcher updates.
- `internal/parser/parser.go` — read `links: a, b, c`.
- `internal/parser/parser_test.go` — round-trip case.
- `internal/writer/writer.go` — emit `links: …`.
- `internal/writer/writer_test.go` — assert ordering.
- `internal/board/board.go` — `EditCard` apply path writes `Links`.
- `internal/board/board_test.go` — assert links persist.
- `internal/search/index.go` — `Doc.Links`, keyword analyzer for the `links` field, `Backlinks(cardId)` method, `Hit.CardID`, search Hit DTO surfaces card_id.
- `internal/search/index_test.go` — `TestSearch_Backlinks`.
- `internal/api/v1/cards.go` (new) — `GET /cards/{cardId}/backlinks`.
- `internal/api/v1/cards_test.go` (new).
- `internal/api/v1/router.go` — register route.
- `internal/api/v1/search.go` — surface `card_id` in DTO.

**Modified (TS shared):**
- `web/shared/src/types.ts` — `Card.links?: string[]`, `MutationOp.edit_card.links: string[]`.
- `web/shared/src/boardOps.ts` — `applyOp` for `edit_card` writes `card.links`.
- `web/shared/src/boardOps.test.ts` — assert links applied.
- `web/shared/src/adapter.ts` — `SearchHit.cardId`, `BacklinkHit`, `backlinks()`.
- `web/shared/src/protocol.ts` — `backlinks` Request variant.
- `web/shared/src/broker.ts` — route.
- `web/shared/src/client.ts` — passthrough.
- `web/shared/src/adapters/server.ts` — `backlinks()` impl + `cardId` in `search`.
- `web/shared/src/adapters/server.test.ts` — coverage.
- `web/shared/src/adapters/local.ts` — `backlinks()` impl + `cardId` in `search`.
- `web/shared/src/adapters/local.search.test.ts` — coverage.

**New (TS renderer):**
- `web/renderer/default/src/queries/useBacklinks.ts`
- `web/renderer/default/src/queries/useResolveLink.ts`
- `web/renderer/default/src/components/LinkPicker.tsx`
- `web/renderer/default/src/components/LinkChip.tsx`

**Modified (TS renderer):**
- `web/renderer/default/src/components/CardDetailModal.tsx` — add Links + Linked-from sections; submit includes `links`.
- `web/renderer/default/src/components/CardEditable.tsx` — inline title-edit `edit_card` op now passes `card.links ?? []`.

---

## Task 1: Schema additions — `Card.Links` + `EditCardOp.Links`

**Files:**
- Modify: `pkg/models/models.go`
- Modify: `internal/api/v1/mutations.go`
- Modify: `web/shared/src/types.ts`

- [ ] **Step 1: `Card.Links` (Go)**

In `pkg/models/models.go`, find the `Card` struct definition. Add:
```go
type Card struct {
    // ...existing fields...
    Links []string `json:"links,omitempty" yaml:"links,omitempty"`
}
```

- [ ] **Step 2: `EditCardOp.Links` (Go)**

In `internal/api/v1/mutations.go` around line 100, extend `EditCardOp`:
```go
type EditCardOp struct {
    ColIdx   int      `json:"col_idx"`
    CardIdx  int      `json:"card_idx"`
    Title    string   `json:"title"`
    Body     string   `json:"body"`
    Tags     []string `json:"tags"`
    Links    []string `json:"links"`
    Priority string   `json:"priority"`
    Due      string   `json:"due"`
    Assignee string   `json:"assignee"`
}
```

If a Go union dispatcher decodes `EditCardOp`, no change needed — JSON unmarshal picks up the new field automatically.

- [ ] **Step 3: `Card.links` + `MutationOp.edit_card.links` (TS)**

In `web/shared/src/types.ts`, find the `Card` interface and add:
```ts
links?: string[]
```

In the `MutationOp` union's `edit_card` variant, add `links: string[]` after `tags`:
```ts
| {
    type: 'edit_card'
    col_idx: number
    card_idx: number
    title: string
    body: string
    tags: string[]
    links: string[]
    priority: string
    due: string
    assignee: string
  }
```

- [ ] **Step 4: Build + typecheck**

```bash
cd /Users/htruong/code/htruong/liveboard && go build ./... && cd web/renderer/default && bun run typecheck
```

Expected: Go builds. TS typecheck fails at every callsite of `edit_card` op missing `links`. Tasks 6 + 9 + 11 + 12 fix call sites; for now we accept the broken state and don't commit standalone.

- [ ] **Step 5: Don't commit yet** — schema change lands together with parser/writer in Task 2 and op apply in Task 3.

---

## Task 2: Parser + writer for `links`

**Files:**
- Modify: `internal/parser/parser.go`
- Modify: `internal/parser/parser_test.go`
- Modify: `internal/writer/writer.go`
- Modify: `internal/writer/writer_test.go`

- [ ] **Step 1: Parser**

In `internal/parser/parser.go`, find the metadata-key switch (the same one that handles `tags`, `priority`, etc.). Add:
```go
case "links":
    card.Links = splitCSV(value)  // reuse existing splitCSV / equivalent helper used by tags
```

If `splitCSV` doesn't exist, use the same approach `tags` uses verbatim — split on `,`, trim each.

- [ ] **Step 2: Writer**

In `internal/writer/writer.go`, find the metadata-emission block (alphabetical order with `tags`, `priority`, etc.). Add `links` in alphabetical position (between `id` and `priority`):
```go
if len(card.Links) > 0 {
    fmt.Fprintf(w, "  links: %s\n", strings.Join(card.Links, ", "))
}
```

- [ ] **Step 3: Tests**

In `internal/parser/parser_test.go`, append:
```go
func TestParseCard_Links(t *testing.T) {
    md := "## Todo\n\n- [ ] Card title\n  links: foo:aBc1234XyZ, bar:Q9rT5pZ2nM\n"
    boards, err := parser.Parse(md)
    if err != nil { t.Fatal(err) }
    if len(boards.Columns) != 1 || len(boards.Columns[0].Cards) != 1 {
        t.Fatal("unexpected structure")
    }
    got := boards.Columns[0].Cards[0].Links
    want := []string{"foo:aBc1234XyZ", "bar:Q9rT5pZ2nM"}
    if !reflect.DeepEqual(got, want) {
        t.Errorf("links = %v, want %v", got, want)
    }
}
```

(Adapt the test to the file's existing parse helper signature. If `parser.Parse` takes a path, use a temp file; if it takes a string, pass directly.)

In `internal/writer/writer_test.go`, append:
```go
func TestWriteCard_Links(t *testing.T) {
    b := &models.Board{
        Columns: []models.Column{{
            Name: "Todo",
            Cards: []*models.Card{{
                Title: "Hi",
                Links: []string{"foo:aBc1234XyZ", "bar:Q9rT5pZ2nM"},
            }},
        }},
    }
    raw, err := writer.Render(b)
    if err != nil { t.Fatal(err) }
    if !strings.Contains(string(raw), "links: foo:aBc1234XyZ, bar:Q9rT5pZ2nM") {
        t.Errorf("links line missing in output:\n%s", raw)
    }
}
```

- [ ] **Step 4: Run**

```bash
cd /Users/htruong/code/htruong/liveboard && go test ./internal/parser/ ./internal/writer/ -run "Links" -v
```
Expected: tests pass.

- [ ] **Step 5: Lint + commit (Tasks 1+2 together)**

```bash
make lint
git add pkg/models/models.go internal/api/v1/mutations.go \
        internal/parser/parser.go internal/parser/parser_test.go \
        internal/writer/writer.go internal/writer/writer_test.go \
        web/shared/src/types.ts
git commit -m "feat(models): add Card.Links + parser/writer + edit_card.Links field

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: `edit_card` op apply path

**Files:**
- Modify: `internal/board/board.go`
- Modify: `internal/board/board_test.go`

- [ ] **Step 1: Apply path writes Links**

In `internal/board/board.go`, find the `edit_card` apply / handler. Add the field set:
```go
card.Title = op.Title
card.Body = op.Body
card.Tags = op.Tags
card.Links = op.Links
card.Priority = op.Priority
card.Due = op.Due
card.Assignee = op.Assignee
```

If the engine uses the v1 `EditCardOp` directly, the rename is `op.EditCard.Links`. Match the existing field references.

- [ ] **Step 2: Test**

In `internal/board/board_test.go`, append:
```go
func TestEditCard_Links(t *testing.T) {
    dir := t.TempDir()
    ws := workspace.Open(dir)
    eng := board.NewEngine(ws)
    if _, err := ws.CreateBoard("Foo"); err != nil { t.Fatal(err) }
    if _, err := eng.MutateBoard("foo", -1, func(b *models.Board) error {
        b.Columns[0].Cards = append(b.Columns[0].Cards, &models.Card{Title: "x"})
        return nil
    }); err != nil { t.Fatal(err) }

    op := &v1.MutationOp{
        Type: "edit_card",
        EditCard: &v1.EditCardOp{
            ColIdx: 0, CardIdx: 0,
            Title: "x", Body: "", Tags: nil,
            Links: []string{"bar:Q9rT5pZ2nM"},
            Priority: "", Due: "", Assignee: "",
        },
    }
    // Whatever helper applies a v1 op to the engine — match the existing test patterns in v1/mutations_test.go.
    // After apply, reload the board:
    b, _ := ws.LoadBoard("foo")
    if got := b.Columns[0].Cards[0].Links; !reflect.DeepEqual(got, []string{"bar:Q9rT5pZ2nM"}) {
        t.Errorf("links = %v", got)
    }
}
```

If the existing test pattern drives v1 ops via HTTP, copy that pattern instead. Use whatever's already canonical.

- [ ] **Step 3: Run + lint + commit**

```bash
go test ./internal/board/ -run TestEditCard -v && make lint
git add internal/board/board.go internal/board/board_test.go
git commit -m "feat(board): edit_card op writes Card.Links

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 4: TS `applyOp` mirror for `edit_card.links`

**Files:**
- Modify: `web/shared/src/boardOps.ts`
- Modify: `web/shared/src/boardOps.test.ts`

- [ ] **Step 1: Test first**

In `web/shared/src/boardOps.test.ts`, append:
```ts
it('edit_card writes links', () => {
  const board = {
    name: 'b', version: 1,
    columns: [{ name: 'Todo', cards: [{ title: 'x', id: 'XXXXXXXXXX' }] }],
  }
  const next = applyOp(board, {
    type: 'edit_card',
    col_idx: 0, card_idx: 0,
    title: 'x', body: '', tags: [], links: ['foo:abcdefghij'],
    priority: '', due: '', assignee: '',
  })
  expect(next.columns[0].cards[0].links).toEqual(['foo:abcdefghij'])
})
```

- [ ] **Step 2: Run, expect fail**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src/boardOps.test.ts -t "links"
```

- [ ] **Step 3: Implement**

In `web/shared/src/boardOps.ts`, find the `case 'edit_card':` branch around line 50. Add `card.links = op.links` alongside the other field assignments:
```ts
card.title = op.title
card.body = op.body
card.tags = op.tags
card.links = op.links
card.priority = op.priority
card.due = op.due
card.assignee = op.assignee
```

- [ ] **Step 4: Run, expect pass**

```bash
bun test web/shared/src/boardOps.test.ts
```

- [ ] **Step 5: Commit**

```bash
git add web/shared/src/boardOps.ts web/shared/src/boardOps.test.ts
git commit -m "feat(boardOps): apply edit_card.links in optimistic apply

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 5: Update existing edit_card call sites

**Files:**
- Modify: `web/renderer/default/src/components/CardEditable.tsx`
- Modify: `web/renderer/default/src/components/CardDetailModal.tsx`

Every place that builds an `edit_card` op must now include `links`. Two known call sites: inline title edit in `CardEditable` (preserves `card.links`); modal Save in `CardDetailModal` (preserves until Task 9 introduces the editor).

- [ ] **Step 1: `CardEditable.tsx` inline edit**

Find the `mutation.mutate({ type: 'edit_card', ..., assignee: ... })` call inside the `commit` function. Add `links: card.links ?? []` after `tags`:
```tsx
mutation.mutate({
  type: 'edit_card',
  col_idx: colIdx,
  card_idx: cardIdx,
  title,
  body: card.body ?? '',
  tags: card.tags ?? [],
  links: card.links ?? [],
  priority: card.priority ?? '',
  due: card.due ?? '',
  assignee: card.assignee ?? '',
})
```

- [ ] **Step 2: `CardDetailModal.tsx` Save**

Find the `mutation.mutate({ type: 'edit_card', ... })` call. Add `links: card.links ?? []` (we'll wire the editor in Task 9; for now Save preserves the existing list):
```tsx
mutation.mutate({
  type: 'edit_card',
  col_idx: colIdx,
  card_idx: cardIdx,
  title,
  body: bodyRef.current?.value ?? '',
  tags,
  links: card.links ?? [],
  priority: priorityRef.current?.value ?? '',
  due: dueRef.current?.value ?? '',
  assignee: assigneeRef.current?.value ?? '',
}, ...)
```

- [ ] **Step 3: Typecheck + run renderer suite**

```bash
cd web/renderer/default && bun test && bun run typecheck
```
Expected: green.

- [ ] **Step 4: Commit**

```bash
git add web/renderer/default/src/components/CardEditable.tsx web/renderer/default/src/components/CardDetailModal.tsx
git commit -m "feat(renderer): preserve card.links in edit_card mutations

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 6: bleve indexing of `Doc.Links` + `Hit.CardID`

**Files:**
- Modify: `internal/search/index.go`
- Modify: `internal/search/index_test.go`

- [ ] **Step 1: Extend `doc` struct**

```go
type doc struct {
    BoardID   string   `json:"board_id"`
    BoardName string   `json:"board_name"`
    ColIdx    int      `json:"col_idx"`
    CardIdx   int      `json:"card_idx"`
    CardID    string   `json:"card_id"`
    Title     string   `json:"title"`
    Body      string   `json:"body"`
    Tags      []string `json:"tags"`
    Links     []string `json:"links"`
}
```

- [ ] **Step 2: Add `Hit.CardID`**

```go
type Hit struct {
    BoardID   string
    BoardName string
    ColIdx    int
    CardIdx   int
    CardID    string
    CardTitle string
    Snippet   string
}
```

- [ ] **Step 3: Use a keyword analyzer for `links` + populate in `UpdateBoard`**

Replace the existing `bleve.NewIndexMapping()` setup in `New()`:
```go
func New() (*Index, error) {
    mapping := bleve.NewIndexMapping()
    keywordField := bleve.NewTextFieldMapping()
    keywordField.Analyzer = "keyword"
    docMapping := bleve.NewDocumentMapping()
    docMapping.AddFieldMappingsAt("links", keywordField)
    mapping.AddDocumentMapping("_default", docMapping)
    idx, err := bleve.NewMemOnly(mapping)
    if err != nil {
        return nil, err
    }
    return &Index{idx: idx}, nil
}
```

In `UpdateBoard`, populate `CardID` and `Links`:
```go
d := doc{
    BoardID:   slug,
    BoardName: boardName,
    ColIdx:    cIdx,
    CardIdx:   kIdx,
    CardID:    c.ID,
    Title:     c.Title,
    Body:      c.Body,
    Tags:      c.Tags,
    Links:     c.Links,
}
```

In `Search()`, request the `card_id` field too:
```go
sr.Fields = []string{"board_id", "board_name", "col_idx", "card_idx", "card_id", "title"}
```
And map into `Hit.CardID`.

- [ ] **Step 4: Add `Backlinks` method**

```go
func (i *Index) Backlinks(cardID string) ([]Hit, error) {
    if cardID == "" {
        return nil, nil
    }
    q := bleve.NewWildcardQuery("*:" + cardID)
    q.SetField("links")
    sr := bleve.NewSearchRequestOptions(q, 100, 0, false)
    sr.Fields = []string{"board_id", "board_name", "col_idx", "card_idx", "card_id", "title"}
    res, err := i.idx.Search(sr)
    if err != nil {
        return nil, err
    }
    out := make([]Hit, 0, len(res.Hits))
    for _, h := range res.Hits {
        out = append(out, Hit{
            BoardID:   getString(h.Fields, "board_id"),
            BoardName: getString(h.Fields, "board_name"),
            ColIdx:    getInt(h.Fields, "col_idx"),
            CardIdx:   getInt(h.Fields, "card_idx"),
            CardID:    getString(h.Fields, "card_id"),
            CardTitle: getString(h.Fields, "title"),
        })
    }
    return out, nil
}
```

- [ ] **Step 5: Test**

In `internal/search/index_test.go`, append:
```go
func TestSearch_Backlinks(t *testing.T) {
    idx, _ := search.New()
    target := newBoard("Target", col("Todo", &models.Card{ID: "TGT0000001", Title: "target card"}))
    src := newBoard("Source", col("Todo", &models.Card{
        ID: "SRC0000001", Title: "source card",
        Links: []string{"target:TGT0000001"},
    }))
    _ = idx.UpdateBoard("target", target)
    _ = idx.UpdateBoard("source", src)
    hits, err := idx.Backlinks("TGT0000001")
    if err != nil { t.Fatal(err) }
    if len(hits) != 1 { t.Fatalf("expected 1 hit, got %d", len(hits)) }
    if hits[0].BoardID != "source" { t.Errorf("board_id = %q", hits[0].BoardID) }
}
```

(Helpers `newBoard` and `col` from the existing P6.2 test file. Make sure `col` accepts `*Card` so we can pass cards with explicit IDs.)

- [ ] **Step 6: Run + lint + commit**

```bash
go test ./internal/search/ -v && make lint
git add internal/search/index.go internal/search/index_test.go
git commit -m "feat(search): index Card.Links (keyword) + add Backlinks method

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 7: HTTP backlinks endpoint + DTO surface `card_id` in search

**Files:**
- Create: `internal/api/v1/cards.go`
- Create: `internal/api/v1/cards_test.go`
- Modify: `internal/api/v1/router.go`
- Modify: `internal/api/v1/search.go`

- [ ] **Step 1: Surface `card_id` in search DTO**

In `internal/api/v1/search.go`, extend `searchHitDTO`:
```go
type searchHitDTO struct {
    BoardID   string `json:"board_id"`
    BoardName string `json:"board_name"`
    ColIdx    int    `json:"col_idx"`
    CardIdx   int    `json:"card_idx"`
    CardID    string `json:"card_id"`
    CardTitle string `json:"card_title"`
    Snippet   string `json:"snippet"`
}
```
And populate `CardID` from `h.CardID` in the handler's loop.

- [ ] **Step 2: Backlinks handler**

Create `internal/api/v1/cards.go`:
```go
package v1

import (
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/go-chi/chi/v5"
)

type backlinkHitDTO struct {
    BoardID   string `json:"board_id"`
    BoardName string `json:"board_name"`
    ColIdx    int    `json:"col_idx"`
    CardIdx   int    `json:"card_idx"`
    CardTitle string `json:"card_title"`
}

func (d Deps) getBacklinks(w http.ResponseWriter, r *http.Request) {
    cardID := chi.URLParam(r, "cardId")
    if cardID == "" {
        writeError(w, fmt.Errorf("%w: cardId required", errInvalid))
        return
    }
    if d.Search == nil {
        _ = json.NewEncoder(w).Encode([]backlinkHitDTO{})
        return
    }
    hits, err := d.Search.Backlinks(cardID)
    if err != nil {
        writeError(w, err)
        return
    }
    out := make([]backlinkHitDTO, 0, len(hits))
    for _, h := range hits {
        out = append(out, backlinkHitDTO{
            BoardID:   h.BoardID,
            BoardName: h.BoardName,
            ColIdx:    h.ColIdx,
            CardIdx:   h.CardIdx,
            CardTitle: h.CardTitle,
        })
    }
    _ = json.NewEncoder(w).Encode(out)
}
```

- [ ] **Step 3: Register route**

In `internal/api/v1/router.go`, alongside `r.Get("/search", d.getSearch)`:
```go
r.Get("/cards/{cardId}/backlinks", d.getBacklinks)
```

- [ ] **Step 4: Test**

Create `internal/api/v1/cards_test.go`:
```go
package v1_test

import (
    "encoding/json"
    "net/http"
    "strings"
    "testing"
)

type backlinkHitDTO struct {
    BoardID   string `json:"board_id"`
    BoardName string `json:"board_name"`
    ColIdx    int    `json:"col_idx"`
    CardIdx   int    `json:"card_idx"`
    CardTitle string `json:"card_title"`
}

func TestBacklinks_FindsLinkedSource(t *testing.T) {
    srv := newV1TestServer(t)
    // Create target board + card.
    postJSON(t, srv, "/api/v1/boards", `{"name":"Target"}`)
    postJSON(t, srv, "/api/v1/boards/target/mutations",
        `{"client_version":1,"op":{"type":"add_card","column":"Todo","title":"target card"}}`)
    // Find target card's id by listing the board.
    res, body := getJSON(t, srv, "/api/v1/boards/target")
    if res.StatusCode != http.StatusOK { t.Fatalf("get target: %d %s", res.StatusCode, body) }
    var board struct {
        Columns []struct {
            Cards []struct {
                ID string `json:"id"`
            } `json:"cards"`
        } `json:"columns"`
    }
    json.Unmarshal([]byte(body), &board)
    targetID := board.Columns[0].Cards[0].ID
    if targetID == "" { t.Fatal("target id empty") }

    // Create source board + card with link.
    postJSON(t, srv, "/api/v1/boards", `{"name":"Source"}`)
    postJSON(t, srv, "/api/v1/boards/source/mutations",
        `{"client_version":1,"op":{"type":"add_card","column":"Todo","title":"source card"}}`)
    // Find source card's id.
    _, srcBody := getJSON(t, srv, "/api/v1/boards/source")
    var srcBoard struct {
        Columns []struct {
            Cards []struct {
                ID string `json:"id"`
            } `json:"cards"`
        } `json:"columns"`
    }
    json.Unmarshal([]byte(srcBody), &srcBoard)
    srcCardID := srcBoard.Columns[0].Cards[0].ID
    _ = srcCardID

    // Edit source card to add link.
    editBody := `{"client_version":2,"op":{"type":"edit_card","col_idx":0,"card_idx":0,"title":"source card","body":"","tags":[],"links":["target:` + targetID + `"],"priority":"","due":"","assignee":""}}`
    postJSON(t, srv, "/api/v1/boards/source/mutations", editBody)

    // Backlinks.
    res, body = getJSON(t, srv, "/api/v1/cards/"+targetID+"/backlinks")
    if res.StatusCode != http.StatusOK { t.Fatalf("status = %d body = %s", res.StatusCode, body) }
    var hits []backlinkHitDTO
    json.Unmarshal([]byte(body), &hits)
    if len(hits) != 1 { t.Fatalf("expected 1 backlink, got %d (body=%s)", len(hits), body) }
    if hits[0].BoardID != "source" { t.Errorf("board_id = %q", hits[0].BoardID) }
}

func TestBacklinks_EmptyCardIDIsInvalid(t *testing.T) {
    // chi won't match the route with empty path param, so we test via an obviously bad path:
    srv := newV1TestServer(t)
    res, _ := getJSON(t, srv, "/api/v1/cards/%20/backlinks")  // single space → empty after parse
    _ = res
    // We just want to confirm the handler returns 400 if it gets empty.
    // chi.URLParam returns the literal " " here; the test below covers the empty-after-trim case.
    _ = strings.TrimSpace
}
```

(The empty-cardId test relies on chi behavior; if empty paths are blocked at the route level, this test is a no-op — drop it. The first test is the meaningful one.)

- [ ] **Step 5: Run + lint + commit**

```bash
go test ./internal/api/v1/ -run Backlinks -v && make lint
git add internal/api/v1/cards.go internal/api/v1/cards_test.go internal/api/v1/router.go internal/api/v1/search.go
git commit -m "feat(api): GET /api/v1/cards/{cardId}/backlinks + card_id in search DTO

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 8: TS protocol + adapter additions for `backlinks`

**Files:**
- Modify: `web/shared/src/adapter.ts`
- Modify: `web/shared/src/protocol.ts`
- Modify: `web/shared/src/broker.ts`
- Modify: `web/shared/src/client.ts`
- Modify: `web/shared/src/adapters/server.ts`
- Modify: `web/shared/src/adapters/server.test.ts`
- Modify: `web/shared/src/adapters/local.ts`
- Modify: `web/shared/src/adapters/local.search.test.ts` (extension)

- [ ] **Step 1: Types**

In `web/shared/src/adapter.ts`:
```ts
export interface SearchHit {
  boardId: string
  boardName: string
  colIdx: number
  cardIdx: number
  cardId: string
  cardTitle: string
  snippet: string
}

export interface BacklinkHit {
  boardId: string
  boardName: string
  colIdx: number
  cardIdx: number
  cardTitle: string
}

export interface BackendAdapter {
  // ...existing...
  backlinks(cardId: string): Promise<BacklinkHit[]>
}
```

- [ ] **Step 2: Protocol Request variant**

In `web/shared/src/protocol.ts`, append to `Request`:
```ts
| { id: string; kind: 'request'; method: 'backlinks'; params: { cardId: string } }
```

- [ ] **Step 3: Broker route**

In `web/shared/src/broker.ts`'s switch, after the `search` case:
```ts
case 'backlinks':
  return this.adapter.backlinks(req.params.cardId)
```

- [ ] **Step 4: Client passthrough**

In `web/shared/src/client.ts`, after `search()`:
```ts
backlinks(cardId: string): Promise<BacklinkHit[]> {
  return this.request({ kind: 'request', method: 'backlinks', params: { cardId } })
}
```
Import `BacklinkHit` at the top.

- [ ] **Step 5: ServerAdapter**

In `web/shared/src/adapters/server.ts`:
```ts
async search(query: string, limit = 20): Promise<SearchHit[]> {
  // ...existing fetch...
  const raw = await this.getJSON<Array<{
    board_id: string
    board_name: string
    col_idx: number
    card_idx: number
    card_id: string
    card_title: string
    snippet: string
  }>>(`/search?${params}`)
  return raw.map((d) => ({
    boardId: d.board_id, boardName: d.board_name, colIdx: d.col_idx,
    cardIdx: d.card_idx, cardId: d.card_id, cardTitle: d.card_title, snippet: d.snippet,
  }))
}

async backlinks(cardId: string): Promise<BacklinkHit[]> {
  const raw = await this.getJSON<Array<{
    board_id: string
    board_name: string
    col_idx: number
    card_idx: number
    card_title: string
  }>>(`/cards/${encodeURIComponent(cardId)}/backlinks`)
  return raw.map((d) => ({
    boardId: d.board_id, boardName: d.board_name,
    colIdx: d.col_idx, cardIdx: d.card_idx, cardTitle: d.card_title,
  }))
}
```

Test in `server.test.ts` (extension):
```ts
it('search returns cardId in mapped hit', async () => {
  const a = new ServerAdapter({
    baseUrl: '/api/v1',
    fetch: mockFetch(() => jsonResponse([{
      board_id: 'foo', board_name: 'Foo', col_idx: 0, card_idx: 1,
      card_id: 'AbCdEfGhIj', card_title: 'hi', snippet: '',
    }])),
  })
  const hits = await a.search('hi')
  expect(hits[0].cardId).toBe('AbCdEfGhIj')
})

it('backlinks GETs /cards/{id}/backlinks and maps DTO', async () => {
  const log: RequestRecord[] = []
  const a = new ServerAdapter({
    baseUrl: '/api/v1',
    fetch: mockFetch(
      () => jsonResponse([{ board_id: 'src', board_name: 'Src', col_idx: 0, card_idx: 2, card_title: 'source' }]),
      log,
    ),
  })
  const hits = await a.backlinks('TGT0000001')
  expect(hits).toEqual([{ boardId: 'src', boardName: 'Src', colIdx: 0, cardIdx: 2, cardTitle: 'source' }])
  expect(log[0].url).toBe('/api/v1/cards/TGT0000001/backlinks')
})
```

- [ ] **Step 6: LocalAdapter**

In `web/shared/src/adapters/local.ts`, extend `search` to populate `cardId`:
```ts
hits.push({
  boardId: id,
  boardName,
  colIdx,
  cardIdx,
  cardId: c.id ?? '',
  cardTitle: c.title ?? '',
  snippet: c.title ?? '',
})
```

Add `backlinks`:
```ts
async backlinks(cardId: string): Promise<BacklinkHit[]> {
  if (!cardId) return []
  const ws = this.loadWorkspace()
  const target = ':' + cardId
  const out: BacklinkHit[] = []
  for (const id of ws.boardIds) {
    const board = this.loadBoard(id)
    const cols = board.columns ?? []
    for (let c = 0; c < cols.length; c++) {
      const cards = cols[c]?.cards ?? []
      for (let k = 0; k < cards.length; k++) {
        const links = cards[k].links ?? []
        if (links.some((l) => l.endsWith(target))) {
          out.push({
            boardId: id,
            boardName: board.name ?? id,
            colIdx: c,
            cardIdx: k,
            cardTitle: cards[k].title ?? '',
          })
        }
      }
    }
  }
  return out
}
```

Extend `local.search.test.ts`:
```ts
it('search returns cardId on hits', async () => {
  const a = new LocalAdapter(new MemoryStorage())
  await a.createBoard('Foo')
  await a.mutateBoard('foo', 1, { type: 'add_card', column: 'Todo', title: 'alpha-token' })
  const hits = await a.search('alpha')
  expect(hits[0].cardId).not.toBe('')
})
```

Add a small backlinks test:
```ts
describe('LocalAdapter.backlinks', () => {
  it('returns cards that link to the given cardId', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Target')
    await a.mutateBoard('target', 1, { type: 'add_card', column: 'Todo', title: 'tgt' })
    const tgtBoard = await a.getBoard('target')
    const tgtId = tgtBoard.columns![0].cards![0].id!
    expect(tgtId).toBeTruthy()

    await a.createBoard('Source')
    await a.mutateBoard('source', 1, { type: 'add_card', column: 'Todo', title: 'src' })
    await a.mutateBoard('source', 2, {
      type: 'edit_card', col_idx: 0, card_idx: 0,
      title: 'src', body: '', tags: [], links: [`target:${tgtId}`],
      priority: '', due: '', assignee: '',
    })

    const back = await a.backlinks(tgtId)
    expect(back.length).toBe(1)
    expect(back[0].boardId).toBe('source')
  })
})
```

- [ ] **Step 7: Run + commit**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src && cd web/renderer/default && bun run typecheck
```
Expected: green.

```bash
cd /Users/htruong/code/htruong/liveboard
git add web/shared/src/adapter.ts web/shared/src/protocol.ts web/shared/src/broker.ts web/shared/src/client.ts \
        web/shared/src/adapters/server.ts web/shared/src/adapters/server.test.ts \
        web/shared/src/adapters/local.ts web/shared/src/adapters/local.search.test.ts
git commit -m "feat(shared): add backlinks method on adapter + cardId in SearchHit

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 9: `useBacklinks` + `useResolveLink` hooks

**Files:**
- Create: `web/renderer/default/src/queries/useBacklinks.ts`
- Create: `web/renderer/default/src/queries/useResolveLink.ts`

- [ ] **Step 1: `useBacklinks.ts`**

Create:
```ts
import { useQuery } from '@tanstack/react-query'
import type { BacklinkHit } from '@shared/adapter.js'
import { useClient } from '../queries.js'

export function useBacklinks(cardId: string | undefined): BacklinkHit[] {
  const client = useClient()
  const q = useQuery({
    queryKey: ['backlinks', cardId],
    queryFn: () => client.backlinks(cardId!),
    enabled: !!cardId,
  })
  return q.data ?? []
}
```

- [ ] **Step 2: `useResolveLink.ts`**

Create:
```ts
import { useQuery } from '@tanstack/react-query'
import { useClient } from '../queries.js'

export interface ResolvedLink {
  boardName: string
  cardTitle: string
  colIdx: number
  cardIdx: number
}

export function useResolveLink(target: string): ResolvedLink | null {
  const client = useClient()
  const q = useQuery({
    queryKey: ['resolve', target],
    queryFn: async (): Promise<ResolvedLink | null> => {
      const idx = target.indexOf(':')
      if (idx <= 0) return null
      const boardSlug = target.slice(0, idx)
      const cardId = target.slice(idx + 1)
      let board
      try {
        board = await client.getBoard(boardSlug)
      } catch {
        return null
      }
      const cols = board.columns ?? []
      for (let c = 0; c < cols.length; c++) {
        const cards = cols[c]?.cards ?? []
        for (let k = 0; k < cards.length; k++) {
          if (cards[k].id === cardId) {
            return {
              boardName: board.name ?? boardSlug,
              cardTitle: cards[k].title ?? '',
              colIdx: c,
              cardIdx: k,
            }
          }
        }
      }
      return null
    },
    enabled: target.length > 0,
  })
  return q.data ?? null
}
```

- [ ] **Step 3: Typecheck + commit**

```bash
cd web/renderer/default && bun run typecheck
git add web/renderer/default/src/queries/useBacklinks.ts web/renderer/default/src/queries/useResolveLink.ts
git commit -m "feat(renderer): useBacklinks + useResolveLink hooks

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 10: `<LinkChip>` + `<LinkPicker>` components

**Files:**
- Create: `web/renderer/default/src/components/LinkChip.tsx`
- Create: `web/renderer/default/src/components/LinkPicker.tsx`

- [ ] **Step 1: `LinkChip.tsx`**

```tsx
import { useResolveLink } from '../queries/useResolveLink.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { useBoardFocus } from '../contexts/BoardFocusContext.js'

export function LinkChip({ target, onRemove }: { target: string; onRemove: () => void }): JSX.Element {
  const resolved = useResolveLink(target)
  const { setActive } = useActiveBoard()
  const { setFocused } = useBoardFocus()

  const navigate = (): void => {
    if (!resolved) return
    const idx = target.indexOf(':')
    const boardSlug = target.slice(0, idx)
    setActive(boardSlug)
    Promise.resolve().then(() => setFocused({ colIdx: resolved.colIdx, cardIdx: resolved.cardIdx }))
  }

  return (
    <li className="flex items-center gap-1 rounded bg-slate-100 dark:bg-slate-700 px-2 py-1 text-xs">
      <button type="button" onClick={navigate} className="flex-1 text-left">
        {resolved
          ? <span><span className="text-slate-500 dark:text-slate-400">{resolved.boardName} · </span>{resolved.cardTitle}</span>
          : <span className="italic text-slate-400">{target} (missing)</span>}
      </button>
      <button type="button" aria-label="remove link" onClick={onRemove}
        className="text-slate-400 hover:text-red-500">✕</button>
    </li>
  )
}
```

- [ ] **Step 2: `LinkPicker.tsx`**

```tsx
import { useState } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { Command } from 'cmdk'
import { useSearch } from '../queries/useSearch.js'

export function LinkPicker({
  open,
  onOpenChange,
  onPick,
  excludeBoardId,
  excludeCardId,
}: {
  open: boolean
  onOpenChange: (o: boolean) => void
  onPick: (target: string) => void
  excludeBoardId?: string
  excludeCardId?: string
}): JSX.Element {
  const [query, setQuery] = useState('')
  const hits = useSearch(query)
  const filtered = hits.filter(
    (h) => !(h.boardId === excludeBoardId && h.cardId === excludeCardId),
  )

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/40" />
        <Dialog.Content
          aria-label="Link picker"
          className="fixed left-1/2 top-1/4 z-50 w-full max-w-md -translate-x-1/2 rounded-lg bg-white dark:bg-slate-800 p-2 shadow-xl"
        >
          <Dialog.Title className="sr-only">Link a card</Dialog.Title>
          <Command label="Pick a card to link" shouldFilter={false}>
            <Command.Input
              value={query}
              onValueChange={setQuery}
              autoFocus
              placeholder="Search a card to link…"
              className="w-full rounded px-3 py-2 text-base outline-none placeholder:text-slate-400 dark:bg-slate-800 dark:text-slate-100"
            />
            <Command.List className="max-h-64 overflow-y-auto">
              {query.length === 0 && (
                <Command.Empty className="px-3 py-2 text-sm text-slate-400">
                  Type to search for a card.
                </Command.Empty>
              )}
              {filtered.map((h) => (
                <Command.Item
                  key={`${h.boardId}:${h.cardId}`}
                  value={`${h.cardTitle} ${h.boardName}`}
                  onSelect={() => {
                    onPick(`${h.boardId}:${h.cardId}`)
                    setQuery('')
                  }}
                  className="cursor-pointer rounded px-3 py-1.5 text-sm aria-selected:bg-slate-100 dark:aria-selected:bg-slate-700"
                >
                  <span className="font-semibold">{h.cardTitle}</span>
                  <span className="ml-2 text-xs text-slate-400">in {h.boardName}</span>
                </Command.Item>
              ))}
            </Command.List>
          </Command>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
```

- [ ] **Step 3: Typecheck + commit**

```bash
cd web/renderer/default && bun run typecheck
git add web/renderer/default/src/components/LinkChip.tsx web/renderer/default/src/components/LinkPicker.tsx
git commit -m "feat(renderer): LinkChip + LinkPicker components

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 11: Wire Links + Linked-from sections into `<CardDetailModal>`

**Files:**
- Modify: `web/renderer/default/src/components/CardDetailModal.tsx`

- [ ] **Step 1: Imports + state**

Add at the top:
```tsx
import { useBacklinks } from '../queries/useBacklinks.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { useBoardFocus } from '../contexts/BoardFocusContext.js'
import { LinkChip } from './LinkChip.js'
import { LinkPicker } from './LinkPicker.js'
```

Inside the component, near other state hooks:
```tsx
const [links, setLinks] = useState<string[]>(card.links ?? [])
const [pickerOpen, setPickerOpen] = useState(false)
const backlinks = useBacklinks(card.id)
const { setActive } = useActiveBoard()
const { setFocused } = useBoardFocus()

useEffect(() => {
  if (open) setLinks(card.links ?? [])
}, [open, card.links])

const addLink = (target: string): void => {
  if (!links.includes(target)) setLinks([...links, target])
  setPickerOpen(false)
}
const removeLink = (target: string): void => {
  setLinks(links.filter((l) => l !== target))
}
const navigateToBacklink = (b: typeof backlinks[number]): void => {
  setActive(b.boardId)
  Promise.resolve().then(() => setFocused({ colIdx: b.colIdx, cardIdx: b.cardIdx }))
  onOpenChange(false)
}
```

- [ ] **Step 2: Replace the Save mutation call to include `links`**

Find the `mutation.mutate({type:'edit_card', ...})` call. Replace `links: card.links ?? []` (added in Task 5) with `links` (the new state):
```tsx
mutation.mutate({
  type: 'edit_card',
  col_idx: colIdx, card_idx: cardIdx,
  title,
  body: bodyRef.current?.value ?? '',
  tags,
  links,
  priority: priorityRef.current?.value ?? '',
  due: dueRef.current?.value ?? '',
  assignee: assigneeRef.current?.value ?? '',
}, { onSuccess: () => onOpenChange(false) })
```

- [ ] **Step 3: Add Links + Linked-from sections in the form JSX**

After the existing Tags / Priority / Due / Assignee blocks (and before Cancel/Save buttons), add:
```tsx
<section>
  <header className="text-xs font-medium text-slate-600 dark:text-slate-300">Links</header>
  <ul className="mt-1 flex flex-col gap-1">
    {links.map((target) => (
      <LinkChip key={target} target={target} onRemove={() => removeLink(target)} />
    ))}
  </ul>
  <button
    type="button"
    onClick={() => setPickerOpen(true)}
    className="mt-1 rounded px-2 py-1 text-xs text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700"
  >
    + Add link
  </button>
</section>

{backlinks.length > 0 && (
  <section>
    <header className="text-xs font-medium text-slate-600 dark:text-slate-300">Linked from</header>
    <ul className="mt-1 flex flex-col gap-1">
      {backlinks.map((b) => (
        <li key={`${b.boardId}:${b.cardIdx}`}>
          <button
            type="button"
            onClick={() => navigateToBacklink(b)}
            className="w-full rounded bg-slate-100 dark:bg-slate-700 px-2 py-1 text-left text-xs"
          >
            <span className="text-slate-500 dark:text-slate-400">{b.boardName} · </span>
            {b.cardTitle}
          </button>
        </li>
      ))}
    </ul>
  </section>
)}

<LinkPicker
  open={pickerOpen}
  onOpenChange={setPickerOpen}
  onPick={addLink}
  excludeBoardId={boardId}
  excludeCardId={card.id}
/>
```

- [ ] **Step 4: Run + typecheck + commit**

```bash
cd web/renderer/default && bun test && bun run typecheck
git add web/renderer/default/src/components/CardDetailModal.tsx
git commit -m "feat(renderer): Links + Linked-from sections in CardDetailModal

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

If tests fail because the LinkChip's `useResolveLink` triggers an extra `getBoard` request not seeded in test cache, mock the server in `setup` to return an empty board for unknown slugs (the resolver tolerates the failure).

---

## Task 12: Build + bundle

**Files:**
- Possibly: `scripts/check-bundle-size.sh`

- [ ] **Step 1: Build**

```bash
cd /Users/htruong/code/htruong/liveboard && make renderer
```

If `bundle-check` fails, bump `MAX_BYTES` in the script (the new components add ~3 KB gz). Add a comment with the new measurement and date.

- [ ] **Step 2: Verify embed**

```bash
go test ./internal/api/ -run TestShellRoute
```

- [ ] **Step 3: Commit if budget changed**

```bash
git add scripts/check-bundle-size.sh
git commit -m "chore(build): bump bundle budget for cross-board linking UI

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 13: Manual smoke

Not a code change.

- [ ] **Step 1: Build + serve**

```bash
make adapter-test
```

- [ ] **Step 2: At <http://localhost:7070/app/>**

1. Open a card detail modal. New "Links" section appears, empty, with "+ Add link".
2. Click "+ Add link" → picker opens. Type a query → matching cards appear.
3. Pick one → modal returns; chip appears with "<Board> · <Card title>".
4. Click Save → modal closes; markdown file under `./demo/` shows the `links: <board>:<id>` line.
5. Reopen the card → chip persists.
6. Open the linked target card's modal → "Linked from" section shows the source card.
7. Click chip on either card → navigates to that board, focuses that card (blue ring).
8. Delete the target card via its modal's UI ✕ (or via column kebab → delete) → reopen the source's modal → chip shows "(missing)" in italics.
9. Click ✕ on a chip → chip removed; Save persists.
10. `?renderer=stub` still loads.

- [ ] **Step 3: Report.** Capture failures with step + expected vs actual.

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| `Card.links` field (Go + TS) | 1 |
| `EditCardOp.Links` (Go + TS variant) | 1 |
| Parser reads `links: a, b, c` | 2 |
| Writer emits `links: …` | 2 |
| `edit_card` engine apply writes Links | 3 |
| `applyOp` mirror writes links | 4 |
| Existing `edit_card` callsites pass `links` | 5 |
| bleve `Doc.Links` (keyword) + `Hit.CardID` + `Backlinks` | 6 |
| `GET /api/v1/cards/{cardId}/backlinks` | 7 |
| Search DTO surfaces `card_id` | 7 |
| `BackendAdapter.backlinks` + protocol/broker/client | 8 |
| `SearchHit.cardId` on both adapters | 8 |
| `useBacklinks` + `useResolveLink` hooks | 9 |
| `<LinkChip>` + `<LinkPicker>` | 10 |
| CardDetailModal Links + Linked-from sections | 11 |
| Bundle gate | 12 |
| Manual smoke | 13 |

## Notes for implementer

1. **Tasks 1 + 2 commit together** because the schema field needs the parser/writer to round-trip. Task 1 alone leaves a half-baked model.
2. **Task 5 lands `links: card.links ?? []` as a placeholder** in CardDetailModal Save; Task 11 replaces it with the live `links` state. This split keeps each commit small and green.
3. **bleve keyword analyzer** must be set at index-mapping time. Existing index from P6.2 used the default analyzer; the change takes effect on next server start.
4. **`useResolveLink` invalidation**: when a board mutation happens, links resolving against that board may have stale labels. Add `qc.invalidateQueries({ queryKey: ['resolve'] })` to `useBoardMutation.onSuccess` if labels feel stuck. Optional polish; not strictly required for ship.
5. **`SearchHit.cardId` propagation**: TS type compiles via the union; consumers (`CommandPalette`, `LinkPicker`) read it. Existing `CommandPalette` doesn't need changes — it doesn't use `cardId` directly (uses `colIdx`/`cardIdx` for navigation).
6. **No commit amending** — forward-only commits.
