# P4c.3 — Keyboard Navigation (Cards) — Design

## Goal

Add roving-tabindex arrow-key navigation across cards in `/app/`'s board view. Tab into the board → first card focused → arrow keys move focus → Enter opens detail modal → Delete/Backspace fires staged delete. No new protocol, no library additions.

**Shippable value:** `/app/` becomes keyboard-operable end-to-end. After P4c.3, P4c is done; only P4d (settings + theme + bundle gate) remains.

## Scope

**In:**
- Roving tabindex across cards in the board area.
- Arrow Up/Down/Left/Right move focus between cards.
- Enter opens the detail modal for the focused card.
- Delete / Backspace fires `stageDelete(...)` for the focused card.
- Empty columns are skipped during left/right traversal.
- Post-mutation focus invariant: after delete, focus clamps to a still-valid position; if the board empties, focus clears.
- Visible focus ring on the focused card.

**Out:**
- Sidebar arrow nav (already keyboard-accessible via native Tab; cmdk handles board jumps).
- Column-level focus / column-header focus (deferred; empty columns are simply skipped).
- Global shortcuts ("n" to add a card, etc.) — competes with cmdk, defer to P4d if real demand emerges.
- Card reorder via keyboard outside dnd-kit's existing Space-grab + arrows path.
- Touch focus rings.

## Architecture

```
BoardView
 └─ <BoardFocusProvider columns={columns}>
     ├─ keeps focused: { colIdx, cardIdx } | null
     ├─ exposes: { focused, setFocused, move, registerCard }
     └─ effect: clamp focused on column/card mutation
        children: <SortableContext> + per-column <SortableColumn>
                   inside which <SortableCard> reads context
```

`<SortableCard>` (existing) gains:
- A `useCardFocus(colIdx, cardIdx)` hook call. Returns `{ isFocused, ref }`.
- `tabIndex={isFocused || (focused === null && colIdx === 0 && cardIdx === 0) ? 0 : -1}` on its wrapper.
- `onFocus={() => setFocused({ colIdx, cardIdx })}`.
- `onKeyDown` handler that switches on `e.key`:
  - Arrow keys → `move(dir)`.
  - Enter → `setModalOpen(true)`.
  - Delete / Backspace → `stageDelete(...)`.
  - Early-return if `e.defaultPrevented` (dnd-kit consumed it) or if focus target is `INPUT`/`TEXTAREA` (inline rename / modal).

`<CardEditable>` (existing) is changed to receive `modalOpen` and `onModalOpenChange` from `SortableCard` instead of holding its own state. Its single-click body handler calls `onModalOpenChange(true)`. This lets the keyboard's Enter and the click both open the same modal.

## File structure

**New:**
- `web/renderer/default/src/contexts/BoardFocusContext.tsx`
- `web/renderer/default/src/contexts/BoardFocusContext.test.tsx`

**Modified:**
- `web/renderer/default/src/components/BoardView.tsx` — wrap children in `<BoardFocusProvider columns={columns}>`.
- `web/renderer/default/src/dnd/SortableCard.tsx` — add focus + key handlers, lift `modalOpen` from CardEditable.
- `web/renderer/default/src/components/CardEditable.tsx` — accept `modalOpen` + `onModalOpenChange` props instead of owning that state.
- `web/renderer/default/src/components/CardEditable.test.tsx` — update tests for the new prop shape.

## Component contracts

### `<BoardFocusProvider columns={columns}>`

```ts
interface BoardFocusCtx {
  focused: { colIdx: number; cardIdx: number } | null
  setFocused: (next: { colIdx: number; cardIdx: number } | null) => void
  move: (dir: 'up' | 'down' | 'left' | 'right') => void
  registerCard: (colIdx: number, cardIdx: number, el: HTMLElement | null) => void
}
```

