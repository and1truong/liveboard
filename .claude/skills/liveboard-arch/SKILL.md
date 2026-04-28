---
name: liveboard-arch
description: Architecture reference for the LiveBoard codebase. Loads the React-shell/renderer + JSON-API layer map, error codes, concurrency model, mutation dispatch, and key file locations to eliminate exploration overhead.
autoTrigger: true
---

# LiveBoard Architecture Reference

## Data Flow

```
Browser (/app/)
  └─ web/shell (Vite SPA)          ← hosts iframe, owns URL + config
      └─ postMessage transport     ← iframe ↔ shell RPC
          └─ web/shared/Broker     ← protocol dispatcher (v1)
              └─ BackendAdapter    ← LocalAdapter | ServerAdapter
                  └─ HTTP JSON     ← /api/v1/*  +  SSE /api/v1/events

Go server
  chi router (internal/api/server.go)
    ├─ GET /app/*                  ← embedded Vite bundles (shell + renderer)
    │    (dev: proxied to Vite via LIVEBOARD_{SHELL,RENDERER}_DEV_URL)
    ├─ /api/v1/*                   ← internal/api/v1 — JSON API
    ├─ /mcp                        ← internal/mcp — Streamable HTTP
    ├─ /api/export                 ← internal/export — ZIP (html | md)
    └─ GET /                       ← 302 /app/
```

Storage: `.md` files in workspace dir. No database.
Realtime: `internal/web/SSEBroker` with per-board, workspace-list, and all-boards fan-out channels.

## JSON Error Shape

All JSON responses use a single shape: `{error, code, status}`. Mapping is `writeError()` in `internal/api/v1/helpers.go`. Codes: `NOT_FOUND`, `VERSION_CONFLICT`, `ALREADY_EXISTS`, `OUT_OF_RANGE`, `INVALID`, `INTERNAL`. Adapter/shell code (`web/shared/src/adapters/server.ts`) reads the `code` field.

**Sentinel errors** (source of truth):
- `board.ErrNotFound`, `board.ErrOutOfRange`, `board.ErrVersionConflict`, `board.ErrInvalidInput`, `board.ErrPartialSourceCleanup`
- `workspace.ErrAlreadyExists`, `workspace.ErrInvalidBoardName`
- `v1.errInvalid` — generic bad-request sentinel internal to the v1 layer

All board/workspace errors wrap sentinels with `%w`.

## Concurrency Model

- `board.Engine` — per-board `*sync.Mutex` stored in a `sync.Map` (`boardLock()`).
- `MutateBoard(path, clientVersion, fn)`:
  1. Acquire per-board lock.
  2. `LoadBoard` → `os.ReadFile` → `parser.Parse`.
  3. If `clientVersion >= 0` and mismatches → `ErrVersionConflict`.
  4. `fn(board)` mutates in place.
  5. `board.Version++`, render, write.
  6. On write failure → `board.Version--` rollback.
- `clientVersion < 0` bypasses the conflict check (idempotent / server-driven ops).
- `MoveCardToBoard` is two-phase, not atomic across boards: dst written first (version bypass), then src (version-checked); partial-success returns `ErrPartialSourceCleanup`.
- SSE broker: `sync.RWMutex`, non-blocking publish, buffered channels (1 for per-board, 8 for board-list, 32 for all-boards, 4 for global). Drops silently when buffer is full.
- `SSEBroker.Shutdown()` closes all subscriber channels — must run before `http.Server.Shutdown` to unblock long-lived streams.

## v1 Mutation Dispatch

`POST /api/v1/boards/mutate/{boardId}` is the single write endpoint.

Body: `{ client_version: int, op: MutationOp }` where `MutationOp` is a tagged union with `type` discriminator and 18 variants (`add_card`, `move_card`, `reorder_card`, `edit_card`, `delete_card`, `complete_card`, `tag_card`, `add_column`, `rename_column`, `delete_column`, `move_column`, `sort_column`, `toggle_column_collapse`, `update_board_meta`, `update_board_members`, `update_board_icon`, `update_board_settings`, `move_card_to_board`).

