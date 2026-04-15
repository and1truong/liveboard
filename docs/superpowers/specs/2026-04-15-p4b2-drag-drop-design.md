# P4b.2 — Drag-Drop (Cards + Columns) — Design

## Goal

Add drag-and-drop to the `/app/` default renderer for:
- Card reorder within a column
- Card move across columns
- Column reorder

All routed through the existing P4b.1a `useBoardMutation` pipeline. No new protocol surface, no changes to `web/shared/*`.

**Shippable value:** `/app/` reaches feature parity with the HTMX UI for spatial card/column manipulation.

## Scope

**In:**
- Pointer + keyboard + touch sensors (mouse, arrow keys + space, finger).
- Card-to-card drop (same or different column).
- Card-to-empty-column drop.
- Column-to-column drop (header doubles as drag handle).
- DragOverlay clone for smooth visual feedback.
- Touch activation constraint (8 px / 200 ms) so taps still fire click handlers.

**Out (later milestones):**
- Multi-select drag (P4d or later).
- Persisted column-collapse during drag.
- Auto-scroll near board edges (dnd-kit AutoScroll plugin — defer if cheap, drop if it bloats bundle).
- Card detail modal (P4b.3).

## Stack additions

| Concern | Choice | Size (gz) |
|---|---|---|
| Drag core | `@dnd-kit/core` | ~10 KB |
| Sortable strategies | `@dnd-kit/sortable` | ~3 KB |
| Utilities (CSS transforms) | `@dnd-kit/utilities` | ~0.5 KB |

Bundle delta: ~13 KB on top of P4b.1a's 99 KB → ~112 KB. Bundle gate stays deferred to P4d per prior decision.

## Architecture

```
BoardView
  └─ <BoardDndContext>            # DndContext + sensors
      └─ <SortableContext items={columnNames} strategy={horizontal}>
          ├─ <SortableColumn col_idx=0>
          │     ├─ ColumnHeader (with drag handle ⋮⋮)
          │     ├─ <SortableContext items={cardIds(col=0)} strategy={vertical}>
          │     │     ├─ <SortableCard col_idx=0 card_idx=0>
          │     │     └─ ...
          │     └─ AddCardButton
          ├─ <SortableColumn col_idx=1>...
          └─ <DragOverlay>          # floating clone of dragged item
```

One outer `DndContext` + per-column `SortableContext` is the canonical dnd-kit kanban pattern. Card drops between columns work because `closestCorners` collision detection finds the target column even when the cursor is between cards.

## Sensor configuration

```ts
useSensors(
  useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
  useSensor(TouchSensor,   { activationConstraint: { delay: 200, tolerance: 5 } }),
  useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
)
```

The pointer 8 px constraint and touch 200 ms delay prevent accidental drags from clicks or taps. Keyboard support uses dnd-kit's built-in sortable coordinate getter — Space starts/ends drag, arrow keys move, Escape cancels.

## Drag identity & data

dnd-kit needs stable string IDs. Encoded by `cardId.ts`:

- Card: `card:${col_idx}:${card_idx}` (composite — IDs change as boards mutate, but only between drags; mid-drag the snapshot is stable).
- Column: `column:${name}` — column names are unique per board.

Each `useSortable` call attaches a `data` payload:
- Card: `{ type: 'card', col_idx, card_idx }`
- Column: `{ type: 'column', name, col_idx }`

`onDragStart`, `onDragOver`, `onDragEnd` discriminate via `active.data.current.type`.

## Drop dispatch (`dispatchDrop`)

Pure function: `(active, over, board) → MutationOp | null`. Tested in isolation.

**Card drop:**

| Active type | Over type | Same col? | Op |
|---|---|---|---|
| card | card | yes, same idx | `null` |
| card | card | yes, different idx | `reorder_card { col_idx, card_idx, before_idx: targetIdx, target_column: <same name> }` |
| card | card | no | `reorder_card { col_idx, card_idx, before_idx: targetIdx, target_column: <other name> }` |
| card | column (header / empty) | — | `move_card { col_idx, card_idx, target_column: <name> }` (appends to end) |

**Column drop:**

| Active type | Over type | Op |
|---|---|---|
| column | column, same name | `null` |
| column | column, different | `move_column { name, after_col }` where `after_col` is computed from drop direction (reuse `moveColumnTarget` semantics) |

`reorder_card`'s `before_idx` is the destination index *in the target column after the source has been removed*. dnd-kit's `arrayMove` follows the same semantics — straightforward conversion.

## Data flow

```
onDragEnd(event)
    │
    ▼
const board = qc.getQueryData(['board', boardId])
const op = dispatchDrop(event.active, event.over, board)
if (op) mutation.mutate(op)        # uses existing useBoardMutation
```

Optimistic apply happens inside `useBoardMutation` via shared `applyOp` — no DnD-specific optimistic logic. dnd-kit handles the *visual* mid-drag transform; the *data* commits only on drop.

## Component contracts

