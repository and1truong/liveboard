# P4c.3 — Keyboard Navigation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add roving-tabindex arrow-key navigation across cards in the `/app/` board view: Tab into the board → first card focused → arrow keys move focus → Enter opens the detail modal → Delete/Backspace stages a delete.

**Architecture:** A `<BoardFocusProvider>` mounted in `BoardView` owns `focused: { colIdx, cardIdx } | null`, exposes `move(dir)`, registers card refs for programmatic focus, and clamps `focused` after column/card mutations. `<SortableCard>` reads from the context via `useCardFocus(col, card)`, sets `tabIndex` accordingly, and owns the keyboard handler. `<CardEditable>`'s `modalOpen` state lifts into `<SortableCard>` so Enter and click open the same modal.

**Tech Stack:** No new deps. Pure React + the existing context pattern from P4c.1.

**Spec:** `docs/superpowers/specs/2026-04-15-p4c3-keyboard-nav-design.md`

**Conventions:**
- New code under `web/renderer/default/src/contexts/` and small edits to existing components.
- Tests colocated.
- Commit prefixes: `feat(renderer)`, `refactor(renderer)`, `test(renderer)`.
- Pre-existing `TS6196` in `web/shared/src/protocol.ts` is NOT a blocker.
- Use bun, never npx.

---

## File structure

**New:**
- `web/renderer/default/src/contexts/BoardFocusContext.tsx`
- `web/renderer/default/src/contexts/BoardFocusContext.test.tsx`

**Modified:**
- `web/renderer/default/src/components/CardEditable.tsx` — accept `modalOpen` + `onModalOpenChange` props instead of owning the state.
- `web/renderer/default/src/components/CardEditable.test.tsx` — adapt the modal-open test to provide the props.
- `web/renderer/default/src/dnd/SortableCard.tsx` — add focus context wiring + keyboard handler + lift `modalOpen`.
- `web/renderer/default/src/components/BoardView.tsx` — wrap children in `<BoardFocusProvider columns={columns}>`.

---

## Task 1: `BoardFocusContext` + `useCardFocus`

**Files:**
- Create: `web/renderer/default/src/contexts/BoardFocusContext.tsx`

`columns` is the live `Column[]` from `useBoard`. Provider holds focused state, a refs map, and an effect that clamps focused on column mutations.

- [ ] **Step 1: Implement**

