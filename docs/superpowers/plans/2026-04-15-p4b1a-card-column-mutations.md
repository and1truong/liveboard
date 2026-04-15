# P4b.1a — Card + Column Mutations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the P4a read-only renderer into an interactive kanban for card and column operations (add/edit/toggle-complete/delete card; add/rename/reorder/delete column), using TanStack `useMutation` for optimistic UI and a 5-second undo toast for destructive ops.

**Architecture:** A single `useBoardMutation` hook routes every mutation through TanStack Query's `useMutation`. `onMutate` applies the op to the cached board via shared `applyOp` (which already clones). `onSuccess` replaces the cache with the server-returned board. `onError` rolls back; `VERSION_CONFLICT` additionally invalidates. Destructive ops fire after a 5s timer that the undo toast can clear. Nothing in `web/shared/*` changes.

**Tech Stack:** Existing P4a stack (React 18, TanStack Query v5, Tailwind v4, Vite, bun test + happy-dom) plus two new deps: `sonner` (toasts) and `@radix-ui/react-dropdown-menu`.

**Spec:** `docs/superpowers/specs/2026-04-15-p4b1a-card-column-mutations-design.md`

**Conventions:**
- All new code under `web/renderer/default/src/`.
- Imports use the `@shared/*` path alias (established in P4a).
- Tests colocated (`*.test.tsx`).
- Commit prefixes: `feat(renderer)`, `test(renderer)`, `chore(build)`.
- Pre-existing `TS6196` in `web/shared/src/protocol.ts` is NOT a blocker.

---

## File structure

**New:**
- `web/renderer/default/src/toast.ts` — sonner re-export + `<ToastHost />` mount
- `web/renderer/default/src/mutations/useBoardMutation.ts`
- `web/renderer/default/src/mutations/useBoardMutation.test.tsx`
- `web/renderer/default/src/mutations/undoable.ts`
- `web/renderer/default/src/mutations/undoable.test.ts`
- `web/renderer/default/src/mutations/moveColumn.ts` — `moveColumnTarget` helper for `after_col`
- `web/renderer/default/src/mutations/moveColumn.test.ts`
- `web/renderer/default/src/components/CardEditable.tsx`
- `web/renderer/default/src/components/CardEditable.test.tsx`
- `web/renderer/default/src/components/AddCardButton.tsx`
- `web/renderer/default/src/components/AddCardButton.test.tsx`
- `web/renderer/default/src/components/ColumnHeader.tsx`
- `web/renderer/default/src/components/ColumnHeader.test.tsx`
- `web/renderer/default/src/components/AddColumnButton.tsx`
- `web/renderer/default/src/components/AddColumnButton.test.tsx`

**Modified:**
- `web/renderer/default/package.json` — add `sonner`, `@radix-ui/react-dropdown-menu`
- `web/renderer/default/src/queries.tsx` — export `useClient`
- `web/renderer/default/src/components/Column.tsx` — use `ColumnHeader`, `CardEditable`, `AddCardButton`; pass `col_idx`, `total_cols`, `boardId`
- `web/renderer/default/src/components/BoardView.tsx` — pass indices to `Column`; render `AddColumnButton`
- `web/renderer/default/src/App.tsx` — mount `<Toaster />`

---

## Task 1: Install sonner + Radix DropdownMenu

**Files:**
- Modify: `web/renderer/default/package.json`

- [ ] **Step 1: Add deps**

Edit `web/renderer/default/package.json`. Under `"dependencies"`, add (alphabetical):
```json
    "@radix-ui/react-dropdown-menu": "^2.1.2",
    "sonner": "^1.7.0",
```
Resulting `dependencies` block looks like:
```json
  "dependencies": {
    "@radix-ui/react-dropdown-menu": "^2.1.2",
    "@tanstack/react-query": "^5.59.0",
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "sonner": "^1.7.0"
  },
```

- [ ] **Step 2: Install**

```bash
cd web/renderer/default && bun install
```
Expected: clean install, `bun.lock` updated.

- [ ] **Step 3: Smoke-import to confirm resolution**

```bash
cd web/renderer/default && bun -e "import('sonner').then(m=>console.log(typeof m.toast)); import('@radix-ui/react-dropdown-menu').then(m=>console.log(typeof m.Root))"
```
Expected: prints `function` twice.

- [ ] **Step 4: Commit**

```bash
git add web/renderer/default/package.json web/renderer/default/bun.lock
git commit -m "chore(build): add sonner and radix dropdown-menu deps

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: Toast host + `toast.ts`

**Files:**
- Create: `web/renderer/default/src/toast.ts`
- Modify: `web/renderer/default/src/App.tsx`

- [ ] **Step 1: `src/toast.ts`**

```ts
import { toast, Toaster } from 'sonner'

export { toast, Toaster }

export function errorToast(code: string): void {
  const copy: Record<string, string> = {
    VERSION_CONFLICT: 'Board changed elsewhere — refreshed',
    NOT_FOUND: 'Card or column not found',
    OUT_OF_RANGE: 'Index out of range',
    INVALID: 'Invalid input',
    INTERNAL: 'Server error — try again',
  }
  toast.error(copy[code] ?? code)
}
```

- [ ] **Step 2: Mount `<Toaster />` in `App.tsx`**

Read current `web/renderer/default/src/App.tsx`. Replace its body with:
```tsx
import { useState } from 'react'
import type { Client } from '@shared/client.js'
import { BoardSidebar } from './components/BoardSidebar.js'
import { BoardView } from './components/BoardView.js'
import { Toaster } from './toast.js'

export function App({ client }: { client: Client }): JSX.Element {
  const [activeId, setActiveId] = useState<string | null>(null)
  return (
    <div className="flex h-screen w-screen">
      <BoardSidebar activeId={activeId} onSelect={setActiveId} />
      <main className="flex-1 overflow-hidden">
        <BoardView boardId={activeId} client={client} />
      </main>
      <Toaster position="bottom-right" richColors closeButton />
    </div>
  )
}
```

- [ ] **Step 3: Run existing tests**

```bash
cd web/renderer/default && bun test && bun run typecheck
```
Expected: 22 pass (same as P4a baseline), no new typecheck errors.

- [ ] **Step 4: Commit**

```bash
git add web/renderer/default/src/toast.ts web/renderer/default/src/App.tsx
git commit -m "feat(renderer): add sonner toast host and errorToast helper

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: Export `useClient` from `queries.tsx`

