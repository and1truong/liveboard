# P6.2 — Full-Text Search (bleve) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-card full-text search end-to-end: bleve in-memory index in Go, `GET /api/v1/search`, `BackendAdapter.search()` on both adapters, debounced `useSearch` hook, and a "Cards" section in `<CommandPalette>`. Selecting a hit switches active board and sets focused card.

**Architecture:** Server-side bleve memory-only index, full reindex at startup, incremental updates after every mutation/CRUD via direct hooks in `internal/api/v1/` handlers. Renderer treats search as just another adapter method; the cmdk palette gains a "Cards" group.

**Tech Stack:** `github.com/blevesearch/bleve/v2` (Go). No new TS deps (`marked`'s sanitizer reused for snippet rendering).

**Spec:** `docs/superpowers/specs/2026-04-15-p6-2-full-text-search-design.md`

**Conventions:**
- Go code under `internal/search/`, `internal/api/v1/`.
- TS under `web/shared/src/adapters/`, `web/renderer/default/src/`.
- Tests colocated.
- Commit prefixes: `feat(search)`, `feat(api)`, `feat(shared)`, `feat(renderer)`, `chore(deps)`.
- `make lint` after every Go change.
- Use bun, never npx (irrelevant in Go tasks).

---

## File structure

**New (Go):**
- `internal/search/index.go`
- `internal/search/index_test.go`
- `internal/api/v1/search.go`
- `internal/api/v1/search_test.go`

**New (TS):**
- `web/renderer/default/src/queries/useSearch.ts`

**Modified (Go):**
- `go.mod` / `go.sum` — add `github.com/blevesearch/bleve/v2`.
- `internal/api/v1/router.go` — add Deps field + register `GET /search`.
- `internal/api/v1/boards.go`, `mutations.go`, etc. — call `Deps.Search.UpdateBoard` / `DeleteBoard` after writes.
- `internal/api/server.go` — build the index at startup; pass into Deps.

**Modified (TS):**
- `web/shared/src/adapter.ts` — add `SearchHit` + `search()`.
- `web/shared/src/adapters/server.ts` — implement.
- `web/shared/src/adapters/local.ts` — implement substring fallback.
- `web/shared/src/client.ts` — add `search()` passthrough on the postMessage `Client` (search needs to traverse the iframe boundary).
- `web/shared/src/protocol.ts` — add `search` to `Request` union; `data: SearchHit[]` for response.
- `web/shared/src/broker.ts` — route `search` → adapter.
- `web/renderer/default/src/components/CommandPalette.tsx` — new "Cards" section.

---

## Task 1: bleve dep

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add bleve**

```bash
cd /Users/htruong/code/htruong/liveboard && go get github.com/blevesearch/bleve/v2@v2.5.4
```

(If a newer minor exists, prefer the latest stable v2.x.)

- [ ] **Step 2: Smoke**

```bash
cd /Users/htruong/code/htruong/liveboard && go build ./...
```
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore(deps): add blevesearch/bleve/v2 for full-text search

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: `internal/search/index.go` + tests

**Files:**
- Create: `internal/search/index.go`
- Create: `internal/search/index_test.go`

- [ ] **Step 1: Failing tests first**

Create `internal/search/index_test.go`:
```go
package search_test

import (
    "testing"

    "github.com/and1truong/liveboard/internal/search"
    "github.com/and1truong/liveboard/pkg/models"
)

func newBoard(name string, columns ...models.Column) *models.Board {
    return &models.Board{Name: name, Version: 1, Columns: columns}
}

func col(name string, cards ...*models.Card) models.Column {
    return models.Column{Name: name, Cards: cards}
}

func card(title, body string, tags ...string) *models.Card {
    return &models.Card{Title: title, Body: body, Tags: tags}
}

func TestSearch_BuildAndQuery(t *testing.T) {
    idx, err := search.New()
    if err != nil { t.Fatal(err) }
    b := newBoard("Welcome", col("Todo", card("Read the docs", "see the wiki", "docs")))
    if err := idx.UpdateBoard("welcome", b); err != nil { t.Fatal(err) }

    hits, err := idx.Search("docs", 10)
    if err != nil { t.Fatal(err) }
    if len(hits) == 0 { t.Fatal("expected at least 1 hit") }
    h := hits[0]
    if h.BoardID != "welcome" { t.Errorf("board_id = %q", h.BoardID) }
    if h.CardIdx != 0 || h.ColIdx != 0 { t.Errorf("indices = (%d,%d)", h.ColIdx, h.CardIdx) }
    if h.CardTitle != "Read the docs" { t.Errorf("title = %q", h.CardTitle) }
}

func TestSearch_UpdateReplaces(t *testing.T) {
    idx, _ := search.New()
    _ = idx.UpdateBoard("foo", newBoard("Foo", col("Todo", card("alpha", ""))))
    _ = idx.UpdateBoard("foo", newBoard("Foo", col("Todo", card("bravo", ""))))
    if hits, _ := idx.Search("alpha", 10); len(hits) != 0 {
        t.Errorf("expected old text gone, got %d hits", len(hits))
    }
    if hits, _ := idx.Search("bravo", 10); len(hits) == 0 {
        t.Errorf("expected new text indexed")
    }
}

func TestSearch_DeleteBoard(t *testing.T) {
    idx, _ := search.New()
    _ = idx.UpdateBoard("foo", newBoard("Foo", col("Todo", card("unique-token", ""))))
    _ = idx.DeleteBoard("foo")
    if hits, _ := idx.Search("unique-token", 10); len(hits) != 0 {
        t.Errorf("expected 0 hits after delete, got %d", len(hits))
    }
}

func TestSearch_TwoBoardsCorrectAttribution(t *testing.T) {
    idx, _ := search.New()
    _ = idx.UpdateBoard("a", newBoard("A", col("Todo", card("hello world", ""))))
    _ = idx.UpdateBoard("b", newBoard("B", col("Todo", card("hello there", ""))))
    hits, _ := idx.Search("hello", 10)
    if len(hits) < 2 { t.Fatalf("expected 2 hits, got %d", len(hits)) }
    seen := map[string]bool{}
    for _, h := range hits { seen[h.BoardID] = true }
    if !seen["a"] || !seen["b"] {
        t.Errorf("expected both boards in hits, got %v", seen)
    }
}
```

- [ ] **Step 2: Run, expect fail (package missing)**

```bash
cd /Users/htruong/code/htruong/liveboard && go test ./internal/search/ -v
```

- [ ] **Step 3: Implement**

Create `internal/search/index.go`:
```go
// Package search wraps bleve to provide per-card full-text indexing.
package search

import (
    "fmt"

    "github.com/blevesearch/bleve/v2"
    "github.com/blevesearch/bleve/v2/search"

    "github.com/and1truong/liveboard/pkg/models"
)

type Hit struct {
    BoardID   string
    BoardName string
    ColIdx    int
    CardIdx   int
    CardTitle string
    Snippet   string
}

type doc struct {
    BoardID   string   `json:"board_id"`
    BoardName string   `json:"board_name"`
    ColIdx    int      `json:"col_idx"`
    CardIdx   int      `json:"card_idx"`
    Title     string   `json:"title"`
    Body      string   `json:"body"`
    Tags      []string `json:"tags"`
}

type Index struct {
    idx bleve.Index
}

func New() (*Index, error) {
    mapping := bleve.NewIndexMapping()
    idx, err := bleve.NewMemOnly(mapping)
    if err != nil {
        return nil, err
    }
    return &Index{idx: idx}, nil
}

// UpdateBoard purges any existing docs for slug and re-indexes the board's cards.
func (i *Index) UpdateBoard(slug string, b *models.Board) error {
    if err := i.DeleteBoard(slug); err != nil {
        return err
    }
    boardName := b.Name
    if boardName == "" {
        boardName = slug
    }
    for cIdx, col := range b.Columns {
        for kIdx, c := range col.Cards {
            if c == nil {
                continue
            }
            d := doc{
                BoardID:   slug,
                BoardName: boardName,
                ColIdx:    cIdx,
                CardIdx:   kIdx,
                Title:     c.Title,
                Body:      c.Body,
                Tags:      c.Tags,
            }
            id := fmt.Sprintf("%s:%d:%d", slug, cIdx, kIdx)
            if err := i.idx.Index(id, d); err != nil {
                return err
            }
        }
    }
    return nil
}

// DeleteBoard removes every doc whose ID has the slug prefix.
func (i *Index) DeleteBoard(slug string) error {
    prefix := slug + ":"
    // Find all IDs to delete via a prefix match on board_id (cheap given small N).
    q := bleve.NewTermQuery(slug)
    q.SetField("board_id")
    sr := bleve.NewSearchRequestOptions(q, 1000, 0, false)
    sr.Fields = []string{"board_id"}
    res, err := i.idx.Search(sr)
    if err != nil {
        return err
    }
    for _, h := range res.Hits {
        if h.ID != "" && (len(h.ID) >= len(prefix) && h.ID[:len(prefix)] == prefix) {
            if err := i.idx.Delete(h.ID); err != nil {
                return err
            }
        }
    }
    return nil
}

// Search runs a query string against the index and returns hits with snippets.
func (i *Index) Search(query string, limit int) ([]Hit, error) {
    if limit <= 0 {
        limit = 20
    }
    q := bleve.NewQueryStringQuery(query)
    sr := bleve.NewSearchRequestOptions(q, limit, 0, false)
    sr.Highlight = bleve.NewHighlight()
    sr.Highlight.AddField("title")
    sr.Highlight.AddField("body")
    sr.Fields = []string{"board_id", "board_name", "col_idx", "card_idx", "title"}
    res, err := i.idx.Search(sr)
    if err != nil {
        return nil, err
    }
    hits := make([]Hit, 0, len(res.Hits))
    for _, h := range res.Hits {
        hits = append(hits, Hit{
            BoardID:   getString(h.Fields, "board_id"),
            BoardName: getString(h.Fields, "board_name"),
            ColIdx:    getInt(h.Fields, "col_idx"),
            CardIdx:   getInt(h.Fields, "card_idx"),
            CardTitle: getString(h.Fields, "title"),
            Snippet:   firstSnippet(h.Fragments),
        })
    }
    return hits, nil
}

func getString(m map[string]interface{}, k string) string {
    if v, ok := m[k]; ok {
        if s, ok := v.(string); ok {
            return s
        }
    }
    return ""
}

func getInt(m map[string]interface{}, k string) int {
    if v, ok := m[k]; ok {
        if f, ok := v.(float64); ok {
            return int(f)
        }
    }
    return 0
}

func firstSnippet(frags search.FieldFragmentMap) string {
    for _, field := range []string{"body", "title"} {
        if list, ok := frags[field]; ok && len(list) > 0 {
            return list[0]
        }
    }
    return ""
}
```

- [ ] **Step 4: Run, expect 4 pass**

```bash
go test ./internal/search/ -v
```

If any test fails because `models.Card` / `models.Column` field shapes differ from what the test assumes (e.g. `Cards` is `[]Card` not `[]*Card`), adapt the test helpers minimally.

If `DeleteBoard` returns 0 hits because the in-memory index isn't fully visible immediately, add an explicit IndexBatch flush — but `bleve.NewMemOnly` is synchronous; this shouldn't be needed.

- [ ] **Step 5: Lint + commit**

```bash
make lint
git add internal/search/index.go internal/search/index_test.go
git commit -m "feat(search): in-memory bleve index for per-card search

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: `Deps.Search` + index init at startup

**Files:**
- Modify: `internal/api/v1/router.go`
- Modify: `internal/api/server.go`

- [ ] **Step 1: Add `Search` field to `Deps`**

Read `internal/api/v1/router.go`. Extend the `Deps` struct:
```go
import (
    // ...existing...
    "github.com/and1truong/liveboard/internal/search"
)

type Deps struct {
    Workspace *workspace.Workspace
    Engine    *board.Engine
    SSE       *web.SSEBroker
    Search    *search.Index
}
```

- [ ] **Step 2: Build index at startup**

Read `internal/api/server.go` near `mountAPIRoutes` (around line 301). Replace the `r.Mount("/api/v1", apiv1.Router(apiv1.Deps{...}))` call with one that constructs the index first:

```go
func (s *Server) mountAPIRoutes(r chi.Router) {
    idx, err := search.New()
    if err != nil {
        log.Printf("search: failed to init index: %v", err)
    }
    if idx != nil {
        if boards, err := s.ws.ListBoards(); err == nil {
            for _, b := range boards {
                slug, _ := s.ws.SlugFor(b.Name)  // adapt to whatever helper exists
                _ = idx.UpdateBoard(slug, &b)
            }
        }
    }
    r.Mount("/api/v1", apiv1.Router(apiv1.Deps{
        Workspace: s.ws,
        Engine:    s.eng,
        SSE:       s.webHandler.SSE,
        Search:    idx,
    }))
    r.Method(http.MethodGet, "/api/versions", apiv1.VersionsHandler())
    // ... rest unchanged ...
}
```

If `Workspace.SlugFor` doesn't exist, derive the slug from the file path the way `ListBoards` does. Inspect the existing call sites in `internal/api/v1/boards.go` for the canonical conversion. If `ListBoards` returns `[]models.Board` without a slug companion, walk via `ListBoardSummaries` instead which returns slugs.

Add the `search` import to `server.go`:
```go
import (
    // ...existing...
    "github.com/and1truong/liveboard/internal/search"
)
```

- [ ] **Step 3: Build + lint**

```bash
cd /Users/htruong/code/htruong/liveboard && go build ./... && make lint
```

If the index isn't required (e.g. `idx == nil` due to init failure), all handlers must guard against that. Simplest: handlers check `if d.Search != nil` before calling `UpdateBoard`. The HTTP search handler returns `[]` if `Search == nil`.

- [ ] **Step 4: Commit**

```bash
git add internal/api/v1/router.go internal/api/server.go
git commit -m "feat(api): wire search index into v1 Deps and build at startup

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 4: `GET /api/v1/search` handler + tests

**Files:**
- Create: `internal/api/v1/search.go`
- Create: `internal/api/v1/search_test.go`
- Modify: `internal/api/v1/router.go`

- [ ] **Step 1: Tests**

Create `internal/api/v1/search_test.go`:
```go
package v1_test

import (
    "encoding/json"
    "net/http"
    "strings"
    "testing"
)

type searchHitDTO struct {
    BoardID   string `json:"board_id"`
    BoardName string `json:"board_name"`
    ColIdx    int    `json:"col_idx"`
    CardIdx   int    `json:"card_idx"`
    CardTitle string `json:"card_title"`
    Snippet   string `json:"snippet"`
}

func TestSearch_FindsCard(t *testing.T) {
    srv := newV1TestServer(t)
    postJSON(t, srv, "/api/v1/boards", `{"name":"Foo"}`)
    postJSON(t, srv, "/api/v1/boards/foo/mutations",
        `{"client_version":1,"op":{"type":"add_card","column":"Todo","title":"alpha-token bravo"}}`)

    res, body := getJSON(t, srv, "/api/v1/search?q=alpha-token")
    if res.StatusCode != http.StatusOK { t.Fatalf("status = %d body = %s", res.StatusCode, body) }
    var hits []searchHitDTO
    if err := json.Unmarshal([]byte(body), &hits); err != nil { t.Fatal(err) }
    if len(hits) == 0 { t.Fatalf("expected hits, got 0") }
    if hits[0].BoardID != "foo" { t.Errorf("board_id = %q", hits[0].BoardID) }
}

func TestSearch_EmptyQueryReturnsEmpty(t *testing.T) {
    srv := newV1TestServer(t)
    res, body := getJSON(t, srv, "/api/v1/search?q=")
    if res.StatusCode != http.StatusOK { t.Fatalf("status = %d", res.StatusCode) }
    if strings.TrimSpace(body) != "[]" { t.Errorf("body = %q", body) }
}

func TestSearch_TooLongIsInvalid(t *testing.T) {
    srv := newV1TestServer(t)
    long := strings.Repeat("x", 257)
    res, _ := getJSON(t, srv, "/api/v1/search?q="+long)
    if res.StatusCode != http.StatusBadRequest {
        t.Errorf("status = %d", res.StatusCode)
    }
}

func TestSearch_DeletedBoardGone(t *testing.T) {
    srv := newV1TestServer(t)
    postJSON(t, srv, "/api/v1/boards", `{"name":"Foo"}`)
    postJSON(t, srv, "/api/v1/boards/foo/mutations",
        `{"client_version":1,"op":{"type":"add_card","column":"Todo","title":"unique-token"}}`)
    deleteReq(t, srv, "/api/v1/boards/foo")
    _, body := getJSON(t, srv, "/api/v1/search?q=unique-token")
    if strings.TrimSpace(body) != "[]" {
        t.Errorf("expected [], got %q", body)
    }
}
```

The test helpers (`newV1TestServer`, `postJSON`, `getJSON`, `deleteReq`) already exist from prior plans. If `newV1TestServer` doesn't include the search index in its Deps, update that helper to call `search.New()` and pass it.

- [ ] **Step 2: Run, expect 404 / fail (handler missing)**

```bash
go test ./internal/api/v1/ -run TestSearch -v
```

- [ ] **Step 3: Handler**

Create `internal/api/v1/search.go`:
```go
package v1

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strconv"
)

type searchHitDTO struct {
    BoardID   string `json:"board_id"`
    BoardName string `json:"board_name"`
    ColIdx    int    `json:"col_idx"`
    CardIdx   int    `json:"card_idx"`
    CardTitle string `json:"card_title"`
    Snippet   string `json:"snippet"`
}

func (d Deps) getSearch(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query().Get("q")
    if len(q) > 256 {
        writeError(w, fmt.Errorf("%w: query too long", errInvalid))
        return
    }
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit <= 0 || limit > 100 {
        limit = 20
    }
    if q == "" || d.Search == nil {
        _ = json.NewEncoder(w).Encode([]searchHitDTO{})
        return
    }
    hits, err := d.Search.Search(q, limit)
    if err != nil {
        writeError(w, err)
        return
    }
    out := make([]searchHitDTO, 0, len(hits))
    for _, h := range hits {
        out = append(out, searchHitDTO{
            BoardID:   h.BoardID,
            BoardName: h.BoardName,
            ColIdx:    h.ColIdx,
            CardIdx:   h.CardIdx,
            CardTitle: h.CardTitle,
            Snippet:   h.Snippet,
        })
    }
    _ = json.NewEncoder(w).Encode(out)
}
```

- [ ] **Step 4: Register route**

In `internal/api/v1/router.go`, inside the `Router` body, add:
```go
r.Get("/search", d.getSearch)
```

- [ ] **Step 5: Run, expect 4 pass**

```bash
go test ./internal/api/v1/ -run TestSearch -v
```

If `TestSearch_FindsCard` fails because the test server's mutation handler doesn't call `UpdateBoard` yet, the next task fixes that. For now: skip / `t.Skip` the FindsCard test temporarily, OR call `idx.UpdateBoard` explicitly in the test setup. Prefer the latter so the test stays meaningful.

- [ ] **Step 6: Lint + commit**

```bash
make lint
git add internal/api/v1/search.go internal/api/v1/search_test.go internal/api/v1/router.go
git commit -m "feat(api): GET /api/v1/search with query-string parsing

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 5: Hook mutation/CRUD handlers into the index

**Files:**
- Modify: `internal/api/v1/boards.go` — `createBoard`, `renameBoard`, `deleteBoard`.
- Modify: `internal/api/v1/mutations.go` — `postMutation`.

- [ ] **Step 1: After each successful write, call into the index**

`createBoard`: after `Workspace.CreateBoard` returns the new board, call:
```go
if d.Search != nil {
    _ = d.Search.UpdateBoard(summary.Slug, board)
}
```
(`board` is the value/pointer returned by `CreateBoard`. Adapt to its actual return shape — if it returns `*models.Board`, pass that.)

`renameBoard`: after success, the response carries the new `BoardSummary`. Call:
```go
if d.Search != nil {
    if oldSlug != summary.Slug {
        _ = d.Search.DeleteBoard(oldSlug)
    }
    if b, err := d.Workspace.LoadBoard(summary.Slug); err == nil {
        _ = d.Search.UpdateBoard(summary.Slug, b)
    }
}
```

`deleteBoard`: after success:
```go
if d.Search != nil {
    _ = d.Search.DeleteBoard(slug)
}
```

`postMutation`: after a successful mutate, the handler has the resulting `*Board`. Call:
```go
if d.Search != nil {
    _ = d.Search.UpdateBoard(slug, board)
}
```

- [ ] **Step 2: Run all v1 tests**

```bash
cd /Users/htruong/code/htruong/liveboard && go test ./internal/api/v1/ -v -timeout 30s
```
Expected: all green. The previously-skipped `TestSearch_FindsCard` (if you skipped it in Task 4) now passes via the natural mutation flow — un-skip it.

- [ ] **Step 3: Lint + commit**

```bash
make lint
git add internal/api/v1/boards.go internal/api/v1/mutations.go
git commit -m "feat(api): keep search index in sync with board mutations + CRUD

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 6: TS protocol + `BackendAdapter.search` type

**Files:**
- Modify: `web/shared/src/adapter.ts`
- Modify: `web/shared/src/protocol.ts`

- [ ] **Step 1: Add `SearchHit` + method**

In `web/shared/src/adapter.ts`, append:
```ts
export interface SearchHit {
  boardId: string
  boardName: string
  colIdx: number
  cardIdx: number
  cardTitle: string
  snippet: string
}
```
Inside `BackendAdapter`, add:
```ts
  search(query: string, limit?: number): Promise<SearchHit[]>
```

- [ ] **Step 2: Add `search` to the protocol Request union**

In `web/shared/src/protocol.ts`, in the `Request` union, append:
```ts
  | { id: string; kind: 'request'; method: 'search'; params: { query: string; limit?: number } }
```

- [ ] **Step 3: Typecheck**

```bash
cd web/renderer/default && bun run typecheck
```
Expected: errors in `LocalAdapter` and `ServerAdapter` because they don't implement `search` yet. Fixed in Tasks 7 + 8. Don't commit yet.

---

## Task 7: `LocalAdapter.search` + tests

**Files:**
- Modify: `web/shared/src/adapters/local.ts`
- Create: `web/shared/src/adapters/local.search.test.ts`

- [ ] **Step 1: Tests**

Create `web/shared/src/adapters/local.search.test.ts`:
```ts
import { describe, expect, it } from 'bun:test'
import { LocalAdapter } from './local.js'
import { MemoryStorage } from './local-storage-driver.js'

describe('LocalAdapter.search', () => {
  it('returns empty for empty query', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    expect(await a.search('')).toEqual([])
    expect(await a.search('   ')).toEqual([])
  })

  it('substring matches across title/body/tags', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    await a.mutateBoard('foo', 1, { type: 'add_card', column: 'Todo', title: 'alpha-token' })
    const hits = await a.search('alpha')
    expect(hits.length).toBe(1)
    expect(hits[0].boardId).toBe('foo')
    expect(hits[0].cardTitle).toBe('alpha-token')
    expect(hits[0].colIdx).toBe(0)
    expect(hits[0].cardIdx).toBe(0)
  })

  it('respects limit', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    for (let i = 0; i < 5; i++) {
      await a.mutateBoard('foo', i + 1, { type: 'add_card', column: 'Todo', title: `xx ${i}` })
    }
    const hits = await a.search('xx', 2)
    expect(hits.length).toBe(2)
  })
})
```

- [ ] **Step 2: Run, expect fail**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src/adapters/local.search.test.ts
```

- [ ] **Step 3: Implement**

In `web/shared/src/adapters/local.ts`, add (alongside other public methods):
```ts
import type { SearchHit } from '../adapter.js'

// inside the LocalAdapter class:
async search(query: string, limit = 20): Promise<SearchHit[]> {
  const q = query.trim().toLowerCase()
  if (!q) return []
  const ws = this.loadWorkspace()
  const hits: SearchHit[] = []
  for (const id of ws.boardIds) {
    const board = this.loadBoard(id)
    const boardName = board.name ?? id
    const cols = board.columns ?? []
    for (let colIdx = 0; colIdx < cols.length; colIdx++) {
      const cards = cols[colIdx]?.cards ?? []
      for (let cardIdx = 0; cardIdx < cards.length; cardIdx++) {
        const c = cards[cardIdx]
        const haystack = `${c.title ?? ''} ${c.body ?? ''} ${(c.tags ?? []).join(' ')}`.toLowerCase()
        if (haystack.includes(q)) {
          hits.push({
            boardId: id,
            boardName,
            colIdx,
            cardIdx,
            cardTitle: c.title ?? '',
            snippet: c.title ?? '',
          })
          if (hits.length >= limit) return hits
        }
      }
    }
  }
  return hits
}
```

- [ ] **Step 4: Run, expect 3 pass**

```bash
bun test web/shared/src/adapters/local.search.test.ts
cd web/renderer/default && bun run typecheck
```
Expected: tests pass; ServerAdapter typecheck still failing — Task 8.

- [ ] **Step 5: Don't commit yet** — wait for ServerAdapter implementation so the tree stays green.

---

## Task 8: `ServerAdapter.search` + tests + Broker route + Client passthrough

**Files:**
- Modify: `web/shared/src/adapters/server.ts`
- Modify: `web/shared/src/adapters/server.test.ts`
- Modify: `web/shared/src/broker.ts`
- Modify: `web/shared/src/client.ts`

- [ ] **Step 1: Add `search` to ServerAdapter**

In `web/shared/src/adapters/server.ts`, add the method:
```ts
async search(query: string, limit = 20): Promise<SearchHit[]> {
  const params = new URLSearchParams({ q: query, limit: String(limit) })
  const raw = await this.getJSON<Array<{
    board_id: string
    board_name: string
    col_idx: number
    card_idx: number
    card_title: string
    snippet: string
  }>>(`/search?${params}`)
  return raw.map((d) => ({
    boardId: d.board_id,
    boardName: d.board_name,
    colIdx: d.col_idx,
    cardIdx: d.card_idx,
    cardTitle: d.card_title,
    snippet: d.snippet,
  }))
}
```

Add `SearchHit` to the imports from `../adapter.js`.

- [ ] **Step 2: Test**

Append to `web/shared/src/adapters/server.test.ts`:
```ts
describe('ServerAdapter.search', () => {
  it('GETs /search?q=&limit= and maps DTO to camelCase', async () => {
    const log: RequestRecord[] = []
    const a = new ServerAdapter({
      baseUrl: '/api/v1',
      fetch: mockFetch(
        () => jsonResponse([
          { board_id: 'foo', board_name: 'Foo', col_idx: 0, card_idx: 2, card_title: 'hi', snippet: 'hi <mark>match</mark>' },
        ]),
        log,
      ),
    })
    const hits = await a.search('match', 5)
    expect(hits[0]).toEqual({
      boardId: 'foo',
      boardName: 'Foo',
      colIdx: 0,
      cardIdx: 2,
      cardTitle: 'hi',
      snippet: 'hi <mark>match</mark>',
    })
    expect(log[0].url).toBe('/api/v1/search?q=match&limit=5')
  })
})
```

- [ ] **Step 3: Add `search` route in Broker**

Read `web/shared/src/broker.ts`. In the `handle` method's switch, add:
```ts
case 'search':
  return this.adapter.search(req.params.query, req.params.limit)
```

- [ ] **Step 4: Add `search` to Client**

In `web/shared/src/client.ts`, after the existing `deleteBoard` method, add:
```ts
search(query: string, limit?: number): Promise<SearchHit[]> {
  return this.request({ kind: 'request', method: 'search', params: { query, limit } })
}
```

Add `SearchHit` to the imports at the top.

- [ ] **Step 5: Run + commit Tasks 6 + 7 + 8 together**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src && cd web/renderer/default && bun run typecheck
```
Expected: all green; only pre-existing failures (boardOps vector test) remain.

```bash
cd /Users/htruong/code/htruong/liveboard
git add web/shared/src/adapter.ts \
        web/shared/src/protocol.ts \
        web/shared/src/adapters/local.ts \
        web/shared/src/adapters/local.search.test.ts \
        web/shared/src/adapters/server.ts \
        web/shared/src/adapters/server.test.ts \
        web/shared/src/broker.ts \
        web/shared/src/client.ts
git commit -m "feat(shared): add search method on adapter + broker + client + server/local impls

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 9: `useSearch` hook

**Files:**
- Create: `web/renderer/default/src/queries/useSearch.ts`

- [ ] **Step 1: Implement**

Create `web/renderer/default/src/queries/useSearch.ts`:
```ts
import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import type { SearchHit } from '@shared/adapter.js'
import { useClient } from '../queries.js'

function useDebouncedValue<T>(value: T, ms: number): T {
  const [v, setV] = useState(value)
  useEffect(() => {
    const t = setTimeout(() => setV(value), ms)
    return () => clearTimeout(t)
  }, [value, ms])
  return v
}

export function useSearch(query: string): SearchHit[] {
  const client = useClient()
  const debounced = useDebouncedValue(query, 200)
  const q = useQuery({
    queryKey: ['search', debounced],
    queryFn: () => client.search(debounced, 20),
    enabled: debounced.trim().length > 0,
  })
  return q.data ?? []
}
```

- [ ] **Step 2: Typecheck + commit**

```bash
cd web/renderer/default && bun run typecheck
git add web/renderer/default/src/queries/useSearch.ts
git commit -m "feat(renderer): add useSearch debounced hook

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 10: Cards section in `<CommandPalette>`

**Files:**
- Modify: `web/renderer/default/src/components/CommandPalette.tsx`

- [ ] **Step 1: Edit imports + state**

In `CommandPalette.tsx`, add at the top of the existing imports:
```tsx
import { useSearch } from '../queries/useSearch.js'
import { useBoardFocus } from '../contexts/BoardFocusContext.js'
import { sanitize } from './markdownPreview.js'
```

Inside the function, near other state hooks:
```tsx
const [query, setQuery] = useState('')
const hits = useSearch(query)
const { setFocused } = useBoardFocus()
```

- [ ] **Step 2: Make `Command.Input` controlled**

Find the existing `<Command.Input ... />` (probably uncontrolled, with a default placeholder). Replace its props with:
```tsx
<Command.Input
  value={query}
  onValueChange={setQuery}
  placeholder="Type a command, board, or card…"
  className="..."  // keep existing class
/>
```

- [ ] **Step 3: Add the Cards group**

In the `<Command.List>` body, after the existing Boards and Actions groups, add:
```tsx
{hits.length > 0 && (
  <Command.Group heading="Cards" className="...">  // keep existing heading style
    {hits.map((h) => (
      <Command.Item
        key={`${h.boardId}:${h.colIdx}:${h.cardIdx}`}
        value={`card ${h.cardTitle} ${query}`}
        onSelect={() => {
          setActive(h.boardId)
          Promise.resolve().then(() => setFocused({ colIdx: h.colIdx, cardIdx: h.cardIdx }))
          close()
        }}
        className="cursor-pointer rounded px-3 py-1.5 text-sm text-slate-800 aria-selected:bg-slate-100 dark:text-slate-100 dark:aria-selected:bg-slate-700"
      >
        <span className="font-semibold">{h.cardTitle}</span>
        <span className="ml-2 text-xs text-slate-400">in {h.boardName}</span>
        {h.snippet && (
          <span
            className="block text-xs text-slate-500 dark:text-slate-400"
            dangerouslySetInnerHTML={{ __html: sanitize(h.snippet) }}
          />
        )}
      </Command.Item>
    ))}
  </Command.Group>
)}
```

`close()` is the existing palette-close function (maps to `setOpen(false)` or `onOpenChange(false)` depending on the host shape).

- [ ] **Step 4: Reset query on close**

In the existing `useEffect` that resets the page on reopen (the `if (open) { setPage('list') ... }` block), also reset the query:
```tsx
useEffect(() => {
  if (open) {
    setPage('list')
    setQuery('')
    committedRef.current = false
  }
}, [open])
```

- [ ] **Step 5: Run renderer suite + typecheck**

```bash
cd web/renderer/default && bun test && bun run typecheck
```
Expected: green. Existing CommandPalette tests should still pass since the Cards group is conditional on `hits.length > 0` and tests don't seed hits.

- [ ] **Step 6: Commit**

```bash
git add web/renderer/default/src/components/CommandPalette.tsx
git commit -m "feat(renderer): Cards section in CommandPalette via useSearch

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 11: Build + bundle measurement

**Files:**
- Possibly: `scripts/check-bundle-size.sh` (only if the new TS surface pushes over the budget).

- [ ] **Step 1: Build**

```bash
cd /Users/htruong/code/htruong/liveboard && make renderer
```
Expected: build + bundle-check. The new code is small (~2 KB gz) and may fit under the existing budget.

- [ ] **Step 2: If gate fails, bump budget**

If `bundle-check` fails, add ~5 KB to `MAX_BYTES` in `scripts/check-bundle-size.sh` and add a comment with the new measurement and date. Re-run.

- [ ] **Step 3: Verify Go embed test**

```bash
go test ./internal/api/ -run TestShellRoute
```
Expected: 3 tests pass.

- [ ] **Step 4: Commit if budget changed; otherwise skip.**

```bash
git add scripts/check-bundle-size.sh
git commit -m "chore(build): bump bundle budget for search hits in palette

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 12: Manual smoke

Not a code change.

- [ ] **Step 1: Server mode**

```bash
make adapter-test
```
Open <http://localhost:7070/app/>:

1. Press Cmd+K → palette opens.
2. Type a word that appears in any card title/body — wait ~200 ms (debounce).
3. A "Cards" section appears under Boards/Actions with one or more matches.
4. Snippet line shows context with `<mark>` tags rendered as highlighted text.
5. Click a card hit → palette closes; sidebar switches to that board; the matching card is focused (blue ring around it).
6. Search for `<script>alert(1)</script>` in some card's body. Snippet renders the literal text safely; no alert fires.
7. Add a new card via the UI; immediately Cmd+K and search for its title — appears in hits (index updated post-mutation).
8. Delete a board; search for one of its cards — no hits.
9. `?renderer=stub` still loads.

- [ ] **Step 2: Local mode**

`make online` and serve the bundle outside Go. Cmd+K → search works via LocalAdapter substring scan. (Slower, but functional.)

- [ ] **Step 3: Report.** Capture failures with step + expected vs actual.

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| `internal/search/Index` (bleve memory-only) | 2 |
| `Build`, `UpdateBoard`, `DeleteBoard`, `Search` methods | 2 |
| Per-card doc with `(col_idx, card_idx)` | 2 |
| Highlight snippet | 2 |
| `GET /api/v1/search?q=&limit=` | 4 |
| Empty `q` → `[]` | 4 |
| `q > 256` → 400 INVALID | 4 |
| Index built at startup | 3 |
| Index updates on every mutation/CRUD | 5 |
| `BackendAdapter.search()` interface addition | 6 |
| Broker + Client routing | 8 |
| ServerAdapter HTTP impl + DTO mapping | 8 |
| LocalAdapter substring impl + (colIdx, cardIdx) | 7 |
| `useSearch` debounced hook | 9 |
| Cards section in CommandPalette | 10 |
| Selection switches active + sets focused | 10 |
| Snippet sanitized via P6.1 helper | 10 |
| Bundle measurement | 11 |
| Manual smoke | 12 |

## Notes for implementer

1. **`go get` in Task 1** pulls a substantial dep tree. If the Go build slows, that's expected. Server binary grows by ~5–10 MB.
2. **Tasks 6+7+8 land in one commit** because the `BackendAdapter` interface change forces both adapters to implement `search`. Splitting earlier breaks `bun run typecheck`. Tasks 6 (interface) and 7 (Local) and 8 (Server + Broker + Client) must commit together.
3. **`firstSnippet` returns the first body fragment, then title fragment** — small heuristic that works well for typical snippets.
4. **DeleteBoard in bleve** does a search-by-prefix then per-ID delete. Cheap for typical workspaces. If a workspace has 10k+ cards, switch to `idx.NewBatch()` for bulk deletes.
5. **`Command.Input` controlled** via `value`/`onValueChange` is supported in cmdk v1+. If a regression appears, revert to uncontrolled and lift query into a separate state via the input ref.
6. **`writeError` in v1** — the existing helper formats `errInvalid` etc. into the JSON envelope. Reuse it; don't introduce a new pattern.
7. **No commit amending** — forward-only commits.
