# P4a — Renderer Scaffold (Read-Only Board View) — Design

## Goal

Replace the P3 stub renderer iframe with a real React SPA that:
- Lists boards in a sidebar.
- Renders the selected board read-only (columns + cards).
- Re-renders live when the shell pushes `board.updated` events.

No mutations, no drag-drop, no chrome beyond sidebar + board view. Those live in P4b/c/d.

**Shippable value:** `/app/` iframe points at a real LiveBoard UI instead of the debug harness. Every pipe from P3 (postMessage protocol, LocalAdapter, Broker, Client) is exercised by a production-shaped consumer.

## Stack

| Concern | Choice | Why |
|---|---|---|
| Framework | React 18 | Ecosystem, type story, TanStack Query assumes it |
| Server state | TanStack Query v5 | Built for async state, invalidation fits event-push model |
| Styling | Tailwind v4 | Already used by the HTMX app |
| Primitives | Radix (as needed) | Accessible, unstyled; P4a uses at most `ScrollArea`, `Separator` |
| Build | Vite 5 | HMR, ecosystem default; run via `bunx --bun vite` |
| Test runner | `bun test` | Consistent with P2/P3 |
| DOM env for tests | happy-dom via `@happy-dom/global-registrator` | Lightweight, fast |
| Component tests | `@testing-library/react` | Standard |

## Directory layout

```
web/renderer/default/
  index.html
  vite.config.ts
  tsconfig.json
  package.json           # bun workspace; link to ../../shared
  .gitignore             # dist/
  src/
    main.tsx             # mounts <App/>, creates Client + QueryClient
    App.tsx              # top-level layout
    client.ts            # Client + TanStack wiring (see below)
    queries.ts           # useBoardList, useBoard, useWorkspaceInfo
    components/
      BoardSidebar.tsx   # board list, active highlight, click to switch
      BoardView.tsx      # column grid, read-only
      Column.tsx
      Card.tsx
      EmptyState.tsx
    styles/
      tailwind.css
  dist/                  # built bundle; gitignored
```

The existing `web/shared/` (P2/P3) is the single source of types, protocol, and Client. The renderer imports from it directly — no duplication.

## Data flow

```
                    ┌─────────────────────┐
                    │  postMessage to      │
                    │  parent shell        │
                    └──────────┬───────────┘
                               │
                     Client (from P3)
                               │
        ┌──────────────────────┼──────────────────────┐
        │                      │                      │
  useBoardList()       useBoard(boardId)      useWorkspaceInfo()
   TanStack Query       TanStack Query         TanStack Query
        │                      │                      │
    BoardSidebar           BoardView                 (topbar — future)
```

On mount:
1. `main.tsx` constructs `Client(iframeTransport(origin))` and `QueryClient`.
2. `Client.ready()` awaited — renders a loading state until resolved.
3. App mounts; hooks fire, queries populate.

On board selected:
1. URL (or in-memory state) updates to the active board id.
2. `useBoard(id)` issues `client.getBoard(id)`.
3. `useEffect` calls `client.subscribe(id)`; cleanup calls `client.unsubscribe(id)`.
4. A listener registered via `client.on('board.updated', ...)` invalidates `['board', id]` when the event's boardId matches.

Version reconciliation is implicit: invalidate re-fetches the latest. No optimistic state in P4a (mutations come in P4b).

## Routing

P4a stays single-page without a router. Active board id is in `useState` at `App` level. P4c adds real routing if needed.

## Build + serve

**Build:** `bunx --bun vite build` — output under `web/renderer/default/dist/` with hashed assets and an emitted `index.html` that references them. Vite's `base` is set so assets load from `/app/renderer/default/`.

**Makefile:** new `renderer` target that runs the Vite build. `shell` target from P3 stays. A `frontend` convenience target runs both.

**Go embed:** new `web/renderer/default/embed.go` with `//go:embed all:dist`. `internal/api/server.go` extends the `/app/*` handler: if the request path starts with `/app/renderer/default/`, serve from the renderer FS; otherwise fall back to shell FS. Both are gated by `LIVEBOARD_APP_SHELL=1`.