**Files:**
- Modify: `web/renderer/default/src/queries.tsx`

The existing `useClient` is module-private; mutation hooks need it.

- [ ] **Step 1: Flip visibility**

Open `web/renderer/default/src/queries.tsx`. Find:
```tsx
function useClient(): Client {
```
Change to:
```tsx
export function useClient(): Client {
```
No other changes.

- [ ] **Step 2: Verify tests still pass**

```bash
cd web/renderer/default && bun test && bun run typecheck
```
Expected: all green.

- [ ] **Step 3: Commit**

```bash
git add web/renderer/default/src/queries.tsx
git commit -m "feat(renderer): export useClient for mutation hooks

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 4: `useBoardMutation` hook + tests

**Files:**
- Create: `web/renderer/default/src/mutations/useBoardMutation.ts`
- Create: `web/renderer/default/src/mutations/useBoardMutation.test.tsx`

- [ ] **Step 1: Write the hook**

Create `web/renderer/default/src/mutations/useBoardMutation.ts`:
```ts
import { useMutation, useQueryClient, type UseMutationResult } from '@tanstack/react-query'
import type { Board, MutationOp } from '@shared/types.js'
import { applyOp } from '@shared/boardOps.js'
import { ProtocolError } from '@shared/protocol.js'
import { useClient } from '../queries.js'
import { errorToast } from '../toast.js'

interface Ctx {
  prev?: Board
}

export function useBoardMutation(
  boardId: string,
): UseMutationResult<Board, Error, MutationOp, Ctx> {
  const client = useClient()
  const qc = useQueryClient()

  return useMutation<Board, Error, MutationOp, Ctx>({
    mutationFn: (op) => {
      const cached = qc.getQueryData<Board>(['board', boardId])
      return client.mutateBoard(boardId, cached?.version ?? -1, op)
    },
    onMutate: async (op) => {
      await qc.cancelQueries({ queryKey: ['board', boardId] })
      const prev = qc.getQueryData<Board>(['board', boardId])
      if (prev) {
        try {
          qc.setQueryData(['board', boardId], applyOp(prev, op))
        } catch {
          // Optimistic apply failed (e.g. stale indices). Keep prev; let server reject.
        }
      }
      return { prev }
    },
    onSuccess: (board) => {
      qc.setQueryData(['board', boardId], board)
    },
    onError: (err, _op, ctx) => {
      if (ctx?.prev) qc.setQueryData(['board', boardId], ctx.prev)
      const code = err instanceof ProtocolError ? err.code : 'INTERNAL'
      if (code === 'VERSION_CONFLICT') {
        void qc.invalidateQueries({ queryKey: ['board', boardId] })
      }
      errorToast(code)
    },
  })
}
```

- [ ] **Step 2: Write the test**

Create `web/renderer/default/src/mutations/useBoardMutation.test.tsx`:
```tsx
import { describe, expect, it } from 'bun:test'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { renderWithQuery } from '../test-utils.js'
import { useBoardMutation } from './useBoardMutation.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return { client, qc }
}

