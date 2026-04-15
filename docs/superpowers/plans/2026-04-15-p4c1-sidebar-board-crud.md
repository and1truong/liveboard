# P4c.1 — Sidebar Board CRUD UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire P4c.0's board CRUD protocol into the renderer sidebar — "+ New board" button, per-board hover-kebab with rename + delete, automatic active-board re-routing on rename and delete-of-active, and cross-tab refresh via `board.list.updated`.

**Architecture:** Active-board state lifts into `<ActiveBoardContext>`. Three thin hooks (`useCreateBoard`/`useRenameBoard`/`useDeleteBoard`) wrap the protocol calls and own their own cache + active-id transitions. The existing `stageDelete` is generalized from `(mutation, op, label)` to `(fire, label)` so all destructive-undo paths share one signature.

**Tech Stack:** P4c.0 protocol + existing renderer stack (React 18, TanStack v5, Radix DropdownMenu, sonner). No new deps.

**Spec:** `docs/superpowers/specs/2026-04-15-p4c1-sidebar-board-crud-design.md`

**Conventions:**
- All new code under `web/renderer/default/src/`.
- Imports use `@shared/*` alias.
- Tests colocated.
- Commit prefixes: `feat(renderer)`, `refactor(renderer)`, `test(renderer)`.
- Pre-existing `TS6196` in `web/shared/src/protocol.ts` is NOT a blocker.
- Use bun, never npx.
- Inputs use uncontrolled pattern (`defaultValue` + ref + `committedRef`) — happy-dom keyDown bug.

---

## File structure

**New:**
- `web/renderer/default/src/contexts/ActiveBoardContext.tsx`
- `web/renderer/default/src/contexts/ActiveBoardContext.test.tsx`
- `web/renderer/default/src/mutations/useBoardCrud.ts`
- `web/renderer/default/src/mutations/useBoardCrud.test.tsx`
- `web/renderer/default/src/mutations/useBoardListEvents.ts`
- `web/renderer/default/src/components/AddBoardButton.tsx`
- `web/renderer/default/src/components/AddBoardButton.test.tsx`
- `web/renderer/default/src/components/BoardRow.tsx`
- `web/renderer/default/src/components/BoardRow.test.tsx`

**Modified:**
- `web/renderer/default/src/mutations/undoable.ts` — generalize `stageDelete`.
- `web/renderer/default/src/mutations/undoable.test.ts` — update tests for new signature.
- `web/renderer/default/src/components/CardEditable.tsx` — call site update.
- `web/renderer/default/src/components/ColumnHeader.tsx` — call site update.
- `web/renderer/default/src/components/BoardSidebar.tsx` — drop props, render `BoardRow` + `AddBoardButton`, read active from context.
- `web/renderer/default/src/components/BoardView.tsx` — drop `boardId` prop, read active from context, self-recover on `NOT_FOUND`.
- `web/renderer/default/src/App.tsx` — wrap in `<ActiveBoardProvider>`, mount `useBoardListEvents`.
- `web/renderer/default/src/toast.ts` — add `ALREADY_EXISTS` copy override.

---

## Task 1: Generalize `stageDelete`

**Files:**
- Modify: `web/renderer/default/src/mutations/undoable.ts`
- Modify: `web/renderer/default/src/mutations/undoable.test.ts`
- Modify: `web/renderer/default/src/components/CardEditable.tsx`
- Modify: `web/renderer/default/src/components/ColumnHeader.tsx`

The new signature is `stageDelete(fire: () => void, label: string)`. Bare wrapper around `scheduleDelete` + sonner toast. All call sites move from `stageDelete(mutation, op, label)` to `stageDelete(() => mutation.mutate(op), label)`.

- [ ] **Step 1: Replace `undoable.ts`**

```ts
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

export function stageDelete(fire: () => void, label: string): void {
  const handle = scheduleDelete(fire)
  toast(`Deleted ${label}`, {
    duration: UNDO_MS,
    action: { label: 'Undo', onClick: handle.cancel },
  })
}
```

- [ ] **Step 2: Update `undoable.test.ts`**

Read the current file. The existing tests target `scheduleDelete` directly (per the P4b.1a refactor). Confirm they still pass as-is — `scheduleDelete` signature didn't change. If any test references the old `stageDelete(mutation, op, label)` signature, replace with `stageDelete(() => fired = true, 'x')` and assert `fired === true` after timeout.

- [ ] **Step 3: Update `CardEditable.tsx` delete call site**

Find:
```tsx
            onClick={(e) => {
              e.stopPropagation()
              stageDelete(
                mutation,
                { type: 'delete_card', col_idx: colIdx, card_idx: cardIdx },
                card.title,
              )
            }}
```
Replace with:
```tsx
            onClick={(e) => {
              e.stopPropagation()
              stageDelete(
                () => mutation.mutate({ type: 'delete_card', col_idx: colIdx, card_idx: cardIdx }),
                card.title,
              )
            }}
```

