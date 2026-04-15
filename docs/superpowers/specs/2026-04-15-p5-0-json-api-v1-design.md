# P5.0 — JSON API v1 (Go) — Design

## Goal

Expose the `BackendAdapter` contract as a clean JSON HTTP + SSE surface under `/api/v1/*`, so a future TypeScript `ServerAdapter` (P5.1) can back the React renderer with the filesystem-backed Go engine instead of the browser's `LocalAdapter`.

**Shippable value:** the Go server publishes a JSON mirror of the adapter, fully tested via `httptest`. No renderer changes yet; curl / Postman can drive every verb. Unblocks P5.1.

## Scope

**In:**
- New Go package `internal/api/v1/` with handlers for every `BackendAdapter` verb.
- New Go type `pkg/models/mutation.go` — discriminated `MutationOp` union that decodes JSON from the renderer and dispatches onto `board.Engine` methods.
- New SSE endpoint `/api/v1/events` multiplexing `board.updated`, `board.list.updated`, `settings.updated`.
- Extension to `internal/web/sse.go`: `PublishBoardList()` + a workspace-level subscribe channel.
- Routing: mount `/api/v1/*` under chi router.
- httptest coverage for each verb + error codes + SSE delivery.

**Out:**
- TypeScript `ServerAdapter` (P5.1).
- Shell toggle between Local/Server adapter (P5.2).
- Auth (documented out of scope).
- CORS (same-origin so N/A).
- OpenAPI / Swagger docs — scoped to P5.3 or later.
- Rate limiting.

## Architecture

```
chi router
 ├─ / (existing HTMX UI)
 ├─ /api/* (existing REST — unchanged)
 ├─ /api/v1/* (NEW)
 │   ├─ GET    /boards                       → listBoards
 │   ├─ POST   /boards                       → createBoard
 │   ├─ GET    /boards/{slug}                → getBoard
 │   ├─ PATCH  /boards/{slug}                → renameBoard
 │   ├─ DELETE /boards/{slug}                → deleteBoard
 │   ├─ POST   /boards/{slug}/mutate         → mutateBoard(clientVersion, op)
 │   ├─ GET    /boards/{slug}/settings       → getSettings
 │   ├─ PUT    /boards/{slug}/settings       → putBoardSettings
 │   ├─ GET    /workspace                    → workspaceInfo
 │   └─ GET    /events                       → SSE multiplex
 └─ /app/*  (existing shell)
```

Handlers delegate to existing plumbing:
- `board.Engine.MutateBoard` / `GetBoard` / etc.
- `workspace.Scanner` for list + create/rename/delete (the existing HTMX layer uses these; P5.0 reuses).
- `web.SSEBroker` for per-board events, extended with workspace-level `BoardList` channel.

No business logic duplicated. The handlers are thin.

## Wire shapes

### Content type

Request + response: `Content-Type: application/json`. SSE stream: `text/event-stream`.

### Field naming

`snake_case` in JSON, matching the existing `pkg/models.Board` JSON tags. `BoardSummary` shape:
```json
{ "id": "welcome", "name": "Welcome", "icon": "🚀", "version": 3 }
```

### Error envelope

Non-2xx responses share a single shape:
```json
{ "error": { "code": "VERSION_CONFLICT", "message": "expected v5, have v6" } }
```

Status mapping (same as existing `handleError`):

| Code | HTTP |
|---|---|
| `NOT_FOUND` | 404 |
| `ALREADY_EXISTS` | 409 |
| `VERSION_CONFLICT` | 409 |
| `INVALID` | 400 |
| `OUT_OF_RANGE` | 400 |
| `INTERNAL` | 500 |

Decode errors (malformed JSON) → 400 with `code: INVALID`.

### Endpoints

**GET `/api/v1/boards`** → `200` + `[]BoardSummary`.

**POST `/api/v1/boards`** body `{ "name": string }` → `201` + `BoardSummary`. Errors: `INVALID` (empty/unslugifiable), `ALREADY_EXISTS`.

**GET `/api/v1/boards/{slug}`** → `200` + full `Board` JSON. Errors: `NOT_FOUND`.

**PATCH `/api/v1/boards/{slug}`** body `{ "new_name": string }` → `200` + updated `BoardSummary` (may have new id). Errors: `NOT_FOUND`, `INVALID`, `ALREADY_EXISTS`.

**DELETE `/api/v1/boards/{slug}`** → `204`. Errors: `NOT_FOUND`.

**POST `/api/v1/boards/{slug}/mutate`** body `{ "client_version": number, "op": MutationOp }` → `200` + updated `Board`. Errors: `NOT_FOUND`, `VERSION_CONFLICT`, `OUT_OF_RANGE`, `INVALID`, `INTERNAL`.

