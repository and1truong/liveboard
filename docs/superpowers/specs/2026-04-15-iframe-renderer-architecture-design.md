# Iframe Renderer Architecture

**Date:** 2026-04-15
**Status:** Design
**Scope:** Full rebuild of the LiveBoard frontend into a four-layer architecture with a REST API, a headless shell, an iframe-hosted renderer, and a browser-storage backend for landing-page demos.

## Goals

1. **Decouple UI from backend** — a stable REST contract so the main app, Wails desktop, and future mobile webviews all consume the same API.
2. **Sandbox the render layer** — the board UI runs inside an iframe so alternative renderers (calendar, timeline, third-party plugins) are first-class peers of the default kanban view.
3. **Backend-agnostic renderer** — the iframe never knows whether it's talking to the Go server or browser storage. This lets a fully-functional demo run on a landing page with zero backend.

## Non-goals

- Offline sync between localStorage and REST. The LocalAdapter exists for the landing demo only; production always uses REST.
- Plugin data store. Browser storage in v1 is used by the LocalAdapter; per-plugin storage is future work.
- Board schema migration tooling. If the board markdown schema itself changes shape, that's a separate effort.
- Multi-user realtime in the demo. BroadcastChannel gives same-browser multi-tab sync; cross-user collab requires a server.

## Architecture

Four layers:

```
┌──────────────────────────────────────────────────────────┐
│ (2) SHELL — headless TypeScript, no view framework       │
│     • postMessage broker between iframe and adapter      │
│     • Session/auth, SSE subscription, BroadcastChannel   │
│     • BackendAdapter interface                           │
│         ├─ RestAdapter   → calls (1) REST API            │
│         └─ LocalAdapter  → reads/writes localStorage     │
│                                                          │
│   ┌──────────────────────────────────────────────────┐   │
│   │ (3) IFRAME — React + TanStack Query SPA          │   │
│   │     • Owns ALL visible UI (chrome + board)       │   │
│   │     • Sidebar, topbar, command palette, modals   │   │
│   │     • Board surface: columns, cards, drag-drop   │   │
│   │     • Talks only via postMessage to parent       │   │
│   │     • Swappable: default / calendar / custom     │   │
│   └──────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────┘
                           │
           (RestAdapter)   │ HTTPS
                           ▼
┌──────────────────────────────────────────────────────────┐
│ (1) REST API — Go backend                                │
│     • JSON over HTTP, versioned (/api/v1/...)            │
│     • board.Engine (unchanged), parser, writer           │
│     • SSE endpoint for realtime                          │
└──────────────────────────────────────────────────────────┘
```

**Key properties:**
- The iframe is backend-agnostic. Same postMessage protocol regardless of adapter.
- The shell is the only layer that knows how to reach a backend. Swap adapter = swap deployment mode.
- Landing-page demo ships: shell + iframe bundle + LocalAdapter. No Go server required.
- Production ships: shell + iframe bundle + RestAdapter + Go REST server.

## postMessage protocol

JSON-RPC-style with correlation IDs plus server-push events.

```js
// Iframe → Shell (request)
{ id: "r1", kind: "request", method: "board.get",
  params: { boardId: "projects" } }

// Shell → Iframe (response)
{ id: "r1", kind: "response", ok: true, data: { /* Board JSON */ } }
{ id: "r1", kind: "response", ok: false,
  error: { code: "VERSION_CONFLICT", message: "..." } }

// Shell → Iframe (push)
{ kind: "event", type: "board.updated",
  data: { boardId: "projects", version: 42 } }

// Iframe → Shell (handshake, first message)
{ kind: "hello", protocols: [2, 1], rendererId: "default-kanban",
  rendererVersion: "1.4.0" }

// Shell → Iframe (handshake reply)
{ kind: "welcome", protocol: 2, shellVersion: "0.9.0",
  capabilities: ["realtime", "local-storage", "multi-board"] }
```

### Methods (v1)

| Method | Purpose |
|---|---|
| `board.list` | List boards in workspace |
| `board.get` | Fetch one board (full JSON) |
| `board.mutate` | Apply a mutation with clientVersion |
| `settings.get` / `settings.put` | Resolved settings for a board |
| `workspace.info` | Current workspace metadata |

### Mutations

One `board.mutate` method takes a tagged-union `MutationOp`:

