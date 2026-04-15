# P4b.3 — Card Detail Modal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Radix Dialog–based card detail modal to the `/app/` default renderer that edits all six fields of `edit_card` (title, body, tags, priority, due, assignee) in one form. Single click on a card body opens it; double-click on the title still inline-edits.

**Architecture:** `<CardDetailModal>` is a controlled Radix Dialog rendered by `<CardEditable>`. Local form state cloned from the `card` prop on each open. Save fires one `edit_card` mutation through the existing `useBoardMutation`. Cancel/Escape/overlay-click discards. No new mutation pathway, no shared changes.

**Tech Stack:** P4b.2 stack + `@radix-ui/react-dialog`.

**Spec:** `docs/superpowers/specs/2026-04-15-p4b3-card-detail-modal-design.md`

**Conventions:**
- All new code under `web/renderer/default/src/components/`.
- Imports use `@shared/*` alias.
- Tests colocated (`*.test.tsx`).
- Commit prefixes: `feat(renderer)`, `chore(build)`.
- Pre-existing `TS6196` in `web/shared/src/protocol.ts` is NOT a blocker.
- Use bun, never npx.
- Inputs use the **uncontrolled pattern** (`defaultValue` + ref reads) — `fireEvent.keyDown` on inputs is broken in happy-dom + React 18; established in P4b.1a.

---

## File structure

**New:**
- `web/renderer/default/src/components/CardDetailModal.tsx`
- `web/renderer/default/src/components/CardDetailModal.test.tsx`

**Modified:**
- `web/renderer/default/package.json` — add `@radix-ui/react-dialog`.
- `web/renderer/default/src/components/CardEditable.tsx` — `modalOpen` state; partition view-mode markup into title region (existing dblclick) and body region (new click); render `<CardDetailModal>`.
- `web/renderer/default/src/components/CardEditable.test.tsx` — add a "click body opens modal" case.

---

## Task 1: Install `@radix-ui/react-dialog`

**Files:**
- Modify: `web/renderer/default/package.json`

- [ ] **Step 1: Add dep**

Edit `web/renderer/default/package.json`. Under `"dependencies"` (alphabetical), add:
```json
    "@radix-ui/react-dialog": "^1.1.2",
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
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "sonner": "^1.7.0"
  },
```

- [ ] **Step 2: Install**

```bash
cd web/renderer/default && bun install
```
Expected: clean install; bun.lock updated.

- [ ] **Step 3: Smoke import**

```bash
cd web/renderer/default && bun -e "import('@radix-ui/react-dialog').then(m=>console.log(typeof m.Root))"
```
Expected: prints `object` or `function` (non-undefined).

- [ ] **Step 4: Commit**

```bash
git add web/renderer/default/package.json web/renderer/default/bun.lock
git commit -m "chore(build): add @radix-ui/react-dialog dep

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: `CardDetailModal` component

**Files:**
- Create: `web/renderer/default/src/components/CardDetailModal.tsx`

- [ ] **Step 1: Component**

Create `web/renderer/default/src/components/CardDetailModal.tsx`:
```tsx
import { useRef, useState, useEffect, type FormEvent } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import type { Card as CardModel } from '@shared/types.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'

