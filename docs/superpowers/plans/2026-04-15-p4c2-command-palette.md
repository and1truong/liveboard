# P4c.2 — Command Palette Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Cmd+K / Ctrl+K command palette to the `/app/` default renderer that lets users jump to any board, create a new board, rename the current board, or delete the current board. Built on `cmdk` + Radix Dialog. Reuses the P4c.1 CRUD hooks.

**Architecture:** Single new component `<CommandPalette />` mounted in `App`. Internal state: `open` (toggled by global Cmd+K listener) + `page` ('list' | 'create' | 'rename'). cmdk handles filterable list and arrow nav; Radix Dialog handles overlay, focus trap, escape. Mutations route through existing `useCreateBoard / useRenameBoard / useDeleteBoard`.

**Tech Stack:** P4c.1 stack + `cmdk`. No other new deps.

**Spec:** `docs/superpowers/specs/2026-04-15-p4c2-command-palette-design.md`

**Conventions:**
- All new code under `web/renderer/default/src/components/`.
- Imports use `@shared/*` alias.
- Tests colocated.
- Commit prefixes: `feat(renderer)`, `chore(build)`.
- Pre-existing `TS6196` in `web/shared/src/protocol.ts` is NOT a blocker.
- Use bun, never npx.
- Inputs: uncontrolled pattern (`defaultValue` + ref + `committedRef`) — happy-dom keyDown bug.

---

## File structure

**New:**
- `web/renderer/default/src/components/CommandPalette.tsx`
- `web/renderer/default/src/components/CommandPalette.test.tsx`

**Modified:**
- `web/renderer/default/package.json` — add `cmdk`.
- `web/renderer/default/src/App.tsx` — mount `<CommandPalette />` inside `<ActiveBoardProvider>`.

---

## Task 1: Install `cmdk`

**Files:**
- Modify: `web/renderer/default/package.json`

- [ ] **Step 1: Add dep**

Edit `web/renderer/default/package.json`. Under `"dependencies"` (alphabetical position), add:
```json
    "cmdk": "^1.0.0",
```
Final dependencies block:
```json
  "dependencies": {
    "@dnd-kit/core": "^6.1.0",
    "@dnd-kit/sortable": "^8.0.0",
    "@dnd-kit/utilities": "^3.2.2",
    "@radix-ui/react-dialog": "^1.1.2",
    "@radix-ui/react-dropdown-menu": "^2.1.2",
    "@tanstack/react-query": "^5.59.0",
    "cmdk": "^1.0.0",
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "sonner": "^1.7.0"
  },
```

- [ ] **Step 2: Install**

```bash
cd web/renderer/default && bun install
```

- [ ] **Step 3: Smoke import**

```bash
cd web/renderer/default && bun -e "import('cmdk').then(m=>console.log(typeof m.Command))"
```
Expected: `function` or `object`.

- [ ] **Step 4: Commit**

```bash
git add web/renderer/default/package.json web/renderer/default/bun.lock
git commit -m "chore(build): add cmdk dep for command palette

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: `CommandPalette` component

**Files:**
- Create: `web/renderer/default/src/components/CommandPalette.tsx`

- [ ] **Step 1: Component**

Create `web/renderer/default/src/components/CommandPalette.tsx`:
```tsx
import { useEffect, useRef, useState, type FormEvent } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { Command } from 'cmdk'
import { useBoardList } from '../queries.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import {
  useCreateBoard,
  useRenameBoard,
  useDeleteBoard,
} from '../mutations/useBoardCrud.js'
import { stageDelete } from '../mutations/undoable.js'

type Page = 'list' | 'create' | 'rename'