- [ ] **Step 4: Update `ColumnHeader.tsx` delete call site**

Find:
```tsx
                onSelect={() =>
                  stageDelete(mutation, { type: 'delete_column', name }, name)
                }
```
Replace with:
```tsx
                onSelect={() =>
                  stageDelete(() => mutation.mutate({ type: 'delete_column', name }), name)
                }
```

- [ ] **Step 5: Run + commit**

```bash
cd /Users/htruong/code/htruong/liveboard && bun --cwd web/renderer/default test && bun --cwd web/renderer/default run typecheck
```
Expected: all green, only pre-existing TS6196.

```bash
git add web/renderer/default/src/mutations/undoable.ts web/renderer/default/src/mutations/undoable.test.ts \
        web/renderer/default/src/components/CardEditable.tsx web/renderer/default/src/components/ColumnHeader.tsx
git commit -m "refactor(renderer): generalize stageDelete to (fire, label)

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: `ActiveBoardContext`

**Files:**
- Create: `web/renderer/default/src/contexts/ActiveBoardContext.tsx`
- Create: `web/renderer/default/src/contexts/ActiveBoardContext.test.tsx`

- [ ] **Step 1: Tests first**

Create `web/renderer/default/src/contexts/ActiveBoardContext.test.tsx`:
```tsx
import { describe, expect, it } from 'bun:test'
import { act, renderHook } from '@testing-library/react'
import { ActiveBoardProvider, useActiveBoard } from './ActiveBoardContext.js'

describe('ActiveBoardContext', () => {
  it('starts with active=null', () => {
    const { result } = renderHook(() => useActiveBoard(), { wrapper: ActiveBoardProvider })
    expect(result.current.active).toBeNull()
  })

  it('setActive updates the context', () => {
    const { result } = renderHook(() => useActiveBoard(), { wrapper: ActiveBoardProvider })
    act(() => result.current.setActive('foo'))
    expect(result.current.active).toBe('foo')
    act(() => result.current.setActive(null))
    expect(result.current.active).toBeNull()
  })

  it('throws when used outside provider', () => {
    expect(() => renderHook(() => useActiveBoard())).toThrow()
  })
})
```

- [ ] **Step 2: Run, expect fail (module missing)**

```bash
cd web/renderer/default && bun test src/contexts/ActiveBoardContext.test.tsx
```

- [ ] **Step 3: Implement**

Create `web/renderer/default/src/contexts/ActiveBoardContext.tsx`:
```tsx
import { createContext, useContext, useState, type ReactNode } from 'react'

interface ActiveBoardCtx {
  active: string | null
  setActive: (next: string | null) => void
}

const Ctx = createContext<ActiveBoardCtx | null>(null)

export function ActiveBoardProvider({ children }: { children: ReactNode }): JSX.Element {
  const [active, setActive] = useState<string | null>(null)
  return <Ctx.Provider value={{ active, setActive }}>{children}</Ctx.Provider>
}

export function useActiveBoard(): ActiveBoardCtx {
  const v = useContext(Ctx)
  if (!v) throw new Error('useActiveBoard must be used within ActiveBoardProvider')
  return v
}
```

- [ ] **Step 4: Run, expect 3 pass**

```bash
cd web/renderer/default && bun test src/contexts/ActiveBoardContext.test.tsx && bun run typecheck
```

- [ ] **Step 5: Commit**

```bash
git add web/renderer/default/src/contexts/ActiveBoardContext.tsx web/renderer/default/src/contexts/ActiveBoardContext.test.tsx
git commit -m "feat(renderer): add ActiveBoardContext for shared active board state

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: `errorToast` `ALREADY_EXISTS` copy

**Files:**
- Modify: `web/renderer/default/src/toast.ts`

- [ ] **Step 1: Extend the map**

Read `web/renderer/default/src/toast.ts`. Inside the `errorToast` function's `copy` map, add:
```ts
    ALREADY_EXISTS: 'A board with that name already exists',
```

- [ ] **Step 2: Verify**

```bash
cd web/renderer/default && bun run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/renderer/default/src/toast.ts
git commit -m "feat(renderer): add ALREADY_EXISTS toast copy

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 4: `useBoardListEvents`

**Files:**
- Create: `web/renderer/default/src/mutations/useBoardListEvents.ts`

Tiny hook that wires `client.on('board.list.updated', ...)` to invalidate `['boards']`. No tests — covered by the integration tests in Task 5.

- [ ] **Step 1: Implement**

Create `web/renderer/default/src/mutations/useBoardListEvents.ts`:
```ts
import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useClient } from '../queries.js'