export function CardDetailModal({
  card,
  colIdx,
  cardIdx,
  boardId,
  open,
  onOpenChange,
}: {
  card: CardModel
  colIdx: number
  cardIdx: number
  boardId: string
  open: boolean
  onOpenChange: (next: boolean) => void
}): JSX.Element {
  const titleRef = useRef<HTMLInputElement>(null)
  const bodyRef = useRef<HTMLTextAreaElement>(null)
  const tagsRef = useRef<HTMLInputElement>(null)
  const priorityRef = useRef<HTMLSelectElement>(null)
  const dueRef = useRef<HTMLInputElement>(null)
  const assigneeRef = useRef<HTMLInputElement>(null)

  const [titleValid, setTitleValid] = useState((card.title ?? '').trim().length > 0)
  const mutation = useBoardMutation(boardId)

  // Reset valid-state on each reopen so disabled state matches current seed.
  useEffect(() => {
    if (open) setTitleValid((card.title ?? '').trim().length > 0)
  }, [open, card.title])

  const submit = (e: FormEvent): void => {
    e.preventDefault()
    const title = (titleRef.current?.value ?? '').trim()
    if (!title) return
    const tags = (tagsRef.current?.value ?? '')
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean)
    mutation.mutate(
      {
        type: 'edit_card',
        col_idx: colIdx,
        card_idx: cardIdx,
        title,
        body: bodyRef.current?.value ?? '',
        tags,
        priority: priorityRef.current?.value ?? '',
        due: dueRef.current?.value ?? '',
        assignee: assigneeRef.current?.value ?? '',
      },
      {
        onSuccess: () => onOpenChange(false),
      },
    )
  }

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/40" />
        <Dialog.Content
          // Reseed default values on every open by remounting the form.
          key={String(open)}
          className="fixed left-1/2 top-1/2 z-50 w-full max-w-lg -translate-x-1/2 -translate-y-1/2 rounded-lg bg-white p-6 shadow-xl"
        >
          <Dialog.Title className="text-lg font-semibold text-slate-800">Edit card</Dialog.Title>
          <form onSubmit={submit} className="mt-4 space-y-3">
            <label className="block">
              <span className="block text-xs font-medium text-slate-600">Title</span>
              <input
                ref={titleRef}
                aria-label="card title"
                defaultValue={card.title ?? ''}
                onInput={(e) => setTitleValid((e.currentTarget.value ?? '').trim().length > 0)}
                className="mt-1 w-full rounded border border-slate-300 px-2 py-1 text-sm outline-none focus:border-blue-400"
              />
            </label>
            <label className="block">
              <span className="block text-xs font-medium text-slate-600">Body</span>
              <textarea
                ref={bodyRef}
                aria-label="card body"
                rows={6}
                defaultValue={card.body ?? ''}
                className="mt-1 w-full rounded border border-slate-300 px-2 py-1 text-sm outline-none focus:border-blue-400"
              />
            </label>
            <label className="block">
              <span className="block text-xs font-medium text-slate-600">Tags (comma separated)</span>
              <input
                ref={tagsRef}
                aria-label="card tags"
                defaultValue={(card.tags ?? []).join(', ')}
                className="mt-1 w-full rounded border border-slate-300 px-2 py-1 text-sm outline-none focus:border-blue-400"
              />
            </label>
            <div className="grid grid-cols-3 gap-3">
              <label className="block">
                <span className="block text-xs font-medium text-slate-600">Priority</span>
                <select
                  ref={priorityRef}
                  aria-label="card priority"
                  defaultValue={card.priority ?? ''}
                  className="mt-1 w-full rounded border border-slate-300 px-2 py-1 text-sm outline-none focus:border-blue-400"
                >
                  <option value="">—</option>
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                  <option value="critical">Critical</option>
                </select>
              </label>
              <label className="block">
                <span className="block text-xs font-medium text-slate-600">Due</span>
                <input
                  ref={dueRef}
                  aria-label="card due"
                  type="date"
                  defaultValue={card.due ?? ''}
                  className="mt-1 w-full rounded border border-slate-300 px-2 py-1 text-sm outline-none focus:border-blue-400"
                />
              </label>
              <label className="block">
                <span className="block text-xs font-medium text-slate-600">Assignee</span>
                <input
                  ref={assigneeRef}
                  aria-label="card assignee"
                  defaultValue={card.assignee ?? ''}
                  className="mt-1 w-full rounded border border-slate-300 px-2 py-1 text-sm outline-none focus:border-blue-400"
                />
              </label>
            </div>
            <div className="mt-2 flex justify-end gap-2">
              <button
                type="button"
                onClick={() => onOpenChange(false)}
                className="rounded px-3 py-1 text-sm text-slate-600 hover:bg-slate-100"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={!titleValid || mutation.isPending}
                className="rounded bg-blue-600 px-3 py-1 text-sm font-medium text-white disabled:cursor-not-allowed disabled:bg-slate-300"
              >
                {mutation.isPending ? 'Saving…' : 'Save'}
              </button>
            </div>
          </form>
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
Expected: no new errors.