```ts
{ method: "board.mutate", params: {
    boardId, clientVersion,
    op: { type: "move_card", from: [c, i], to: [c, i] } } }
```

This mirrors the existing `eng.MutateBoard(fn)` internal API and keeps the protocol surface small.

### Events (shell → iframe)

- `board.updated { boardId, version }` — iframe refetches
- `settings.updated`
- `connection.status { online: bool }`

### Error codes

`VERSION_CONFLICT`, `NOT_FOUND`, `OUT_OF_RANGE`, `INVALID`, `INTERNAL` — lifted from current Go sentinel errors.

### Origin security

- Shell validates `event.origin` against its own origin.
- Iframe validates parent origin against an allowlist baked at build time.

## Versioning

Two independent version axes.

### postMessage protocol (iframe ↔ shell)

Negotiated at handshake. Iframe advertises `protocols: [newest, ..., oldest]`; shell picks the highest mutual version and returns it in `welcome`. No match → shell returns `{ error: "PROTOCOL_UNSUPPORTED", minSupported: 1 }`.

Capabilities are orthogonal feature flags (`realtime`, `local-storage`) so the iframe doesn't infer features from the version number.

Breaking change policy: bump the protocol integer. Shell keeps N-1 handlers for one release cycle, then drops.

### REST API (shell ↔ Go backend)

URL-prefixed: `/api/v1/boards`, `/api/v1/boards/{id}/mutations`, `/api/v1/events`. Breaking changes → `/api/v2/...` with a deprecation window. `/api/versions` advertises supported versions.

The iframe never sees REST, so REST version bumps don't force an iframe rebuild.

### Board schema version

Separate and already exists (`version:` in frontmatter — optimistic locking counter). If the board schema itself changes shape, add a `schemaVersion` field with migration on load; out of scope for v1.

## Backend Adapter interface

The shell defines one interface; both adapters implement it. The iframe never sees this — the shell's postMessage broker translates iframe requests into adapter calls.

```ts
interface BackendAdapter {
  listBoards(): Promise<BoardSummary[]>
  getWorkspaceInfo(): Promise<WorkspaceInfo>
  getBoard(boardId: string): Promise<Board>
  mutateBoard(boardId: string,
              clientVersion: number,
              op: MutationOp): Promise<Board>
  getSettings(boardId: string): Promise<ResolvedSettings>
  putBoardSettings(boardId: string,
                   patch: Partial<BoardSettings>): Promise<void>
  subscribe(boardId: string): Subscription
}
```

`MutationOp` is a tagged union — one-to-one with existing Go mutations. Collapses dozens of REST endpoints into a single `POST /api/v1/boards/{id}/mutations` with `{ clientVersion, op }`.

### RestAdapter

- `getBoard` → `GET /api/v1/boards/{id}`
- `mutateBoard` → `POST /api/v1/boards/{id}/mutations` — 200 → new board; 409 → VERSION_CONFLICT; 404 → NOT_FOUND
- `subscribe` → `EventSource('/api/v1/events?board={id}')`, forwards board-updated events
- Auth: cookie-based session (shell is same-origin as the Go server)

### LocalAdapter (demo)

- Storage key: `liveboard:workspace:{name}:board:{id}` → serialized JSON
- `mutateBoard`: read → version check → apply op via shared `boardOps.ts` → bump version → write
- `subscribe` uses `BroadcastChannel('liveboard')` for same-browser multi-tab sync
- Seed data: a preloaded "welcome" board on first run
- Mutation logic lives in a shared TS module (`boardOps.ts`), not duplicated per-op

### Go/TS parity

The same mutation semantics exist in two languages:
- `internal/board/board.go` (Go, canonical)
- `web/shell/boardOps.ts` (TS, for LocalAdapter)

**Mitigation:** a shared JSON test-vector suite. Each vector: `{ board_before, op, board_after }`. Both Go unit tests and TS vitest run the same vectors. Drift is caught immediately.

## Iframe renderer (React + TanStack Query)

### Bundle contents

- `index.html` — empty skeleton with mount points
- `renderer.js` — React app, TanStack Query client, postMessage client
- `renderer.css` — styles extracted from current `web/css/`
- React + TanStack Query + dependencies bundled (no CDN; demo must work without extra network)

### Boot sequence