Implementation outline:
```tsx
const [focused, setFocused] = useState<...>(null)
const refs = useRef(new Map<string, HTMLElement>())

const registerCard = (col, card, el) => {
  const key = `${col}:${card}`
  if (el) refs.current.set(key, el)
  else refs.current.delete(key)
}

useEffect(() => {
  if (!focused) return
  const el = refs.current.get(`${focused.colIdx}:${focused.cardIdx}`)
  if (el && document.activeElement !== el) el.focus()
}, [focused])

// Clamp on column structure change.
useEffect(() => {
  if (!focused) return
  const col = columns[focused.colIdx]
  const len = col?.cards?.length ?? 0
  if (!col) {
    const lastCol = Math.max(0, columns.length - 1)
    const lastLen = columns[lastCol]?.cards?.length ?? 0
    if (columns.length === 0 || lastLen === 0) setFocused(null)
    else setFocused({ colIdx: lastCol, cardIdx: lastLen - 1 })
    return
  }
  if (focused.cardIdx >= len) {
    if (len === 0) {
      // Find next non-empty column (left first, then right) or null.
      const target = nearestNonEmpty(columns, focused.colIdx)
      setFocused(target)
    } else {
      setFocused({ colIdx: focused.colIdx, cardIdx: len - 1 })
    }
  }
}, [columns, focused])

const move = (dir) => { /* see semantics below */ }
```

`nearestNonEmpty(columns, fromIdx)` searches outward (`fromIdx-1`, `fromIdx+1`, `fromIdx-2`, `fromIdx+2`, …) and returns `{ colIdx, cardIdx: 0 }` for the first column with cards, or `null`.

### `useCardFocus(colIdx, cardIdx)`

```tsx
function useCardFocus(colIdx, cardIdx) {
  const { focused, registerCard } = useBoardFocus()
  const isFocused = focused?.colIdx === colIdx && focused?.cardIdx === cardIdx
  const ref = useCallback((el) => registerCard(colIdx, cardIdx, el), [colIdx, cardIdx, registerCard])
  return { isFocused, ref }
}
```

### `move` semantics

Same logic from the design conversation, formalized:

```ts
function move(dir) {
  if (!focused) {
    setFocused({ colIdx: 0, cardIdx: 0 })
    return
  }
  const { colIdx, cardIdx } = focused
  switch (dir) {
    case 'up':
      if (cardIdx > 0) setFocused({ colIdx, cardIdx: cardIdx - 1 })
      return
    case 'down': {
      const len = columns[colIdx]?.cards?.length ?? 0
      if (cardIdx < len - 1) setFocused({ colIdx, cardIdx: cardIdx + 1 })
      return
    }
    case 'left':
    case 'right': {
      const step = dir === 'left' ? -1 : 1
      let next = colIdx + step
      while (next >= 0 && next < columns.length && (columns[next]?.cards?.length ?? 0) === 0) {
        next += step
      }
      if (next < 0 || next >= columns.length) return
      const newLen = columns[next]?.cards?.length ?? 0
      setFocused({ colIdx: next, cardIdx: Math.min(cardIdx, newLen - 1) })
      return
    }
  }
}
```

### `<SortableCard>` keyboard handler

```tsx
const { isFocused, ref: focusRef } = useCardFocus(colIdx, cardIdx)
const { focused, setFocused, move } = useBoardFocus()
const [modalOpen, setModalOpen] = useState(false)
const mutation = useBoardMutation(boardId)

const onKeyDown = (e: React.KeyboardEvent) => {
  if (e.defaultPrevented) return
  const tag = (e.target as HTMLElement).tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA') return
  switch (e.key) {
    case 'ArrowUp':    e.preventDefault(); move('up'); break
    case 'ArrowDown':  e.preventDefault(); move('down'); break
    case 'ArrowLeft':  e.preventDefault(); move('left'); break
    case 'ArrowRight': e.preventDefault(); move('right'); break
    case 'Enter':      e.preventDefault(); setModalOpen(true); break
    case 'Delete':
    case 'Backspace':
      e.preventDefault()
      stageDelete(
        () => mutation.mutate({ type: 'delete_card', col_idx: colIdx, card_idx: cardIdx }),
        card.title,
      )
      break
  }
}

const showFocusedTabStop = isFocused || (focused === null && colIdx === 0 && cardIdx === 0)
```

