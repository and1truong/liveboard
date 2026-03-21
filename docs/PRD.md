# LiveBoard — Product Requirements Document

> Current state of the web application as of 2026-03-21.

---

## 1. Product Overview

LiveBoard is a **Markdown-native, local-first Kanban board** served as a real-time web application. All data is stored as plain `.md` files in a user-owned directory — no database, no proprietary format. The web UI provides live, WebSocket-driven interactivity with automatic Git versioning on every write.

**Core philosophy:** Markdown is the single source of truth. The UI is a view layer over files you already own.

---

## 2. Technology Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.24 |
| HTTP Router | chi/v5 |
| Real-time UI | jfyne/live (LiveView over WebSocket) |
| Templating | Go `html/template` |
| Frontend | Vanilla JS, CSS (no build tooling) |
| Git | go-git/v5 (auto-commit) |
| Config | YAML frontmatter + JSON settings |

---

## 3. Data Model

### 3.1 Board

A board is a single `.md` file with YAML frontmatter:

```markdown
---
name: Product Roadmap
description: Planning features
icon: 🚀
tags: [product, roadmap]
members: [alice, bob]
list-collapse: [false, true, false]
settings:
  show-checkbox: true
  card-position: prepend
  view-mode: board
---
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Display name |
| `description` | string | Short description |
| `icon` | string | Emoji icon |
| `tags` | []string | Board-level tags |
| `members` | []string | Assigned members |
| `list-collapse` | []bool | Per-column collapsed state |
| `settings` | object | Per-board setting overrides |

### 3.2 Column

An H2 heading (`## Column Name`) in the Markdown file. Contains an ordered list of cards.

### 3.3 Card

A list item (`- [ ] Title`) under a column heading:

```markdown
- [ ] Task title #tag1 #tag2
  tags: extra-tag
  assignee: alice
  priority: high
  due: 2025-12-31
  custom-field: value
  Multi-line body text here
```

| Field | Type | Source |
|-------|------|--------|
| `title` | string | List item text |
| `completed` | bool | Checkbox `[x]` vs `[ ]` |
| `tags` | []string | Inline `#hashtags` + `tags:` metadata |
| `assignee` | string | `assignee:` metadata |
| `priority` | string | `priority:` metadata |
| `due` | date | `due:` metadata |
| `body` | string | Indented text (non-metadata) |
| `metadata` | map | Arbitrary `key: value` pairs |

### 3.4 Settings (`settings.json`)

Global application preferences persisted in workspace root.

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `siteName` | string | `"LiveBoard"` | Displayed in sidebar and title |
| `theme` | enum | `"system"` | `system`, `dark`, `light` |
| `colorTheme` | enum | `"default"` | One of 9 color themes |
| `columnWidth` | int | `320` | Column width in px (200–800) |
| `sidebarPosition` | enum | `"left"` | `left` or `right` |
| `showCheckbox` | bool | `true` | Show card checkboxes |
| `newLineTrigger` | enum | `"enter"` | `enter` or `shift-enter` for card body newlines |
| `cardPosition` | enum | `"append"` | `append` or `prepend` new cards |
| `defaultColumns` | []string | `["not now","maybe?","done"]` | Columns for new boards |

Per-board settings (stored in frontmatter) override globals for: `showCheckbox`, `cardPosition`, `expandColumns`, `viewMode`.

---

## 4. Pages & Navigation

### 4.1 Layout Shell

All pages share a common layout:

- **Sidebar** (collapsible, position configurable left/right)
  - Board list with emoji icons — click to navigate
  - "New Board" quick-create
  - Settings link
  - About/branding footer with site name
- **Emoji picker** popup (39 built-in emojis) for setting board icons
- **Connection status indicator** (WebSocket health)
- **Theme-aware SVG favicon** generated dynamically

### 4.2 Board List (`/`)

The home page showing all boards.

| Element | Behavior |
|---------|----------|
| Board grid | Cards showing name, icon, description, card count |
| "+ New Board" button | Reveals inline form; creates board with default columns |
| Delete button | Per-board, with confirmation |
| Board card click | Navigates to `/board/{name}` |
| Empty state | Message when no boards exist |

### 4.3 Board View (`/board/{name}`)

The main workspace. Supports two view modes:

#### 4.3.1 Board Mode (Kanban)

Drag-and-drop column layout:

| Element | Behavior |
|---------|----------|
| **Column** | Vertical lane with header + card list |
| Column header | Name (editable), collapse toggle, menu (rename/delete/sort) |
| **Card** | Draggable item within or between columns |
| Card display | Checkbox (optional), title, tags as chips, metadata badges (priority/assignee/due) |
| Card checkbox | Click toggles completion state |
| Card drag | Drag between columns or reorder within column |
| Card right-click | Context menu: Edit, Move to [column], Delete |
| Card double-click | Opens quick-edit overlay |
| **Add card form** | Per-column inline form at bottom of each column |
| **Add column FAB** | Floating button to create new column |
| **Board topbar** | Board name (editable), description, tags, icon picker, settings toggle |

