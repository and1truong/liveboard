# LiveBoard

**Markdown-native, local-first Kanban system with REST API, Web UI, TUI, CLI, and AI agent support.**

> Obsidian Kanban + Trello + Git + AI agent — without the lock-in.

-----

## Overview

LiveBoard treats plain Markdown files as Kanban boards. There is no database, no proprietary format, no sync server. Your tasks live as `.md` files in a folder you own. Everything else — the API, the UI, the TUI, the AI agent — is built on top of that single source of truth.

**Core design goals:**

- Markdown is the database
- Git-friendly by default
- Human-readable without any tooling
- REST API + Web UI + TUI + CLI — all first-class
- AI-controllable via structured tool calls

-----

## Why LiveBoard

|Feature             |LiveBoard|Obsidian Kanban|Trello  |Linear  |
|--------------------|---------|---------------|--------|--------|
|Plain Markdown files|✅        |✅              |❌       |❌       |
|Local-first         |✅        |✅              |❌       |❌       |
|Git integration     |✅        |manual         |❌       |❌       |
|REST API            |✅        |❌              |✅       |✅       |
|CLI                 |✅        |❌              |❌       |partial |
|TUI                 |✅        |❌              |❌       |❌       |
|AI agent tools      |✅        |❌              |limited |limited |
|Automatable         |✅        |❌              |webhooks|webhooks|

-----

## Board Format

A board is a single Markdown file.

```markdown
---
name: Product Roadmap
description: Planning upcoming features
tags: [product, roadmap]
---

## Backlog

- [ ] Add OAuth login
  tags: auth, backend
  priority: high

- [ ] Build mobile layout
  tags: ui

## In Progress

- [ ] Implement billing integration
<!-- liveboard:id=abc123 -->
  tags: payments
  assignee: hong

## Done

- [x] Create landing page
<!-- liveboard:id=def456 -->
```

### Column rule

```
H2 heading = column
```

Column order in the file is the display order. Columns can be renamed, reordered, and moved across boards.

### Card rule

```
List item = card
```

Extended card syntax supports inline metadata below the list item:

```markdown
- [ ] Implement billing integration
<!-- liveboard:id=abc123 -->
  tags: payments, billing
  assignee: hong
  priority: high
  due: 2025-07-01
  relates: board-infra#xyz789

  Notes go here.
  Supports full **Markdown** including `code blocks`.
```

Parsed card model:

```
Card
  id          string        (uuidv7, stored as HTML comment)
  title       string
  completed   bool
  column      string
  tags        []string
  metadata    map[string]string
  body        string        (markdown)
```

### Card ID strategy

IDs are stable `uuidv7` values stored as an HTML comment directly below the list item:

```markdown
- [ ] Fix deployment issue
<!-- liveboard:id=01j4k... -->
```

This ensures IDs survive moves, renames, and external edits. The comment is invisible when rendered.

### Tag syntax

Two equivalent forms — the system normalizes both:

```markdown
tags: backend, urgent
```

```markdown
- [ ] Fix bug #backend #urgent
```

### Cross-board card links

```markdown
- [ ] Fix deployment issue
<!-- liveboard:id=abc123 -->
  relates: board-infra#xyz789
```

Rendered as:

```
Fix deployment issue
↳ relates to: infra / deploy pipeline  [In Progress]
```

The engine tracks dependencies and blocking relationships across boards.

-----

## Workspace Layout

A workspace is any folder containing `.md` board files and an optional config directory:

```
~/boards/
    board-product.md
    board-infra.md
    board-personal.md
    .liveboard/
        config.yaml
```

Multiple workspaces are supported. The default workspace is configured globally.

-----

## Configuration

### Global config

`~/.config/liveboard/config.yaml`

```yaml
llm:
  provider: anthropic        # or: openai, ollama
  model: claude-sonnet-4-20250514

workspace:
  default: ~/boards

git:
  auto_commit: true
  commit_format: "{operation}: {detail}"
```

### Project config

`.liveboard/config.yaml` (inside workspace folder)

```yaml
board:
  default_columns:
    - Backlog
    - In Progress
    - Review
    - Done
```

-----

## CLI

