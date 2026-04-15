# P6.2 — Full-Text Search (bleve) — Design

## Goal

Add per-card full-text search to LiveBoard:

- Server-side **in-memory bleve index** built at startup, kept fresh by hooking every mutation handler.
- New `GET /api/v1/search?q=&limit=` endpoint returning hits with snippets.
- New `BackendAdapter.search()` method; `ServerAdapter` calls the HTTP endpoint, `LocalAdapter` does in-process substring matching.
- Renderer integrates into the existing `Cmd+K` palette as a **Cards** section. Selecting a hit switches active board and focuses the matching card.

**Shippable value:** users can find a card by its title/body/tag from anywhere via Cmd+K, both in server mode (bleve-backed) and local mode (substring fallback).

## Scope

**In:**
- New Go package `internal/search/` wrapping bleve in-memory index.
- `Build`, `UpdateBoard`, `DeleteBoard`, `Search` methods.
- `GET /api/v1/search` route + handler.
- Hooks in mutation/CRUD handlers to keep the index fresh.
- `BackendAdapter.search()` on the TS interface; ServerAdapter HTTP impl; LocalAdapter substring impl.
- `useSearch` debounced React Query hook (200 ms).
- "Cards" section in `<CommandPalette>`; `onSelect` switches board + sets `BoardFocusContext.focused`.
- Tests covering index logic, the HTTP handler, and both adapter `search()` paths.

**Out:**
- Persisted bleve index (rebuild on each startup).
- fsnotify watcher for external `.md` edits (documented limitation).
- Auto-opening the card detail modal after selection (defer; for P6.2 selection just switches board + sets focus).
- Highlighting the matched card with a flash animation.
- Search filters (board: prefix, tag: prefix, etc.) — defer.
- Cross-server federation.

## Architecture

```
Go side
  internal/search/         (new)
   ├─ Index struct wrapping bleve.NewMemOnly()
   ├─ Build(workspace) — full reindex on startup
   ├─ UpdateBoard(slug, *Board) — delete + re-index this board's cards
   ├─ DeleteBoard(slug) — purge by slug prefix
   └─ Search(query, limit) → []Hit

  internal/api/v1/
   ├─ search.go — GET /search → JSON
   ├─ Hooks in postMutation / createBoard / renameBoard / deleteBoard
   └─ Deps gains Search *search.Index

  internal/api/server.go
   └─ Build search index at startup, inject into Deps

TS side
  web/shared/src/adapter.ts        adds search() to BackendAdapter + SearchHit type
  web/shared/src/adapters/server.ts ServerAdapter.search() = GET /search
  web/shared/src/adapters/local.ts  LocalAdapter.search()  = substring scan over loaded boards

  web/renderer/default/src/queries/useSearch.ts
   ├─ useDebouncedValue(query, 200)
   └─ useQuery(['search', q]) → client.search(q, 20)

  web/renderer/default/src/components/CommandPalette.tsx
   ├─ adds <Command.Group heading="Cards"> when hits.length > 0
   └─ onSelect → setActive(boardId), setFocused({colIdx,cardIdx}), close palette
```

The renderer doesn't talk HTTP directly — it uses `client.search()` from the postMessage `Client`. ServerAdapter on the shell side handles the HTTP. LocalAdapter handles substring. Renderer is adapter-agnostic.

## Wire shapes

### `GET /api/v1/search?q=<query>&limit=<n>`

Defaults: `limit=20`. Empty `q` → `200 []`. `q` longer than 256 chars → `400 INVALID`.

Response body — JSON array of:
```json
{
  "board_id": "welcome",
  "board_name": "Welcome",
  "card_idx": 3,
  "card_title": "Read the docs",
  "snippet": "the project <mark>docs</mark> are in the wiki"
}
```

`snippet` may contain `<mark>` tags from bleve's highlighter. Renderer renders via the existing P6.1 sanitizer.

### TS types

`web/shared/src/adapter.ts`:
```ts
export interface SearchHit {
  boardId: string
  boardName: string
  cardIdx: number
  cardTitle: string
  snippet: string
}

export interface BackendAdapter {
  // ...existing...
  search(query: string, limit?: number): Promise<SearchHit[]>
}
```

The wire DTO (snake_case `board_id`/`card_idx`/`card_title`) is mapped to camelCase in `ServerAdapter`.

## Doc layout in bleve

Each card → one bleve doc. ID format: `<slug>:<card_idx>`. Doc fields:
```go
type Doc struct {
    BoardID    string   `json:"board_id"`
    BoardName  string   `json:"board_name"`
    CardIdx    int      `json:"card_idx"`
    Title      string   `json:"title"`
    Body       string   `json:"body"`
    Tags       []string `json:"tags"`
}
```

`UpdateBoard(slug, board)`:
1. Delete all docs with ID prefix `<slug>:` (bleve `Search` then iterate IDs and `Delete`).
2. Iterate columns and cards, indexing each as a fresh doc.

`DeleteBoard(slug)`: same prefix-purge, no re-add.