#### 4.3.2 Table Mode

Spreadsheet-style rows:

| Element | Behavior |
|---------|----------|
| Rows | One per card, grouped by column |
| Columns | Status, Title, Tags, Assignee, Priority, Due |

#### 4.3.3 Quick Edit Overlay

Modal for editing a card in-place:

| Field | Control |
|-------|---------|
| Title | Textarea (newline behavior configurable) |
| Tags | Chip input with autocomplete from existing board tags |
| Priority | Text input |
| Assignee | Text input |
| Due | Date input |
| Body | Textarea (multi-line) |
| Actions | Save, Cancel |

#### 4.3.4 Board Settings Panel

Per-board configuration overlay:

| Setting | Control |
|---------|---------|
| Show checkbox | Toggle (with reset-to-global) |
| Card position | Toggle append/prepend (with reset-to-global) |
| Expand all columns | Toggle (with reset-to-global) |
| View mode | Toggle board/table (with reset-to-global) |

### 4.4 Settings (`/settings`)

Global application settings page:

| Section | Settings |
|---------|----------|
| **General** | Site name |
| **Appearance** | Theme (system/dark/light), Color theme (9 options), Column width (slider), Sidebar position |
| **Board Behavior** | Show checkbox, Newline trigger, Card position |
| **New Board Defaults** | Default columns (chip input, add/remove) |

All changes are applied immediately and persisted.

---

## 5. Color Themes

9 built-in themes, each supporting dark and light modes:

| Theme | Description |
|-------|-------------|
| Default | Neutral blue-gray |
| GitHub | GitHub-inspired greens |
| GitLab | GitLab-inspired oranges |
| Emerald | Green tones |
| Rose | Pink/rose tones |
| Sunset | Warm orange/amber |
| Aqua | Teal/cyan tones |
| Graphite | Monochrome grays |
| macOS | Apple-inspired blues |

---

## 6. Real-Time Behavior

- All UI interactions go through **LiveView WebSocket** — no full page reloads
- Server pushes HTML diffs to client on state change
- **PubSub** broadcasts updates across all connected sockets (multi-tab/multi-user)
- Event flow: User action → JS event → WebSocket → Go handler → file write → Git commit → PubSub → all clients re-render

---

## 7. Drag & Drop

Implemented in vanilla JS (`drag.js`):

| Operation | Behavior |
|-----------|----------|
| Card → different column | Moves card to target column (appended) |
| Card → same column | Reorders card within column at drop position |
| Visual feedback | Drag preview, drop zone highlighting |
| Persistence | Triggers LiveView `move-card` or `reorder-card` event |

---

## 8. Git Integration

- Every write operation auto-commits if workspace is a Git repository
- Structured commit messages: `"card: add ..."`, `"card: move → Column"`, `"column: add ..."`, etc.
- Gracefully degrades — works without Git

---

## 9. Column Operations

| Operation | Behavior |
|-----------|----------|
| Create | Add new column (H2 heading) |
| Rename | Edit column name in-place |
| Delete | Remove column and all cards within it (with confirmation) |
| Collapse/Expand | Toggle column visibility; state persisted in frontmatter |
| Sort | Sort cards by: title, priority, due date, assignee |
| Reorder | Move column position relative to others |

---

## 10. Board Operations

| Operation | Behavior |
|-----------|----------|
| Create | New `.md` file with default columns from settings |
| Delete | Removes `.md` file (with confirmation) |
| Edit metadata | Update name, description, tags inline |
| Set icon | Emoji picker (39 options) |
| Per-board settings | Override global defaults for checkbox, card position, expand, view mode |

---

## 11. Persistence Model

```
workspace/
├── settings.json          # Global app settings
├── board-one.md           # Board file
├── board-two.md           # Board file
├── .git/                  # Auto-commit history
└── .liveboard/
    └── config.yaml        # Optional workspace config
```

- **No database** — all state lives in files
- **Atomic writes** — full file rewrite on each operation (load → modify → serialize → write)
- **Git history** — full audit trail of every change
- **Portable** — copy the folder, you have everything

---

## 12. Client-Side Storage

`localStorage` is used for UI-only preferences:

| Key | Purpose |
|-----|---------|
| `theme` | Dark/light/system toggle |
| `colorTheme` | Active color theme |
| `columnWidth` | Column width in px |
| `sidebarPosition` | Left/right |
| `sidebarCollapsed` | Sidebar open/closed state |

These mirror server-side settings and are applied immediately on page load to prevent flash of unstyled content.

---

## 13. Non-Functional Characteristics

| Aspect | Detail |
|--------|--------|
| **Local-first** | Runs on localhost, no external dependencies |
| **Zero build** | No npm, no webpack — plain CSS and JS |
| **Single binary** | One Go binary serves everything |
| **File-portable** | Markdown files work in any editor |
| **Git-native** | Every change is a commit |
| **Real-time** | WebSocket push, no polling |
| **Multi-tab safe** | PubSub keeps all tabs in sync |
| **Configurable** | Global + per-board settings with inheritance |