```
liveboard board list
liveboard board create <name>
liveboard board delete <name>

liveboard card add <board> "<title>"
liveboard card move <id> "<column>"
liveboard card complete <id>
liveboard card tag <id> <tag> [tag...]
liveboard card link <id> <board>#<id>
liveboard card show <id>

liveboard column add <board> "<name>"
liveboard column move <board> "<name>" --after "<other>"
liveboard column delete <board> "<name>"

liveboard ai "<prompt>"
liveboard ai plan-day
liveboard ai session         # interactive agent session
```

Examples:

```sh
liveboard board create roadmap
liveboard card add roadmap "Implement OAuth login"
liveboard card move abc123 "In Progress"
liveboard card tag abc123 backend auth
liveboard ai "summarize board roadmap"
liveboard ai "which cards should I work on today?"
```

-----

## REST API

The server runs locally (default: `http://localhost:7070`).

### Boards

```
GET    /boards
POST   /boards
GET    /boards/{board}
DELETE /boards/{board}
```

### Columns

```
POST   /boards/{board}/columns
PATCH  /columns/{id}
DELETE /columns/{id}
POST   /columns/{id}/move
```

### Cards

```
POST   /columns/{id}/cards
GET    /cards/{id}
PATCH  /cards/{id}
DELETE /cards/{id}
POST   /cards/{id}/move
POST   /cards/{id}/complete
POST   /cards/{id}/tag
```

### Search

```
GET    /search?q=<query>&board=<board>&tags=<tags>
```

### Event stream

```
GET    /events          (SSE)
GET    /events/ws       (WebSocket)
```

-----

## Event Bus

All internal state changes emit structured events. The event stream is exposed externally via SSE for UI and agent consumption.

```
Event
  type        string
  board       string
  entity_id   string
  payload     map[string]any
  timestamp   time.Time
```

Event types:

```
board.created   board.updated   board.deleted
column.created  column.renamed  column.moved   column.deleted
card.created    card.updated    card.moved     card.completed
card.tagged     card.linked     card.deleted
```

-----

## Web UI

The built-in web UI is a LiveView application served by the LiveBoard server itself. No separate frontend deployment needed.

Main views:

- **Boards list** — name, description, tags, card count
- **Board view** — Kanban columns with drag-and-drop
- **Card detail** — full Markdown rendering, metadata, linked cards
- **Search** — full-text across all boards

Board view features:

```
drag card        (within column and across columns)
drag column      (reorder columns)
collapse column
tag filter
hide completed
full-text search
```