export function useBoardListEvents(): void {
  const client = useClient()
  const qc = useQueryClient()
  useEffect(() => {
    const off = client.on('board.list.updated', () => {
      void qc.invalidateQueries({ queryKey: ['boards'] })
    })
    return () => {
      off()
    }
  }, [client, qc])
}
```

Note: `client.on` returns an unsubscribe function — verify by reading `web/shared/src/client.ts`. If it returns something else (e.g. a `Subscription` with `.close()`), adapt the cleanup. The existing P4a `useBoard` `useEffect` already wires `client.on('board.updated', ...)` — match that pattern.

- [ ] **Step 2: Typecheck + commit**

```bash
cd web/renderer/default && bun run typecheck
git add web/renderer/default/src/mutations/useBoardListEvents.ts
git commit -m "feat(renderer): add useBoardListEvents hook

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 5: `useBoardCrud` hooks + tests

**Files:**
- Create: `web/renderer/default/src/mutations/useBoardCrud.ts`
- Create: `web/renderer/default/src/mutations/useBoardCrud.test.tsx`

Three hooks: `useCreateBoard`, `useRenameBoard`, `useDeleteBoard`. Each fires `errorToast(code)` on protocol failure.

- [ ] **Step 1: Implement**

Create `web/renderer/default/src/mutations/useBoardCrud.ts`:
```ts
import { useMutation, useQueryClient, type UseMutationResult } from '@tanstack/react-query'
import type { BoardSummary } from '@shared/adapter.js'
import { ProtocolError } from '@shared/protocol.js'
import { useClient } from '../queries.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { errorToast } from '../toast.js'

function code(err: unknown): string {
  return err instanceof ProtocolError ? err.code : 'INTERNAL'
}

export function useCreateBoard(): UseMutationResult<BoardSummary, Error, string> {
  const client = useClient()
  const qc = useQueryClient()
  const { setActive } = useActiveBoard()
  return useMutation<BoardSummary, Error, string>({
    mutationFn: (name) => client.createBoard(name),
    onSuccess: (summary) => {
      void qc.invalidateQueries({ queryKey: ['boards'] })
      setActive(summary.id)
    },
    onError: (err) => errorToast(code(err)),
  })
}

interface RenameVars {
  boardId: string
  newName: string
}

export function useRenameBoard(): UseMutationResult<BoardSummary, Error, RenameVars> {
  const client = useClient()
  const qc = useQueryClient()
  const { active, setActive } = useActiveBoard()
  return useMutation<BoardSummary, Error, RenameVars>({
    mutationFn: ({ boardId, newName }) => client.renameBoard(boardId, newName),
    onSuccess: (summary, { boardId }) => {
      void qc.invalidateQueries({ queryKey: ['boards'] })
      if (active === boardId) setActive(summary.id)
    },
    onError: (err) => errorToast(code(err)),
  })
}

interface DeleteCtx {
  fallbackActive: string | null
}

export function useDeleteBoard(): UseMutationResult<void, Error, string, DeleteCtx> {
  const client = useClient()
  const qc = useQueryClient()
  const { active, setActive } = useActiveBoard()
  return useMutation<void, Error, string, DeleteCtx>({
    mutationFn: (boardId) => client.deleteBoard(boardId),
    onMutate: (boardId) => {
      const list = qc.getQueryData<BoardSummary[]>(['boards']) ?? []
      const fallbackActive = list.find((b) => b.id !== boardId)?.id ?? null
      return { fallbackActive }
    },
    onSuccess: (_void, boardId, ctx) => {
      void qc.invalidateQueries({ queryKey: ['boards'] })
      if (active === boardId) setActive(ctx?.fallbackActive ?? null)
    },
    onError: (err) => errorToast(code(err)),
  })
}
```

- [ ] **Step 2: Tests**