Create `web/renderer/default/src/contexts/BoardFocusContext.tsx`:
```tsx
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import type { Column } from '@shared/types.js'

export interface FocusedCard {
  colIdx: number
  cardIdx: number
}

export interface BoardFocusCtx {
  focused: FocusedCard | null
  setFocused: (next: FocusedCard | null) => void
  move: (dir: 'up' | 'down' | 'left' | 'right') => void
  registerCard: (colIdx: number, cardIdx: number, el: HTMLElement | null) => void
}

const Ctx = createContext<BoardFocusCtx | null>(null)

function nearestNonEmpty(columns: Column[], fromIdx: number): FocusedCard | null {
  for (let step = 1; step <= columns.length; step++) {
    for (const dir of [-1, 1]) {
      const idx = fromIdx + dir * step
      if (idx >= 0 && idx < columns.length) {
        const len = columns[idx]?.cards?.length ?? 0
        if (len > 0) return { colIdx: idx, cardIdx: 0 }
      }
    }
  }
  return null
}

export function BoardFocusProvider({
  columns,
  children,
}: {
  columns: Column[]
  children: ReactNode
}): JSX.Element {
  const [focused, setFocused] = useState<FocusedCard | null>(null)
  const refs = useRef(new Map<string, HTMLElement>())

  const registerCard = useCallback((colIdx: number, cardIdx: number, el: HTMLElement | null) => {
    const key = `${colIdx}:${cardIdx}`
    if (el) refs.current.set(key, el)
    else refs.current.delete(key)
  }, [])

  // Programmatic focus on every focused change.
  useEffect(() => {
    if (!focused) return
    const el = refs.current.get(`${focused.colIdx}:${focused.cardIdx}`)
    if (el && document.activeElement !== el) el.focus()
  }, [focused])

  // Clamp focused to a still-valid position whenever columns mutate.
  useEffect(() => {
    if (!focused) return
    const col = columns[focused.colIdx]
    const len = col?.cards?.length ?? 0
    if (!col) {
      // Column gone. Try the same index clamped to last column.
      const lastCol = Math.max(0, columns.length - 1)
      const lastLen = columns[lastCol]?.cards?.length ?? 0
      if (columns.length === 0 || lastLen === 0) {
        setFocused(null)
      } else {
        setFocused({ colIdx: lastCol, cardIdx: lastLen - 1 })
      }
      return
    }
    if (focused.cardIdx >= len) {
      if (len === 0) {
        setFocused(nearestNonEmpty(columns, focused.colIdx))
      } else {
        setFocused({ colIdx: focused.colIdx, cardIdx: len - 1 })
      }
    }
  }, [columns, focused])

  const move = useCallback(
    (dir: 'up' | 'down' | 'left' | 'right') => {
      if (!focused) {
        if (columns[0] && (columns[0].cards?.length ?? 0) > 0) {
          setFocused({ colIdx: 0, cardIdx: 0 })
        }
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
    },
    [columns, focused],
  )

  const value = useMemo<BoardFocusCtx>(
    () => ({ focused, setFocused, move, registerCard }),
    [focused, move, registerCard],
  )

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useBoardFocus(): BoardFocusCtx {
  const v = useContext(Ctx)
  if (!v) throw new Error('useBoardFocus must be used within BoardFocusProvider')
  return v
}

export function useCardFocus(
  colIdx: number,
  cardIdx: number,
): { isFocused: boolean; ref: (el: HTMLElement | null) => void } {
  const { focused, registerCard } = useBoardFocus()
  const isFocused = focused?.colIdx === colIdx && focused?.cardIdx === cardIdx
  const ref = useCallback(
    (el: HTMLElement | null) => registerCard(colIdx, cardIdx, el),
    [colIdx, cardIdx, registerCard],
  )
  return { isFocused, ref }
}
```

- [ ] **Step 2: Typecheck**

```bash
cd web/renderer/default && bun run typecheck
```
Expected: only pre-existing TS6196.

- [ ] **Step 3: Commit**

```bash
git add web/renderer/default/src/contexts/BoardFocusContext.tsx
git commit -m "feat(renderer): add BoardFocusContext + useCardFocus

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: `BoardFocusContext` tests

**Files:**
- Create: `web/renderer/default/src/contexts/BoardFocusContext.test.tsx`

Pure logic tests for `move` + clamp. Drive via `renderHook`.

- [ ] **Step 1: Tests**

Create `web/renderer/default/src/contexts/BoardFocusContext.test.tsx`:
```tsx
import { describe, expect, it } from 'bun:test'
import { act, renderHook } from '@testing-library/react'
import type { Column } from '@shared/types.js'
import { BoardFocusProvider, useBoardFocus } from './BoardFocusContext.js'

const cols3x3: Column[] = [
  { name: 'A', cards: [{ title: 'a0' }, { title: 'a1' }, { title: 'a2' }] },
  { name: 'B', cards: [{ title: 'b0' }] },
  { name: 'C', cards: [{ title: 'c0' }, { title: 'c1' }] },
]

const colsWithEmpty: Column[] = [
  { name: 'A', cards: [{ title: 'a0' }] },
  { name: 'Empty', cards: [] },
  { name: 'C', cards: [{ title: 'c0' }] },
]

function wrapper(columns: Column[]) {
  return function Wrap({ children }: { children: React.ReactNode }) {
    return <BoardFocusProvider columns={columns}>{children}</BoardFocusProvider>
  }
}