### `<BoardDndContext children boardId />`
Mounts `<DndContext sensors collisionDetection={closestCorners} onDragEnd={dispatch}>`. Reads QueryClient + `useBoardMutation(boardId)`. Renders children + `<DragOverlay>` showing a clone of the active item.

### `<SortableColumn column colIdx allColumnNames boardId />`
Wraps `Column`'s contents with `useSortable({ id: column:${column.name}, data: { type, name, col_idx } })`. Applies `transform` + `transition` styles. Drag handle = a small `⋮⋮` icon next to `ColumnHeader`'s name; the kebab menu trigger stays clickable.

### `<SortableCard card colIdx cardIdx boardId />`
Wraps `CardEditable` with `useSortable({ id: card:${colIdx}:${cardIdx}, data: { type, col_idx, card_idx } })`. Drag handle = a small grip icon on the left, visible on hover. The card body (title, complete button, edit click) stays interactive.

### `dispatchDrop(active, over, board): MutationOp | null`
See table above. Pure; no React or dnd-kit imports beyond types.

### `cardId.ts`
- `encodeCardId(colIdx, cardIdx): string`
- `decodeCardId(id): { colIdx, cardIdx } | null` (null if not a card id)

## Visual feedback

- Active item: 50 % opacity in place; full clone in `<DragOverlay>`.
- Sortable items: `transition: transform 200ms ease`.
- Empty drop target: subtle dashed border on the receiving column when a card hovers over it.

## Drag handle vs. existing interactions

This is the riskiest UX detail. The card already responds to:
- double-click (edit)
- click on circle (complete)
- click on ✕ (delete)
- click anywhere else (no-op today; reserved for P4b.3 modal)

**Decision:** drag must originate on a dedicated grip icon (`⋮⋮` left side, opacity 0 → 100 on hover). The card body forwards no drag listeners. This keeps double-click and button clicks intact and prevents touch-start ambiguity.

Same logic for columns — drag handle is a separate `⋮⋮` next to the column name; `ColumnHeader`'s kebab menu and rename input are unaffected.

## Error handling

All errors flow through `useBoardMutation`'s existing handler:
- `VERSION_CONFLICT` → rollback + invalidate + toast (board may have changed elsewhere mid-drag).
- `OUT_OF_RANGE` → toast "Index out of range" — most likely if two simultaneous drags collide on the same boardId.
- Other codes → existing copy.

No DnD-specific error handling needed.

## Testing

**Unit:**
- `cardId.test.ts` — encode/decode roundtrip, returns null on bad input.
- `dispatchDrop.test.ts` — exhaustive table:
  - card → same-position no-op
  - card → same column different position
  - card → different column at index
  - card → empty column
  - column → adjacent
  - column → far position
  - bad/null over → null

**Component tests skipped.** happy-dom does not faithfully simulate `pointerdown` + `pointermove` + `pointerup` sequences that dnd-kit listens to (same family of issues as the `keyDown` bug in P4b.1a). Manual smoke covers the integration. If we add Playwright in P4d we'll backfill.

**Manual smoke** (added to existing P4b.1a manual checklist):
- Drag a card from "Todo" position 0 → position 2 → reflects after refresh.
- Drag a card from "Todo" → "Doing" → reflects.
- Drag a card to an empty column → appears in that column.
- Drag a column header left/right → order changes.
- Try to start a drag from the card title → no drag fires; double-click still edits.
- Try to start a drag with a 4 px mouse jiggle → no drag (8 px constraint).
- Tab to a card, press Space, arrow keys, Space again → keyboard reorder works.
- On a touch device (or DevTools touch emulation) — long-press 200 ms then drag.

## Risks

- **Bundle**: another ~13 KB on top of 99 KB. Already accepted via P4b.1a's deferred gate.
- **closestCorners vs closestCenter**: kanban with mixed orientations (vertical lists in horizontal columns) needs `closestCorners`. The dnd-kit kanban example confirms this.
- **DragOverlay re-render**: rendering a full `Card` clone in the overlay is fine; rendering a full `Column` clone (with all its cards) on column drag could lag on huge boards. Acceptable for P4b.2; revisit only if observed.
- **Mid-drag invalidation race**: if `board.updated` arrives mid-drag, optimistic apply could shuffle the drop target. P4a's invalidate-on-event behavior plus useBoardMutation's `cancelQueries` mitigates this; still possible in pathological cases. Document as a known limitation; rare in single-user local-first usage.

## Open questions

None blocking. Pre-decided:
- Drag handle is a separate icon, not the whole card/column.
- No multi-select.
- Touch supported (200 ms / 5 px constraint).
- DragOverlay used for visual clone (smoother than transform-only).

## Dependencies on prior work

- P2: shared `move_card`, `reorder_card`, `move_column` ops + `applyOp` semantics.
- P4a: `BoardView`, `Column`, query infrastructure.
- P4b.1a: `useBoardMutation`, `errorToast`, `moveColumnTarget`.