**Shell iframe target:** the shell's `index.html` currently hardcodes `src="/app/renderer-stub/"`. Add an env-flag-driven switch so the same shell can point at either stub or default renderer. Simplest approach: shell reads a query param or a build-time define and picks the src. For P4a we default to `/app/renderer/default/` and keep the stub available at `/app/?renderer=stub` for debugging.

## Tests

**Unit / component (bun test + happy-dom + @testing-library/react):**
- `BoardSidebar.test.tsx` — renders list from mocked `useBoardList`, highlights active, fires `onSelect` on click.
- `BoardView.test.tsx` — renders columns + cards from mocked board data, handles empty-board case (EmptyState).
- `Column.test.tsx` — renders card titles, handles collapsed state (purely visual in P4a — column.cards is the list).
- `Card.test.tsx` — renders title + optional metadata (tags, priority badge).
- `queries.test.ts` — inject a stub Client with `createMemoryPair` + in-test Broker + MemoryStorage; verify `useBoard` resolves, `useBoardList` resolves, and `board.updated` invalidates the cache.

**Integration:** no Playwright. Browser smoke: load `/app/` and verify sidebar + board render. Documented as a manual step.

**Bundle size check:** CI runs `du -bs web/renderer/default/dist/assets/*.js | awk '{s+=$1} END {print s}'` and gzips; logs the number. No hard cap yet — hard gate in P4d.

## Visual spec

Minimal, for the scaffold:

- **Sidebar**: fixed 240px, left side. Board name + icon per row. Active board has a subtle background.
- **Main area**: horizontally scrolling columns. Each column is fixed 280px wide with a header showing column name + card count, then a vertical list of cards.
- **Card**: white surface, rounded, small shadow, title in semibold, tags as small pills, priority as a colored dot.
- **Theme**: light mode only for P4a. P4d adds the theme layer.

No drag affordances, no hover edit buttons, no modals. It looks like a Kanban board; you can't change anything.

## Error and empty states

- **Client handshake fails** → full-page "Couldn't connect to shell" message.
- **No boards** → sidebar shows "No boards yet" text.
- **Empty board** → `EmptyState` in the main area: "This board has no columns yet."
- **Query error** → inline banner in the main area with the error code from `ProtocolError`.

## What P4a does NOT ship

- Mutations: add/edit/delete cards, column operations, settings writes.
- Drag-and-drop.
- Modals (card detail, settings panel, command palette).
- Keyboard navigation.
- Theme switching, color themes, dark mode.
- Calendar view.
- Board creation, rename, delete.
- Optimistic UI.
- Client-side routing.

All of the above are P4b/c/d scope. Keep the surface small so the pipeline is proven before the UI volume grows.

## Open questions

1. **Should the default renderer fully replace the stub, or run alongside it?** Proposed: keep both reachable during P4a-b. Delete the stub when P4 is done and the default is dogfoodable.
2. **Does `vite build` emit the manifest we need for Go embed?** Likely yes (Vite always emits `dist/index.html` with hashed asset refs) — worth verifying at plan-write time.
3. **`bunx --bun vite` vs `bun x vite`** — pick whichever resolves reliably in CI. Plan will settle this.

## Dependencies on P3

- `Client`, `iframeTransport`, protocol types, `ProtocolError` — imported from `web/shared`.
- `/app/*` route infrastructure in `internal/api/server.go` and the embed pattern in `web/shell/embed.go` — extended, not replaced.

## Risk

- **Vite + bun workspace edge cases.** If `bun install` mis-resolves workspace deps from `../../shared`, fall back to a path alias in `vite.config.ts` resolving `@shared/*` to `web/shared/src/*`.
- **Bundle size.** React + ReactDOM + TanStack Query is roughly 55kb gz at rest. Within budget but leaves no room for careless additions. Every dep added in P4a must be justified.
- **Subscribe race.** If `client.on('board.updated')` fires before `useBoard` has a cache entry for that board, the invalidation is a no-op (fine) but the UI won't show the change until someone fetches. Mitigate by subscribing in the same `useEffect` that the query is mounted in.