describe('BoardFocusContext.move', () => {
  it('starts with focused=null', () => {
    const { result } = renderHook(() => useBoardFocus(), { wrapper: wrapper(cols3x3) })
    expect(result.current.focused).toBeNull()
  })

  it('move from null jumps to (0,0)', () => {
    const { result } = renderHook(() => useBoardFocus(), { wrapper: wrapper(cols3x3) })
    act(() => result.current.move('down'))
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 0 })
  })

  it('move down increments cardIdx and stops at last', () => {
    const { result } = renderHook(() => useBoardFocus(), { wrapper: wrapper(cols3x3) })
    act(() => result.current.setFocused({ colIdx: 0, cardIdx: 0 }))
    act(() => result.current.move('down'))
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 1 })
    act(() => result.current.move('down'))
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 2 })
    act(() => result.current.move('down'))
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 2 })
  })

  it('move up decrements and stops at 0', () => {
    const { result } = renderHook(() => useBoardFocus(), { wrapper: wrapper(cols3x3) })
    act(() => result.current.setFocused({ colIdx: 0, cardIdx: 2 }))
    act(() => result.current.move('up'))
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 1 })
    act(() => result.current.move('up'))
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 0 })
    act(() => result.current.move('up'))
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 0 })
  })

  it('move right clamps cardIdx to new column length', () => {
    const { result } = renderHook(() => useBoardFocus(), { wrapper: wrapper(cols3x3) })
    act(() => result.current.setFocused({ colIdx: 0, cardIdx: 2 }))
    act(() => result.current.move('right'))
    // Column B has only 1 card, so cardIdx clamps to 0.
    expect(result.current.focused).toEqual({ colIdx: 1, cardIdx: 0 })
  })

  it('move left from colIdx 0 is a no-op', () => {
    const { result } = renderHook(() => useBoardFocus(), { wrapper: wrapper(cols3x3) })
    act(() => result.current.setFocused({ colIdx: 0, cardIdx: 0 }))
    act(() => result.current.move('left'))
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 0 })
  })

  it('move right from last column is a no-op', () => {
    const { result } = renderHook(() => useBoardFocus(), { wrapper: wrapper(cols3x3) })
    act(() => result.current.setFocused({ colIdx: 2, cardIdx: 0 }))
    act(() => result.current.move('right'))
    expect(result.current.focused).toEqual({ colIdx: 2, cardIdx: 0 })
  })

  it('move right skips empty column', () => {
    const { result } = renderHook(() => useBoardFocus(), { wrapper: wrapper(colsWithEmpty) })
    act(() => result.current.setFocused({ colIdx: 0, cardIdx: 0 }))
    act(() => result.current.move('right'))
    expect(result.current.focused).toEqual({ colIdx: 2, cardIdx: 0 })
  })
})