`RenameBoard(oldSlug, newBoard)`: `DeleteBoard(oldSlug)` then `UpdateBoard(newSlug, newBoard)`.

## Search query

```go
func (i *Index) Search(query string, limit int) ([]Hit, error) {
    sr := bleve.NewSearchRequestOptions(bleve.NewQueryStringQuery(query), limit, 0, false)
    sr.Highlight = bleve.NewHighlight()
    sr.Highlight.AddField("title")
    sr.Highlight.AddField("body")
    sr.Fields = []string{"board_id", "board_name", "card_idx", "title"}
    res, err := i.idx.Search(sr)
    if err != nil { return nil, err }

    hits := make([]Hit, 0, len(res.Hits))
    for _, h := range res.Hits {
        snippet := firstSnippet(h.Fragments)
        hits = append(hits, Hit{
            BoardID:   getString(h.Fields["board_id"]),
            BoardName: getString(h.Fields["board_name"]),
            CardIdx:   getInt(h.Fields["card_idx"]),
            CardTitle: getString(h.Fields["title"]),
            Snippet:   snippet,
        })
    }
    return hits, nil
}
```

`firstSnippet` picks the first fragment from `body` (or `title` if no body match), preserving `<mark>` tags.

## Hooks in handlers

After each successful write in `internal/api/v1/`:
- `postMutation` → `deps.Search.UpdateBoard(slug, board)` with the returned `*Board`.
- `createBoard` → `UpdateBoard(slug, newBoard)`.
- `renameBoard` → `DeleteBoard(oldSlug)` + `UpdateBoard(newSlug, board)` (or just `UpdateBoard` if the new slug == old).
- `deleteBoard` → `DeleteBoard(slug)`.

`putBoardSettings` doesn't change card text — skip.

Failures from `index.UpdateBoard` are logged but NOT propagated to the API response. Index drift is recoverable; the user's mutation succeeded.

## TS adapter implementations

### `ServerAdapter.search`

```ts
search(query: string, limit = 20): Promise<SearchHit[]> {
  const params = new URLSearchParams({ q: query, limit: String(limit) })
  return this.getJSON<SearchHitDTO[]>(`/search?${params}`).then(arr => arr.map(toSearchHit))
}
```

Where:
```ts
interface SearchHitDTO {
  board_id: string
  board_name: string
  card_idx: number
  card_title: string
  snippet: string
}
function toSearchHit(d: SearchHitDTO): SearchHit {
  return { boardId: d.board_id, boardName: d.board_name, cardIdx: d.card_idx, cardTitle: d.card_title, snippet: d.snippet }
}
```

### `LocalAdapter.search`

```ts
async search(query: string, limit = 20): Promise<SearchHit[]> {
  const q = query.trim().toLowerCase()
  if (!q) return []
  const ws = this.loadWorkspace()
  const hits: SearchHit[] = []
  for (const id of ws.boardIds) {
    const board = this.loadBoard(id)
    const boardName = board.name ?? id
    const cards = (board.columns ?? []).flatMap((c) => c.cards ?? [])
    let i = 0
    for (const c of cards) {
      const haystack = `${c.title ?? ''} ${c.body ?? ''} ${(c.tags ?? []).join(' ')}`.toLowerCase()
      if (haystack.includes(q)) {
        hits.push({
          boardId: id,
          boardName,
          cardIdx: i,
          cardTitle: c.title ?? '',
          snippet: c.title ?? '',
        })
        if (hits.length >= limit) return hits
      }
      i++
    }
  }
  return hits
}
```

LocalAdapter `cardIdx` mirrors the linear index used by mutations — must match the renderer's per-column `card_idx`. **Caveat:** the existing LocalAdapter's mutation ops use `(col_idx, card_idx)`. Linear-flat indexing breaks this. Fix: track `(colIdx, cardIdx)` per card hit:

```ts
async search(query, limit = 20): Promise<SearchHit[]> {
  // ... iterate columns and cards with both indices ...
  for (let colIdx = 0; colIdx < cols.length; colIdx++) {
    for (let cardIdx = 0; cardIdx < cards.length; cardIdx++) {
      // match → hit { ..., cardIdx, /* implicit colIdx in board navigation */ }
    }
  }
}
```

The `SearchHit` type needs `colIdx` too — call it out:
```ts
export interface SearchHit {
  boardId: string
  boardName: string
  colIdx: number       // NEW — needed for BoardFocusContext.setFocused
  cardIdx: number
  cardTitle: string
  snippet: string
}
```

Server side: index docs gain `col_idx`. Adjust `Doc` struct + `UpdateBoard` loop + `Search` field projection accordingly.

## `useSearch` hook

```ts
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

## CommandPalette integration

```tsx
const [query, setQuery] = useState('')
const hits = useSearch(query)
const { setFocused } = useBoardFocus()

