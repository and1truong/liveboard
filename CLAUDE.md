# LiveBoard

Markdown-powered Kanban board with real-time collaboration.

## Tech Stack

- **Backend**: Go 1.24, chi/v5 router, cobra CLI
- **Frontend**: React shell + renderer (TypeScript) mounted at `/app/`, served as embedded Vite bundles. `web/shell/` is the iframe host; `web/renderer/default/` is the UI; `web/shared/` is the backend-adapter protocol. Browser talks to Go via `/api/v1/*` JSON + SSE at `/api/v1/events`.
  - tools: use bun if we can, instead of node/npm/pnpm
- **Storage**: Markdown files with YAML frontmatter — no database
- **Dev**: `make dev` (Go + air live reload, port 7070). `make adapter-test` for renderer HMR. `make frontend` to rebuild the embedded shell+renderer bundles.

## Domain Concepts

- **Workspace**: A directory of `.md` files — the root container for all boards
- **Board**: A single `.md` file. YAML frontmatter holds metadata (name, description, icon, tags, members, settings). Versioned for optimistic concurrency control
- **Column**: An H2 heading (`##`) in the markdown. Contains cards. Can be collapsed, reordered, sorted
- **Card**: A markdown list item (`- [ ]` or `- `). Has: title, body, tags, priority (critical/high/medium/low), due date, assignee, completed state
- **Settings**: Two-tier hierarchy — global (`settings.json` in workspace) with per-board overrides (in YAML frontmatter). Includes: theme, color theme, site name, column width, sidebar position, card display mode, default columns
- **Command Palette**: Cmd+K / Ctrl+K — navigates between boards and pages
- **Parser/Writer**: Roundtrips between markdown text and Go structs (`pkg/models/models.go`)

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

**Parsing rules**: metadata lines match `^  (\w+): (.+)$` (exactly 2-space indent). Non-matching indented lines become body. HTML comments are skipped. Inline `#tags` in title are stripped after extraction.

## Architecture

- `cmd/liveboard/` — CLI entrypoint (cobra)
- `internal/api/` — chi router, middleware, shell/renderer mount, `/api/export`, legacy REST
- `internal/api/v1/` — JSON API consumed by the renderer (boards, mutations, settings, search, SSE events)
- `internal/web/` — settings persistence (`settings.go`) and SSE broker (`sse.go`); no HTTP handlers
- `internal/board/` — CRUD engine, mutex-per-board, optimistic locking
- `internal/parser/` — markdown + YAML frontmatter parsing
- `internal/writer/` — struct-to-markdown serialization
- `internal/workspace/` — directory scanning, board listing
- `internal/templates/` — Go HTML templates for static export only (`export_*.html`)
- `internal/export/` — workspace → static HTML/ZIP export
- `web/shell/`, `web/renderer/default/`, `web/shared/` — TypeScript SPA (shell iframe-hosts the renderer; shared defines the BackendAdapter protocol)
- `web/img/` — logos / icons used by desktop bundle (`make generate-icon`)
- `pkg/models/` — shared data structs

## On commit

`make lint` to check & fix for lint errors