The wrapper `<div>` ref combines the existing dnd-kit `setNodeRef` and the `focusRef`:
```tsx
ref={(el) => { setNodeRef(el); focusRef(el) }}
tabIndex={showFocusedTabStop ? 0 : -1}
onFocus={() => setFocused({ colIdx, cardIdx })}
onKeyDown={onKeyDown}
className={`... ${isFocused ? 'ring-2 ring-blue-400' : ''}`}
```

### `<CardEditable>` prop change

Old: owns `const [modalOpen, setModalOpen] = useState(false)`.

New: receives via props.
```ts
interface CardEditableProps {
  card: CardModel
  colIdx: number
  cardIdx: number
  boardId: string
  modalOpen: boolean
  onModalOpenChange: (next: boolean) => void
}
```

The single-click "open card details" button calls `onModalOpenChange(true)`. The `<CardDetailModal>` receives `open={modalOpen}` and `onOpenChange={onModalOpenChange}`. Everything else unchanged.

Existing `CardEditable.test.tsx` cases that drive the modal click need to assert via parent state instead of internal state — wrap in a small harness component that holds `modalOpen` in `useState` and forwards.

## Visual spec

- Focused card wrapper gets `outline: none` (browser default) replaced with `ring-2 ring-blue-400 ring-offset-2`.
- No animation on focus change.
- Standard browser focus-visible behavior preserved (focus ring shows after keyboard nav, not after click).

## Testing

`BoardFocusContext.test.tsx` — pure logic tests for `move` + clamp effect:
- Build a fake `columns` array.
- `move('down')` from (0,0) → (0,1).
- `move('down')` at last card → no-op (focus unchanged).
- `move('right')` clamps cardIdx to new column's length.
- `move('right')` skips an empty column.
- `move('left')` from colIdx 0 → no-op.
- `move(...)` from null `focused` → (0,0).
- After columns mutate so focused position is invalid, the clamp effect moves to a valid one (or `null` for empty board).

End-to-end keyboard interaction (focus rings, Tab in, arrow movement) verified by manual smoke. happy-dom's `tabIndex` + arrow event behavior is fragile; pure-logic coverage is the better gate.

## Risks

- **Roving tabindex + dnd-kit Space-grab**: dnd-kit's KeyboardSensor activates on the drag handle (the `⋮⋮` grip), not on the card body. So Space on the card body does nothing, Arrows on the card body go to our handler. Once dnd-kit's drag is active, it `preventDefault`s its own keys; our `if (e.defaultPrevented) return` guards against double-handling.
- **Inline rename mode**: focus is in the `<input>`. Our keyDown is on the wrapper; events bubble. `INPUT`/`TEXTAREA` early-return prevents interference.
- **Modal open**: Radix Dialog portals the modal outside the SortableCard subtree; events inside the modal don't bubble to the wrapper. Verified.
- **Drag handle and inner buttons (complete, delete, kebab)**: each is a `<button>` and steals focus on click. They're inside the focused wrapper but `e.target` matches them, not the wrapper. The keyDown handler still runs, but Space/Enter on a button has its own native semantics. The `tag === 'BUTTON'` case isn't excluded today; if Enter on the complete button accidentally also opens the modal, add `BUTTON` to the early-return list. **Decision:** include `BUTTON` in the early-return for keyboard handler — buttons handle their own activation.
- **Default-focus rule on first render**: if `focused === null` and the first card mounts, its `tabIndex={0}` is set by the rule. Once any card receives focus (via Tab), `focused` becomes non-null and the rule is moot.
- **Cross-column move clamping with mismatched lengths**: covered by `move`'s `Math.min(cardIdx, newLen - 1)`. Tests cover this explicitly.

## Open questions

None blocking. Pre-decided:
- Roving tabindex on cards only (columns excluded; empty columns skipped).
- Enter opens modal; Delete/Backspace stages delete.
- `CardEditable` modal state lifts to `SortableCard`.
- Inputs and other buttons skip the keyboard handler via `tag` check.

## Dependencies on prior work

- P4b.1a: `useBoardMutation`, `stageDelete` (generalized in P4c.1), `errorToast`.
- P4b.2: `<SortableCard>`, drag handle pattern.
- P4b.3: `<CardDetailModal>`.
- P4c.1: roving-tabindex pattern not strictly required, but `stageDelete(fire, label)` shape used.