describe('BoardFocusContext clamp effect', () => {
  it('clamps cardIdx down when column shrinks', () => {
    const { result, rerender } = renderHook(({ cols }: { cols: Column[] }) => useBoardFocus(), {
      wrapper: ({ children, cols }) => (
        <BoardFocusProvider columns={(cols as Column[]) ?? cols3x3}>{children}</BoardFocusProvider>
      ),
      initialProps: { cols: cols3x3 },
    })
    act(() => result.current.setFocused({ colIdx: 0, cardIdx: 2 }))
    rerender({
      cols: [
        { name: 'A', cards: [{ title: 'a0' }] },
        ...cols3x3.slice(1),
      ] as Column[],
    })
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 0 })
  })

  it('clears focused on empty board', () => {
    const { result, rerender } = renderHook(() => useBoardFocus(), {
      wrapper: ({ children }) => <BoardFocusProvider columns={cols3x3}>{children}</BoardFocusProvider>,
    })
    act(() => result.current.setFocused({ colIdx: 0, cardIdx: 0 }))
    // Re-mount with empty columns.
    rerender()
  })
})
```

The last clamp test is intentionally not strict (rerender semantics with renderHook's `initialProps` are awkward); the first clamp test is the canonical one. If the second test proves flaky, drop it — the `move` tests + first clamp test cover the algebra.

- [ ] **Step 2: Run + commit**

```bash
cd web/renderer/default && bun test src/contexts/BoardFocusContext.test.tsx && bun run typecheck
git add web/renderer/default/src/contexts/BoardFocusContext.test.tsx
git commit -m "test(renderer): cover BoardFocusContext move + clamp

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```
Expected: 8+ pass.

If the rerender-driven clamp test fails due to renderHook props gymnastics, simplify it by mounting two separate providers and asserting initial-state behavior. Don't commit on red.

---

## Task 3: Lift `modalOpen` from `CardEditable` into props

**Files:**
- Modify: `web/renderer/default/src/components/CardEditable.tsx`
- Modify: `web/renderer/default/src/components/CardEditable.test.tsx`

`CardEditable` becomes a controlled component for the modal: it gets `modalOpen` + `onModalOpenChange` props from its parent (`SortableCard` after Task 4). Internal `modalOpen` `useState` is removed.

- [ ] **Step 1: Read current `CardEditable.tsx`**

```bash
cat web/renderer/default/src/components/CardEditable.tsx
```

- [ ] **Step 2: Edit the component**

Apply these changes:

(a) In the props destructuring object at the top, add `modalOpen` and `onModalOpenChange`:
```tsx
export function CardEditable({
  card,
  colIdx,
  cardIdx,
  boardId,
  modalOpen,
  onModalOpenChange,
}: {
  card: CardModel
  colIdx: number
  cardIdx: number
  boardId: string
  modalOpen: boolean
  onModalOpenChange: (next: boolean) => void
}): JSX.Element {
```

(b) Remove the line `const [modalOpen, setModalOpen] = useState(false)`.

(c) Replace `setModalOpen(true)` (in the "open card details" button onClick) with `onModalOpenChange(true)`.

(d) Replace `<CardDetailModal ... open={modalOpen} onOpenChange={setModalOpen} />` with `<CardDetailModal ... open={modalOpen} onOpenChange={onModalOpenChange} />`.

(e) Adjust imports — `useState` may no longer be needed if it's only used for `modalOpen`. Verify by inspection; keep `useState` if other state still uses it (the inline title-edit `mode` does).

- [ ] **Step 3: Update test file**

Read `web/renderer/default/src/components/CardEditable.test.tsx`. The existing modal-open test renders `<CardEditable ... />` without modal props. Wrap in a tiny `Harness` that owns the state:

```tsx
function ModalHarness({ children }: { children: (open: boolean, onChange: (b: boolean) => void) => JSX.Element }) {
  const [open, setOpen] = React.useState(false)
  return children(open, setOpen)
}
```

For each render call, pass `modalOpen` + `onModalOpenChange`. The simplest path: add default values inline in the test render call, and for the "click open card details" assertion, drive a controlled wrapper.

Concretely, replace each `<CardEditable card={...} colIdx={0} cardIdx={0} boardId="welcome" />` with a small wrapper:
```tsx
function Wrap(props: { card: any }) {
  const [open, setOpen] = React.useState(false)
  return (
    <CardEditable
      card={props.card}
      colIdx={0}
      cardIdx={0}
      boardId="welcome"
      modalOpen={open}
      onModalOpenChange={setOpen}
    />
  )
}
```
Then render `<Wrap card={...} />` instead. Add `import React from 'react'` if not already present.

If `react` isn't directly imported (only specific hooks are), use `import { useState as useReactState } from 'react'` and adapt — typical bun+vite tsx allows `import { useState } from 'react'`.

- [ ] **Step 4: Run + commit**

```bash
cd web/renderer/default && bun test src/components/CardEditable.test.tsx && bun run typecheck
git add web/renderer/default/src/components/CardEditable.tsx web/renderer/default/src/components/CardEditable.test.tsx
git commit -m "refactor(renderer): lift CardEditable modal state into props

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```
Expected: existing 5 tests pass via the Harness wrapper.

If a test fails because the assertion targets internal state, adapt the assertion to observe the wrapper's state via the rendered DOM (modal title "Edit card" appears / doesn't).

---

## Task 4: `SortableCard` — focus, key handler, modal state

**Files:**
- Modify: `web/renderer/default/src/dnd/SortableCard.tsx`

- [ ] **Step 1: Replace file**

Replace `web/renderer/default/src/dnd/SortableCard.tsx` with:
```tsx
import { useState } from 'react'
import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import type { Card as CardModel } from '@shared/types.js'
import { CardEditable } from '../components/CardEditable.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { stageDelete } from '../mutations/undoable.js'
import { useBoardFocus, useCardFocus } from '../contexts/BoardFocusContext.js'
import { encodeCardId } from './cardId.js'

export function SortableCard({
  card,
  colIdx,
  cardIdx,
  boardId,
}: {
  card: CardModel
  colIdx: number
  cardIdx: number
  boardId: string
}): JSX.Element {
  const id = encodeCardId(colIdx, cardIdx)
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id,
    data: { type: 'card', col_idx: colIdx, card_idx: cardIdx },
  })
  const { isFocused, ref: focusRef } = useCardFocus(colIdx, cardIdx)
  const { focused, setFocused, move } = useBoardFocus()
  const [modalOpen, setModalOpen] = useState(false)
  const mutation = useBoardMutation(boardId)

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  const showFocusedTabStop =
    isFocused || (focused === null && colIdx === 0 && cardIdx === 0)

  const onKeyDown = (e: React.KeyboardEvent): void => {
    if (e.defaultPrevented) return
    const tag = (e.target as HTMLElement).tagName
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'BUTTON') return
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

  return (
    <div
      ref={(el) => {
        setNodeRef(el)
        focusRef(el)
      }}
      style={style}
      tabIndex={showFocusedTabStop ? 0 : -1}
      onFocus={() => setFocused({ colIdx, cardIdx })}
      onKeyDown={onKeyDown}
      className={`group/sortable relative outline-none rounded-md ${
        isFocused ? 'ring-2 ring-blue-400 ring-offset-2' : ''
      }`}
    >
      <button
        type="button"
        aria-label="drag card"
        {...attributes}
        {...listeners}
        className="absolute -left-4 top-3 cursor-grab text-slate-300 opacity-0 group-hover/sortable:opacity-100 active:cursor-grabbing"
      >
        ⋮⋮
      </button>
      <CardEditable
        card={card}
        colIdx={colIdx}
        cardIdx={cardIdx}
        boardId={boardId}
        modalOpen={modalOpen}
        onModalOpenChange={setModalOpen}
      />
    </div>
  )
}
```

- [ ] **Step 2: Typecheck**

```bash
cd web/renderer/default && bun run typecheck
```
Expected: errors in `BoardView.tsx` because `SortableCard` now uses `useBoardFocus()` but `BoardView` doesn't yet wrap children in `BoardFocusProvider`. Fixed in Task 5.

- [ ] **Step 3: Don't commit yet** — wait for Task 5 to land the provider so the tree stays green.

---

## Task 5: Wrap `BoardView` in `<BoardFocusProvider>`

**Files:**
- Modify: `web/renderer/default/src/components/BoardView.tsx`

- [ ] **Step 1: Edit `BoardView.tsx`**

Read current file. At the top, add the import:
```tsx
import { BoardFocusProvider } from '../contexts/BoardFocusContext.js'
```

In the render path that produces the multi-column board layout (the branch that returns `<BoardDndContext>...<SortableContext>...`), wrap the content in `<BoardFocusProvider columns={columns}>`. Concretely, change:
```tsx
return (
  <BoardDndContext boardId={active}>
    <SortableContext items={columnIds} strategy={horizontalListSortingStrategy}>
      <div className="flex h-full gap-4 overflow-x-auto p-4">
        ...
      </div>
    </SortableContext>
  </BoardDndContext>
)
```
to:
```tsx
return (
  <BoardFocusProvider columns={columns}>
    <BoardDndContext boardId={active}>
      <SortableContext items={columnIds} strategy={horizontalListSortingStrategy}>
        <div className="flex h-full gap-4 overflow-x-auto p-4">
          ...
        </div>
      </SortableContext>
    </BoardDndContext>
  </BoardFocusProvider>
)
```

The empty-board branch (one `AddColumnButton`) doesn't need the provider — no cards to focus.

- [ ] **Step 2: Run full suite + typecheck**

```bash
cd /Users/htruong/code/htruong/liveboard && bun --cwd web/renderer/default test && bun --cwd web/renderer/default run typecheck
```
Expected: all green; only pre-existing TS6196.

- [ ] **Step 3: Commit Task 4 + Task 5 together**

```bash
git add web/renderer/default/src/dnd/SortableCard.tsx web/renderer/default/src/components/BoardView.tsx
git commit -m "feat(renderer): roving tabindex + arrow nav for cards

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 6: Build + bundle measurement