Create `web/renderer/default/src/mutations/useBoardCrud.test.tsx`:
```tsx
import { describe, expect, it } from 'bun:test'
import { act, renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { ActiveBoardProvider, useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { useCreateBoard, useRenameBoard, useDeleteBoard } from './useBoardCrud.js'

async function setup(): Promise<{ client: Client; qc: QueryClient; wrap: (children: ReactNode) => JSX.Element }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  // Seed the boards list cache so onMutate snapshots are realistic.
  qc.setQueryData(['boards'], await client.listBoards())
  const wrap = (children: ReactNode): JSX.Element => (
    <ClientProvider client={client}>
      <QueryClientProvider client={qc}>
        <ActiveBoardProvider>{children}</ActiveBoardProvider>
      </QueryClientProvider>
    </ClientProvider>
  )
  return { client, qc, wrap }
}

function combined(): { create: ReturnType<typeof useCreateBoard>; rename: ReturnType<typeof useRenameBoard>; del: ReturnType<typeof useDeleteBoard>; ab: ReturnType<typeof useActiveBoard> } {
  return {
    create: useCreateBoard(),
    rename: useRenameBoard(),
    del: useDeleteBoard(),
    ab: useActiveBoard(),
  }
}

describe('useBoardCrud', () => {
  it('useCreateBoard sets new board active', async () => {
    const { wrap } = await setup()
    const { result } = renderHook(combined, { wrapper: ({ children }) => wrap(children) })
    await act(async () => {
      result.current.create.mutate('Foo')
    })
    await waitFor(() => expect(result.current.create.isSuccess).toBe(true))
    expect(result.current.ab.active).toBe('foo')
  })

  it('useRenameBoard switches active to new id when active was renamed', async () => {
    const { wrap } = await setup()
    const { result } = renderHook(combined, { wrapper: ({ children }) => wrap(children) })
    await act(async () => {
      result.current.create.mutate('Foo')
    })
    await waitFor(() => expect(result.current.ab.active).toBe('foo'))
    await act(async () => {
      result.current.rename.mutate({ boardId: 'foo', newName: 'Bar' })
    })
    await waitFor(() => expect(result.current.ab.active).toBe('bar'))
  })

  it('useRenameBoard leaves active untouched when renaming a different board', async () => {
    const { wrap } = await setup()
    const { result } = renderHook(combined, { wrapper: ({ children }) => wrap(children) })
    await act(async () => {
      result.current.create.mutate('Foo')
    })
    await waitFor(() => expect(result.current.ab.active).toBe('foo'))
    // Active stays 'foo' while we rename 'welcome' → 'Welcomed'.
    await act(async () => {
      result.current.rename.mutate({ boardId: 'welcome', newName: 'Welcomed' })
    })
    await waitFor(() => expect(result.current.rename.isSuccess).toBe(true))
    expect(result.current.ab.active).toBe('foo')
  })

  it('useDeleteBoard switches active to first remaining when active was deleted', async () => {
    const { wrap, qc, client } = await setup()
    const { result } = renderHook(combined, { wrapper: ({ children }) => wrap(children) })
    await act(async () => {
      result.current.create.mutate('Foo')
    })
    await waitFor(() => expect(result.current.ab.active).toBe('foo'))
    // Refresh the boards cache so onMutate sees the up-to-date list.
    qc.setQueryData(['boards'], await client.listBoards())
    await act(async () => {
      result.current.del.mutate('foo')
    })
    await waitFor(() => expect(result.current.del.isSuccess).toBe(true))
    expect(result.current.ab.active).toBe('welcome')
  })

  it('useCreateBoard surfaces ALREADY_EXISTS via toast (no throw)', async () => {
    const { wrap } = await setup()
    const { result } = renderHook(combined, { wrapper: ({ children }) => wrap(children) })
    await act(async () => {
      result.current.create.mutate('Welcome')
    })
    await waitFor(() => expect(result.current.create.isError).toBe(true))
  })
})
```

- [ ] **Step 3: Run + commit**

```bash
cd /Users/htruong/code/htruong/liveboard && bun --cwd web/renderer/default test src/mutations/useBoardCrud.test.tsx && bun --cwd web/renderer/default run typecheck
git add web/renderer/default/src/mutations/useBoardCrud.ts web/renderer/default/src/mutations/useBoardCrud.test.tsx
git commit -m "feat(renderer): add useCreateBoard / useRenameBoard / useDeleteBoard hooks

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```
Expected: 5 pass.

---

## Task 6: `AddBoardButton`

**Files:**
- Create: `web/renderer/default/src/components/AddBoardButton.tsx`
- Create: `web/renderer/default/src/components/AddBoardButton.test.tsx`

Mirrors `AddColumnButton`. Uncontrolled input + committedRef + deferred close.

- [ ] **Step 1: Component**

Create `web/renderer/default/src/components/AddBoardButton.tsx`:
```tsx
import { useState, useRef, useEffect } from 'react'
import { useCreateBoard } from '../mutations/useBoardCrud.js'

export function AddBoardButton(): JSX.Element {
  const [open, setOpen] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useCreateBoard()
  const committedRef = useRef(false)

  useEffect(() => {
    if (open) {
      committedRef.current = false
      inputRef.current?.focus()
    }
  }, [open])

  const commit = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    const name = (inputRef.current?.value ?? '').trim()
    if (name) mutation.mutate(name)
    Promise.resolve().then(() => setOpen(false))
  }

  const cancel = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    Promise.resolve().then(() => setOpen(false))
  }

  if (open) {
    return (
      <div className="border-t border-slate-200 p-2">
        <input
          ref={inputRef}
          aria-label="new board name"
          defaultValue=""
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === 'Enter') { e.preventDefault(); commit() }
            else if (e.key === 'Escape') { e.preventDefault(); cancel() }
          }}
          placeholder="Board name…"
          className="w-full rounded bg-white px-2 py-1 text-sm outline-none ring-1 ring-slate-200 focus:ring-blue-400"
        />
      </div>
    )
  }

  return (
    <button
      type="button"
      onClick={() => setOpen(true)}
      className="border-t border-slate-200 px-3 py-2 text-left text-sm text-slate-500 hover:bg-slate-50"
    >
      + New board
    </button>
  )
}
```

- [ ] **Step 2: Tests**

