# Renderer Focus Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port the HTMX board's "Focus" feature to the React renderer — a single-column zoom view triggered from the column menu, with a card grid and exit-bar.

**Architecture:** New `FocusedColumnContext` holds the currently focused column name (or `null`). Provider wraps the board subtree in `BoardView`. `ColumnHeader` gets a "Focus" menu item; `BoardView` renders a `FocusExitBar` and filters columns; `SortableColumn` accepts an `isFocusMode` prop that switches card layout to a CSS grid. State is ephemeral — clears on board switch, on column rename/delete, and on `Esc`.

**Tech Stack:** React 18, TypeScript, Tailwind 4, Radix UI (`@radix-ui/react-dropdown-menu`), `@dnd-kit/sortable`, bun test + `@testing-library/react`.

**Spec:** `docs/superpowers/specs/2026-04-16-renderer-focus-mode-design.md`.

---

## File Structure

Files to create:

- `web/renderer/default/src/contexts/FocusedColumnContext.tsx` — provider, hook, Esc handler, auto-clear effects.
- `web/renderer/default/src/contexts/FocusedColumnContext.test.tsx` — unit tests for the provider.
- `web/renderer/default/src/components/FocusExitBar.tsx` — top bar shown during focus mode.
- `web/renderer/default/src/components/FocusExitBar.test.tsx` — tests for the bar.

Files to modify:

- `web/renderer/default/src/components/ColumnHeader.tsx` — add "Focus" menu item.
- `web/renderer/default/src/components/ColumnHeader.test.tsx` — add tests for the new item.
- `web/renderer/default/src/components/BoardView.tsx` — wrap in provider, render exit bar, filter columns.
- `web/renderer/default/src/components/BoardView.test.tsx` — add tests for focus-mode rendering.
- `web/renderer/default/src/dnd/SortableColumn.tsx` — accept `isFocusMode` prop, swap layout.

Test commands (run from `web/renderer/default/`):

- Run a single file: `bun test src/contexts/FocusedColumnContext.test.tsx`
- Run all renderer tests: `bun test`
- Typecheck: `bun run typecheck`

---

## Task 1: Create `FocusedColumnContext` with basic state

**Files:**
- Create: `web/renderer/default/src/contexts/FocusedColumnContext.tsx`
- Create: `web/renderer/default/src/contexts/FocusedColumnContext.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `web/renderer/default/src/contexts/FocusedColumnContext.test.tsx`:

```tsx
import { describe, expect, it } from 'bun:test'
import { act, renderHook } from '@testing-library/react'
import type { Column } from '@shared/types.js'
import {
  FocusedColumnProvider,
  useFocusedColumn,
} from './FocusedColumnContext.js'

const cols: Column[] = [
  { name: 'Todo', cards: [] },
  { name: 'Doing', cards: [] },
  { name: 'Done', cards: [] },
]

function wrapper(columns: Column[], active: string | null = 'b1') {
  return function Wrap({ children }: { children: React.ReactNode }) {
    return (
      <FocusedColumnProvider columns={columns} active={active}>
        {children}
      </FocusedColumnProvider>
    )
  }
}