describe('useBoardMutation', () => {
  it('optimistically applies add_card and server confirms', async () => {
    const { client, qc } = await setup()
    // Seed cache by fetching once
    qc.setQueryData(['board', 'welcome'], await client.getBoard('welcome'))

    const { result } = renderHook(() => useBoardMutation('welcome'), {
      wrapper: ({ children }) => (
        <ClientProvider client={client}>
          {renderWithQueryWrapper(children, qc)}
        </ClientProvider>
      ),
    })

    result.current.mutate({ type: 'add_card', column: 'Todo', title: 'OPT' })

    // Optimistic: title visible in cache before mutation resolves
    await waitFor(() => {
      const b = qc.getQueryData<any>(['board', 'welcome'])
      const todo = b?.columns?.find((c: any) => c.name === 'Todo')
      expect(todo?.cards?.some((c: any) => c.title === 'OPT')).toBe(true)
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    // Server result wrote into cache with bumped version
    const final = qc.getQueryData<any>(['board', 'welcome'])
    expect(final.version).toBeGreaterThanOrEqual(2)
  })

  it('rolls back on VERSION_CONFLICT and invalidates', async () => {
    const { client, qc } = await setup()
    const real = await client.getBoard('welcome')
    // Seed cache with a stale version
    qc.setQueryData(['board', 'welcome'], { ...real, version: 0 })

    const { result } = renderHook(() => useBoardMutation('welcome'), {
      wrapper: ({ children }) => (
        <ClientProvider client={client}>
          {renderWithQueryWrapper(children, qc)}
        </ClientProvider>
      ),
    })

    result.current.mutate({ type: 'add_card', column: 'Todo', title: 'BAD' })
    await waitFor(() => expect(result.current.isError).toBe(true))

    // Rollback: cache version matches the seeded stale snapshot (pre-mutation)
    const after = qc.getQueryData<any>(['board', 'welcome'])
    expect(after.version).toBe(0)
  })
})

// Small helper: wrap in QueryClientProvider without importing renderWithQuery twice
import { QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
function renderWithQueryWrapper(children: ReactNode, qc: QueryClient): JSX.Element {
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}
```

- [ ] **Step 3: Run**

```bash
cd web/renderer/default && bun test src/mutations/useBoardMutation.test.tsx && bun run typecheck
```
Expected: 2 pass, no new typecheck errors.

If the rollback test fails because the stale-version cache triggered optimistic apply into an invalid state, note that the try/catch in `onMutate` swallows apply errors — that's intentional.

- [ ] **Step 4: Commit**

```bash
git add web/renderer/default/src/mutations/useBoardMutation.ts web/renderer/default/src/mutations/useBoardMutation.test.tsx
git commit -m "feat(renderer): add useBoardMutation with optimistic + rollback

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 5: `undoable.ts` — staged delete

**Files:**
- Create: `web/renderer/default/src/mutations/undoable.ts`
- Create: `web/renderer/default/src/mutations/undoable.test.ts`

- [ ] **Step 1: Write the module**

Create `web/renderer/default/src/mutations/undoable.ts`:
```ts
import type { MutationOp } from '@shared/types.js'
import type { UseMutationResult } from '@tanstack/react-query'
import { toast } from '../toast.js'

const UNDO_MS = 5000

export function stageDelete(
  mutation: UseMutationResult<unknown, unknown, MutationOp, unknown>,
  op: MutationOp,
  label: string,
): void {
  let undone = false
  const timer = setTimeout(() => {
    if (!undone) mutation.mutate(op)
  }, UNDO_MS)

  toast(`Deleted ${label}`, {
    duration: UNDO_MS,
    action: {
      label: 'Undo',
      onClick: () => {
        undone = true
        clearTimeout(timer)
      },
    },
  })
}
```

- [ ] **Step 2: Write tests using bun mock timers**

Create `web/renderer/default/src/mutations/undoable.test.ts`:
```ts
import { describe, expect, it, mock } from 'bun:test'
import { stageDelete } from './undoable.js'

function fakeMutation() {
  return { mutate: mock(() => {}) } as any
}

describe('stageDelete', () => {
  it('fires mutation after 5s when undo not clicked', async () => {
    const m = fakeMutation()
    stageDelete(m, { type: 'delete_card', col_idx: 0, card_idx: 0 }, 'x')
    await new Promise((r) => setTimeout(r, 5100))
    expect(m.mutate).toHaveBeenCalledTimes(1)
  })

  it('does not fire when undo is clicked before timeout', async () => {
    const m = fakeMutation()
    // Stub sonner toast to capture the action and invoke it immediately.
    const { toast } = await import('../toast.js')
    const originalToast = toast
    let capturedOnClick: (() => void) | undefined
    ;(toast as any) = Object.assign(
      (_msg: string, opts: any) => {
        capturedOnClick = opts?.action?.onClick
      },
      originalToast,
    )
    try {
      stageDelete(m, { type: 'delete_card', col_idx: 0, card_idx: 0 }, 'x')
      capturedOnClick?.()
      await new Promise((r) => setTimeout(r, 5100))
      expect(m.mutate).not.toHaveBeenCalled()
    } finally {
      ;(toast as any) = originalToast
    }
  })
})
```

Note: this test monkey-patches `toast` because sonner's `<Toaster />` isn't mounted in a bun test. It's brittle. If it proves flaky, replace with a direct refactor: split out the timer logic into a testable function `scheduleDelete(mutation, op, onAction)` and call it from `stageDelete`. Test the scheduler directly; leave `stageDelete` as the thin UI wrapper.

- [ ] **Step 3: Run**

```bash
cd web/renderer/default && bun test src/mutations/undoable.test.ts
```
Expected: 2 pass (each takes ~5s — don't worry about the wait).

- [ ] **Step 4: If Step 3 is flaky, refactor**

If the monkey-patch approach is unreliable, replace `undoable.ts` with:
```ts
import type { MutationOp } from '@shared/types.js'
import type { UseMutationResult } from '@tanstack/react-query'
import { toast } from '../toast.js'

const UNDO_MS = 5000

export function scheduleDelete(
  fire: () => void,
  ms: number = UNDO_MS,
): { cancel: () => void } {
  let done = false
  const timer = setTimeout(() => {
    if (!done) fire()
  }, ms)
  return {
    cancel: () => {
      done = true
      clearTimeout(timer)
    },
  }
}

export function stageDelete(
  mutation: UseMutationResult<unknown, unknown, MutationOp, unknown>,
  op: MutationOp,
  label: string,
): void {
  const handle = scheduleDelete(() => mutation.mutate(op))
  toast(`Deleted ${label}`, {
    duration: UNDO_MS,
    action: { label: 'Undo', onClick: handle.cancel },
  })
}
```

And replace the tests with tests for `scheduleDelete` directly:
```ts
import { describe, expect, it, mock } from 'bun:test'
import { scheduleDelete } from './undoable.js'

describe('scheduleDelete', () => {
  it('fires after timeout when not cancelled', async () => {
    const fire = mock(() => {})
    scheduleDelete(fire, 30)
    await new Promise((r) => setTimeout(r, 60))
    expect(fire).toHaveBeenCalledTimes(1)
  })

  it('does not fire when cancelled before timeout', async () => {
    const fire = mock(() => {})
    const h = scheduleDelete(fire, 50)
    h.cancel()
    await new Promise((r) => setTimeout(r, 80))
    expect(fire).not.toHaveBeenCalled()
  })
})
```

Rerun `bun test src/mutations/undoable.test.ts`. Expected: 2 pass.

- [ ] **Step 5: Commit**

```bash
git add web/renderer/default/src/mutations/undoable.ts web/renderer/default/src/mutations/undoable.test.ts
git commit -m "feat(renderer): add staged delete with 5s undo toast

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 6: `moveColumnTarget` helper

**Files:**
- Create: `web/renderer/default/src/mutations/moveColumn.ts`
- Create: `web/renderer/default/src/mutations/moveColumn.test.ts`

`move_column` takes `{ name, after_col }` where `after_col: ''` means "first position." Computing the target for move-left/move-right is easy to get wrong; pin it with tests.

- [ ] **Step 1: Write tests first (TDD)**

Create `web/renderer/default/src/mutations/moveColumn.test.ts`:
```ts
import { describe, expect, it } from 'bun:test'
import { moveColumnTarget } from './moveColumn.js'

describe('moveColumnTarget', () => {
  const names = ['A', 'B', 'C', 'D']

  it('move left from index 1 → first position', () => {
    expect(moveColumnTarget(names, 1, 'left')).toBe('')
  })
  it('move left from index 2 → after col 0', () => {
    expect(moveColumnTarget(names, 2, 'left')).toBe('A')
  })
  it('move left from index 3 → after col 1', () => {
    expect(moveColumnTarget(names, 3, 'left')).toBe('B')
  })
  it('move right from index 0 → after col 1', () => {
    expect(moveColumnTarget(names, 0, 'right')).toBe('B')
  })
  it('move right from index 2 → after col 3', () => {
    expect(moveColumnTarget(names, 2, 'right')).toBe('D')
  })
  it('returns null for move-left at index 0 (disabled edge)', () => {
    expect(moveColumnTarget(names, 0, 'left')).toBeNull()
  })
  it('returns null for move-right at last index (disabled edge)', () => {
    expect(moveColumnTarget(names, 3, 'right')).toBeNull()
  })
})
```

- [ ] **Step 2: Run to fail**

```bash
cd web/renderer/default && bun test src/mutations/moveColumn.test.ts
```
Expected: fail (module missing).

- [ ] **Step 3: Implement**

Create `web/renderer/default/src/mutations/moveColumn.ts`:
```ts
export function moveColumnTarget(
  columnNames: string[],
  colIdx: number,
  dir: 'left' | 'right',
): string | null {
  if (dir === 'left') {
    if (colIdx <= 0) return null
    // New position: after the column two slots earlier (or '' for first).
    return columnNames[colIdx - 2] ?? ''
  }
  // dir === 'right'
  if (colIdx >= columnNames.length - 1) return null
  return columnNames[colIdx + 1]!
}
```

- [ ] **Step 4: Run to pass**

```bash
cd web/renderer/default && bun test src/mutations/moveColumn.test.ts
```
Expected: 7 pass.

- [ ] **Step 5: Commit**

```bash
git add web/renderer/default/src/mutations/moveColumn.ts web/renderer/default/src/mutations/moveColumn.test.ts
git commit -m "feat(renderer): add moveColumnTarget helper

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 7: `CardEditable` component

**Files:**
- Create: `web/renderer/default/src/components/CardEditable.tsx`
- Create: `web/renderer/default/src/components/CardEditable.test.tsx`

- [ ] **Step 1: Write the component**

Create `web/renderer/default/src/components/CardEditable.tsx`:
```tsx
import { useState, useRef, useEffect } from 'react'
import type { Card as CardModel } from '@shared/types.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { stageDelete } from '../mutations/undoable.js'
import { Card } from './Card.js'

export function CardEditable({
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
  const [mode, setMode] = useState<'view' | 'edit'>('view')
  const [draft, setDraft] = useState(card.title)
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useBoardMutation(boardId)

  useEffect(() => {
    if (mode === 'edit') {
      inputRef.current?.focus()
      inputRef.current?.select()
    }
  }, [mode])

  const commit = (): void => {
    const title = draft.trim()
    if (title && title !== card.title) {
      mutation.mutate({
        type: 'edit_card',
        col_idx: colIdx,
        card_idx: cardIdx,
        title,
      })
    }
    setMode('view')
  }

  const cancel = (): void => {
    setDraft(card.title)
    setMode('view')
  }

  if (mode === 'edit') {
    return (
      <div className="rounded-md bg-white p-3 shadow-sm ring-2 ring-blue-400">
        <input
          ref={inputRef}
          aria-label="card title"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === 'Enter') commit()
            else if (e.key === 'Escape') cancel()
          }}
          className="w-full bg-transparent text-sm font-semibold outline-none"
        />
      </div>
    )
  }

  return (
    <div
      className="group relative"
      onDoubleClick={() => {
        setDraft(card.title)
        setMode('edit')
      }}
    >
      <div className="flex items-start gap-2">
        <button
          type="button"
          aria-label={card.completed ? 'mark incomplete' : 'mark complete'}
          onClick={() =>
            mutation.mutate({
              type: 'complete_card',
              col_idx: colIdx,
              card_idx: cardIdx,
            })
          }
          className={`mt-3 h-4 w-4 shrink-0 rounded-full border ${
            card.completed ? 'bg-slate-400 border-slate-400' : 'border-slate-300'
          }`}
        />
        <div className="flex-1">
          <Card card={card} />
        </div>
        <button
          type="button"
          aria-label="delete card"
          onClick={() =>
            stageDelete(
              mutation,
              { type: 'delete_card', col_idx: colIdx, card_idx: cardIdx },
              card.title,
            )
          }
          className="opacity-0 group-hover:opacity-100 mt-1 text-xs text-slate-400 hover:text-red-500"
        >
          ✕
        </button>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Write tests**

Create `web/renderer/default/src/components/CardEditable.test.tsx`:
```tsx
import { describe, expect, it } from 'bun:test'
import { fireEvent, waitFor } from '@testing-library/react'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { renderWithQuery } from '../test-utils.js'
import { CardEditable } from './CardEditable.js'
import { QueryClient } from '@tanstack/react-query'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  // Seed the cache with the real board (so useBoardMutation has a prev to read).
  qc.setQueryData(['board', 'welcome'], await client.getBoard('welcome'))
  return { client, qc }
}

describe('CardEditable', () => {
  it('double-click switches to edit mode', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardEditable
          card={{ title: 'hello' }}
          colIdx={0}
          cardIdx={0}
          boardId="welcome"
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.doubleClick(getByText('hello'))
    await waitFor(() => expect(getByLabelText('card title')).toBeDefined())
  })

  it('Escape cancels edit without mutation', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText, queryByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardEditable
          card={{ title: 'hello' }}
          colIdx={0}
          cardIdx={0}
          boardId="welcome"
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.doubleClick(getByText('hello'))
    const input = await waitFor(() => getByLabelText('card title'))
    fireEvent.change(input, { target: { value: 'CHANGED' } })
    fireEvent.keyDown(input, { key: 'Escape' })
    await waitFor(() => expect(queryByLabelText('card title')).toBeNull())
    expect(getByText('hello')).toBeDefined()
  })

  it('Enter commits edit_card mutation', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardEditable
          card={{ title: 'Welcome to LiveBoard' }}
          colIdx={0}
          cardIdx={0}
          boardId="welcome"
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.doubleClick(getByText('Welcome to LiveBoard'))
    const input = await waitFor(() => getByLabelText('card title'))
    fireEvent.change(input, { target: { value: 'NEW TITLE' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    // Wait for the optimistic update to reflect in the live cache.
    await waitFor(() => {
      const b = qc.getQueryData<any>(['board', 'welcome'])
      const todo = b?.columns?.find((c: any) => c.name === 'Todo')
      expect(todo?.cards?.[0]?.title).toBe('NEW TITLE')
    })
  })

  it('complete button fires complete_card mutation', async () => {
    const { client, qc } = await setup()
    const { getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardEditable
          card={{ title: 'Welcome to LiveBoard' }}
          colIdx={0}
          cardIdx={0}
          boardId="welcome"
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByLabelText('mark complete'))
    await waitFor(() => {
      const b = qc.getQueryData<any>(['board', 'welcome'])
      const todo = b?.columns?.find((c: any) => c.name === 'Todo')
      expect(todo?.cards?.[0]?.completed).toBe(true)
    })
  })
})
```

Note: `CardEditable` expects real data at `col_idx 0` / `card_idx 0` of the welcome seed. If the seeded Todo column has cards, the indices point at its first card. Read `web/shared/src/adapters/local-seed.ts` to confirm the seed shape; adjust `title` strings in the test if the first Todo card isn't "Welcome to LiveBoard".

- [ ] **Step 3: Run**

```bash
cd web/renderer/default && bun test src/components/CardEditable.test.tsx && bun run typecheck
```
Expected: 4 pass.

- [ ] **Step 4: Commit**

```bash
git add web/renderer/default/src/components/CardEditable.tsx web/renderer/default/src/components/CardEditable.test.tsx
git commit -m "feat(renderer): add CardEditable with inline edit, complete, delete

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 8: `AddCardButton` component

**Files:**
- Create: `web/renderer/default/src/components/AddCardButton.tsx`
- Create: `web/renderer/default/src/components/AddCardButton.test.tsx`

- [ ] **Step 1: Component**

Create `web/renderer/default/src/components/AddCardButton.tsx`:
```tsx
import { useState, useRef, useEffect } from 'react'
import { useBoardMutation } from '../mutations/useBoardMutation.js'

export function AddCardButton({
  columnName,
  boardId,
}: {
  columnName: string
  boardId: string
}): JSX.Element {
  const [open, setOpen] = useState(false)
  const [draft, setDraft] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useBoardMutation(boardId)

  useEffect(() => {
    if (open) inputRef.current?.focus()
  }, [open])

  const commit = (): void => {
    const title = draft.trim()
    if (title) {
      mutation.mutate({ type: 'add_card', column: columnName, title })
    }
    setDraft('')
    setOpen(false)
  }

  if (open) {
    return (
      <input
        ref={inputRef}
        aria-label={`new card in ${columnName}`}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={commit}
        onKeyDown={(e) => {
          if (e.key === 'Enter') commit()
          else if (e.key === 'Escape') {
            setDraft('')
            setOpen(false)
          }
        }}
        placeholder="Card title…"
        className="mt-2 w-full rounded-md bg-white p-2 text-sm shadow-sm ring-1 ring-slate-200 outline-none focus:ring-blue-400"
      />
    )
  }

  return (
    <button
      type="button"
      onClick={() => setOpen(true)}
      className="mt-2 w-full rounded-md px-2 py-1 text-left text-xs text-slate-500 hover:bg-slate-200"
    >
      + Add card
    </button>
  )
}
```

- [ ] **Step 2: Tests**

Create `web/renderer/default/src/components/AddCardButton.test.tsx`:
```tsx
import { describe, expect, it } from 'bun:test'
import { fireEvent, waitFor } from '@testing-library/react'
import { QueryClient } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { renderWithQuery } from '../test-utils.js'
import { AddCardButton } from './AddCardButton.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['board', 'welcome'], await client.getBoard('welcome'))
  return { client, qc }
}

describe('AddCardButton', () => {
  it('click reveals input', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <AddCardButton columnName="Todo" boardId="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByText('+ Add card'))
    await waitFor(() => expect(getByLabelText('new card in Todo')).toBeDefined())
  })

  it('Enter commits add_card', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <AddCardButton columnName="Todo" boardId="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByText('+ Add card'))
    const input = await waitFor(() => getByLabelText('new card in Todo'))
    fireEvent.change(input, { target: { value: 'NEW' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    await waitFor(() => {
      const b = qc.getQueryData<any>(['board', 'welcome'])
      const todo = b.columns.find((c: any) => c.name === 'Todo')
      expect(todo.cards.some((c: any) => c.title === 'NEW')).toBe(true)
    })
  })

  it('Escape cancels and clears draft', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText, queryByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <AddCardButton columnName="Todo" boardId="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByText('+ Add card'))
    const input = await waitFor(() => getByLabelText('new card in Todo'))
    fireEvent.change(input, { target: { value: 'CANCELME' } })
    fireEvent.keyDown(input, { key: 'Escape' })
    await waitFor(() => expect(queryByLabelText('new card in Todo')).toBeNull())
  })
})
```

- [ ] **Step 3: Run + commit**

```bash
cd web/renderer/default && bun test src/components/AddCardButton.test.tsx && bun run typecheck
git add web/renderer/default/src/components/AddCardButton.tsx web/renderer/default/src/components/AddCardButton.test.tsx
git commit -m "feat(renderer): add AddCardButton with inline input

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```
Expected: 3 pass.

---

## Task 9: `ColumnHeader` with Radix menu

**Files:**
- Create: `web/renderer/default/src/components/ColumnHeader.tsx`
- Create: `web/renderer/default/src/components/ColumnHeader.test.tsx`

- [ ] **Step 1: Component**

Create `web/renderer/default/src/components/ColumnHeader.tsx`:
```tsx
import { useState, useRef, useEffect } from 'react'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { stageDelete } from '../mutations/undoable.js'
import { moveColumnTarget } from '../mutations/moveColumn.js'

export function ColumnHeader({
  name,
  cardCount,
  colIdx,
  allColumnNames,
  boardId,
}: {
  name: string
  cardCount: number
  colIdx: number
  allColumnNames: string[]
  boardId: string
}): JSX.Element {
  const [mode, setMode] = useState<'view' | 'edit'>('view')
  const [draft, setDraft] = useState(name)
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useBoardMutation(boardId)

  useEffect(() => {
    if (mode === 'edit') {
      inputRef.current?.focus()
      inputRef.current?.select()
    }
  }, [mode])

  const commitRename = (): void => {
    const next = draft.trim()
    if (next && next !== name) {
      mutation.mutate({ type: 'rename_column', old_name: name, new_name: next })
    }
    setMode('view')
  }

  const move = (dir: 'left' | 'right'): void => {
    const target = moveColumnTarget(allColumnNames, colIdx, dir)
    if (target === null) return
    mutation.mutate({ type: 'move_column', name, after_col: target })
  }

  const leftDisabled = colIdx === 0
  const rightDisabled = colIdx === allColumnNames.length - 1

  if (mode === 'edit') {
    return (
      <header className="mb-3 flex items-center justify-between">
        <input
          ref={inputRef}
          aria-label={`rename column ${name}`}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={commitRename}
          onKeyDown={(e) => {
            if (e.key === 'Enter') commitRename()
            else if (e.key === 'Escape') {
              setDraft(name)
              setMode('view')
            }
          }}
          className="w-full bg-white px-1 text-sm font-semibold outline-none ring-1 ring-blue-400 rounded"
        />
      </header>
    )
  }

  return (
    <header className="mb-3 flex items-center justify-between">
      <h2 className="text-sm font-semibold text-slate-800">{name}</h2>
      <div className="flex items-center gap-2">
        <span className="text-xs text-slate-500">{cardCount}</span>
        <DropdownMenu.Root>
          <DropdownMenu.Trigger
            aria-label={`column menu ${name}`}
            className="rounded p-1 text-slate-500 hover:bg-slate-200"
          >
            ⋮
          </DropdownMenu.Trigger>
          <DropdownMenu.Portal>
            <DropdownMenu.Content
              sideOffset={4}
              className="z-50 min-w-40 rounded-md bg-white p-1 shadow-lg ring-1 ring-slate-200"
            >
              <DropdownMenu.Item
                onSelect={() => {
                  setDraft(name)
                  setMode('edit')
                }}
                className="cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-slate-100"
              >
                Rename
              </DropdownMenu.Item>
              <DropdownMenu.Item
                disabled={leftDisabled}
                onSelect={() => move('left')}
                className="cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-slate-100 data-[disabled]:text-slate-300 data-[disabled]:cursor-not-allowed"
              >
                Move left
              </DropdownMenu.Item>
              <DropdownMenu.Item
                disabled={rightDisabled}
                onSelect={() => move('right')}
                className="cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-slate-100 data-[disabled]:text-slate-300 data-[disabled]:cursor-not-allowed"
              >
                Move right
              </DropdownMenu.Item>
              <DropdownMenu.Separator className="my-1 h-px bg-slate-200" />
              <DropdownMenu.Item
                onSelect={() =>
                  stageDelete(
                    mutation,
                    { type: 'delete_column', name },
                    name,
                  )
                }
                className="cursor-pointer rounded px-2 py-1 text-sm text-red-600 outline-none hover:bg-red-50"
              >
                Delete
              </DropdownMenu.Item>
            </DropdownMenu.Content>
          </DropdownMenu.Portal>
        </DropdownMenu.Root>
      </div>
    </header>
  )
}
```

- [ ] **Step 2: Tests**

Create `web/renderer/default/src/components/ColumnHeader.test.tsx`:
```tsx
import { describe, expect, it } from 'bun:test'
import { fireEvent, waitFor } from '@testing-library/react'
import { QueryClient } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { renderWithQuery } from '../test-utils.js'
import { ColumnHeader } from './ColumnHeader.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['board', 'welcome'], await client.getBoard('welcome'))
  return { client, qc }
}

describe('ColumnHeader', () => {
  it('renders name and count', async () => {
    const { client, qc } = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ColumnHeader
          name="Todo"
          cardCount={3}
          colIdx={0}
          allColumnNames={['Todo', 'Doing', 'Done']}
          boardId="welcome"
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    expect(getByText('Todo')).toBeDefined()
    expect(getByText('3')).toBeDefined()
  })

  it('menu → Rename enters edit mode', async () => {
    const { client, qc } = await setup()
    const { getByLabelText, getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ColumnHeader
          name="Todo"
          cardCount={0}
          colIdx={0}
          allColumnNames={['Todo', 'Doing']}
          boardId="welcome"
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByLabelText('column menu Todo'))
    const rename = await waitFor(() => getByText('Rename'))
    fireEvent.click(rename)
    await waitFor(() => expect(getByLabelText('rename column Todo')).toBeDefined())
  })

  it('rename Enter dispatches rename_column', async () => {
    const { client, qc } = await setup()
    const { getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <ColumnHeader
          name="Todo"
          cardCount={0}
          colIdx={0}
          allColumnNames={['Todo', 'Doing', 'Done']}
          boardId="welcome"
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByLabelText('column menu Todo'))
    fireEvent.click(await waitFor(() => (document.querySelector('[role="menuitem"]') as HTMLElement)))
    const input = await waitFor(() => getByLabelText('rename column Todo'))
    fireEvent.change(input, { target: { value: 'Pending' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    await waitFor(() => {
      const b = qc.getQueryData<any>(['board', 'welcome'])
      expect(b.columns.some((c: any) => c.name === 'Pending')).toBe(true)
    })
  })
})
```

Note on Radix + happy-dom: Radix DropdownMenu uses Portals and focus traps. happy-dom supports this but timing can be flaky. If a test hangs or times out, wrap assertions in `await waitFor(...)` and consider adding `await new Promise(r => setTimeout(r, 0))` before the first menu interaction to let Radix finish mounting.

- [ ] **Step 3: Run + commit**

```bash
cd web/renderer/default && bun test src/components/ColumnHeader.test.tsx && bun run typecheck
git add web/renderer/default/src/components/ColumnHeader.tsx web/renderer/default/src/components/ColumnHeader.test.tsx
git commit -m "feat(renderer): add ColumnHeader with Radix menu + inline rename

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```
Expected: 3 pass.

---

## Task 10: `AddColumnButton` component

**Files:**
- Create: `web/renderer/default/src/components/AddColumnButton.tsx`
- Create: `web/renderer/default/src/components/AddColumnButton.test.tsx`

- [ ] **Step 1: Component**

Create `web/renderer/default/src/components/AddColumnButton.tsx`:
```tsx
import { useState, useRef, useEffect } from 'react'
import { useBoardMutation } from '../mutations/useBoardMutation.js'

export function AddColumnButton({ boardId }: { boardId: string }): JSX.Element {
  const [open, setOpen] = useState(false)
  const [draft, setDraft] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useBoardMutation(boardId)

  useEffect(() => {
    if (open) inputRef.current?.focus()
  }, [open])

  const commit = (): void => {
    const name = draft.trim()
    if (name) {
      mutation.mutate({ type: 'add_column', name })
    }
    setDraft('')
    setOpen(false)
  }

  if (open) {
    return (
      <div className="flex w-72 shrink-0 flex-col rounded-lg bg-slate-100 p-3">
        <input
          ref={inputRef}
          aria-label="new column name"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === 'Enter') commit()
            else if (e.key === 'Escape') {
              setDraft('')
              setOpen(false)
            }
          }}
          placeholder="Column name…"
          className="w-full rounded bg-white px-2 py-1 text-sm outline-none ring-1 ring-slate-200 focus:ring-blue-400"
        />
      </div>
    )
  }

  return (
    <button
      type="button"
      onClick={() => setOpen(true)}
      className="flex w-72 shrink-0 items-center justify-center rounded-lg border-2 border-dashed border-slate-300 p-3 text-sm text-slate-500 hover:border-slate-400 hover:text-slate-700"
    >
      + Add column
    </button>
  )
}
```

- [ ] **Step 2: Tests**

Create `web/renderer/default/src/components/AddColumnButton.test.tsx`:
```tsx
import { describe, expect, it } from 'bun:test'
import { fireEvent, waitFor } from '@testing-library/react'
import { QueryClient } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { renderWithQuery } from '../test-utils.js'
import { AddColumnButton } from './AddColumnButton.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['board', 'welcome'], await client.getBoard('welcome'))
  return { client, qc }
}

describe('AddColumnButton', () => {
  it('click reveals input', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <AddColumnButton boardId="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByText('+ Add column'))
    await waitFor(() => expect(getByLabelText('new column name')).toBeDefined())
  })

  it('Enter commits add_column', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <AddColumnButton boardId="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByText('+ Add column'))
    const input = await waitFor(() => getByLabelText('new column name'))
    fireEvent.change(input, { target: { value: 'Review' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    await waitFor(() => {
      const b = qc.getQueryData<any>(['board', 'welcome'])
      expect(b.columns.some((c: any) => c.name === 'Review')).toBe(true)
    })
  })
})
```

- [ ] **Step 3: Run + commit**

```bash
cd web/renderer/default && bun test src/components/AddColumnButton.test.tsx && bun run typecheck
git add web/renderer/default/src/components/AddColumnButton.tsx web/renderer/default/src/components/AddColumnButton.test.tsx
git commit -m "feat(renderer): add AddColumnButton

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```
Expected: 2 pass.

---

## Task 11: Wire `Column.tsx` and `BoardView.tsx`

**Files:**
- Modify: `web/renderer/default/src/components/Column.tsx`
- Modify: `web/renderer/default/src/components/BoardView.tsx`

- [ ] **Step 1: Replace `Column.tsx`**

Read current `web/renderer/default/src/components/Column.tsx`. Replace entire file with:
```tsx
import type { Column as ColumnModel } from '@shared/types.js'
import { CardEditable } from './CardEditable.js'
import { ColumnHeader } from './ColumnHeader.js'
import { AddCardButton } from './AddCardButton.js'

export function Column({
  column,
  colIdx,
  allColumnNames,
  boardId,
}: {
  column: ColumnModel
  colIdx: number
  allColumnNames: string[]
  boardId: string
}): JSX.Element {
  const cards = column.cards ?? []
  return (
    <section className="flex w-72 shrink-0 flex-col rounded-lg bg-slate-100 p-3">
      <ColumnHeader
        name={column.name}
        cardCount={cards.length}
        colIdx={colIdx}
        allColumnNames={allColumnNames}
        boardId={boardId}
      />
      <ul className="flex flex-col gap-2">
        {cards.map((card, i) => (
          <li key={`${column.name}-${i}`}>
            <CardEditable card={card} colIdx={colIdx} cardIdx={i} boardId={boardId} />
          </li>
        ))}
      </ul>
      <AddCardButton columnName={column.name} boardId={boardId} />
    </section>
  )
}
```

- [ ] **Step 2: Update `Column.test.tsx`**

The existing `Column.test.tsx` calls `<Column column={...} />` without the new props. It will break. Replace its contents with a version that provides the new props and wraps in `ClientProvider` (needed because `CardEditable` calls `useBoardMutation`):

Read current `web/renderer/default/src/components/Column.test.tsx`. Replace with:
```tsx
import { describe, expect, it } from 'bun:test'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { QueryClient } from '@tanstack/react-query'
import { ClientProvider } from '../queries.js'
import { renderWithQuery } from '../test-utils.js'
import { Column } from './Column.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['board', 'welcome'], await client.getBoard('welcome'))
  return { client, qc }
}

describe('Column', () => {
  it('renders name and count', async () => {
    const { client, qc } = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <Column
          column={{ name: 'Todo', cards: [{ title: 'a' }, { title: 'b' }] }}
          colIdx={0}
          allColumnNames={['Todo']}
          boardId="welcome"
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    expect(getByText('Todo')).toBeDefined()
    expect(getByText('2')).toBeDefined()
  })

  it('renders all cards', async () => {
    const { client, qc } = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <Column
          column={{ name: 'x', cards: [{ title: 'A' }, { title: 'B' }] }}
          colIdx={0}
          allColumnNames={['x']}
          boardId="welcome"
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    expect(getByText('A')).toBeDefined()
    expect(getByText('B')).toBeDefined()
  })

  it('handles empty cards array', async () => {
    const { client, qc } = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <Column
          column={{ name: 'Empty', cards: [] }}
          colIdx={0}
          allColumnNames={['Empty']}
          boardId="welcome"
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    expect(getByText('0')).toBeDefined()
  })
})
```

- [ ] **Step 3: Replace `BoardView.tsx`**

Read current `web/renderer/default/src/components/BoardView.tsx`. Replace with:
```tsx
import { useEffect } from 'react'
import type { Client } from '@shared/client.js'
import { useBoard } from '../queries.js'
import { Column } from './Column.js'
import { EmptyState } from './EmptyState.js'
import { AddColumnButton } from './AddColumnButton.js'

export function BoardView({
  boardId,
  client,
}: {
  boardId: string | null
  client: Client
}): JSX.Element {
  const { data, isLoading, error } = useBoard(boardId)

  useEffect(() => {
    if (!boardId) return
    void client.subscribe(boardId)
    return () => {
      void client.unsubscribe(boardId)
    }
  }, [boardId, client])

  if (!boardId) return <EmptyState title="Select a board" />
  if (isLoading) return <EmptyState title="Loading…" />
  if (error) return <EmptyState title="Failed to load board" detail={String(error)} />
  if (!data) return <EmptyState title="Board not found" />

  const columns = data.columns ?? []
  if (columns.length === 0 && boardId) {
    return (
      <div className="flex h-full gap-4 overflow-x-auto p-4">
        <AddColumnButton boardId={boardId} />
      </div>
    )
  }

  const names = columns.map((c) => c.name)

  return (
    <div className="flex h-full gap-4 overflow-x-auto p-4">
      {columns.map((col, i) => (
        <Column
          key={`${col.name}-${i}`}
          column={col}
          colIdx={i}
          allColumnNames={names}
          boardId={boardId}
        />
      ))}
      <AddColumnButton boardId={boardId} />
    </div>
  )
}
```

- [ ] **Step 4: Update `BoardView.test.tsx` assertions**

The existing BoardView test uses `getByText('Todo')` etc. — those still work because `ColumnHeader` renders the column name as an `<h2>`. No changes needed. Just confirm by running.

- [ ] **Step 5: Full suite**

```bash
cd web/renderer/default && bun test && bun run typecheck
```
Expected: everything green. Total test count should rise to ~40+.

- [ ] **Step 6: Commit**

```bash
git add web/renderer/default/src/components/Column.tsx web/renderer/default/src/components/Column.test.tsx web/renderer/default/src/components/BoardView.tsx
git commit -m "feat(renderer): wire Column and BoardView to interactive components

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 12: Build + bundle size check

**Files:**
- None modified; just runs the existing `make renderer` target.

- [ ] **Step 1: Build**

```bash
make renderer
```
Expected: clean build.

- [ ] **Step 2: Measure**

```bash
ls -la web/renderer/default/dist/assets/*.js
gzip -c web/renderer/default/dist/assets/*.js | wc -c
```
Record the gzipped byte count. Target: **≤ 75,000 bytes** (75 KB). Fail the task if > 80 KB — something pulled in a surprise dependency.

- [ ] **Step 3: Rebuild Go embed**

```bash
go test ./internal/api/ -run TestShellRoute -v
```
Expected: 3 tests pass. The embedded `dist/` now contains the new bundle, but the route test only checks for `<div id="root">` — still green.

- [ ] **Step 4: No commit**

Build output is gitignored. This task is a measurement gate, not a code change.

---

## Task 13: Manual browser smoke

Not a code change. Gate before merging.

- [ ] **Step 1: Rebuild everything**

```bash
make shell && make renderer
```

- [ ] **Step 2: Serve**

```bash
LIVEBOARD_APP_SHELL=1 go run ./cmd/liveboard serve --dir ./demo --port 7070
```

- [ ] **Step 3: Open http://localhost:7070/app/ and verify**

For each action, check: visual feedback is instant, reload preserves state.

1. Click "+ Add card" in Todo, type "smoke-test", Enter → card appears at bottom of Todo.
2. Double-click "smoke-test" title → input appears. Type "edited", Enter → card shows "edited".
3. Click the round button next to a card → card shows strikethrough / completed styling.
4. Hover a card → ✕ button appears; click → toast "Deleted edited" with Undo action. Click Undo within 5s → card returns. Wait out the 5s → card gone permanently.
5. Click column "⋮" menu on Todo → Rename → input appears; type "Pending", Enter → column header updates. Rename it back.
6. Open column menu → Move right → Todo moves to position 2. Move left twice → Todo moves back to position 0 (and Move left is then disabled).
7. Click column menu → Delete → toast with Undo. Wait out the 5s → column gone.
8. Click "+ Add column" at the end → input → type "Review", Enter → new column appears.
9. Open devtools. Run: `__liveboardDebugForceConflict ?? ''` — no-op. Instead:
   - In the console: `document.querySelectorAll('[aria-label^=\\"new card\\"]')` — just a DOM poke to confirm the app is live.
10. Open `http://localhost:7070/app/?renderer=stub` → P3 harness still loads.

- [ ] **Step 4: Report**

If any step fails, capture the console output + which step + expected vs actual. Fix before marking P4b.1a done.

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| Optimistic UI for non-destructive ops | 4 (useBoardMutation) |
| Undo toast for destructive ops | 5, 7, 9 |
| VERSION_CONFLICT rollback + refetch | 4 |
| Error toasts for protocol codes | 2 (errorToast), 4 |
| Card: add | 8 (AddCardButton) |
| Card: inline-edit title | 7 (CardEditable) |
| Card: toggle complete | 7 |
| Card: delete | 7 + 5 |
| Column: add | 10 (AddColumnButton) |
| Column: rename | 9 (ColumnHeader) |
| Column: move left / right | 6 (helper), 9 (menu) |
| Column: delete | 9 + 5 |
| Toast primitive (`sonner`) | 1, 2 |
| Radix DropdownMenu | 1, 9 |
| Bundle ≤ 75 KB gz | 12 |
| Browser smoke | 13 |

## Notes for implementer

1. **`useClient` export (Task 3) is not optional** — mutation hooks import it.
2. **Radix + happy-dom portal flakiness** — if `ColumnHeader` menu tests hang, see the Task 9 note about `waitFor` + microtask yields. Do not wrap tests in arbitrary `setTimeout` delays > 50ms; if you need that, the test is wrong.
3. **`applyOp` throws `OpError`, not `ProtocolError`** — the `try/catch` in `onMutate` swallows it. Server errors come through `mutationFn` as `ProtocolError` and are handled in `onError`. Keep those two paths separate.
4. **Version is NEVER bumped optimistically** — only `onSuccess` writes the server board's version. Do not be tempted to guess `version + 1` during `onMutate`.
5. **Seed data** — every test that seeds the cache via `await client.getBoard('welcome')` depends on `LocalAdapter` returning a real welcome board. Confirmed by P4a tests. If seed changes, update tests accordingly.
6. **No commit amending** — the plan assumes forward-only commits. If a task commit fails hooks, fix the issue and make a NEW commit.