Variants are defined in a single registry — `mutationRegistry` in `internal/api/v1/mutation_registry.go` — which `MarshalJSON`, `UnmarshalJSON`, and `Apply` all dispatch through. Adding a variant: add a typed pointer field to `MutationOp` plus one registry entry.

Tag colors are NOT a mutation — they live in `AppSettings` (workspace-level) and are written via `PUT /api/v1/settings`.

Flow:
1. `postMutation` (`internal/api/v1/mutations.go`) decodes request.
2. `Dispatch(eng, path, clientVersion, op)` wraps a single `MutateBoard` call.
3. `Apply(b, op)` (pure, no IO/locking) looks up the registry and routes to `board.Apply*` functions in `internal/board/board.go`.
4. On success: `SSE.Publish(slug)`, `Search.UpdateBoard(slug, updated)`, return mutated board JSON.

`move_card_to_board` is special-cased: the handler calls `Engine.MoveCardToBoard` directly (two-phase cross-board write) and fans out SSE to both src and dst.

The TS side mirrors this contract in `web/shared/src/types.ts` (`MutationOp`) and `web/shared/src/boardOps.ts` (local-adapter equivalent of `Apply`).

## Frontend Wiring

**Three packages, three roles:**

| Package | Role |
|---|---|
| `web/shell/` | Vite SPA mounted at `/app/`. Owns browser URL, reads `__LIVEBOARD_CONFIG__`, picks adapter, hosts renderer iframe, runs `Broker`. |
| `web/renderer/default/` | React app loaded in iframe at `/app/renderer/default/`. TanStack Query + custom `Client` class over postMessage. |
| `web/shared/` | Non-package TS consumed as relative imports. Defines `BackendAdapter`, protocol, transports, `Broker`, `Client`, `boardOps`, types. |

**Config injection**: `server.go` swaps the marker `/*__LIVEBOARD_CONFIG__*/ { adapter: 'local' }` in the shell `index.html` for `{ adapter: 'server', baseUrl: '/api/v1' }` at serve time (both embedded and Vite-dev-proxy paths). Locally-built bundles ship the `'local'` default so they work offline as a static page.

**Adapters** (`web/shared/src/adapters/`):
- `LocalAdapter` — browser-storage persistence, for standalone/offline use. Applies mutations via `boardOps.ts`.
- `ServerAdapter` — HTTP client for `/api/v1/*`, EventSource for `/api/v1/events`.

**Protocol** (`web/shared/src/protocol.ts`) — postMessage wire format:
- `PROTOCOL_VERSION = 1`. Hello/welcome handshake. Request/response correlate by `id`. Events are one-way (shell → iframe).
- Request methods: `board.list`, `board.listLite`, `board.get`, `board.mutate`, `board.create`, `board.rename`, `board.delete`, `board.pin`, `workspace.info`, `workspace.exportUrl`, `settings.get`, `settings.put`, `appSettings.get`, `appSettings.put`, `subscribe`, `unsubscribe`, `folder.{list,create,rename,delete}`, `search`, `backlinks`.
- Event types: `board.updated`, `settings.updated`, `connection.status`, `board.list.updated`, `active.changed`, `active.set`, `title.changed`, `key.forward`.
- Error codes: `NOT_FOUND`, `OUT_OF_RANGE`, `INVALID`, `ALREADY_EXISTS`, `INTERNAL`, `VERSION_CONFLICT`, `PROTOCOL_UNSUPPORTED`.

**Renderer URL conventions**: `/app/b/{boardId}` → board; `/f/{column}` suffix → focused column; `/c/{colIdx}-{cardIdx}` suffix → deep-linked card. Board id may contain a single `/` (folder/stem). Shell parses with `BOARD_PATH_RE` in `shell/src/main.ts`.

**Realtime**: shell's `ServerAdapter` subscribes to `/api/v1/events` once; Broker fans `board.updated` events to the renderer iframe. Renderer invalidates React Query cache keys (`['board', id]`, `['boards']`).

## Settings Hierarchy