- [ ] **Step 3: Commit**

```bash
git add web/renderer/default/src/components/CardDetailModal.tsx
git commit -m "feat(renderer): add CardDetailModal with Radix Dialog form

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: `CardDetailModal` tests

**Files:**
- Create: `web/renderer/default/src/components/CardDetailModal.test.tsx`

- [ ] **Step 1: Tests**

Create `web/renderer/default/src/components/CardDetailModal.test.tsx`:
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
import { CardDetailModal } from './CardDetailModal.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['board', 'welcome'], await client.getBoard('welcome'))
  return { client, qc }
}

const seed = {
  title: 'Hello',
  body: 'orig body',
  tags: ['a', 'b'],
  priority: 'high',
  due: '2026-05-01',
  assignee: 'alice',
}

describe('CardDetailModal', () => {
  it('renders form seeded from card prop when open', async () => {
    const { client, qc } = await setup()
    const { getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardDetailModal
          card={seed}
          colIdx={0}
          cardIdx={0}
          boardId="welcome"
          open={true}
          onOpenChange={() => {}}
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    expect((getByLabelText('card title') as HTMLInputElement).value).toBe('Hello')
    expect((getByLabelText('card body') as HTMLTextAreaElement).value).toBe('orig body')
    expect((getByLabelText('card tags') as HTMLInputElement).value).toBe('a, b')
    expect((getByLabelText('card priority') as HTMLSelectElement).value).toBe('high')
    expect((getByLabelText('card due') as HTMLInputElement).value).toBe('2026-05-01')
    expect((getByLabelText('card assignee') as HTMLInputElement).value).toBe('alice')
  })

  it('Save fires edit_card with form values and closes modal', async () => {
    const { client, qc } = await setup()
    const calls: boolean[] = []
    const { getByLabelText, getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardDetailModal
          card={seed}
          colIdx={0}
          cardIdx={0}
          boardId="welcome"
          open={true}
          onOpenChange={(next) => calls.push(next)}
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.input(getByLabelText('card title'), { target: { value: 'NEW TITLE' } })
    fireEvent.input(getByLabelText('card tags'), { target: { value: 'x, y, z' } })
    fireEvent.click(getByText('Save'))

    await waitFor(() => expect(calls).toContain(false))

    const b = qc.getQueryData<any>(['board', 'welcome'])
    const updated = b.columns[0].cards[0]
    expect(updated.title).toBe('NEW TITLE')
    expect(updated.tags).toEqual(['x', 'y', 'z'])
  })

  it('Cancel closes without firing mutation', async () => {
    const { client, qc } = await setup()
    const before = qc.getQueryData<any>(['board', 'welcome'])
    const beforeTitle = before.columns[0].cards[0].title
    const calls: boolean[] = []
    const { getByLabelText, getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardDetailModal
          card={seed}
          colIdx={0}
          cardIdx={0}
          boardId="welcome"
          open={true}
          onOpenChange={(next) => calls.push(next)}
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.input(getByLabelText('card title'), { target: { value: 'WONT SAVE' } })
    fireEvent.click(getByText('Cancel'))

    expect(calls).toContain(false)
    const after = qc.getQueryData<any>(['board', 'welcome'])
    expect(after.columns[0].cards[0].title).toBe(beforeTitle)
  })

  it('empty title disables Save button', async () => {
    const { client, qc } = await setup()
    const { getByLabelText, getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardDetailModal
          card={seed}
          colIdx={0}
          cardIdx={0}
          boardId="welcome"
          open={true}
          onOpenChange={() => {}}
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.input(getByLabelText('card title'), { target: { value: '   ' } })
    expect((getByText('Save') as HTMLButtonElement).disabled).toBe(true)
  })
})
```

- [ ] **Step 2: Run + commit**