describe('FocusedColumnContext', () => {
  it('starts with focused=null', () => {
    const { result } = renderHook(() => useFocusedColumn(), {
      wrapper: wrapper(cols),
    })
    expect(result.current.focused).toBeNull()
  })

  it('setFocused updates the value', () => {
    const { result } = renderHook(() => useFocusedColumn(), {
      wrapper: wrapper(cols),
    })
    act(() => result.current.setFocused('Todo'))
    expect(result.current.focused).toBe('Todo')
    act(() => result.current.setFocused(null))
    expect(result.current.focused).toBeNull()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web/renderer/default && bun test src/contexts/FocusedColumnContext.test.tsx`
Expected: FAIL — cannot find module `./FocusedColumnContext.js`.

- [ ] **Step 3: Write minimal implementation**

Create `web/renderer/default/src/contexts/FocusedColumnContext.tsx`:

```tsx
import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import type { Column } from '@shared/types.js'

export interface FocusedColumnCtx {
  focused: string | null
  setFocused: (name: string | null) => void
}

const Ctx = createContext<FocusedColumnCtx | null>(null)

export function FocusedColumnProvider({
  columns: _columns,
  active: _active,
  children,
}: {
  columns: Column[]
  active: string | null
  children: ReactNode
}): JSX.Element {
  const [focused, setFocused] = useState<string | null>(null)

  const value = useMemo<FocusedColumnCtx>(
    () => ({ focused, setFocused }),
    [focused],
  )

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useFocusedColumn(): FocusedColumnCtx {
  const v = useContext(Ctx)
  if (!v) throw new Error('useFocusedColumn must be used within FocusedColumnProvider')
  return v
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd web/renderer/default && bun test src/contexts/FocusedColumnContext.test.tsx`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add web/renderer/default/src/contexts/FocusedColumnContext.tsx web/renderer/default/src/contexts/FocusedColumnContext.test.tsx
git commit -m "feat(renderer): add FocusedColumnContext skeleton"
```

---

## Task 2: Clear focused on active board change

**Files:**
- Modify: `web/renderer/default/src/contexts/FocusedColumnContext.tsx`
- Modify: `web/renderer/default/src/contexts/FocusedColumnContext.test.tsx`

- [ ] **Step 1: Add the failing test**

Append to `FocusedColumnContext.test.tsx` (after the last `it`, inside the same `describe`):

```tsx
  it('clears focused when active board changes', () => {
    let active: string | null = 'b1'
    const { result, rerender } = renderHook(() => useFocusedColumn(), {
      wrapper: ({ children }: { children: React.ReactNode }) => (
        <FocusedColumnProvider columns={cols} active={active}>
          {children}
        </FocusedColumnProvider>
      ),
    })
    act(() => result.current.setFocused('Todo'))
    expect(result.current.focused).toBe('Todo')
    active = 'b2'
    rerender()
    expect(result.current.focused).toBeNull()
  })
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web/renderer/default && bun test src/contexts/FocusedColumnContext.test.tsx`
Expected: FAIL — third test fails because `focused` is still `'Todo'` after the rerender.

- [ ] **Step 3: Implement**

In `FocusedColumnContext.tsx`, replace the provider body so it resets on `active` change. Add the `useEffect` import and rename the underscored params:

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

export interface FocusedColumnCtx {
  focused: string | null
  setFocused: (name: string | null) => void
}

const Ctx = createContext<FocusedColumnCtx | null>(null)

export function FocusedColumnProvider({
  columns: _columns,
  active,
  children,
}: {
  columns: Column[]
  active: string | null
  children: ReactNode
}): JSX.Element {
  const [focused, setFocused] = useState<string | null>(null)
  const activeRef = useRef(active)

  useEffect(() => {
    if (activeRef.current !== active) {
      activeRef.current = active
      setFocused(null)
    }
  }, [active])

  const value = useMemo<FocusedColumnCtx>(
    () => ({ focused, setFocused }),
    [focused],
  )

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useFocusedColumn(): FocusedColumnCtx {
  const v = useContext(Ctx)
  if (!v) throw new Error('useFocusedColumn must be used within FocusedColumnProvider')
  return v
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd web/renderer/default && bun test src/contexts/FocusedColumnContext.test.tsx`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add web/renderer/default/src/contexts/FocusedColumnContext.tsx web/renderer/default/src/contexts/FocusedColumnContext.test.tsx
git commit -m "feat(renderer): clear focused column on active-board change"
```

---

## Task 3: Clear focused when column no longer exists

**Files:**
- Modify: `web/renderer/default/src/contexts/FocusedColumnContext.tsx`
- Modify: `web/renderer/default/src/contexts/FocusedColumnContext.test.tsx`

- [ ] **Step 1: Add the failing tests**

Append to the existing `describe('FocusedColumnContext', ...)` in `FocusedColumnContext.test.tsx`:

```tsx
  it('clears focused when the focused column is removed', () => {
    let columns: Column[] = cols
    const { result, rerender } = renderHook(() => useFocusedColumn(), {
      wrapper: ({ children }: { children: React.ReactNode }) => (
        <FocusedColumnProvider columns={columns} active="b1">
          {children}
        </FocusedColumnProvider>
      ),
    })
    act(() => result.current.setFocused('Doing'))
    expect(result.current.focused).toBe('Doing')
    columns = [cols[0]!, cols[2]!]
    rerender()
    expect(result.current.focused).toBeNull()
  })

  it('clears focused when the focused column is renamed', () => {
    let columns: Column[] = cols
    const { result, rerender } = renderHook(() => useFocusedColumn(), {
      wrapper: ({ children }: { children: React.ReactNode }) => (
        <FocusedColumnProvider columns={columns} active="b1">
          {children}
        </FocusedColumnProvider>
      ),
    })
    act(() => result.current.setFocused('Doing'))
    columns = [cols[0]!, { name: 'In Progress', cards: [] }, cols[2]!]
    rerender()
    expect(result.current.focused).toBeNull()
  })
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd web/renderer/default && bun test src/contexts/FocusedColumnContext.test.tsx`
Expected: FAIL — both new tests fail (focused stays at `'Doing'`).

- [ ] **Step 3: Implement**

In `FocusedColumnContext.tsx`, add a second effect that watches `columns` + `focused`. Put it directly after the first `useEffect`:

```tsx
  useEffect(() => {
    if (focused === null) return
    const exists = _columns.some((c) => c.name === focused)
    if (!exists) setFocused(null)
  }, [_columns, focused])
```

Rename `_columns` to `columns` in the props destructure for readability (both the param and the effect dep):

```tsx
export function FocusedColumnProvider({
  columns,
  active,
  children,
}: {
  columns: Column[]
  active: string | null
  children: ReactNode
}): JSX.Element {
  const [focused, setFocused] = useState<string | null>(null)
  const activeRef = useRef(active)

  useEffect(() => {
    if (activeRef.current !== active) {
      activeRef.current = active
      setFocused(null)
    }
  }, [active])

  useEffect(() => {
    if (focused === null) return
    const exists = columns.some((c) => c.name === focused)
    if (!exists) setFocused(null)
  }, [columns, focused])

  const value = useMemo<FocusedColumnCtx>(
    () => ({ focused, setFocused }),
    [focused],
  )

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd web/renderer/default && bun test src/contexts/FocusedColumnContext.test.tsx`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add web/renderer/default/src/contexts/FocusedColumnContext.tsx web/renderer/default/src/contexts/FocusedColumnContext.test.tsx
git commit -m "feat(renderer): auto-clear focused column on rename/delete"
```

---

## Task 4: Global `Esc` handler (with modal/input guards)

**Files:**
- Modify: `web/renderer/default/src/contexts/FocusedColumnContext.tsx`
- Modify: `web/renderer/default/src/contexts/FocusedColumnContext.test.tsx`

- [ ] **Step 1: Add the failing tests**

Append to the `describe('FocusedColumnContext', ...)` block:

```tsx
  it('Escape keydown clears focused', () => {
    const { result } = renderHook(() => useFocusedColumn(), {
      wrapper: wrapper(cols),
    })
    act(() => result.current.setFocused('Todo'))
    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    })
    expect(result.current.focused).toBeNull()
  })

  it('Escape is ignored while an input is focused', () => {
    const input = document.createElement('input')
    document.body.appendChild(input)
    input.focus()
    try {
      const { result } = renderHook(() => useFocusedColumn(), {
        wrapper: wrapper(cols),
      })
      act(() => result.current.setFocused('Todo'))
      act(() => {
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
      })
      expect(result.current.focused).toBe('Todo')
    } finally {
      input.remove()
    }
  })

  it('Escape is ignored while a Radix dialog is open', () => {
    const dialog = document.createElement('div')
    dialog.setAttribute('role', 'dialog')
    dialog.setAttribute('data-state', 'open')
    document.body.appendChild(dialog)
    try {
      const { result } = renderHook(() => useFocusedColumn(), {
        wrapper: wrapper(cols),
      })
      act(() => result.current.setFocused('Todo'))
      act(() => {
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
      })
      expect(result.current.focused).toBe('Todo')
    } finally {
      dialog.remove()
    }
  })

  it('does not handle Escape when no column is focused', () => {
    const { result } = renderHook(() => useFocusedColumn(), {
      wrapper: wrapper(cols),
    })
    // No-op should not throw.
    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    })
    expect(result.current.focused).toBeNull()
  })
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd web/renderer/default && bun test src/contexts/FocusedColumnContext.test.tsx`
Expected: FAIL on the "Escape keydown clears focused" test (others may pass incidentally because focused is already `null` or stays at `Todo`, but the first will fail).

- [ ] **Step 3: Implement the Esc listener**

In `FocusedColumnContext.tsx`, add a third `useEffect` that attaches a window `keydown` listener while `focused !== null`:

```tsx
  useEffect(() => {
    if (focused === null) return
    function onKey(e: KeyboardEvent): void {
      if (e.key !== 'Escape') return
      // Ignore when typing in an input/textarea/contenteditable.
      const el = document.activeElement as HTMLElement | null
      if (el) {
        const tag = el.tagName
        if (tag === 'INPUT' || tag === 'TEXTAREA' || el.isContentEditable) return
      }
      // Ignore when a Radix (or compatible) dialog is open.
      if (document.querySelector('[role="dialog"][data-state="open"]')) return
      setFocused(null)
    }
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('keydown', onKey)
    }
  }, [focused])
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd web/renderer/default && bun test src/contexts/FocusedColumnContext.test.tsx`
Expected: PASS (9 tests).

- [ ] **Step 5: Commit**

```bash
git add web/renderer/default/src/contexts/FocusedColumnContext.tsx web/renderer/default/src/contexts/FocusedColumnContext.test.tsx
git commit -m "feat(renderer): wire Esc to exit focus mode (input/dialog guarded)"
```

---

## Task 5: `FocusExitBar` component

**Files:**
- Create: `web/renderer/default/src/components/FocusExitBar.tsx`
- Create: `web/renderer/default/src/components/FocusExitBar.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `web/renderer/default/src/components/FocusExitBar.test.tsx`:

```tsx
import { describe, expect, it } from 'bun:test'
import { act, fireEvent, render } from '@testing-library/react'
import type { Column } from '@shared/types.js'
import {
  FocusedColumnProvider,
  useFocusedColumn,
} from '../contexts/FocusedColumnContext.js'
import { FocusExitBar } from './FocusExitBar.js'

const cols: Column[] = [{ name: 'Todo', cards: [] }]

function Probe({ initial }: { initial: string }): JSX.Element {
  const { focused, setFocused } = useFocusedColumn()
  // Set once on first render.
  if (focused === null && initial) {
    queueMicrotask(() => setFocused(initial))
  }
  return <FocusExitBar />
}

describe('FocusExitBar', () => {
  it('renders with the focused column name', async () => {
    const { findByText } = render(
      <FocusedColumnProvider columns={cols} active="b1">
        <Probe initial="Todo" />
      </FocusedColumnProvider>,
    )
    expect(await findByText(/Focusing:/)).toBeDefined()
    expect(await findByText('Todo')).toBeDefined()
  })

  it('renders nothing when no column is focused', () => {
    const { container } = render(
      <FocusedColumnProvider columns={cols} active="b1">
        <FocusExitBar />
      </FocusedColumnProvider>,
    )
    expect(container.firstChild).toBeNull()
  })

  it('exit button clears focused', async () => {
    function Host(): JSX.Element {
      const { focused } = useFocusedColumn()
      return (
        <>
          <Probe initial="Todo" />
          <span data-testid="state">{focused ?? 'null'}</span>
        </>
      )
    }
    const { findByText, getByTestId } = render(
      <FocusedColumnProvider columns={cols} active="b1">
        <Host />
      </FocusedColumnProvider>,
    )
    const btn = await findByText(/Exit Focus/)
    act(() => {
      fireEvent.click(btn)
    })
    expect(getByTestId('state').textContent).toBe('null')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web/renderer/default && bun test src/components/FocusExitBar.test.tsx`
Expected: FAIL — cannot find module `./FocusExitBar.js`.

- [ ] **Step 3: Implement**

Create `web/renderer/default/src/components/FocusExitBar.tsx`:

```tsx
import { useFocusedColumn } from '../contexts/FocusedColumnContext.js'

export function FocusExitBar(): JSX.Element | null {
  const { focused, setFocused } = useFocusedColumn()
  if (focused === null) return null
  return (
    <div className="mb-3 flex shrink-0 items-center justify-between rounded-md border border-slate-200 bg-white px-4 py-2 dark:border-slate-700 dark:bg-slate-900">
      <span className="text-sm font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
        Focusing: <span className="text-slate-800 dark:text-slate-100">{focused}</span>
      </span>
      <button
        type="button"
        onClick={() => setFocused(null)}
        className="inline-flex items-center gap-1.5 rounded border border-slate-200 bg-slate-50 px-3 py-1 text-xs font-semibold text-slate-600 transition-colors hover:border-[color:var(--accent-500)] hover:bg-[color:var(--accent-500)] hover:text-white dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300"
      >
        Exit Focus
        <span className="rounded border border-current px-1 py-0.5 text-[10px] opacity-60">Esc</span>
      </button>
    </div>
  )
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd web/renderer/default && bun test src/components/FocusExitBar.test.tsx`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add web/renderer/default/src/components/FocusExitBar.tsx web/renderer/default/src/components/FocusExitBar.test.tsx
git commit -m "feat(renderer): add FocusExitBar component"
```

---

## Task 6: Add "Focus" item to the column menu

**Files:**
- Modify: `web/renderer/default/src/components/ColumnHeader.tsx`
- Modify: `web/renderer/default/src/components/ColumnHeader.test.tsx`

- [ ] **Step 1: Add failing tests**

At the top of `ColumnHeader.test.tsx`, add the provider import:

```tsx
import type { Column } from '@shared/types.js'
import { FocusedColumnProvider } from '../contexts/FocusedColumnContext.js'
```

Then append two new tests to the existing `describe('ColumnHeader', ...)`:

```tsx
  it('shows a Focus menu item that sets the focused column', async () => {
    const { client, qc } = await setup()
    const cols: Column[] = [
      { name: 'Todo', cards: [] },
      { name: 'Doing', cards: [] },
    ]
    let currentFocused: string | null = null
    function Spy(): null {
      // Capture the current focused value after each render.
      const { useFocusedColumn } = require('../contexts/FocusedColumnContext.js')
      currentFocused = useFocusedColumn().focused
      return null
    }
    const { getByLabelText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <FocusedColumnProvider columns={cols} active="welcome">
          <Spy />
          <ColumnHeader name="Todo" cardCount={0} colIdx={0} allColumnNames={['Todo','Doing']} boardId="welcome" />
        </FocusedColumnProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.pointerDown(getByLabelText('column menu Todo'), { button: 0, pointerType: 'mouse' })
    fireEvent.click(await findByText('Focus'))
    expect(currentFocused).toBe('Todo')
  })

  it('hides the Focus item when this column is already focused', async () => {
    const { client, qc } = await setup()
    const cols: Column[] = [{ name: 'Todo', cards: [] }]
    function Seed(): null {
      const { useFocusedColumn } = require('../contexts/FocusedColumnContext.js')
      const { setFocused, focused } = useFocusedColumn()
      if (focused === null) queueMicrotask(() => setFocused('Todo'))
      return null
    }
    const { getByLabelText, queryByText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <FocusedColumnProvider columns={cols} active="welcome">
          <Seed />
          <ColumnHeader name="Todo" cardCount={0} colIdx={0} allColumnNames={['Todo']} boardId="welcome" />
        </FocusedColumnProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.pointerDown(getByLabelText('column menu Todo'), { button: 0, pointerType: 'mouse' })
    // Rename should still be present in both states.
    expect(await findByText('Rename')).toBeDefined()
    expect(queryByText('Focus')).toBeNull()
  })
```

Note: the existing tests in this file already render `ColumnHeader` without a `FocusedColumnProvider`. The implementation in Step 3 must therefore treat missing context as "no focus mode" (use `useContext` directly, not the throwing `useFocusedColumn`).

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd web/renderer/default && bun test src/components/ColumnHeader.test.tsx`
Expected: FAIL on the two new tests — "Focus" item doesn't exist yet.

- [ ] **Step 3: Implement**

In `ColumnHeader.tsx`, add an optional focus hook and a new menu item. Near the top:

```tsx
import { useContext } from 'react'
// Use the raw context handle so ColumnHeader can be rendered outside a
// FocusedColumnProvider (some tests, embedded previews).
import { FocusedColumnContext } from '../contexts/FocusedColumnContext.js'
```

In the component body, before the `if (mode === 'edit')` branch, read the context:

```tsx
  const focusCtx = useContext(FocusedColumnContext)
  const isFocused = focusCtx?.focused === name
```

Inside the `DropdownMenu.Content`, add a new `DropdownMenu.Item` before the "Rename" item:

```tsx
              {focusCtx && !isFocused && (
                <DropdownMenu.Item
                  onSelect={() => focusCtx.setFocused(name)}
                  className="cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-slate-100 dark:hover:bg-slate-700"
                >
                  Focus
                </DropdownMenu.Item>
              )}
```

Also export the context from `FocusedColumnContext.tsx` so ColumnHeader can import it. In `FocusedColumnContext.tsx`, change:

```tsx
const Ctx = createContext<FocusedColumnCtx | null>(null)
```

to:

```tsx
export const FocusedColumnContext = createContext<FocusedColumnCtx | null>(null)
```

and update the two internal references (`Ctx.Provider`, `useContext(Ctx)`) to use `FocusedColumnContext`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd web/renderer/default && bun test src/components/ColumnHeader.test.tsx src/contexts/FocusedColumnContext.test.tsx`
Expected: PASS (all 5 ColumnHeader tests + all 9 FocusedColumnContext tests).

- [ ] **Step 5: Commit**

```bash
git add web/renderer/default/src/components/ColumnHeader.tsx web/renderer/default/src/components/ColumnHeader.test.tsx web/renderer/default/src/contexts/FocusedColumnContext.tsx
git commit -m "feat(renderer): add 'Focus' item to column menu"
```

---

## Task 7: `SortableColumn` focus-mode layout

**Files:**
- Modify: `web/renderer/default/src/dnd/SortableColumn.tsx`

No direct test file for `SortableColumn` exists today (it's exercised via `BoardView.test.tsx` in Task 8). We'll change the rendering logic here, then verify via Task 8's tests.

- [ ] **Step 1: Add `isFocusMode` prop and branch layout**

Edit `web/renderer/default/src/dnd/SortableColumn.tsx`:

Add the new prop to the signature:

```tsx
export function SortableColumn({
  column,
  colIdx,
  allColumnNames,
  boardId,
  collapsed = false,
  filterQuery = '',
  hideCompleted = false,
  isFocusMode = false,
}: {
  column: ColumnModel
  colIdx: number
  allColumnNames: string[]
  boardId: string
  collapsed?: boolean
  filterQuery?: string
  hideCompleted?: boolean
  isFocusMode?: boolean
}): JSX.Element {
```

Immediately after the destructure of visible cards and the call to `useSortable`, short-circuit the collapsed branch when focus mode is active. Replace:

```tsx
  if (collapsed) {
    return (
      ...
    )
  }
```

with:

```tsx
  if (collapsed && !isFocusMode) {
    return (
      ...  // existing collapsed rendering unchanged
    )
  }
```

Change the outer `<section>`'s className in the default branch to honour focus mode:

```tsx
  const sectionClass = isFocusMode
    ? 'flex w-full flex-1 flex-col rounded-lg bg-slate-100 p-3 dark:bg-slate-900'
    : `flex ${settings.expand_columns ? 'min-w-[200px] flex-[1_1_0]' : 'w-72 shrink-0'} flex-col rounded-lg bg-slate-100 p-3 dark:bg-slate-900`
```

and use `className={sectionClass}` on the `<section>`.

Inside the default branch, hide the column drag handle when `isFocusMode`:

```tsx
        {!isFocusMode && (
          <button
            type="button"
            aria-label={`drag column ${column.name}`}
            {...attributes}
            {...listeners}
            className="cursor-grab text-slate-400 hover:text-slate-600 active:cursor-grabbing"
          >
            ⋮⋮
          </button>
        )}
```

Swap the cards `<ul>` classes:

```tsx
        <ul
          className={
            isFocusMode
              ? 'grid grid-cols-[repeat(auto-fill,minmax(280px,1fr))] gap-2.5 overflow-y-auto'
              : 'flex flex-col gap-2'
          }
        >
```

- [ ] **Step 2: Typecheck**

Run: `cd web/renderer/default && bun run typecheck`
Expected: no errors.

- [ ] **Step 3: Run full test suite — existing tests must still pass**

Run: `cd web/renderer/default && bun test`
Expected: all pre-existing tests still pass (new `isFocusMode` prop has a default of `false`, so callers that don't set it are unaffected).

- [ ] **Step 4: Commit**

```bash
git add web/renderer/default/src/dnd/SortableColumn.tsx
git commit -m "feat(renderer): add isFocusMode layout branch to SortableColumn"
```

---

## Task 8: Wire focus mode into `BoardView`

**Files:**
- Modify: `web/renderer/default/src/components/BoardView.tsx`
- Modify: `web/renderer/default/src/components/BoardView.test.tsx`

- [ ] **Step 1: Add failing tests**

At the top of `BoardView.test.tsx`, add imports for the dropdown trigger test:

```tsx
import { fireEvent } from '@testing-library/react'
```

Append new tests to `describe('BoardView', ...)`:

```tsx
  it('renders only the focused column and an exit bar while in focus mode', async () => {
    const client = await setup()
    const { getByText, queryByText, getByLabelText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <BoardView client={client} />
        </ActiveBoardProvider>
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Todo')).toBeDefined())
    // Enter focus mode via the Todo column menu.
    fireEvent.pointerDown(getByLabelText('column menu Todo'), { button: 0, pointerType: 'mouse' })
    fireEvent.click(await findByText('Focus'))
    // Exit bar is visible.
    await waitFor(() => expect(getByText(/Focusing:/)).toBeDefined())
    // Other columns are no longer in the DOM.
    expect(queryByText('Doing')).toBeNull()
    expect(queryByText('Done')).toBeNull()
    // "Add list" button is hidden.
    expect(queryByText('+ Add list')).toBeNull()
  })

  it('exits focus mode via the exit bar button', async () => {
    const client = await setup()
    const { getByText, queryByText, getByLabelText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <BoardView client={client} />
        </ActiveBoardProvider>
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Todo')).toBeDefined())
    fireEvent.pointerDown(getByLabelText('column menu Todo'), { button: 0, pointerType: 'mouse' })
    fireEvent.click(await findByText('Focus'))
    await waitFor(() => expect(getByText(/Focusing:/)).toBeDefined())
    fireEvent.click(getByText(/Exit Focus/))
    // All three columns back.
    await waitFor(() => expect(getByText('Doing')).toBeDefined())
    expect(getByText('Done')).toBeDefined()
    expect(queryByText(/Focusing:/)).toBeNull()
  })
```

Helper `setup` needs to also set query data (copy from `ColumnHeader.test.tsx` pattern) so that columns render synchronously. Update the existing `setup`:

```tsx
async function setup(): Promise<Client> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  return client
}
```

(Leave as-is — it already works for the existing tests that `await waitFor`. Keep the new tests using `await waitFor` similarly.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd web/renderer/default && bun test src/components/BoardView.test.tsx`
Expected: FAIL on the two new tests — "Focus" menu item isn't wired into `BoardView`'s `SortableColumn` render because `BoardView` doesn't yet host the `FocusedColumnProvider`.

- [ ] **Step 3: Implement in `BoardView.tsx`**

At the top of `BoardView.tsx`, add the imports:

```tsx
import { FocusedColumnProvider, useFocusedColumn } from '../contexts/FocusedColumnContext.js'
import { FocusExitBar } from './FocusExitBar.js'
```

Extract the inner board render into a component that consumes the focus hook, because hooks can't be called next to the provider that owns them. Add this helper component in the same file:

```tsx
function BoardContent({
  data,
  active,
  columns,
  filterQuery,
  setFilterQuery,
  hideCompleted,
  toggleHideCompleted,
  setSettingsOpen,
}: {
  data: NonNullable<ReturnType<typeof useBoard>['data']>
  active: string
  columns: ReturnType<typeof useBoard>['data'] extends infer T
    ? T extends { columns?: infer C } ? NonNullable<C> : never
    : never
  filterQuery: string
  setFilterQuery: (s: string) => void
  hideCompleted: boolean
  toggleHideCompleted: () => void
  setSettingsOpen: (open: boolean) => void
}): JSX.Element {
  const { focused } = useFocusedColumn()
  const names = columns.map((c) => c.name)
  const visibleColumns = focused !== null
    ? columns.filter((c) => c.name === focused)
    : columns
  const columnIds = names.map(encodeColumnId)

  const columnsRow = (
    <div className="flex flex-1 gap-4 overflow-x-auto p-4">
      {visibleColumns.map((col) => {
        const i = columns.indexOf(col)
        return (
          <SortableColumn
            key={`${col.name}-${i}`}
            column={col}
            colIdx={i}
            allColumnNames={names}
            boardId={active}
            collapsed={data.list_collapse?.[i] ?? false}
            filterQuery={filterQuery}
            hideCompleted={hideCompleted}
            isFocusMode={focused !== null}
          />
        )
      })}
      {focused === null && <AddColumnButton boardId={active} />}
    </div>
  )

  return (
    <div className="flex h-full flex-col">
      <div className="flex h-12 shrink-0 items-center gap-3 border-b border-slate-200 px-4 dark:border-slate-800">
        {/* header unchanged — copy the existing header JSX here */}
      </div>
      <div className="flex flex-1 flex-col p-4 pt-0">
        <FocusExitBar />
        {focused !== null ? (
          columnsRow
        ) : (
          <SortableContext items={columnIds} strategy={horizontalListSortingStrategy}>
            {columnsRow}
          </SortableContext>
        )}
      </div>
    </div>
  )
}
```

Then rewrite the `BoardView` return to host the provider and render `BoardContent`:

```tsx
  return (
    <BoardFocusProvider columns={columns}>
      <FocusedColumnProvider columns={columns} active={active}>
        <BoardDndContext boardId={active}>
          <BoardContent
            data={data}
            active={active}
            columns={columns}
            filterQuery={filterQuery}
            setFilterQuery={setFilterQuery}
            hideCompleted={hideCompleted}
            toggleHideCompleted={toggleHideCompleted}
            setSettingsOpen={setSettingsOpen}
          />
        </BoardDndContext>
        <Suspense fallback={null}>
          <BoardSettingsModal
            boardId={active}
            boardName={data.name}
            open={settingsOpen}
            onOpenChange={setSettingsOpen}
          />
        </Suspense>
      </FocusedColumnProvider>
    </BoardFocusProvider>
  )
```

Paste the existing header JSX (icon, title, settings trigger, filter input, hideCompleted toggle) in place of the `{/* header unchanged ... */}` placeholder in `BoardContent`, wiring the props through. The only behaviour change is that the whole header keeps rendering the same way in focus mode — matching the HTMX reference.

Keep both `SortableContext` and `BoardDndContext` mounted when focus is off (unchanged DnD). When `focused !== null`, we skip the horizontal `SortableContext` (only card-level DnD inside `SortableColumn` matters).

- [ ] **Step 4: Typecheck**

Run: `cd web/renderer/default && bun run typecheck`
Expected: no errors. If `columns` type helper in `BoardContent` props is too fancy, replace with `Column[]` imported from `@shared/types.js` and `data: BoardData` (use whatever shape `useBoard` returns — check `queries.tsx`).

- [ ] **Step 5: Run the BoardView tests**

Run: `cd web/renderer/default && bun test src/components/BoardView.test.tsx`
Expected: PASS — all existing BoardView tests + 2 new ones.

- [ ] **Step 6: Run the full renderer test suite**

Run: `cd web/renderer/default && bun test`
Expected: all tests pass.

- [ ] **Step 7: Manual smoke test**

Start the dev environment:

```bash
make dev
```

In the browser:

1. Open a board with 3+ columns.
2. Click a column's `⋮` menu → click **Focus**.
3. Confirm: only that column is visible, exit bar at top reads "Focusing: X", cards arranged as a responsive grid, "+ Add list" button hidden.
4. Press **Esc** → focus exits, all columns return.
5. Re-enter focus, switch boards via sidebar → focus clears on the new board.
6. Re-enter focus, rename the focused column via its menu → focus exits (column name no longer exists in the list).
7. Re-enter focus, drag a card within the focused column → still works.
8. Open a card modal while in focus mode, press Esc → modal closes (not focus mode).

- [ ] **Step 8: Commit**

```bash
git add web/renderer/default/src/components/BoardView.tsx web/renderer/default/src/components/BoardView.test.tsx
git commit -m "feat(renderer): wire focus mode into BoardView"
```

---

## Task 9: Final housekeeping

- [ ] **Step 1: Run lint**

Run from the repo root: `make lint`
Expected: no errors.

- [ ] **Step 2: Typecheck everything**

```bash
cd web/renderer/default && bun run typecheck
```

Expected: no errors.

- [ ] **Step 3: Run the full renderer test suite one more time**

```bash
cd web/renderer/default && bun test
```

Expected: all green.

- [ ] **Step 4: Commit any lint fixups**

If `make lint` produced changes:

```bash
git add -u
git commit -m "chore: lint fixups for focus mode"
```

---

## Self-review

Against the spec (`docs/superpowers/specs/2026-04-16-renderer-focus-mode-design.md`):

- [x] New `FocusedColumnContext` with `{ focused, setFocused }` — Task 1
- [x] Ephemeral; clears on active board change — Task 2
- [x] Clears when focused column disappears (rename/delete) — Task 3
- [x] Esc handler with input + dialog guards — Task 4
- [x] `FocusExitBar` with "Focusing: X" label and "Exit Focus [Esc]" button — Task 5
- [x] "Focus" item in `ColumnHeader` dropdown, hidden when already focused — Task 6
- [x] `SortableColumn` `isFocusMode` — full width, card grid, collapsed bypass, drag handle hidden — Task 7
- [x] `BoardView` wraps subtree with provider, renders exit bar, filters to single column, hides `AddColumnButton`, skips horizontal `SortableContext` — Task 8
- [x] DnD within focused column still works (card-level `SortableContext` unchanged inside `SortableColumn`) — Task 7 does not touch the card context
- [x] Filter query and hide-completed still apply (reuse existing `SortableColumn` filtering) — no change needed; existing code path runs
- [x] Tests cover state, Esc guards, menu item, rename/delete clears, focus-mode rendering in BoardView — Tasks 1–8

No placeholder text remains. Method signatures used in later tasks (`setFocused`, `useFocusedColumn`, `FocusedColumnContext`, `isFocusMode`) match their definitions.
