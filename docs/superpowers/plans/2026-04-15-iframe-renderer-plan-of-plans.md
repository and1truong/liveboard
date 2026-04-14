# Iframe Renderer Architecture — Plan of Plans

> **Scope:** This is a sequencing/milestone document. Each milestone below gets its own detailed implementation plan written at the time it starts. Do not execute tasks directly from this document.

**Design spec:** `docs/superpowers/specs/2026-04-15-iframe-renderer-architecture-design.md`

**Goal:** Rebuild the LiveBoard frontend into a four-layer architecture (REST API / headless shell / iframe renderer / browser-storage adapter) without a flag day, keeping the current HTMX app live throughout.

---

## Milestone sequencing

```
P1 REST API ──────────────┐
                          ├──► P3 Shell + LocalAdapter ──► P4 Renderer ──┐
P2 Mutation vectors ──────┘                                              │
                                                                         ├──► P5 RestAdapter ──► P6 Landing demo ──► P7 Cutover
                                                                         │
```

P1 and P2 can run in parallel. P3 needs P2 (LocalAdapter consumes `boardOps.ts`). P4 needs P3 (renderer talks postMessage to shell). P5 needs P1 + P4. P6 needs P4. P7 is last.

---

## P1 — REST API

**Output:** `/api/v1/*` endpoints alongside existing web handlers, returning JSON. SSE at `/api/v1/events`. `/api/versions` probe.

**Shippable value:** External API consumers (desktop, future mobile, third-party) have a stable contract. Current HTMX app is unaffected.

**Depends on:** Nothing.

**Scope:**
- `GET /api/v1/boards` — list
- `GET /api/v1/boards/{id}` — full board JSON
- `POST /api/v1/boards/{id}/mutations` — body `{ clientVersion, op }`, returns new board; 409 on conflict
- `GET /api/v1/boards/{id}/settings`, `PUT /api/v1/boards/{id}/settings`
- `GET /api/v1/workspace`
- `GET /api/v1/events?board={id}` — SSE, forwards `board-updated`
- `GET /api/versions` — supported REST versions
- JSON shapes documented and contract-tested

**Out of scope:** Auth beyond current cookie session. Rate limiting. Any UI change.

**Done when:** All endpoints implemented, contract tests green, existing app still functional, `curl` smoke suite passes.

---

## P2 — Shared mutation vectors (Go ↔ TS parity)

**Output:** A canonical `MutationOp` schema, a shared JSON vector suite, and the `boardOps.ts` module used by LocalAdapter in P3.

**Shippable value:** The mechanism that prevents Go/TS drift forever. Without this, LocalAdapter is a time bomb.

**Depends on:** Nothing (can run in parallel with P1).

**Scope:**
- Define `MutationOp` union shape (tagged union, one variant per existing `board.Engine` mutation)
- Go types (`pkg/models/` or new package) matching the schema
- TS types in `web/shared/` or equivalent
- `boardOps.ts` — pure functions, one per op, mirroring `internal/board/board.go` semantics
- Vector suite: `testdata/mutations/*.json`, each `{ description, board_before, op, board_after, expected_error? }`
- Go test runner: loads vectors, applies op via engine, asserts
- TS test runner (vitest): same vectors, applies op via `boardOps.ts`, asserts
- CI: both runners must pass; adding a vector requires both sides green

**Out of scope:** Code generation from schema. Hand-maintain both sides; vectors are the guarantee.

**Done when:** Vector suite covers every existing mutation, both test runners pass in CI.

---

## P3 — Shell + LocalAdapter

**Output:** Headless TS shell served at `/app/` (flag-gated). postMessage broker, `BackendAdapter` interface, `LocalAdapter` backed by localStorage + BroadcastChannel. No renderer yet — a stub iframe that logs messages for testing.

**Shippable value:** Proves the adapter/protocol boundary works end-to-end without a backend.

**Depends on:** P2 (`boardOps.ts`).

**Scope:**
- Shell host HTML page at `/app/`
- TS entrypoint: build `BackendAdapter`, instantiate iframe, run postMessage broker
- `BackendAdapter` interface (exact signature from spec)
- `LocalAdapter` implementation
- Seed workspace on first load
- BroadcastChannel multi-tab sync
- Protocol handshake (hello/welcome) with version negotiation
- Stub renderer iframe that exercises every method, logs results, used as integration test harness
- Origin validation on both ends

**Out of scope:** Real renderer (P4). RestAdapter (P5).

**Done when:** Stub iframe can list boards, get a board, perform every mutation type, receive `board.updated` events across tabs, handle VERSION_CONFLICT.

---

## P4 — Renderer v1 (default kanban)