```bash
cd web/renderer/default && bun test src/components/CardDetailModal.test.tsx && bun run typecheck
git add web/renderer/default/src/components/CardDetailModal.test.tsx
git commit -m "test(renderer): cover CardDetailModal seeding, save, cancel, validation

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```
Expected: 4 pass.

If a test hangs because Radix Dialog Portal mounting under happy-dom needs a tick, wrap initial assertions in `await waitFor(...)` — same workaround used in P4b.1a Radix tests. Don't add arbitrary `setTimeout`.

---

## Task 4: Wire `CardEditable` to open modal

**Files:**
- Modify: `web/renderer/default/src/components/CardEditable.tsx`

The view-mode markup needs:
- A title region wrapping `card.title` rendering with `onDoubleClick` (existing).
- A body region (everything else clickable inside the card) with `onClick={() => setModalOpen(true)}`.
- Action buttons (complete circle, delete ✕) that `stopPropagation` so they don't trigger the modal.
- `<CardDetailModal open={modalOpen} onOpenChange={setModalOpen} ... />` rendered after the card markup.

The existing component renders `<Card card={card} />` as a black box for the body — but `<Card>` includes the title, so we can't just slap onClick around the whole thing without losing the title-vs-body partition.

**Decision:** stop using `<Card>` from inside `<CardEditable>` view-mode and inline the markup. The presentational `Card.tsx` stays for the DragOverlay use in P4b.2 (which renders a full clone). This is a small duplication, but it gives us per-region click control without bending `Card.tsx` into accepting click props.

- [ ] **Step 1: Replace `CardEditable.tsx`**

Read current `web/renderer/default/src/components/CardEditable.tsx`. Replace whole file with:
```tsx
import { useState, useRef, useEffect } from 'react'
import type { Card as CardModel } from '@shared/types.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { stageDelete } from '../mutations/undoable.js'
import { CardDetailModal } from './CardDetailModal.js'

const PRIORITY_DOT: Record<string, string> = {
  critical: 'bg-red-600',
  high: 'bg-orange-500',
  medium: 'bg-yellow-400',
  low: 'bg-slate-300',
}

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
  const [modalOpen, setModalOpen] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useBoardMutation(boardId)
  const committedRef = useRef(false)

  useEffect(() => {
    if (mode === 'edit') {
      committedRef.current = false
      inputRef.current?.focus()
      inputRef.current?.select()
    }
  }, [mode])

  const commit = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    const title = (inputRef.current?.value ?? '').trim()
    if (title && title !== card.title) {
      mutation.mutate({
        type: 'edit_card',
        col_idx: colIdx,
        card_idx: cardIdx,
        title,
        body: card.body ?? '',
        tags: card.tags ?? [],
        priority: card.priority ?? '',
        due: card.due ?? '',
        assignee: card.assignee ?? '',
      })
    }
    Promise.resolve().then(() => setMode('view'))
  }

  const cancel = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    Promise.resolve().then(() => setMode('view'))
  }

  if (mode === 'edit') {
    return (
      <div className="rounded-md bg-white p-3 shadow-sm ring-2 ring-blue-400">
        <input
          ref={inputRef}
          aria-label="card title"
          defaultValue={card.title}
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === 'Enter') { e.preventDefault(); commit() }
            else if (e.key === 'Escape') { e.preventDefault(); cancel() }
          }}
          className="w-full bg-transparent text-sm font-semibold outline-none"
        />
      </div>
    )
  }

  return (
    <>
      <div className="group relative rounded-md bg-white p-3 shadow-sm ring-1 ring-slate-200">
        <div className="flex items-start gap-2">
          <button
            type="button"
            aria-label={card.completed ? 'mark incomplete' : 'mark complete'}
            onClick={(e) => {
              e.stopPropagation()
              mutation.mutate({
                type: 'complete_card',
                col_idx: colIdx,
                card_idx: cardIdx,
              })
            }}
            className={`mt-1 h-4 w-4 shrink-0 rounded-full border ${
              card.completed ? 'bg-slate-400 border-slate-400' : 'border-slate-300'
            }`}
          />
          <div className="flex-1">
            <div
              onDoubleClick={() => setMode('edit')}
              className="flex items-start gap-2"
            >
              {card.priority && (
                <span
                  aria-label={`priority ${card.priority}`}
                  className={`mt-1.5 h-2 w-2 shrink-0 rounded-full ${PRIORITY_DOT[card.priority] ?? 'bg-slate-300'}`}
                />
              )}
              <h3 className={`text-sm font-semibold ${card.completed ? 'line-through text-slate-400' : ''}`}>
                {card.title}
              </h3>
            </div>
            <button
              type="button"
              aria-label="open card details"
              onClick={() => setModalOpen(true)}
              className="mt-1 block w-full text-left"
            >
              {card.tags && card.tags.length > 0 ? (
                <ul className="flex flex-wrap gap-1">
                  {card.tags.map((t) => (
                    <li key={t} className="rounded bg-slate-100 px-1.5 py-0.5 text-xs text-slate-700">
                      {t}
                    </li>
                  ))}
                </ul>
              ) : (
                <span className="block h-2 w-full" />
              )}
            </button>
          </div>
          <button
            type="button"
            aria-label="delete card"
            onClick={(e) => {
              e.stopPropagation()
              stageDelete(
                mutation,
                { type: 'delete_card', col_idx: colIdx, card_idx: cardIdx },
                card.title,
              )
            }}
            className="opacity-0 group-hover:opacity-100 mt-1 text-xs text-slate-400 hover:text-red-500"
          >
            ✕
          </button>
        </div>
      </div>
      <CardDetailModal
        card={card}
        colIdx={colIdx}
        cardIdx={cardIdx}
        boardId={boardId}
        open={modalOpen}
        onOpenChange={setModalOpen}
      />
    </>
  )
}
```