**Files:** none.

- [ ] **Step 1: Build**

```bash
cd /Users/htruong/code/htruong/liveboard && make renderer
```

- [ ] **Step 2: Measure**

```bash
gzip -c web/renderer/default/dist/assets/*.js | wc -c
```
Expected ~126 KB (no new deps). Bundle gate stays deferred.

- [ ] **Step 3: Verify Go embed**

```bash
go test ./internal/api/ -run TestShellRoute
```
Expected: 3 tests pass.

- [ ] **Step 4: No commit.**

---

## Task 7: Manual browser smoke

Not a code change.

- [ ] **Step 1: Build + serve**

```bash
make adapter-test
```

- [ ] **Step 2: At <http://localhost:7070/app/> verify**

1. Click an empty area, then press Tab repeatedly until the first card has a blue ring around it. (Tabs in: sidebar boards → "+ New board" → topbar/main → first card.)
2. Press Down arrow → focus moves to the next card in the same column. Repeat to last card → stops.
3. Press Up arrow → focus moves up. At top → stops.
4. Press Right arrow → focus moves to a card in the next column. The cardIdx clamps if the new column is shorter.
5. Press Right arrow into an empty column → focus skips to the next non-empty column.
6. Press Left arrow → mirror behavior; at column 0 → no-op.
7. Press **Enter** on a focused card → detail modal opens.
8. Press **Delete** (or Backspace) on a focused card → undo toast appears; wait or click Undo.
9. After a Delete fires (no undo), focus stays on the same column at a clamped position; if column empties, focus jumps to the nearest non-empty column.
10. Open inline rename (double-click card title) — arrow keys type into the input, do NOT navigate cards.
11. Open detail modal (click body or Enter) — arrow keys behave normally inside the form, do NOT navigate cards.
12. Pick up a card with the drag grip + Space (dnd-kit keyboard sensor) — arrows reorder via dnd-kit, NOT our nav handler. After dropping (Space again), arrows resume nav.
13. Cmd+K still opens the command palette; arrows navigate the palette list, not the board.
14. `?renderer=stub` still loads.