**GET `/api/v1/boards/{slug}/settings`** → `200` + `ResolvedSettings`. Errors: `NOT_FOUND`.

**PUT `/api/v1/boards/{slug}/settings`** body = partial `BoardSettings` (snake_case keys) → `204`. Errors: `NOT_FOUND`, `INVALID`.

**GET `/api/v1/workspace`** → `200` + `{ "name": string, "board_count": number }`.

### SSE `/api/v1/events`

Headers:
```
Content-Type: text/event-stream
Cache-Control: no-cache
```

Keepalive: a comment line `: keepalive\n\n` every 30s.

Event frames:
```
event: board.updated
data: {"board_id":"welcome","version":7}

event: board.list.updated
data: null

event: settings.updated
data: {"board_id":"welcome"}
```

A `retry: 5000\n` header is sent once at connection open so browsers' `EventSource` reconnects after 5s on drop.

## `MutationOp` Go type

`pkg/models/mutation.go` defines the discriminated union. JSON decoder inspects `type`, decodes into the right struct, and wraps them behind a common interface:

```go
type MutationOp interface {
    Apply(*Board) error // delegates to package-level ops in internal/board
}

type AddCardOp struct {
    Column string `json:"column"`
    Title  string `json:"title"`
    Prepend bool   `json:"prepend,omitempty"`
}
func (o AddCardOp) Apply(b *Board) error { ... }

// ...one struct per TS union variant...

type opEnvelope struct {
    Type   string          `json:"type"`
    // All fields flattened here for decode; the typed struct is picked from Type.
}

func DecodeMutationOp(raw json.RawMessage) (MutationOp, error) {
    var t struct { Type string `json:"type"` }
    if err := json.Unmarshal(raw, &t); err != nil { return nil, wrap("INVALID", err) }
    switch t.Type {
    case "add_card":          var o AddCardOp;          json.Unmarshal(raw, &o); return o, nil
    // ... all other variants
    default:
        return nil, fmt.Errorf("unknown op type %q", t.Type)
    }
}
```

Unknown / missing `type` → `INVALID`.

`Apply` methods wrap the existing methods on `board.Engine` / operations in `internal/board`. Where the Go engine already has `AddCard(board, col, title)`, the variant `Apply` just calls through. The existing TS `applyOp` is the source of truth for semantics; Go mirrors.

Each TS variant listed in `web/shared/src/types.ts` gets one Go struct:
`add_card`, `move_card`, `reorder_card`, `edit_card`, `delete_card`, `complete_card`, `tag_card`, `add_column`, `rename_column`, `delete_column`, `move_column`, `sort_column`, `toggle_column_collapse`, `update_board_meta`, `update_board_members`, `update_board_icon`, `update_board_settings`.

Any that the Go engine doesn't support yet return `INVALID` with a clear message — they're feature flags to close later, not silent no-ops.

## SSE broker extension

`internal/web/sse.go` currently tracks per-board subscribers. Add:

```go
// Workspace-level (non-board-scoped) events.
boardListSubs map[chan Event]struct{}
boardListMu   sync.RWMutex

func (b *Broker) PublishBoardList() {
    b.boardListMu.RLock()
    defer b.boardListMu.RUnlock()
    for ch := range b.boardListSubs { select { case ch <- Event{Type: "board.list.updated"}: default: } }
}

func (b *Broker) SubscribeBoardList() (ch chan Event, cancel func())
```

Callers: `createBoard`, `renameBoard`, `deleteBoard` handlers in v1 package invoke `broker.PublishBoardList()` after success.

For the `settings.updated` event (currently absent), also add:

```go
func (b *Broker) PublishSettings(boardID string)
```
called by `putBoardSettings` handler. Subscribers for `settings.updated` share the workspace channel (matches the adapter contract: settings events aren't board-scoped for subscription purposes, though the payload identifies the board).

### `/api/v1/events` handler

```go
func (h *V1Handler) events(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    flusher := w.(http.Flusher)
    fmt.Fprintf(w, "retry: 5000\n\n")
    flusher.Flush()

    listCh, cancelList := h.broker.SubscribeBoardList()
    boardCh, cancelBoards := h.broker.SubscribeAllBoards() // new: fan-in for all per-board events
    defer cancelList()
    defer cancelBoards()

    keepalive := time.NewTicker(30 * time.Second)
    defer keepalive.Stop()

    for {
        select {
        case <-r.Context().Done():
            return
        case <-keepalive.C:
            fmt.Fprintf(w, ": keepalive\n\n")
            flusher.Flush()
        case ev := <-listCh:
            writeSSE(w, flusher, ev.Type, ev.Data)
        case ev := <-boardCh:
            writeSSE(w, flusher, ev.Type, ev.Data)
        }
    }
}
```

`SubscribeAllBoards()` is another broker addition — a single channel that receives every per-board event. Implementation: the broker's `Publish(boardID, version)` fan-outs to both per-board subscribers AND any "all-boards" subscribers.

## Handler file layout

```
internal/api/v1/
  handler.go        // V1Handler struct, dependencies injection (engine, scanner, broker)
  routes.go         // Mount function that registers chi routes
  boards.go         // list, get, create, rename, delete, mutate
  settings.go       // getSettings, putBoardSettings
  workspace.go      // workspaceInfo
  events.go         // SSE handler
  helpers.go        // writeJSON, writeError, statusFromError, decodeJSON
  *_test.go         // httptest coverage per file
```

Route mounting in `internal/api/server.go`:
```go
r.Route("/api/v1", v1.Mount(engine, scanner, broker))
```

## Testing

Each `*_test.go` spins up an `httptest.Server` with a real `board.Engine` backed by a temp-dir workspace.

Representative cases:

**`boards_test.go`**:
- `GET /boards` on a fresh tmp dir → `[]`, 200.
- `POST /boards {name:"Foo"}` → 201 + BoardSummary with `id:"foo"`.
- `POST /boards {name:"Foo"}` twice → 409 ALREADY_EXISTS.
- `POST /boards {name:"  "}` → 400 INVALID.
- `GET /boards/foo` → 200 + full Board.
- `PATCH /boards/foo {new_name:"Bar"}` → 200 + summary with `id:"bar"`; GET foo → 404.
- `DELETE /boards/foo` → 204; GET → 404.

**`mutate_test.go`**:
- `POST /boards/foo/mutate { client_version: 1, op: { type: "add_card", column: "Todo", title: "Hi" } }` → 200 + Board with the new card and `version: 2`.
- Same with `client_version: 0` against a v1 board → 409 VERSION_CONFLICT.
- Unknown op type → 400 INVALID.
- Every other op variant → 200 happy path (parametrized test table).

**`settings_test.go`**:
- `PUT /boards/foo/settings { show_checkbox: false }` → 204; subsequent GET returns `show_checkbox: false`.
- Invalid JSON body → 400.

**`events_test.go`** (the hairiest):
- Connect an `EventSource`-like reader, block on the stream.
- Fire `POST /boards/{slug}/mutate` from another goroutine.
- Assert a `board.updated` event arrives within 1 second.
- Fire `POST /boards` → `board.list.updated` arrives.
- Disconnect the reader → broker eventually releases the subscriber (verify no goroutine leak via `runtime.NumGoroutine` before/after).

## Dependencies on prior work

- `board.Engine` — CRUD methods.
- `workspace.Scanner` — list/create/rename/delete board files on disk.
- `web.SSEBroker` — extended with `PublishBoardList`, `PublishSettings`, `SubscribeAllBoards`, `SubscribeBoardList`.
- Shared TypeScript `MutationOp` union — frozen in `web/shared/src/types.ts`; Go side mirrors.

## Risks

- **`MutationOp` type-decoder drift**: adding a TS variant without a matching Go variant returns `INVALID`. Mitigation: a contract test enumerates all TS variant type strings and asserts Go decodes each (happy-path fixture per variant).
- **SSE goroutine leaks**: every long-lived connection is a goroutine. If `SubscribeAllBoards` doesn't cancel cleanly on context cancel, leaks accumulate. Test explicitly (see `events_test.go`).
- **Concurrent workspace scanner + engine operations**: engine holds per-board mutex; workspace scanner reads the disk list. Stale list between scan and board access is fine — the access returns `NOT_FOUND`. Documented.
- **Filesystem slug vs in-memory id**: the Go engine keys boards by filename (slug). Matches the TS adapter's id model (slug = id). No id migration needed.
- **`settings.updated` semantics**: TS adapter's contract is that settings events don't include the board body. Our SSE event carries `{ board_id }` and matches. Renderer invalidates its `['settings', boardId]` query.
- **Engine has 17 ops but may not implement all of them**: if the Go `board.Engine` doesn't handle `update_board_members` / `update_board_icon` today, the `Apply` method returns `INVALID` with a clear message. Those variants fail gracefully rather than silently mutating nothing.

## Open questions

None blocking. Pre-decided:
- One `/mutate` endpoint; per-verb routes for the non-mutation CRUD.
- One `/events` SSE stream multiplexing all event types.
- No auth in P5.0.
- `MutationOp` Go type mirrors TS union; unknown variants surface as `INVALID`.
- SSE broker extended with `PublishBoardList`, `PublishSettings`, `SubscribeAllBoards`, `SubscribeBoardList`.

## Dependencies on later work

- P5.1: TS `ServerAdapter` consumes this contract.
- P5.2: shell picks `ServerAdapter` via runtime flag.