1. Shell creates iframe with `src="/renderer/default/index.html?boardId=projects"`
2. Iframe loads → sends `{ kind: "hello", protocols: [1] }` to `window.parent`
3. Shell responds with `{ kind: "welcome", protocol: 1, capabilities }`
4. Iframe `useQuery(['board', boardId])` → posts `board.get` → receives Board JSON → renders
5. Shell pushes `board.updated` events → iframe invalidates the query → refetches

### Data layer

TanStack Query handles caching, staleness, refetch, and optimistic mutations:

- `useQuery(['board', id])` wraps `board.get`
- `useMutation` with `onMutate` / `onError` rollback handles optimistic drag-drop
- `queryClient.invalidateQueries(['board', id])` on every `board.updated` event

### UI surfaces in the iframe

- **Chrome:** sidebar (board list), topbar (board switcher, view toggle, user menu), command palette (Cmd+K), global modals
- **Board:** columns, cards, drag-drop, column menus, quick edit, card modal
- **Settings panel:** per-board overrides

All React components. Shell has zero DOM.

### Optimistic mutation flow

1. User drags card → `useMutation.mutate(op)` called
2. `onMutate` applies op to cached board locally
3. postMessage `board.mutate` sent with `clientVersion`
4. On success → replace cache with returned board
5. On `VERSION_CONFLICT` → rollback, invalidate, TanStack Query refetches
6. On other error → rollback + toast

### Multiple renderers

Identified by URL: `/renderer/{name}/index.html`.

- `/renderer/default/` — kanban view (v1)
- `/renderer/calendar/` — alternative view, same protocol (future)
- `/renderer/custom-{slug}/` — plugin slot (future)

Switching view = shell swaps iframe `src`.

### Sandbox enforcement

- CSP: `connect-src 'none'` on the iframe — only postMessage works
- No direct localStorage access from iframe (shell brokers if needed)
- Cross-board requests rejected by the shell's scope check

## Shell (headless)

Vanilla TypeScript. No view framework. Responsibilities:

- Host the iframe element
- Instantiate the correct `BackendAdapter` based on build/runtime config
- postMessage broker: validate origin, route requests to adapter, push events to iframe
- Session/auth (cookie forwarding is automatic in the browser for same-origin REST)
- SSE subscription management and fan-out to iframe
- BroadcastChannel for LocalAdapter multi-tab sync

Expected size: a few KB of TS.

## Migration / rollout

Seven phases. The current HTMX app stays live until Phase 7.

1. **REST API (additive)** — `/api/v1/*` alongside existing web handlers. Contract tested. SSE endpoint. `/api/versions` probe.
2. **Shared mutation vectors** — define `MutationOp` schema, generate or hand-write Go + TS types, build `boardOps.ts`, vector test suite runs in both languages.
3. **Shell + LocalAdapter** — new route `/app/` serves headless shell host page. postMessage broker, LocalAdapter, seed workspace.
4. **Renderer (React + TanStack Query), default kanban only** — reach parity with current HTMX app. Feature-by-feature checklist. `/app/` opt-in behind a flag.
5. **RestAdapter** — swap LocalAdapter for RestAdapter when shell runs on the Go server. Dogfood.
6. **Landing-page demo** — static build (shell + renderer + LocalAdapter), deployed to landing page with a "data stays in this browser" disclaimer.
7. **Flip the default** — `/` serves the new shell. Delete `internal/web/` handlers, Alpine components, old JS, old templates.

### Risks

- **Go/TS mutation drift** — vector suite is the only safeguard. Must be non-skippable in CI.
- **Bundle size** — React + TanStack Query + chrome + board + dependencies has a budget. CI check ≤120kb gz for the landing demo.
- **Parity debt** — subtle UX behaviors (keyboard nav, drag affordances, focus management) are easy to miss in a rewrite. Capture as tests/specs *before* Phase 4.
- **Parallel-frontend cost** — Phases 3–6 run alongside current HTMX app. Keep Phase 4 focused; resist scope creep.

## Open questions

- Exact bundle budget for landing demo — needs a prototype measurement.
- Whether to generate TS types from Go (via a schema) or hand-maintain with vector tests. Recommendation: hand-maintain + vectors; codegen adds build complexity that pays off only with more types than we have.
- Plugin renderer discovery (v2 concern).