```
AppSettings (settings.json in workspace root)
  └─ BoardSettings (YAML frontmatter, nullable *bool / *string pointers)
      └─ resolveSettings() → ResolvedSettings (concrete values)
```

`internal/web/settings.go` owns load/save/merge. `ResolvedSettings` is mirrored in `web/shared/src/adapter.ts`. Key fields: `ShowCheckbox`, `NewLineTrigger`, `CardPosition`, `ExpandColumns`, `ViewMode`, `CardDisplayMode`, `WeekStart`.

## Parser/Writer Contract

- **Columns**: H2 headings (`## Name`).
- **Cards**: `- [ ] Title` (checkbox) or `- Title` (`NoCheckbox: true`).
- **Metadata**: exactly 2-space indented `key: value` under a card.
- **Body**: 2-space indented non-metadata lines.
- **Inline tags**: `#tag` in card title → extracted to `InlineTags`, merged into `Tags`.
- **Roundtrip**: `Parse → Render → Parse` is stable (tested).
- **Metadata order**: alphabetically sorted keys (deterministic output).

## Key Files

| Concern | File |
|---|---|
| Router, shell mount, config injection | `internal/api/server.go` |
| v1 JSON API router | `internal/api/v1/router.go` |
| v1 error mapping (`code` + status) | `internal/api/v1/helpers.go` |
| v1 mutation dispatcher + `MutationOp` tagged union | `internal/api/v1/mutations.go` |
| v1 workspace-wide SSE stream | `internal/api/v1/events.go` |
| v1 board read/create/rename/delete | `internal/api/v1/boards.go` |
| v1 settings endpoints | `internal/api/v1/settings.go`, `app_settings.go` |
| v1 search + backlinks | `internal/api/v1/search.go` |
| Board engine, mutex, MutateBoard, Apply* | `internal/board/board.go` |
| Settings load/save/merge | `internal/web/settings.go` |
| SSE broker (per-board, board-list, all-boards, global) | `internal/web/sse.go` |
| Workspace directory scanning + board paths | `internal/workspace/workspace.go` |
| Markdown → structs | `internal/parser/parser.go` |
| Structs → markdown | `internal/writer/writer.go` |
| Static HTML/ZIP export | `internal/export/` |
| MCP Streamable-HTTP server + tools | `internal/mcp/server.go`, `tools_*.go` |
| Bleve per-card search index | `internal/search/index.go` |
| Parity vector runner (TS ↔ Go `Apply` equivalence) | `internal/parity/runner_test.go` |
| Go HTML templates (export only) | `internal/templates/export_*.html` |
| Data models | `pkg/models/models.go` |
| Shell entrypoint, URL parser, adapter selection | `web/shell/src/main.ts` |
| Shell embed FS | `web/shell/embed.go` |
| Renderer React app | `web/renderer/default/src/App.tsx` |
| Renderer transport + React Query | `web/renderer/default/src/client.ts` |
| Renderer embed FS | `web/renderer/default/embed.go` |
| BackendAdapter interface + types | `web/shared/src/adapter.ts` |
| postMessage protocol (v1) | `web/shared/src/protocol.ts` |
| Broker (shell side) | `web/shared/src/broker.ts` |
| Client (renderer side) | `web/shared/src/client.ts` |
| Local adapter mutation engine (TS mirror of `board.Apply*`) | `web/shared/src/boardOps.ts` |
| Server adapter (HTTP + EventSource) | `web/shared/src/adapters/server.ts` |
| Local adapter (browser storage) | `web/shared/src/adapters/local.ts` |

## Dev Loop

- `make dev` — Go + air on :7070, serves embedded shell/renderer bundles.
- `make adapter-test` — runs Vite dev servers for shell and renderer; Go proxies `/app/*` to them via `LIVEBOARD_SHELL_DEV_URL` / `LIVEBOARD_RENDERER_DEV_URL` for HMR.
- `make frontend` — rebuilds embedded bundles.
- `make lint` — runs on commit.
- Bun is preferred over npm/pnpm (see project CLAUDE.md).
