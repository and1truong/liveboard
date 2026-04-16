# Renderer Focus Mode — Design

## Summary

Port the HTMX board's "Focus" feature to the React renderer. Focus mode lets
the user zoom in on a single column: other columns are hidden, the focused
column spans the full width of the board area, and cards are laid out in a
responsive CSS grid rather than a single vertical list.

Behaviour mirrors the HTMX implementation exactly (per user decision):
ephemeral state, triggered from the column's `⋮` menu, exited via a top bar
or `Esc`, and cleared when the active board changes.

## Background

The HTMX board (`internal/templates/board_view.html`, `web/css/focus-mode.css`,
`web/js/liveboard.column-menu.js`) exposes focus mode through an Alpine store
`focusedColumn` (string; `""` means off):

- Column menu `⋮` → **Focus** sets `$store.ui.focusedColumn = columnName`.
- Container gets `.focus-mode`; other columns are hidden via `x-show`.
- The focused column's `.cards` list gets `.focus-grid`, switching to
  `grid-template-columns: repeat(auto-fill, minmax(280px, 1fr))`.
- A `.focus-exit-bar` with "Focusing: {name}" and an Esc button is rendered.
- `Esc` at the layout level clears the store.
- Board switch (boosted nav) clears the store in `liveboard.core.js`.

The React renderer (`web/renderer/default/`) has no equivalent. Its existing
`BoardFocusContext` is unrelated: it tracks which *card* has keyboard focus
for `j/k` / arrow navigation — not the same concept.

## Goals

- 1:1 port of HTMX focus mode behaviour into the renderer.
- No new persistence layer (state stays in memory, dies on reload).
- No breaking changes to existing DnD, keyboard nav, filter, or hide-completed.

## Non-goals

- Persisting focused column across reload or in the board settings file.
- URL-level focus (e.g. `?focus=Todo`).
- Multi-column focus (focusing N > 1 columns).
- Changing the card layout outside focus mode.

## Architecture

### New context: `FocusedColumnContext`

`web/renderer/default/src/contexts/FocusedColumnContext.tsx`

```ts
interface FocusedColumnCtx {
  focused: string | null          // column name, or null when off
  setFocused: (name: string | null) => void
}
```

Provider responsibilities:

1. Hold `useState<string | null>(null)`.
2. Reset to `null` when the active board changes (effect on `active` from
   `useActiveBoard`).
3. Reset to `null` when the current `focused` name is no longer present in
   the column list (handles rename/delete).
4. Register a window `keydown` listener while `focused !== null`:
   `Escape` → `setFocused(null)`. Skipped when `document.activeElement` is an
   input/textarea/contenteditable, and skipped when any Radix dialog is open
   (check via `document.querySelector('[role="dialog"][data-state="open"]')`).

Hook: `useFocusedColumn(): FocusedColumnCtx`.

### `BoardView.tsx` changes

- Wrap the board subtree with `<FocusedColumnProvider columns={columns}>`.
- Inside the columns container, read `focused` via hook.
- If `focused !== null`:
  - Render `<FocusExitBar name={focused} />` above the columns row.
  - Filter `columns` to the single matching column (by name).
  - Skip the horizontal `SortableContext` (no column reordering makes sense
    when only one column is visible). Card-level `SortableContext` inside the
    column still mounts.
  - Hide `<AddColumnButton />`.
- Else: unchanged behaviour.

### `SortableColumn.tsx` changes

New prop `isFocusMode?: boolean` (default `false`).

When `isFocusMode`:

- Section container uses `w-full flex-1` instead of the usual `w-72 shrink-0`
  / `min-w-[200px] flex-[1_1_0]`.
- Cards `<ul>` switches class from `flex flex-col gap-2` to
  `grid grid-cols-[repeat(auto-fill,minmax(280px,1fr))] gap-2.5`.
- `collapsed` rendering branch is bypassed (focused column always shows
  expanded; underlying `list_collapse` state is untouched).
- Column drag handle is hidden (no reordering while focused).

### `ColumnHeader.tsx` changes

Add a new `DropdownMenu.Item` at the top of the menu list:

```tsx
<DropdownMenu.Item onSelect={() => setFocused(name)} ...>Focus</DropdownMenu.Item>
```

Placed above "Rename". Hidden when the current column is already the focused
column (exit is done via the top bar or `Esc`, matching HTMX).

### New component: `FocusExitBar.tsx`

`web/renderer/default/src/components/FocusExitBar.tsx`

Small presentational component:

- Label: `Focusing: {name}` (uppercase, muted).
- Button: `Exit Focus [Esc]`, calls `setFocused(null)`.
- Styling approximates `web/css/focus-mode.css` rules but expressed as
  Tailwind utilities, matching the renderer's existing style conventions.

## Data flow

```
BoardView
  ├── FocusedColumnProvider(columns)
  │     └── (columns, active) → clears focused when stale
  │
  ├── [when focused !== null]
  │     └── FocusExitBar → setFocused(null)
  │
  └── SortableColumn × N  ── reads focused via useFocusedColumn
        └── ColumnHeader  ── Focus item → setFocused(name)
```

## Edge cases

| Case | Behaviour |
|------|-----------|
| Focused column deleted | Provider effect clears `focused` to `null`. |
| Focused column renamed | Same (the name no longer exists in `columns`). |
| Active board switched | Provider effect clears `focused` to `null`. |
| Column drag while focused | Horizontal `SortableContext` not mounted → impossible. |
| Card drag within focused column | Works normally — card-level context still mounts. |
| Modal open (card detail, board settings) | `Esc` handler skipped; modal handles its own Esc. |
| Input focused | `Esc` handler skipped. |
| Filter query / hide-completed | Still applied to the focused column's cards (reuses existing `SortableColumn` filtering). |
| Previously-collapsed column focused | Renders expanded; `list_collapse[i]` remains unchanged. |

## Testing

**Unit: `FocusedColumnContext.test.tsx`**

- `setFocused` / `focused` round-trip.
- Clears when `active` changes.
- Clears when focused column is removed from `columns`.
- Esc keydown clears focused.
- Esc is ignored while an input is focused.
- Esc is ignored while a Radix dialog is open.

**Component: `ColumnHeader.test.tsx`** (extend existing test file if present,
else new)

- "Focus" menu item present and calls `setFocused(name)`.
- "Focus" menu item not rendered for the currently-focused column.

**Component: `BoardView.test.tsx`** (extend existing)

- In focus mode, only the focused column is rendered.
- `FocusExitBar` is visible and exits focus on click.
- `AddColumnButton` is hidden in focus mode.
- Exiting focus restores the full column list.

## Out of scope / follow-ups

- Persistence (reload-survivable focus) — can be added later via
  localStorage if requested.
- Keyboard shortcut to *enter* focus mode on the currently selected card's
  column — belongs with the broader kbd-nav skill.
- Animation of column zoom in/out — not in HTMX either.
