# LiveBoard

Markdown-powered Kanban board with real-time collaboration.

## Tech Stack

- **Backend**: Go 1.24, chi/v5 router, cobra CLI
- **Frontend**: HTMX + SSE (real-time), Alpine.js (client reactivity), vanilla JS (drag-and-drop, command palette)
- **Storage**: Markdown files with YAML frontmatter — no database
- **Git**: Auto-commits every mutation
- **Dev**: `make dev` (air live reload), port 7070

## Domain Concepts

- **Workspace**: A directory of `.md` files — the root container for all boards
- **Board**: A single `.md` file. YAML frontmatter holds metadata (name, description, icon, tags, members, settings). Versioned for optimistic concurrency control
- **Column**: An H2 heading (`##`) in the markdown. Contains cards. Can be collapsed, reordered, sorted
- **Card**: A markdown list item (`- [ ]` or `- `). Has: title, body, tags, priority (critical/high/medium/low), due date, assignee, completed state
- **Settings**: Two-tier hierarchy — global (`settings.json` in workspace) with per-board overrides (in YAML frontmatter). Includes: theme, color theme, site name, column width, sidebar position, card display mode, default columns
- **Command Palette**: Cmd+K / Ctrl+K — navigates between boards and pages
- **Parser/Writer**: Roundtrips between markdown text and Go structs (`pkg/models/models.go`)

## Architecture

- `cmd/liveboard/` — CLI entrypoint (cobra)
- `internal/api/` — chi router, middleware, SSE broker
- `internal/web/` — HTMX handlers, template rendering, settings
- `internal/board/` — CRUD engine, mutex-per-board, optimistic locking
- `internal/parser/` — markdown + YAML frontmatter parsing
- `internal/writer/` — struct-to-markdown serialization
- `internal/workspace/` — directory scanning, board listing
- `internal/git/` — auto-commit on mutations
- `internal/templates/` — Go HTML templates
- `web/` — static assets (JS, CSS)
- `pkg/models/` — shared data structs
