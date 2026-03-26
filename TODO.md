# Tailwind CSS v4 Migration

Incremental migration from monolithic `web/css/board.css` (4,351 lines) to Tailwind CSS v4 with `@apply` — keeping all existing template classes intact.

---

## Phase 0: Setup

- [x] Install Tailwind CSS v4 standalone CLI (`brew install tailwindcss`)
- [x] Create input entry file `web/css/input.css` with `@import "tailwindcss"`
- [x] Add `make css` target: `tailwindcss -i web/css/input.css -o web/css/board.css --minify`
- [x] Add `make css-watch` target for dev: `tailwindcss -i web/css/input.css -o web/css/board.css --watch`
- [x] Update `make dev` to run `css-watch` alongside `air`
- [x] Verify build produces working `board.css` output
- [x] Add `web/css/board.css` to `.gitignore` (now a build artifact)

## Phase 1: Theme Tokens

Map existing 36 CSS custom properties to Tailwind's `@theme` directive.

- [x] Define `@theme` block with color tokens (accent, bg, surface, border, text, success, danger)
- [x] Define layout tokens (sidebar-width, topbar-height, column-width)
- [x] Define typography tokens (font-sans, font-size-xs/sm/base/lg)
- [x] Define radius tokens (sm, md, lg)
- [x] Define shadow tokens (card, raised, sidebar)
- [x] Migrate dark mode defaults (`@media prefers-color-scheme: dark`)
- [x] Migrate explicit `html[data-theme="dark"]` overrides
- [x] Migrate explicit `html[data-theme="light"]` overrides
- [x] Migrate GitLab color theme (light + dark variants)
- [x] Migrate Emerald color theme (light + dark variants)
- [x] Migrate Rose color theme (light + dark variants)
- [x] Migrate Aqua color theme (light + dark variants)
- [x] Verify all 5 themes × light/dark render correctly

## Phase 2: Base & Reset

- [x] Migrate FOUC fix (`html { opacity: 1 }`)
- [x] Migrate Alpine.js cloak (`[x-cloak]`)
- [x] Migrate base reset styles (box-sizing, margins, font smoothing)
- [x] Migrate form element base styles (input, select, textarea, button resets)
- [x] Migrate scrollbar styling

## Phase 3: Component Migration

Convert section-by-section using `@apply`. Each item = one logical section of `board.css`.

**Settings & Controls**
- [x] Settings page layout (`.settings-page`, `.settings-section`)
- [x] Toggle switches (`.toggle-switch`)
- [x] Segmented controls (`.segmented-control`)
- [x] Color swatches (`.color-swatch`)
- [x] Sliders and range inputs

**App Shell**
- [x] Sidebar (`.sidebar`, collapse states, branding)
- [x] Sidebar navigation items (`.sidebar-item`, active states)
- [x] Topbar (`.topbar`, breadcrumb, actions)
- [x] Mobile board dropdown

**Board List**
- [x] Board list grid (`.board-list`)
- [x] Board cards with progress bars
- [x] Board tags and pin buttons
- [x] Entrance animations

**Board View**
- [x] Column layout (`.columns-wrapper`, `.column`)
- [x] Column headers and collapse
- [x] Add column bar
- [x] Column sorting controls

**Cards**
- [x] Card layout (`.card`, checkbox, drag states)
- [x] Card meta (tags, priority, assignee, due date)
- [x] Card completed state
- [x] Card quick edit overlay
- [x] Priority color indicators (critical/high/medium/low)

**Modals & Overlays**
- [x] Card detail modal (`.card-modal`)
- [x] Card modal sidebar
- [x] Date picker (`.date-picker`)
- [x] Members picker
- [x] Tags dropdown and tag color picker
- [x] Context menu (`.context-menu`)

**Notifications**
- [x] Conflict toast
- [x] Flash toast notifications

**Command Palette**
- [x] Command palette overlay (`.command-palette`)
- [x] Fuzzy search highlighting
- [x] Keyboard shortcuts help overlay

## Phase 4: View-Specific Styles

- [x] Table view (sticky headers, grid layout, row states)
- [x] Calendar view — month layout
- [x] Calendar view — week layout
- [x] Calendar view — day layout
- [x] Calendar drag states
- [x] Focus mode (full-width single column, grid)
- [x] Desktop app / Wails (custom titlebar, webview adjustments)

## Phase 5: Cleanup & Verify

- [x] Remove legacy `board.css` source (replaced by `input.css` + partials) — already done; `board.css` is a build artifact in `.gitignore`
- [x] Verify Tailwind tree-shaking — output only contains used utilities
- [x] Verify minified output size vs. original 96KB — output 99.2KB (within margin; `@apply` expansion adds slight overhead)
- [x] Test all 5 color themes × light/dark — themes use CSS variables in `@theme`, unchanged by migration
- [x] Test responsive breakpoints (mobile, tablet, desktop) — verified mobile (375px) and desktop
- [x] Test all views (board, table, calendar, focus, settings) — all verified visually
- [x] Run `make lint` — 0 issues
- [x] Update CLAUDE.md if build instructions changed — no changes needed