**Output:** React + TanStack Query SPA at `/renderer/default/`. Feature parity with the current HTMX app — chrome (sidebar, topbar, command palette, modals) + board surface (columns, cards, drag-drop, settings).

**Shippable value:** `/app/` becomes a real LiveBoard UI. Current HTMX at `/` still the default.

**Depends on:** P3.

**Scope summary** (detailed plan written at milestone start):
- React + TanStack Query project scaffold in `web/renderer/default/`
- Bundle pipeline (likely Vite), CI bundle-size check (≤120kb gz)
- Parity checklist driving task decomposition:
  - Board list sidebar, board switcher, board create/rename/delete
  - Board view: columns, cards, drag-drop (card + column), collapse, sort
  - Card: create, edit (modal), quick-edit (context menu), delete, complete toggle, priority, due, tags, assignee, body
  - Column: create, rename, delete, reorder, sort
  - Settings panel (global + per-board)
  - Command palette (Cmd+K)
  - Keyboard navigation
  - Theme/color theme
  - Calendar sub-view (if in scope for v1 — confirm at plan time)
- Optimistic mutations via `useMutation` with `onMutate`/`onError` rollback
- Query invalidation on `board.updated` events
- Origin-locked postMessage client

**Out of scope:** RestAdapter wiring (P5). Non-default renderers (future).

**Critical pre-work:** Before starting, capture parity acceptance criteria as explicit tests or scenarios. Subtle UX behaviors (focus, drag affordances, keyboard) must not silently regress.

**Done when:** Every item in the parity checklist passes, bundle budget met, `/app/?flag=1` is dogfoodable.

---

## P5 — RestAdapter + internal dogfood

**Output:** `RestAdapter` implementation; shell auto-selects Rest vs Local based on environment. Internal team uses `/app/` on real deployments behind a flag.

**Shippable value:** New stack proven on real data and real backend.

**Depends on:** P1, P4.

**Scope:**
- `RestAdapter` implementing `BackendAdapter` against `/api/v1/*`
- EventSource-based `subscribe` wired to `/api/v1/events`
- Auto-detection: if page is served from the Go server, use Rest; if standalone (demo), use Local
- Error mapping (HTTP status → protocol error code)
- Retry/backoff on SSE disconnect
- Flag gate kept on in production
- Bug-fix tasks slot into this phase as dogfood surfaces issues

**Done when:** Internal team uses `/app/` for a full week with no critical regressions.

---

## P6 — Landing-page demo

**Output:** Static build (shell + default renderer + LocalAdapter, no backend) deployed to the landing page. Seed board, disclaimer banner.

**Shippable value:** Prospective users can try LiveBoard with zero signup.

**Depends on:** P4.

**Scope:**
- Static build configuration (no Go server)
- Seed board(s) tuned for demo (shows features, not empty)
- Disclaimer: "Data stored in this browser only"
- Landing-page embed instructions
- Analytics hook (optional) for demo interaction

**Done when:** Landing page loads demo, full kanban interaction works, no network calls beyond initial asset load.

---

## P7 — Cutover + cleanup

**Output:** `/` serves the new shell. Old HTMX routes removed. Dead code deleted.

**Shippable value:** One frontend, not two. Maintenance cost drops.

**Depends on:** P5 (dogfood green) + P6 (demo live).

**Scope:**
- `/` now serves new shell host page
- Old HTMX routes: redirect or 410
- Delete `internal/web/` handlers, Alpine components in `web/js/`, old templates
- Delete obsolete CSS (or move what's still used into the renderer bundle)
- Update docs, CLAUDE.md, README

**Risk gate:** Do not start P7 until P5 has run a full internal dogfood window with no open critical bugs.

**Done when:** `internal/web/` is deleted, all CI green, no references to old UI paths.

---

## Decision points between phases

- **After P1:** Review the API shape with anyone who'll consume it externally. Lock v1 or iterate *before* P5 locks in a client.
- **After P2:** Confirm the vector suite has enough coverage — gaps here cause divergence later.
- **After P3:** Protocol version 1 is now a commitment. Any changes before P4 are cheap; after P4, they're a breaking change.
- **After P4:** Parity gap triage. What's missing vs. the current app? Decide: fix now, defer, or kill.
- **After P5:** Dogfood bug triage. Decide cutover readiness honestly; don't rush P7.

---

## What I need from you before writing P1's detailed plan

1. Confirm P1 is the right place to start, or whether P2 should go first (they can run in parallel, but one detailed plan at a time).
2. Confirm REST API shape is settled enough to commit to `/api/v1/` — any endpoints or response shapes you want to revisit now, before they become a contract.
3. Confirm auth scope: cookie session only, or does P1 need anything else (API tokens for external clients)?