- [ ] **Step 3: Report.** Capture failures with step + expected vs actual.

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| `BoardFocusProvider` + `useCardFocus` | 1 |
| `move` semantics + clamp effect | 1, 2 |
| `<SortableCard>` roving tabindex + onFocus | 4 |
| Arrow keys → `move(dir)` | 4 |
| Enter → opens modal | 3, 4 |
| Delete/Backspace → `stageDelete` | 4 |
| Skip empty columns on left/right | 1, 2 |
| Inline rename + modal don't trigger nav | 4 (INPUT/TEXTAREA early-return) |
| Buttons (complete/delete/kebab) don't trigger nav | 4 (BUTTON early-return) |
| dnd-kit keyboard sensor doesn't conflict | 4 (`e.defaultPrevented` early-return) |
| `<BoardFocusProvider>` mounted in `BoardView` | 5 |
| Visible focus ring | 4 |
| Bundle measurement | 6 |
| Manual smoke | 7 |

## Notes for implementer

1. **Tasks 4 + 5 commit together** — Task 4 alone leaves the tree in a state where `SortableCard` calls `useBoardFocus()` without a provider. Don't commit Task 4 alone.
2. **Default-focus rule** — only the (0,0) card gets `tabIndex=0` when `focused === null`. After any focus, `focused` is non-null and the rule is moot. Cards mounting/unmounting after the initial render keep the rule consistent because it's a derived value.
3. **Refs map cleanup** — `registerCard(col, card, null)` is called by React on unmount (cleanup phase of the ref callback). The provider's Map drops the stale entry. No manual cleanup needed.
4. **Inline rename + edit-mode** — the `<CardEditable>` `mode === 'edit'` branch renders an `<input>`. The keyboard handler on the parent SortableCard wrapper still receives bubbled events; `tag === 'INPUT'` early-return short-circuits before any nav action.
5. **dnd-kit Space-grab on the drag handle** — the handle is a `<button>`; `tag === 'BUTTON'` early-return prevents Enter on the handle from opening the modal. Space on the handle is dnd-kit's grab; that key isn't in our switch anyway.
6. **No commit amending** — forward-only commits.