Create `web/renderer/default/src/components/AddBoardButton.test.tsx`:
```tsx
import { describe, expect, it } from 'bun:test'
import { fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { ActiveBoardProvider } from '../contexts/ActiveBoardContext.js'
import { renderWithQuery } from '../test-utils.js'
import { AddBoardButton } from './AddBoardButton.js'

async function setup() {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['boards'], await client.listBoards())
  return { client, qc }
}

describe('AddBoardButton', () => {
  it('click reveals input', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <AddBoardButton />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByText('+ New board'))
    await waitFor(() => expect(getByLabelText('new board name')).toBeDefined())
  })

  it('blur with text creates a new board (sidebar list grows)', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <AddBoardButton />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByText('+ New board'))
    const input = await waitFor(() => getByLabelText('new board name')) as HTMLInputElement
    fireEvent.change(input, { target: { value: 'Foo' } })
    fireEvent.blur(input)
    await waitFor(async () => {
      const list = await client.listBoards()
      expect(list.map((s) => s.id)).toContain('foo')
    })
  })
})
```

- [ ] **Step 3: Run + commit**

```bash
cd web/renderer/default && bun test src/components/AddBoardButton.test.tsx && bun run typecheck
git add web/renderer/default/src/components/AddBoardButton.tsx web/renderer/default/src/components/AddBoardButton.test.tsx
git commit -m "feat(renderer): add AddBoardButton with inline name input

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```
Expected: 2 pass.

---

## Task 7: `BoardRow`

**Files:**
- Create: `web/renderer/default/src/components/BoardRow.tsx`
- Create: `web/renderer/default/src/components/BoardRow.test.tsx`

Single row: click selects, kebab → Rename / Delete. Inline rename input uses uncontrolled pattern.

- [ ] **Step 1: Component**

Create `web/renderer/default/src/components/BoardRow.tsx`:
```tsx
import { useState, useRef, useEffect } from 'react'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import type { BoardSummary } from '@shared/adapter.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { useRenameBoard, useDeleteBoard } from '../mutations/useBoardCrud.js'
import { stageDelete } from '../mutations/undoable.js'

export function BoardRow({ board }: { board: BoardSummary }): JSX.Element {
  const { active, setActive } = useActiveBoard()
  const [mode, setMode] = useState<'view' | 'edit'>('view')
  const inputRef = useRef<HTMLInputElement>(null)
  const renameMut = useRenameBoard()
  const deleteMut = useDeleteBoard()
  const committedRef = useRef(false)
  const isActive = active === board.id

  useEffect(() => {
    if (mode === 'edit') {
      committedRef.current = false
      inputRef.current?.focus()
      inputRef.current?.select()
    }
  }, [mode])

  const commitRename = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    const next = (inputRef.current?.value ?? '').trim()
    if (next && next !== board.name) {
      renameMut.mutate({ boardId: board.id, newName: next })
    }
    Promise.resolve().then(() => setMode('view'))
  }

  const cancelRename = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    Promise.resolve().then(() => setMode('view'))
  }

  if (mode === 'edit') {
    return (
      <li>
        <input
          ref={inputRef}
          aria-label={`rename board ${board.name}`}
          defaultValue={board.name}
          onBlur={commitRename}
          onKeyDown={(e) => {
            if (e.key === 'Enter') { e.preventDefault(); commitRename() }
            else if (e.key === 'Escape') { e.preventDefault(); cancelRename() }
          }}
          className="block w-full rounded bg-white px-2 py-1 text-sm outline-none ring-1 ring-blue-400"
        />
      </li>
    )
  }

  return (
    <li className="group flex items-center gap-1">
      <button
        type="button"
        onClick={() => setActive(board.id)}
        className={`flex flex-1 items-center gap-2 rounded px-2 py-1.5 text-left text-sm ${
          isActive ? 'bg-slate-200 text-slate-900' : 'text-slate-700 hover:bg-slate-100'
        }`}
      >
        {board.icon && <span aria-hidden>{board.icon}</span>}
        <span className="truncate">{board.name}</span>
      </button>
      <DropdownMenu.Root>
        <DropdownMenu.Trigger
          aria-label={`board menu ${board.name}`}
          className="rounded p-1 text-slate-400 opacity-0 group-hover:opacity-100 hover:bg-slate-200"
        >
          ⋮
        </DropdownMenu.Trigger>
        <DropdownMenu.Portal>
          <DropdownMenu.Content
            sideOffset={4}
            className="z-50 min-w-32 rounded-md bg-white p-1 shadow-lg ring-1 ring-slate-200"
          >
            <DropdownMenu.Item
              onSelect={() => setMode('edit')}
              className="cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-slate-100"
            >
              Rename
            </DropdownMenu.Item>
            <DropdownMenu.Separator className="my-1 h-px bg-slate-200" />
            <DropdownMenu.Item
              onSelect={() =>
                stageDelete(() => deleteMut.mutate(board.id), board.name)
              }
              className="cursor-pointer rounded px-2 py-1 text-sm text-red-600 outline-none hover:bg-red-50"
            >
              Delete
            </DropdownMenu.Item>
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
    </li>
  )
}
```