Notes on the layout decision:
- The whole card body is no longer one big clickable region. Instead an explicit `aria-label="open card details"` button wraps the tags strip (and a small empty placeholder when there are no tags). This gives a discoverable, accessible click target without forcing the title region into the modal-open path.
- Title region keeps `onDoubleClick`. Single-click on the title area does nothing.
- Complete and delete buttons `stopPropagation` defensively (the new "open details" button is a sibling, not an ancestor — but stopping propagation is cheap insurance against future restructuring).

- [ ] **Step 2: Typecheck**

```bash
cd web/renderer/default && bun run typecheck
```
Expected: no new errors.

- [ ] **Step 3: Commit**

```bash
git add web/renderer/default/src/components/CardEditable.tsx
git commit -m "feat(renderer): wire CardEditable to open detail modal

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 5: Update `CardEditable` tests for new structure

**Files:**
- Modify: `web/renderer/default/src/components/CardEditable.test.tsx`

The existing tests assume `<Card>` rendered inside, so they `getByText('hello')` to find the title. The title is still rendered as text in an `<h3>` so that selector still works. The `mark complete` and `delete card` buttons keep the same aria labels.

- [ ] **Step 1: Read current `CardEditable.test.tsx`**

```bash
cat web/renderer/default/src/components/CardEditable.test.tsx
```

- [ ] **Step 2: Run as-is to confirm it still passes**

```bash
cd web/renderer/default && bun test src/components/CardEditable.test.tsx
```
Expected: existing 4 cases still pass. If any fail because of the new layout, fix the selector minimally (e.g., the old test may use `getByText` against the priority dot's parent — adapt to find the `<h3>` directly).

- [ ] **Step 3: Append new test**

Add this test inside the existing `describe('CardEditable', ...)` block in `web/renderer/default/src/components/CardEditable.test.tsx`:
```tsx
  it('clicking "open card details" reveals the modal', async () => {
    const { client, qc } = await setup()
    const { getByLabelText, getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardEditable card={{ title: 'hello' }} colIdx={0} cardIdx={0} boardId="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByLabelText('open card details'))
    await waitFor(() => expect(getByText('Edit card')).toBeDefined())
  })
```

If `waitFor` is not already imported in the file, add it to the import line:
```tsx
import { fireEvent, waitFor } from '@testing-library/react'
```

- [ ] **Step 4: Run + commit**

```bash
cd web/renderer/default && bun test src/components/CardEditable.test.tsx && bun run typecheck
git add web/renderer/default/src/components/CardEditable.test.tsx
git commit -m "test(renderer): cover CardEditable opening detail modal

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```
Expected: 5 pass total in this file.

---

## Task 6: Full suite + bundle check

**Files:** none.

- [ ] **Step 1: Full suite**

```bash
cd web/renderer/default && bun test && bun run typecheck
```
Expected: all green; total test count rises by ~5.

- [ ] **Step 2: Build**

```bash
make renderer
```
Expected: clean build.

- [ ] **Step 3: Measure bundle**

```bash
gzip -c web/renderer/default/dist/assets/*.js | wc -c
```
Record the gzipped byte count. Expected ~120 KB. Bundle gate is deferred to P4d — log the number, don't fail on size.

- [ ] **Step 4: Embed sanity**

```bash
go test ./internal/api/ -run TestShellRoute
```
Expected: 3 tests pass.

- [ ] **Step 5: No commit** — measurement only.

---

## Task 7: Manual browser smoke

Not a code change. Final gate.

- [ ] **Step 1: Rebuild + serve**

```bash
make shell && make renderer
LIVEBOARD_APP_SHELL=1 go run ./cmd/liveboard serve --dir ./demo --port 7070
```

- [ ] **Step 2: At <http://localhost:7070/app/> verify**

1. Click the tag-strip area below a card title → modal opens with all six fields seeded from the card.
2. Edit title to "manual-smoke", set priority to High, type some body text, set due to a date, add `qa, smoke` to tags → click Save → modal closes; card now shows new title and tags; refresh persists.
3. Open the modal again → click Cancel → modal closes; card unchanged.
4. Open modal → press Escape → closes; card unchanged.
5. Open modal → clear title to whitespace → Save button is disabled.
6. Double-click on the card title → inline edit appears (NOT the modal). Escape cancels.
7. Click the round complete button → toggles strikethrough; modal does NOT open.
8. Click the ✕ button → undo toast appears; modal does NOT open.
9. Drag the card via its grip → drag works as in P4b.2; modal does NOT open.
10. `?renderer=stub` still loads.

- [ ] **Step 3: Report**

If any step fails, capture which step + expected vs actual + console output. Fix before marking P4b.3 done.

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| `@radix-ui/react-dialog` dep | 1 |
| Modal with title, body, tags, priority, due, assignee | 2 |
| Save fires single `edit_card` mutation | 2, 3 |
| Cancel/Escape/overlay close without mutation | 2, 3 |
| Save disabled when title empty | 2, 3 |
| Optimistic + rollback through `useBoardMutation` | 2 (uses existing hook) |
| Modal stays open on error (errorToast already fires) | 2 (no special-casing) |
| Single click on body opens modal | 4 |
| Double-click on title still inline-edits | 4, 5 |
| Action buttons don't open modal | 4, 7 |
| Tags round-trip via comma-split | 2, 3 |
| Bundle measurement | 6 |
| Manual browser smoke | 7 |

## Notes for implementer

1. **Uncontrolled inputs are mandatory** — every text input/textarea in this plan uses `defaultValue` + ref reads. This is the established workaround for the `fireEvent.keyDown` bug in happy-dom + React 18.
2. **Reseed on reopen** — `key={String(open)}` on `Dialog.Content` is what actually reseeds the form, because uncontrolled inputs only honor `defaultValue` on mount. Don't drop the `key`.
3. **Title-validity state** is a separate `useState`, kept in sync via `onInput`. We need it as a controlled shadow for the disabled-Save behavior even though the input is uncontrolled.
4. **Stop using `<Card>` from `<CardEditable>` view mode** — Task 4 inlines the markup so we can partition click vs. doubleclick by region. `<Card>` itself is unchanged and still used by `<DragOverlay>` in `BoardDndContext`.
5. **No commit amending** — forward-only commits.