export function CommandPalette(): JSX.Element {
  const [open, setOpen] = useState(false)
  const [page, setPage] = useState<Page>('list')
  const inputRef = useRef<HTMLInputElement>(null)
  const committedRef = useRef(false)

  const boards = useBoardList()
  const { active, setActive } = useActiveBoard()
  const createMut = useCreateBoard()
  const renameMut = useRenameBoard()
  const deleteMut = useDeleteBoard()

  const activeBoard = boards.data?.find((b) => b.id === active) ?? null
  const activeName = activeBoard?.name ?? ''

  // Global Cmd+K / Ctrl+K toggle.
  useEffect(() => {
    const handler = (e: KeyboardEvent): void => {
      if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        setOpen((v) => !v)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  // Reset to list page on every (re)open.
  useEffect(() => {
    if (open) {
      setPage('list')
      committedRef.current = false
    }
  }, [open])

  // Focus input on page change to create/rename.
  useEffect(() => {
    if (open && (page === 'create' || page === 'rename')) {
      committedRef.current = false
      // Defer to next tick so the input has mounted.
      setTimeout(() => {
        inputRef.current?.focus()
        inputRef.current?.select()
      }, 0)
    }
  }, [open, page])

  const close = (): void => {
    setOpen(false)
  }

  const submitCreate = (e: FormEvent): void => {
    e.preventDefault()
    if (committedRef.current) return
    committedRef.current = true
    const name = (inputRef.current?.value ?? '').trim()
    if (name) createMut.mutate(name)
    close()
  }

  const submitRename = (e: FormEvent): void => {
    e.preventDefault()
    if (committedRef.current) return
    committedRef.current = true
    const next = (inputRef.current?.value ?? '').trim()
    if (active && next && next !== activeName) {
      renameMut.mutate({ boardId: active, newName: next })
    }
    close()
  }

  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/40" />
        <Dialog.Content
          aria-label="Command palette"
          className="fixed left-1/2 top-[20%] z-50 w-full max-w-lg -translate-x-1/2 rounded-lg bg-white p-2 shadow-xl"
        >
          {page === 'list' && (
            <Command label="Command palette" className="flex flex-col gap-1">
              <Command.Input
                placeholder="Type a command or board name…"
                className="w-full rounded px-3 py-2 text-base outline-none placeholder:text-slate-400"
              />
              <Command.List className="max-h-80 overflow-y-auto">
                <Command.Empty className="px-3 py-2 text-sm text-slate-400">
                  No matches.
                </Command.Empty>
                {boards.data && boards.data.length > 0 && (
                  <Command.Group heading="Boards" className="text-xs uppercase text-slate-400 [&_[cmdk-group-heading]]:px-3 [&_[cmdk-group-heading]]:py-1">
                    {boards.data.map((b) => (
                      <Command.Item
                        key={b.id}
                        value={`board ${b.name}`}
                        onSelect={() => {
                          setActive(b.id)
                          close()
                        }}
                        className="cursor-pointer rounded px-3 py-1.5 text-sm text-slate-800 aria-selected:bg-slate-100"
                      >
                        {b.icon && <span aria-hidden className="mr-2">{b.icon}</span>}
                        {b.name}
                      </Command.Item>
                    ))}
                  </Command.Group>
                )}
                <Command.Group heading="Actions" className="text-xs uppercase text-slate-400 [&_[cmdk-group-heading]]:px-3 [&_[cmdk-group-heading]]:py-1">
                  <Command.Item
                    value="action create board"
                    onSelect={() => setPage('create')}
                    className="cursor-pointer rounded px-3 py-1.5 text-sm text-slate-800 aria-selected:bg-slate-100"
                  >
                    Create board
                  </Command.Item>
                  {active !== null && (
                    <>
                      <Command.Item
                        value="action rename current board"
                        onSelect={() => setPage('rename')}
                        className="cursor-pointer rounded px-3 py-1.5 text-sm text-slate-800 aria-selected:bg-slate-100"
                      >
                        Rename current board
                      </Command.Item>
                      <Command.Item
                        value="action delete current board"
                        onSelect={() => {
                          stageDelete(() => deleteMut.mutate(active), activeName)
                          close()
                        }}
                        className="cursor-pointer rounded px-3 py-1.5 text-sm text-red-600 aria-selected:bg-red-50"
                      >
                        Delete current board
                      </Command.Item>
                    </>
                  )}
                </Command.Group>
              </Command.List>
            </Command>
          )}

          {page === 'create' && (
            <form onSubmit={submitCreate} className="flex flex-col gap-1">
              <div className="px-3 pt-1 text-xs uppercase text-slate-400">New board</div>
              <input
                ref={inputRef}
                aria-label="new board name"
                defaultValue=""
                placeholder="Board name…"
                onKeyDown={(e) => {
                  if (e.key === 'Escape') { e.preventDefault(); close() }
                }}
                className="w-full rounded px-3 py-2 text-base outline-none placeholder:text-slate-400"
              />
            </form>
          )}

          {page === 'rename' && (
            <form onSubmit={submitRename} className="flex flex-col gap-1">
              <div className="px-3 pt-1 text-xs uppercase text-slate-400">Rename board</div>
              <input
                ref={inputRef}
                aria-label="rename current board"
                defaultValue={activeName}
                onKeyDown={(e) => {
                  if (e.key === 'Escape') { e.preventDefault(); close() }
                }}
                className="w-full rounded px-3 py-2 text-base outline-none"
              />
            </form>
          )}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
```

- [ ] **Step 2: Typecheck**

```bash
cd web/renderer/default && bun run typecheck
```
Expected: only pre-existing TS6196.

- [ ] **Step 3: Commit**

```bash
git add web/renderer/default/src/components/CommandPalette.tsx
git commit -m "feat(renderer): add CommandPalette (Cmd+K) with cmdk + Radix Dialog

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: Mount in `App`

**Files:**
- Modify: `web/renderer/default/src/App.tsx`

- [ ] **Step 1: Update `App.tsx`**

Read current `web/renderer/default/src/App.tsx`. Replace with:
```tsx
import type { Client } from '@shared/client.js'
import { BoardSidebar } from './components/BoardSidebar.js'
import { BoardView } from './components/BoardView.js'
import { CommandPalette } from './components/CommandPalette.js'
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
        <CommandPalette />
        <Toaster position="bottom-right" richColors closeButton />
      </div>
    </ActiveBoardProvider>
  )
}
```

- [ ] **Step 2: Run renderer suite**

```bash
cd web/renderer/default && bun test && bun run typecheck
```
Expected: all green.

- [ ] **Step 3: Commit**

```bash
git add web/renderer/default/src/App.tsx
git commit -m "feat(renderer): mount CommandPalette in App

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 4: `CommandPalette` tests

**Files:**
- Create: `web/renderer/default/src/components/CommandPalette.test.tsx`

cmdk runs filter logic synchronously, but Radix Portal mounts on a tick under happy-dom; wrap initial assertions in `waitFor`.

- [ ] **Step 1: Tests**

Create `web/renderer/default/src/components/CommandPalette.test.tsx`:
```tsx
import { describe, expect, it } from 'bun:test'
import { useEffect } from 'react'
import { fireEvent, waitFor } from '@testing-library/react'
import { QueryClient } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { ActiveBoardProvider, useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { renderWithQuery } from '../test-utils.js'
import { CommandPalette } from './CommandPalette.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['boards'], await client.listBoards())
  return { client, qc }
}

function SeedActive({ id }: { id: string | null }): null {
  const { setActive } = useActiveBoard()
  useEffect(() => { setActive(id) }, [id, setActive])
  return null
}

function ActiveProbe({ onChange }: { onChange: (id: string | null) => void }): null {
  const { active } = useActiveBoard()
  useEffect(() => { onChange(active) }, [active, onChange])
  return null
}

describe('CommandPalette', () => {
  it('Cmd+K opens the palette', async () => {
    const { client, qc } = await setup()
    const { queryByPlaceholderText, findByPlaceholderText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <CommandPalette />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    expect(queryByPlaceholderText('Type a command or board name…')).toBeNull()
    fireEvent.keyDown(window, { key: 'k', metaKey: true })
    await findByPlaceholderText('Type a command or board name…')
  })

  it('lists boards from cache', async () => {
    const { client, qc } = await setup()
    const { findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <CommandPalette />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.keyDown(window, { key: 'k', metaKey: true })
    await findByText('Welcome')
  })

  it('selecting a board sets active and closes', async () => {
    const { client, qc } = await setup()
    let activeSeen: string | null = null
    const { findByText, queryByPlaceholderText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <ActiveProbe onChange={(v) => { activeSeen = v }} />
          <CommandPalette />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.keyDown(window, { key: 'k', metaKey: true })
    const item = await findByText('Welcome')
    fireEvent.click(item)
    await waitFor(() => expect(activeSeen).toBe('welcome'))
    await waitFor(() => expect(queryByPlaceholderText('Type a command or board name…')).toBeNull())
  })

  it('Rename current board hidden when no active board', async () => {
    const { client, qc } = await setup()
    const { queryByText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <CommandPalette />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.keyDown(window, { key: 'k', metaKey: true })
    await findByText('Create board')
    expect(queryByText('Rename current board')).toBeNull()
    expect(queryByText('Delete current board')).toBeNull()
  })

  it('Rename current board visible when active is set', async () => {
    const { client, qc } = await setup()
    const { findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <CommandPalette />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.keyDown(window, { key: 'k', metaKey: true })
    await findByText('Rename current board')
  })
})
```

- [ ] **Step 2: Run + commit**

```bash
cd web/renderer/default && bun test src/components/CommandPalette.test.tsx && bun run typecheck
git add web/renderer/default/src/components/CommandPalette.test.tsx
git commit -m "test(renderer): cover CommandPalette open, list, select, conditional actions

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```
Expected: 5 pass.

If a test fails because cmdk's filter scoring ran asynchronously, wrap the assertion in a longer `waitFor`. If Radix Portal isn't ready, `await new Promise(r => setTimeout(r, 0))` after the keydown helps. Don't commit on red.

---

## Task 5: Build + bundle measurement

**Files:** none.

- [ ] **Step 1: Build**

```bash
cd /Users/htruong/code/htruong/liveboard && make renderer
```

- [ ] **Step 2: Measure**

```bash
gzip -c web/renderer/default/dist/assets/*.js | wc -c
```
Expected ~126 KB. Bundle gate stays deferred.

- [ ] **Step 3: Verify Go embed**

```bash
go test ./internal/api/ -run TestShellRoute
```
Expected: 3 tests pass.

- [ ] **Step 4: No commit.**

---

## Task 6: Manual browser smoke

Not a code change.

- [ ] **Step 1: Build + serve**

```bash
make adapter-test
```
(One-shot target added in P4c.1: builds shell + renderer, then runs `go run ./cmd/liveboard serve --dir ./demo --port 7070` with shell flag.)

- [ ] **Step 2: At <http://localhost:7070/app/> verify**

1. Press **Cmd+K** (or **Ctrl+K**) → palette appears with input + boards list + "Create board" action.
2. Press Cmd+K again → palette closes.
3. Open palette, type "wel" → only "Welcome" remains visible.
4. Press Down arrow until a board is highlighted → press Enter → palette closes, that board becomes active.
5. With no board active, open palette → "Rename current board" and "Delete current board" are NOT in the list. Only "Create board".
6. Select an active board first, then open palette → both Rename and Delete are visible.
7. Open palette → "Create board" → input page → type "Palette Test" → press Enter → board created and active.
8. Open palette → "Rename current board" → input prefilled with current name → edit → Enter → renames.
9. Open palette → "Delete current board" → undo toast appears → wait or click Undo.
10. Press Escape while open → closes.
11. `?renderer=stub` still loads.

- [ ] **Step 3: Report.** Capture failures with step + expected vs actual.

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| `cmdk` dep | 1 |
| `<CommandPalette>` controlled by Cmd/Ctrl+K | 2 |
| Three pages: list / create / rename | 2 |
| Boards filterable list | 2 |
| Three actions in list (Create / Rename / Delete) | 2 |
| Rename / Delete hidden when active is null | 2 |
| Delete fires `stageDelete` immediately | 2 |
| Mount in App alongside Toaster | 3 |
| Tests for open, list, select, conditional visibility | 4 |
| Bundle measurement | 5 |
| Manual smoke | 6 |

## Notes for implementer

1. **Page reset on reopen** — `useEffect` keyed on `open` resets `page` to `'list'`. Don't move it into a render path.
2. **Uncontrolled inputs** — create/rename inputs use `defaultValue` + `inputRef` reads + `committedRef`. Same pattern as P4b.1a / P4c.1 for happy-dom safety.
3. **Submit handlers use `<form onSubmit>`** for Enter handling — happy-dom dispatches `submit` reliably even when `keydown` fails. Prevent default in handler. Escape handled in `onKeyDown`.
4. **cmdk `value` strings** — each `Command.Item` has a unique `value` for filter matching. We use a prefix (`board ...`, `action ...`) so action items don't get filtered out by board-name typing — cmdk matches against `value`, so `"board welcome"` and `"action create board"` both contain "board" and remain visible during a "wel" filter. Acceptable.
5. **Bundle gate** still deferred to P4d.
6. **No commit amending** — forward-only commits.