- [ ] **Step 2: Tests**

Create `web/renderer/default/src/components/BoardRow.test.tsx`:
```tsx
import { describe, expect, it } from 'bun:test'
import { fireEvent } from '@testing-library/react'
import { QueryClient } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { ActiveBoardProvider } from '../contexts/ActiveBoardContext.js'
import { renderWithQuery } from '../test-utils.js'
import { BoardRow } from './BoardRow.js'

async function setup() {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['boards'], await client.listBoards())
  return { client, qc }
}

const board = { id: 'foo', name: 'Foo', version: 1 }

describe('BoardRow', () => {
  it('renders name and menu trigger', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <ul><BoardRow board={board} /></ul>
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    expect(getByText('Foo')).toBeDefined()
    expect(getByLabelText('board menu Foo')).toBeDefined()
  })

  it('clicking the row attempts to set active (smoke)', async () => {
    const { client, qc } = await setup()
    // Just verify no throw and the row button is clickable.
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <ul><BoardRow board={board} /></ul>
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByText('Foo'))
  })
})
```

(Radix menu interactions tend to be flaky under happy-dom — match the P4b.1a `ColumnHeader.test.tsx` minimal-coverage approach. Manual smoke covers the rename/delete UX in Task 11.)

- [ ] **Step 3: Run + commit**

```bash
cd web/renderer/default && bun test src/components/BoardRow.test.tsx && bun run typecheck
git add web/renderer/default/src/components/BoardRow.tsx web/renderer/default/src/components/BoardRow.test.tsx
git commit -m "feat(renderer): add BoardRow with kebab menu and inline rename

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```
Expected: 2 pass.

---

## Task 8: Wire `BoardSidebar` and drop its props

**Files:**
- Modify: `web/renderer/default/src/components/BoardSidebar.tsx`

Remove `activeId` + `onSelect` props. Render `BoardRow` per board, render `AddBoardButton` at the bottom.

- [ ] **Step 1: Replace file**

Replace `web/renderer/default/src/components/BoardSidebar.tsx` with:
```tsx
import { useBoardList, useWorkspaceInfo } from '../queries.js'
import { EmptyState } from './EmptyState.js'
import { BoardRow } from './BoardRow.js'
import { AddBoardButton } from './AddBoardButton.js'

export function BoardSidebar(): JSX.Element {
  const boards = useBoardList()
  const ws = useWorkspaceInfo()

  return (
    <aside className="flex h-full w-60 shrink-0 flex-col border-r border-slate-200 bg-white">
      <header className="border-b border-slate-200 p-3">
        <p className="text-xs uppercase tracking-wide text-slate-500">Workspace</p>
        <p className="truncate text-sm font-semibold text-slate-800">
          {ws.data?.name ?? '—'}
        </p>
      </header>
      <div className="flex-1 overflow-y-auto p-2">
        {boards.isLoading ? (
          <EmptyState title="Loading…" />
        ) : boards.error ? (
          <EmptyState title="Failed to load" detail={String(boards.error)} />
        ) : !boards.data || boards.data.length === 0 ? (
          <EmptyState title="No boards yet" />
        ) : (
          <ul className="flex flex-col gap-1">
            {boards.data.map((b) => (
              <BoardRow key={b.id} board={b} />
            ))}
          </ul>
        )}
      </div>
      <AddBoardButton />
    </aside>
  )
}
```

- [ ] **Step 2: Existing `BoardSidebar.test.tsx` will break — update it**

Read `web/renderer/default/src/components/BoardSidebar.test.tsx`. The test currently passes `activeId` + `onSelect` props. Update the render call to drop those props, and wrap in `<ActiveBoardProvider>` so `BoardRow` (used inside) has access to context.

Replace the test file with:
```tsx
import { describe, expect, it } from 'bun:test'
import { waitFor } from '@testing-library/react'
import { QueryClient } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { ActiveBoardProvider } from '../contexts/ActiveBoardContext.js'
import { renderWithQuery } from '../test-utils.js'
import { BoardSidebar } from './BoardSidebar.js'

async function setup() {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return { client, qc }
}

describe('BoardSidebar', () => {
  it('renders boards from the cache', async () => {
    const { client, qc } = await setup()
    const { findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <BoardSidebar />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    await findByText('Welcome')
    await findByText('+ New board')
  })

  it('renders workspace name', async () => {
    const { client, qc } = await setup()
    const { findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <BoardSidebar />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    await waitFor(() => findByText('Demo'))
  })
})
```