<Command label="Command palette" shouldFilter={true}>
  <Command.Input value={query} onValueChange={setQuery} placeholder="Type a command, board, or card…" />
  <Command.List>
    <Command.Empty>No matches.</Command.Empty>
    {/* Boards group (existing) */}
    {/* Actions group (existing) */}
    {hits.length > 0 && (
      <Command.Group heading="Cards">
        {hits.map((h) => (
          <Command.Item
            key={`${h.boardId}:${h.colIdx}:${h.cardIdx}`}
            value={`card ${h.cardTitle} ${query}`}  // ensure cmdk doesn't filter out
            onSelect={() => onSelectCard(h)}
          >
            <span className="font-semibold">{h.cardTitle}</span>
            <span className="ml-2 text-xs text-slate-400">in {h.boardName}</span>
            {h.snippet && (
              <span
                className="block text-xs text-slate-500"
                dangerouslySetInnerHTML={{ __html: sanitize(h.snippet) }}
              />
            )}
          </Command.Item>
        ))}
      </Command.Group>
    )}
  </Command.List>
</Command>
```

`onSelectCard`:
```ts
const onSelectCard = (h: SearchHit) => {
  setActive(h.boardId)
  // Defer focus until the next render, after the new board mounts.
  Promise.resolve().then(() => setFocused({ colIdx: h.colIdx, cardIdx: h.cardIdx }))
  close()  // existing palette close handler
}
```

`sanitize` is imported from `markdownPreview.ts` (P6.1).

`Command.Input` becoming controlled means cmdk loses some of its own state — verify `value`/`onValueChange` work as expected with cmdk v1.0+. Keep `shouldFilter` so client-side board+action filtering still works.

## Testing

### Go

- `internal/search/index_test.go`:
  - `Build` over a 2-board fixture from a temp workspace; `Search("hello")` finds the matching card.
  - `UpdateBoard` replacing card text → old text no longer matches, new does.
  - `DeleteBoard` → no hits from that slug.
  - Snippet contains `<mark>`.
  - Empty query → empty result (or handler-level guard).

- `internal/api/v1/search_test.go`:
  - `GET /search?q=hello` → 200 + JSON array.
  - `GET /search?q=` → 200 + `[]`.
  - `GET /search` (no `q`) → 200 + `[]`.
  - `q` length > 256 → 400 INVALID.
  - Mutation via `/mutations` then immediate search reflects the new card.
  - `DELETE /boards/{slug}` then search returns no hits from that board.

### TS

- `ServerAdapter.search` test: mock fetch with hits → assert URL/params + camelCase mapping.
- `LocalAdapter.search` test: substring match across a small in-memory workspace; verify (colIdx, cardIdx) accuracy.
- `CommandPalette.test.tsx` (extension): mount with a Local adapter pre-seeded with cards; type a query; assert "Cards" group renders.

## Visual

- Cards group heading: same uppercase grey style as Boards/Actions.
- Hit row: bold card title + faint "in <board name>" + dimmer snippet line.
- Snippet: small text, `<mark>` tags styled inline (default browser yellow background is fine).

## Risks

- **bleve binary size**: adds ~5–10 MB to the Go binary. Acceptable for server distribution; documented.
- **In-memory index lost on restart**: full reindex on startup. For ~1000 cards: <100 ms. Above that, persistence becomes desirable — future work.
- **External `.md` edits skip the index**: documented limitation. fsnotify follow-up if real users hit it.
- **cmdk client-side filtering hides server hits**: setting `value="card <title> <query>"` ensures the typed query is in the value. Verify with the manual smoke.
- **`Command.Input` controlled value + cmdk**: cmdk v1 supports controlled `value`/`onValueChange`. If a regression appears, revert to uncontrolled and lift the query into a separate state via `useEffect` watching the input ref.
- **`<mark>` snippet sanitization**: bleve's snippet text is the user's own content. Same threat model as P6.1; same regex sanitizer applied.
- **Index updates on every mutation**: cost is small (delete + reindex one board), but all mutations now block on the index. For typical sizes this is sub-millisecond. Documented.
- **`useSearch` query key churn**: every keystroke before debounce produces a key. The 200ms debounce caps at ~5 queries/sec. TanStack Query dedupes parallel fetches.
- **LocalAdapter loads all boards on every search**: O(N boards). For typical workspaces (<50 boards) negligible. Documented.

## Open questions

None blocking. Pre-decided:
- Per-card index, in-memory, full reindex on startup, incremental on mutations.
- New `/api/v1/search` HTTP endpoint with `q` + `limit`.
- `BackendAdapter.search()` on both adapters; LocalAdapter does substring.
- Cmd+K integration as inline "Cards" group.
- Selection switches active board + sets focused card; modal auto-open deferred.
- `SearchHit` includes `colIdx` so focused is set correctly.

## Dependencies on prior work

- P5.0: `/api/v1/*` infrastructure + Deps wiring.
- P5.1: `ServerAdapter` HTTP plumbing.
- P5.2: shell adapter selection.
- P4c.2: `<CommandPalette>` exists with cmdk.
- P4c.3: `BoardFocusContext` exposes `setFocused({ colIdx, cardIdx })`.
- P6.1: `sanitize` from `markdownPreview.ts`.
