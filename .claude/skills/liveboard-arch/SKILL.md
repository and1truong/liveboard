---
name: liveboard-arch
description: Architecture reference for the LiveBoard codebase. Loads layer map, error paths, concurrency model, frontend wiring, and key file locations to eliminate exploration overhead.
autoTrigger: true
---

# LiveBoard Architecture Reference

## Data Flow

```
Request → chi router (internal/api/server.go)
  ├─ Web UI handlers (internal/web/) → Go HTML templates → HTMX partials
  │   └─ mutateBoard() → eng.MutateBoard() → parse/write .md → SSE publish
  └─ REST API handlers (internal/api/) → JSON
      └─ handleError() → HTTP status via errors.Is() on sentinels
```

Storage: `.md` files in workspace dir. No database.
Real-time: SSE broker (`internal/web/sse.go`) with per-board channels.
Client: HTMX swaps `#board-content` on `sse:board-update`. Alpine.js manages modals/state.

## Two Error Paths (Critical)

These are **separate** paths. Do not confuse them.

| Path | Package | Errors become | Key function |
|---|---|---|---|
| REST API | `internal/api/` | HTTP status (404/409/400/500) | `handleError()` in `helpers.go` — uses `errors.Is()` |
| Web UI | `internal/web/` | Error banner in template (`model.Error` string) | `mutateBoard()` in `board_view.go` — wraps as `BoardViewModel.Error` |

**Sentinel errors** (defined in `board/board.go` and `workspace/workspace.go`):
- `board.ErrNotFound` — column/card not found
- `board.ErrOutOfRange` — invalid column/card index
- `board.ErrVersionConflict` — optimistic locking failure
- `workspace.ErrAlreadyExists` — board name collision
- `workspace.ErrInvalidBoardName` — unsafe filename

All errors in `board.go` and `workspace.go` use `%w` wrapping with these sentinels.

## Concurrency Model

- `board.Engine` — per-board mutex via `boardLock()`. Global mutex protects only the lock map.
- `MutateBoard(path, clientVersion, fn)`:
  1. Acquire per-board lock
  2. Read board from disk (`os.ReadFile` → `parser.Parse`)
  3. If `clientVersion >= 0` and mismatches → `ErrVersionConflict`
  4. Apply `fn(board)`
  5. `board.Version++`
  6. Write to disk. On write failure → rollback `Version--`
- `clientVersion < 0` bypasses conflict check (REST API idempotent ops)
- SSE broker: `sync.RWMutex`, non-blocking publish (buffer=1, drops silently)

## Frontend Wiring

**HTMX mutations**: All `hx-post` targets swap `#board-content` via `innerHTML`.

**Version management** (`liveboard.core.js`):
- Hidden input `<input id="board-version">` holds current version
- `htmx:configRequest` auto-injects version into all HTMX requests
- 409 conflict → auto-retry once with fresh version, then show toast

**Alpine.js stores** (`liveboard.store.js`):
- `board`: slug, version, tags, members — refreshed by DOM scan after each swap
- `ui`: activeModal, isDragging, searchQuery, hideCompleted, sidebarCollapsed

**Event handling** (`liveboard.drag.js`):
- Card click, contextmenu, column menu, board title dblclick → **delegated** on `document` (survive HTMX swaps)
- Drag start/end → per-element wiring in `attach()`, re-runs after `htmx:afterSettle`

**Alpine components** (defined in `web/js/`, mounted in `layout.html`):
- `cardModal()` — full card editor dialog
- `quickEdit()` — right-click context menu + inline editor
- `columnMenu()` — column actions (sort, rename, delete, focus)
- `boardSettings()` — per-board settings panel
- `cmdPalette()` — Cmd+K navigation
- `calendarView()` — calendar sub-view

## Settings Hierarchy

```
Global (settings.json in workspace root)
  └─ Per-board override (YAML frontmatter `settings:` field, nullable *bool pointers)
      └─ resolveSettings() merges → ResolvedSettings (concrete values)
```

Key settings: ShowCheckbox, CardPosition, ExpandColumns, ViewMode, CardDisplayMode, WeekStart.

## Key Files

| Concern | File |
|---|---|
| Router, middleware | `internal/api/server.go` |
| Error → HTTP status | `internal/api/helpers.go` |
| Board mutations (web) | `internal/web/board_view.go` |
| Handler struct, templates | `internal/web/handler.go` |
| Settings (load/save/merge) | `internal/web/settings.go` |
| SSE broker | `internal/web/sse.go` |
| Board engine, locking | `internal/board/board.go` |
| Markdown → structs | `internal/parser/parser.go` |
| Structs → markdown | `internal/writer/writer.go` |
| Data models | `pkg/models/models.go` |
| Board template | `internal/templates/board_view.html` |
| Layout + all modals | `internal/templates/layout.html` |
| Card modal component | `web/js/liveboard.card-modal.js` |
| Drag + event delegation | `web/js/liveboard.drag.js` |
| Version + conflict | `web/js/liveboard.core.js` |
| Alpine stores | `web/js/liveboard.store.js` |
| Keyboard navigation | `web/js/liveboard.keyboard.js` |
| Reminders engine | `internal/reminder/` |
| Reminder handlers | `internal/web/reminder_handler.go` |

## Parser/Writer Contract

- **Columns**: H2 headings (`## Name`)
- **Cards**: `- [ ] Title` (checkbox) or `- Title` (plain, `NoCheckbox: true`)
- **Metadata**: exactly 2-space indented `key: value` lines under a card
- **Body**: 2-space indented non-metadata lines
- **Inline tags**: `#tag` in card title → extracted to `InlineTags`, merged into `Tags`
- **Roundtrip**: `Parse(md) → Board → Render(board) → md` must be stable (tested)
- **Metadata order**: alphabetically sorted keys (deterministic output)