(If the seeded workspace name isn't `'Demo'`, swap to whatever `WORKSPACE_NAME` is in `web/shared/src/adapters/local-seed.ts` — read the file to confirm.)

- [ ] **Step 3: Typecheck — expect App.tsx to break**

```bash
cd web/renderer/default && bun run typecheck
```
Expected: errors in `App.tsx` because `BoardSidebar` no longer accepts the props it's passing. That's covered in Task 9.

- [ ] **Step 4: Don't commit yet** — `BoardSidebar` change + `App.tsx` change must land together to keep the tree green.

---

## Task 9: Wire `App` and `BoardView` to context; mount `useBoardListEvents`; self-recover on NOT_FOUND

**Files:**
- Modify: `web/renderer/default/src/App.tsx`
- Modify: `web/renderer/default/src/components/BoardView.tsx`

App owns the provider and mounts the cross-tab event listener. BoardView reads from context and recovers from `NOT_FOUND`.

- [ ] **Step 1: Replace `App.tsx`**

```tsx
import type { Client } from '@shared/client.js'
import { BoardSidebar } from './components/BoardSidebar.js'
import { BoardView } from './components/BoardView.js'
import { Toaster } from './toast.js'
import { ActiveBoardProvider } from './contexts/ActiveBoardContext.js'
import { useBoardListEvents } from './mutations/useBoardListEvents.js'

function ListEventsBridge(): null {
  useBoardListEvents()
  return null
}

export function App({ client }: { client: Client }): JSX.Element {
  return (
    <ActiveBoardProvider>
      <ListEventsBridge />
      <div className="flex h-screen w-screen">
        <BoardSidebar />
        <main className="flex-1 overflow-hidden">
          <BoardView client={client} />
        </main>
        <Toaster position="bottom-right" richColors closeButton />
      </div>
    </ActiveBoardProvider>
  )
}
```

- [ ] **Step 2: Replace `BoardView.tsx`**

Read current `web/renderer/default/src/components/BoardView.tsx`. Replace the prop-based `boardId` with context, and add a self-recovery effect on `NOT_FOUND`:
```tsx
import { useEffect } from 'react'
import { SortableContext, horizontalListSortingStrategy } from '@dnd-kit/sortable'
import type { Client } from '@shared/client.js'
import { ProtocolError } from '@shared/protocol.js'
import { useBoard } from '../queries.js'
import { EmptyState } from './EmptyState.js'
import { AddColumnButton } from './AddColumnButton.js'
import { BoardDndContext } from '../dnd/BoardDndContext.js'
import { SortableColumn } from '../dnd/SortableColumn.js'
import { encodeColumnId } from '../dnd/cardId.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'

export function BoardView({ client }: { client: Client }): JSX.Element {
  const { active, setActive } = useActiveBoard()
  const { data, isLoading, error } = useBoard(active)

  useEffect(() => {
    if (!active) return
    void client.subscribe(active)
    return () => {
      void client.unsubscribe(active)
    }
  }, [active, client])

  // Self-recover if the active board was deleted (e.g. cross-tab).
  useEffect(() => {
    if (error instanceof ProtocolError && error.code === 'NOT_FOUND') {
      setActive(null)
    }
  }, [error, setActive])

  if (!active) return <EmptyState title="Select a board" />
  if (isLoading) return <EmptyState title="Loading…" />
  if (error) return <EmptyState title="Failed to load board" detail={String(error)} />
  if (!data) return <EmptyState title="Board not found" />

  const columns = data.columns ?? []
  if (columns.length === 0) {
    return (
      <div className="flex h-full gap-4 overflow-x-auto p-4">
        <AddColumnButton boardId={active} />
      </div>
    )
  }

  const names = columns.map((c) => c.name)
  const columnIds = names.map(encodeColumnId)

  return (
    <BoardDndContext boardId={active}>
      <SortableContext items={columnIds} strategy={horizontalListSortingStrategy}>
        <div className="flex h-full gap-4 overflow-x-auto p-4">
          {columns.map((col, i) => (
            <SortableColumn
              key={`${col.name}-${i}`}
              column={col}
              colIdx={i}
              allColumnNames={names}
              boardId={active}
            />
          ))}
          <AddColumnButton boardId={active} />
        </div>
      </SortableContext>
    </BoardDndContext>
  )
}
```

- [ ] **Step 3: Existing `BoardView.test.tsx` may pass props that no longer exist**

Read `web/renderer/default/src/components/BoardView.test.tsx`. If it passes `boardId={...}` to `<BoardView>`, drop that prop and instead seed `ActiveBoardProvider` with the active id by wrapping render in:
```tsx
<ActiveBoardProvider>
  <SeedActive id="welcome" />
  <BoardView client={client} />
</ActiveBoardProvider>
```
where `SeedActive` is a small helper:
```tsx
function SeedActive({ id }: { id: string }): null {
  const { active, setActive } = useActiveBoard()
  if (active !== id) setActive(id)
  return null
}
```
Add `SeedActive` near the top of the test file. (Only adapt as needed — if the existing test was minimal and didn't depend on a specific board id, just wrap in `<ActiveBoardProvider>` and assert the "Select a board" empty state instead.)

- [ ] **Step 4: Run full suite + typecheck**

```bash
cd /Users/htruong/code/htruong/liveboard && bun --cwd web/renderer/default test && bun --cwd web/renderer/default run typecheck
```
Expected: all green; only pre-existing TS6196.

- [ ] **Step 5: Single commit** for Tasks 8 + 9

```bash
git add web/renderer/default/src/components/BoardSidebar.tsx \
        web/renderer/default/src/components/BoardSidebar.test.tsx \
        web/renderer/default/src/App.tsx \
        web/renderer/default/src/components/BoardView.tsx \
        web/renderer/default/src/components/BoardView.test.tsx
git commit -m "feat(renderer): wire sidebar to ActiveBoardContext + cross-tab events

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 10: Build + bundle measurement

**Files:** none.

- [ ] **Step 1: Build**

```bash
cd /Users/htruong/code/htruong/liveboard && make renderer
```
Expected: clean build.

- [ ] **Step 2: Measure**

```bash
gzip -c web/renderer/default/dist/assets/*.js | wc -c
```
Record the byte count. Expected ~120 KB (negligible delta from P4b.3 since no new deps).

- [ ] **Step 3: Verify Go embed**

```bash
go test ./internal/api/ -run TestShellRoute
```
Expected: 3 tests pass.

- [ ] **Step 4: No commit.**

---

## Task 11: Manual browser smoke

Not a code change. Final gate.

- [ ] **Step 1: Rebuild + serve**

```bash
make shell && make renderer
LIVEBOARD_APP_SHELL=1 go run ./cmd/liveboard serve --dir ./demo --port 7070
```

- [ ] **Step 2: At <http://localhost:7070/app/> verify**

1. Sidebar shows existing boards + "+ New board" at the bottom.
2. Click "+ New board" → input appears → type "Manual Smoke" → Enter → new "Manual Smoke" row appears, becomes active, BoardView shows it.
3. Hover a board row → kebab `⋮` appears at right.
4. Open kebab on any non-active board → click Rename → input replaces label → type new name → Enter → row updates; not active.
5. Open kebab on the *active* board → Rename → new name → Enter → row updates AND BoardView still shows the renamed board (active id followed the rename).
6. Open kebab on the active board → Delete → undo toast → wait 5s → board gone, BoardView switches to first remaining board.
7. Open kebab on a non-active board → Delete → undo toast → click Undo within 5s → board stays.
8. Try to create a board with an existing name → toast "A board with that name already exists".
9. Open a second tab on `/app/` → in tab A, create a new board → tab B's sidebar refreshes (no manual reload) and shows the new board.
10. In tab A, delete the active board (tab B is also viewing it) → tab B's BoardView falls back to the empty state without crashing (NOT_FOUND self-recovery effect).
11. `?renderer=stub` still loads.

- [ ] **Step 3: Report.** If anything fails, capture step + expected vs actual.

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| `<ActiveBoardContext>` + `useActiveBoard()` | 2 |
| `useCreateBoard` / `useRenameBoard` / `useDeleteBoard` | 5 |
| `useBoardListEvents` cross-tab refresh | 4, 9 (mounted) |
| `<AddBoardButton>` | 6 |
| `<BoardRow>` with kebab + Rename + Delete | 7 |
| Generalized `stageDelete(fire, label)` | 1 |
| `errorToast` `ALREADY_EXISTS` copy | 3 |
| `<BoardSidebar>` rewires to context | 8 |
| `App.tsx` provider + bridge | 9 |
| `<BoardView>` reads from context + NOT_FOUND self-recovery | 9 |
| Manual browser smoke | 11 |
| Bundle check | 10 |

## Notes for implementer

1. **Tasks 8 and 9 land in one commit** because `BoardSidebar` losing its props breaks `App.tsx` until `App.tsx` is rewired. Don't commit Task 8 alone.
2. **Inputs use uncontrolled pattern** with `committedRef` + deferred `setMode/setOpen` via `Promise.resolve().then(...)` — established in P4b.1a to dodge happy-dom's `keyDown` bug and to defer state changes out of event-handler stacks.
3. **Radix DropdownMenu tests are minimal** — happy-dom's portal/focus-trap support is shaky. Match `ColumnHeader.test.tsx`'s coverage: assert the trigger renders, leave menu interactions to manual smoke.
4. **`client.on` return type** — verify via `web/shared/src/client.ts`. If it's a `Subscription` (`{ close() {} }`), adapt `useBoardListEvents`'s cleanup accordingly. The P4a `useBoard` already wires this pattern.
5. **`NOT_FOUND` self-recovery** in `BoardView` is a simple `useEffect`. Only fires when `error instanceof ProtocolError` with `code === 'NOT_FOUND'` — not on transient network errors.
6. **No commit amending** — forward-only commits.