Framework: [`go-live-view`](https://github.com/go-live-view/go-live-view)

-----

## TUI

The terminal UI is read + light-edit mode. It connects to the same API as the web UI.

```
bubbletea model → API client → event stream
```

Views:

```
Boards list
Board view (columns + cards)
Card detail
Command palette
```

Keybindings:

```
j / k          navigate up/down
h / l          switch columns
enter          open card
m              move card
t              tag card
a              add card
/              search
q              quit / back
```

Framework: [`bubbletea`](https://github.com/charmbracelet/bubbletea)

-----

## AI Layer

The AI agent operates exclusively through the internal API. It never reads or writes Markdown directly.

```
User prompt
    ↓
Agent (LLM)
    ↓
Tool calls → REST API
    ↓
Markdown transform
    ↓
Git commit
```

### Agent tools

```
list_boards         → GET /boards
get_board           → GET /boards/{board}
create_card         → POST /columns/{id}/cards
move_card           → POST /cards/{id}/move
complete_card       → POST /cards/{id}/complete
tag_card            → POST /cards/{id}/tag
search_cards        → GET /search
summarize_board     → read + synthesize
find_blockers       → read + analyze links
archive_done        → batch complete + move
suggest_tags        → read + cluster
plan_day            → read + prioritize
```

### Example prompts

```
liveboard ai "summarize progress and blockers on board roadmap"
liveboard ai "which cards should I work on today?"
liveboard ai "archive completed tasks older than 7 days"
liveboard ai "group similar tasks and suggest tags"
liveboard ai "what is blocking the billing integration card?"
```

### LLM compatibility

Supports any OpenAI-compatible endpoint and the Anthropic API. Configure via `~/.config/liveboard/config.yaml`.

-----

## Git Integration

Every write operation commits the changed Markdown file with a structured message:

```
card: add "Implement OAuth login" → Backlog
card: move "billing integration" Backlog → In Progress
card: complete "Create landing page"
column: add "Review" to board-product
board: create board-infra
```

Optional features:

```yaml
git:
  auto_commit: true       # commit on every write
  branch_per_board: false # isolate board history
```

Manual commit mode is also supported for users who prefer batching changes.

-----

## Architecture

```
             ┌───────────────────────┐
             │      CLI / TUI        │
             │  liveboard card add   │
             └──────────┬────────────┘
                        │
             ┌──────────▼────────────┐
             │       API Layer       │
             │   REST + SSE + WS     │
             └──────────┬────────────┘
                        │
         ┌──────────────┼──────────────┐
         │              │              │
 ┌───────▼──────┐ ┌─────▼──────┐ ┌────▼────────┐
 │ Board Engine │ │  AI Agent  │ │  Event Bus  │
 └───────┬──────┘ └─────┬──────┘ └────┬────────┘
         │              │             │
         └──────────────┼─────────────┘
                        │
               ┌────────▼────────┐
               │ Markdown Store  │
               │  + Git layer    │
               └─────────────────┘
```

**Markdown is the single source of truth. Everything else is derived.**

-----

## Internal Modules

### `workspace/`

Folder-level management: scan boards, detect changes, resolve paths.

```
ListBoards()
LoadBoard(path)
CreateBoard(name)
DeleteBoard(name)
```

### `parser/`

Converts a board Markdown file into the structured `Board` model.

```
Markdown file
    ↓
goldmark AST
    ↓
Board / Column / Card models
```

### `writer/`

Applies mutations back to the Markdown file.

**Design rule: never rewrite the entire file.** Instead:

```
parse → transform AST → render diff
```

This preserves formatting, comments, and custom content — and keeps Git diffs clean.

### `event/`

Internal pub/sub bus (`chan Event` or lightweight pub/sub). Bridges internal state changes to the external SSE stream.

### `git/`

Wraps `go-git` to stage and commit changed board files after each write operation.

### `search/`

Full-text search index using `bleve`. Indexed fields: `title`, `body`, `tags`, `board`, `column`.

### `ai/`

LLM client + tool dispatcher. Translates agent tool calls into API requests.

-----

## Concurrency

File writes use file-level locking to prevent simultaneous edits corrupting board state. The writer also performs an optimistic hash check before committing to catch conflicts from external edits (e.g. another editor saving the file mid-operation).

-----

## Technology Stack

|Component       |Library                                                       |
|----------------|--------------------------------------------------------------|
|Language        |Go                                                            |
|Web UI framework|[`go-live-view`](https://github.com/go-live-view/go-live-view)|
|CLI             |[`cobra`](https://github.com/spf13/cobra)                     |
|TUI             |[`bubbletea`](https://github.com/charmbracelet/bubbletea)     |
|Markdown parsing|[`goldmark`](https://github.com/yuin/goldmark)                |
|File watching   |[`fsnotify`](https://github.com/fsnotify/fsnotify)            |
|Git integration |[`go-git`](https://github.com/go-git/go-git)                  |
|Search index    |[`bleve`](https://github.com/blevesearch/bleve)               |
|Event streaming |SSE (default), WebSocket (optional)                           |
|LLM             |OpenAI-compatible + Anthropic API                             |

-----

## Project Layout

```
liveboard/
    cmd/
        liveboard/
            main.go         CLI entrypoint
            serve.go        server command

    internal/
        workspace/          folder scanning, board resolution
        board/              board engine, operations
        parser/             markdown → model
        writer/             model → markdown (AST transform)
        git/                go-git integration
        api/                REST handlers
        templates/          HTML templates
        web/                web UI handlers and views

    pkg/
        models/             shared Board, Column, Card types

    web/
        css/                stylesheets

    README.md
    LICENSE
```

**Planned modules** (not yet implemented):
- `internal/event/` — internal event bus + SSE
- `internal/search/` — bleve index
- `internal/ai/` — LLM client, tool dispatcher
- `tui/` — bubbletea models and views

-----

## Roadmap

- [ ] Core parser + writer (read/write Markdown ↔ model)
- [ ] Board engine (card/column CRUD operations)
- [ ] REST API server
- [ ] CLI (`cobra`)
- [ ] Git integration
- [ ] File watcher + event bus
- [ ] Web UI (basic board view)
- [ ] TUI (`bubbletea`)
- [ ] Search index (`bleve`)
- [ ] AI agent layer
- [ ] Cross-card linking + dependency tracking
- [ ] WebSocket event stream

-----

## License

MIT
