# LiveBoard

Simple personal Kanban, markdown-based.

## Tech Stack

- **Backend**: Go 1.24, chi/v5 router, cobra CLI
- **Frontend**: React shell + renderer (TS) mounted at `/app/`, served as embedded Vite bundles. `web/shell/` = iframe host; `web/renderer/default/` = UI; `web/shared/` = backend-adapter protocol. Browser → Go via `/api/v1/*` JSON + SSE at `/api/v1/events`.
  - tools: bun, not node/npm/pnpm
- **Storage**: Markdown + YAML frontmatter — no DB
- **Dev**: `make dev` (Go + air live reload, port 7070). `make adapter-test` for renderer HMR. `make frontend` rebuilds embedded shell+renderer bundles.

## Domain Concepts

- **Workspace**: dir of `.md` files — root container for boards
- **Board**: single `.md`. YAML frontmatter = metadata (name, description, icon, tags, members, settings). Versioned for optimistic concurrency
- **Column**: H2 (`##`). Holds cards. Can collapse, reorder, sort
- **Card**: markdown list item (`- [ ]` or `- `). Fields: title, body, tags, priority (critical/high/medium/low), due, assignee, completed, attachments
- **Attachment**: file on card. Blob in `<workspace>/.attachments/` keyed sha256+ext. Card carries descriptors (`{h,n,s,m}`); body can embed via `attachment:<hash>` URLs. See `docs/attachments.md`. GC via `liveboard gc`
- **Settings**: two-tier — global (`settings.json` in workspace) + per-board overrides (YAML frontmatter). Includes: theme, color theme, site name, column width, sidebar position, card display mode, default columns
- **Command Palette**: Cmd+K / Ctrl+K — nav boards + pages
- **Parser/Writer**: roundtrips markdown text ↔ Go structs (`pkg/models/models.go`)

## Board File Format

```markdown
---
version: 1                        # optimistic locking counter
name: My Board
description: optional description
icon: "🚀"                        # emoji
tags: [product, q1]
members: [alice, bob]
list-collapse: [false, false, true]  # per-column collapse state
settings:                         # per-board setting overrides
  show-checkbox: true
  card-display-mode: compact
  expand-columns: false
  view-mode: board
---

## Column Name                    # H2 = column

- [ ] Card title #inline-tag      # unchecked card; #hashtags extracted as tags
  tags: backend, api              # comma-separated; merged with inline tags
  assignee: alice
  priority: high                  # critical | high | medium | low
  due: 2026-03-25                 # YYYY-MM-DD
  custom-key: any value           # arbitrary metadata
  Body text starts here.          # 2-space indented non-metadata lines = body
  Newlines preserved.

- [x] Completed card              # [x] or [X] = done
```

**Parsing rules**: metadata lines match `^  (\w+): (.+)$` (exact 2-space indent). Non-matching indented lines = body. HTML comments skipped. Inline `#tags` in title stripped after extraction.

## Architecture

- `cmd/liveboard/` — CLI entrypoint (cobra). `liveboard gc` removes orphan attachment blobs.
- `internal/api/` — chi router, middleware, shell/renderer mount, `/api/export`
- `internal/api/v1/` — JSON API for renderer (boards, mutations, settings, search, SSE events, attachments)
- `internal/web/` — settings persistence (`settings.go`) + SSE broker (`sse.go`); no HTTP handlers
- `internal/board/` — CRUD engine, mutex-per-board, optimistic locking
- `internal/attachments/` — content-addressed blob pool, ref scanning, GC, JPEG thumbs
- `internal/parser/` — markdown + YAML frontmatter parsing
- `internal/writer/` — struct → markdown serialization
- `internal/workspace/` — dir scanning, board listing
- `internal/templates/` — Go HTML templates, static export only (`export_*.html`)
- `internal/export/` — workspace → static HTML/ZIP export (bundles `.attachments/` default; `?attachments=false` opts out)
- `web/shell/`, `web/renderer/default/`, `web/shared/` — TS SPA (shell iframe-hosts renderer; shared = BackendAdapter protocol)
- `web/img/` — logos / icons for desktop bundle (`make generate-icon`)
- `pkg/models/` — shared data structs

## On commit

`make lint` to check + fix lint errors
