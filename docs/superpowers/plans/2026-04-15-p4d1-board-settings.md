# P4d.1 — Per-Board Settings UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Surface two per-board settings (`show_checkbox`, `card_display_mode`) in `/app/`'s default renderer via a Radix Dialog modal opened from `BoardRow`'s kebab. Wire `CardEditable` to honor those settings. Fix the `LocalAdapter.getSettings` bug that ignores stored `board.settings`.

**Architecture:** A `useBoardSettings(boardId)` query hook returns resolved settings (defaults during loading). A `useUpdateSettings(boardId)` mutation wraps `client.putBoardSettings`. `<BoardSettingsModal>` is a Radix Dialog form with a checkbox + select that fires the mutation. `BoardRow`'s kebab gains a Settings item that opens the modal. `CardEditable` reads the hook to (a) hide the complete-circle when `show_checkbox === false` and (b) apply tighter spacing when `card_display_mode === 'compact'`.

**Tech Stack:** No new deps. Reuses `@radix-ui/react-dialog` (P4b.3), TanStack Query (P4a), existing `errorToast`.

**Spec:** `docs/superpowers/specs/2026-04-15-p4d1-board-settings-design.md`

**Conventions:**
- Renderer code under `web/renderer/default/src/`.
- Adapter fix in `web/shared/src/adapters/local.ts`.
- Tests colocated.
- Commit prefixes: `fix(shared)`, `feat(renderer)`, `test(...)`.
- Pre-existing `TS6196` in `web/shared/src/protocol.ts` is NOT a blocker.
- Use bun, never npx.

---

## File structure

**New:**
- `web/shared/src/adapters/local.settings.test.ts` — round-trip get-after-put test.
- `web/renderer/default/src/queries/useBoardSettings.ts` — `useBoardSettings` + `useUpdateSettings` hooks.
- `web/renderer/default/src/components/BoardSettingsModal.tsx`
- `web/renderer/default/src/components/BoardSettingsModal.test.tsx`

**Modified:**
- `web/shared/src/adapters/local.ts` — `getSettings` reads `board.settings` and merges with defaults.
- `web/renderer/default/src/components/BoardRow.tsx` — add Settings menu item + modal mount + state.
- `web/renderer/default/src/components/CardEditable.tsx` — consult settings; conditional complete-circle + compact class.

---

## Task 1: Fix `LocalAdapter.getSettings`

**Files:**
- Modify: `web/shared/src/adapters/local.ts`
- Create: `web/shared/src/adapters/local.settings.test.ts`

TDD: failing round-trip test first, then fix the read.

- [ ] **Step 1: Failing test**

Create `web/shared/src/adapters/local.settings.test.ts`:
```ts
import { describe, expect, it } from 'bun:test'
import { LocalAdapter } from './local.js'
import { MemoryStorage } from './local-storage-driver.js'

describe('LocalAdapter settings round-trip', () => {
  it('getSettings reflects putBoardSettings', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.putBoardSettings('welcome', { show_checkbox: false, card_display_mode: 'compact' })
    const s = await a.getSettings('welcome')
    expect(s.show_checkbox).toBe(false)
    expect(s.card_display_mode).toBe('compact')
    // Other defaults preserved.
    expect(s.expand_columns).toBe(false)
    expect(s.view_mode).toBe('board')
  })

  it('getSettings returns defaults for a fresh board', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const s = await a.getSettings('welcome')
    expect(s.show_checkbox).toBe(true)
    expect(s.card_display_mode).toBe('normal')
  })
})
```

- [ ] **Step 2: Run, expect first test fail**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src/adapters/local.settings.test.ts
```
Expected: "getSettings reflects putBoardSettings" fails (returns hardcoded defaults). "getSettings returns defaults for a fresh board" passes coincidentally.

- [ ] **Step 3: Fix `getSettings`**

Open `web/shared/src/adapters/local.ts`. Replace the existing `getSettings` method:
```ts
  async getSettings(boardId: string): Promise<ResolvedSettings> {
    this.loadBoard(boardId) // 404 check
    return {
      show_checkbox: true,
      card_position: 'bottom',
      expand_columns: false,
      view_mode: 'board',
      card_display_mode: 'normal',
      week_start: 'monday',
    }
  }
```
with:
```ts
  async getSettings(boardId: string): Promise<ResolvedSettings> {
    const board = this.loadBoard(boardId)
    return {
      show_checkbox: board.settings?.show_checkbox ?? true,
      card_position: board.settings?.card_position ?? 'bottom',
      expand_columns: board.settings?.expand_columns ?? false,
      view_mode: board.settings?.view_mode ?? 'board',
      card_display_mode: board.settings?.card_display_mode ?? 'normal',
      week_start: board.settings?.week_start ?? 'monday',
    }
  }
```

- [ ] **Step 4: Run, expect 2 pass**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src/adapters/local.settings.test.ts && cd web/renderer/default && bun run typecheck
```
Expected: 2 pass; only pre-existing TS6196 in typecheck.

If types complain because `board.settings.X` shape doesn't match (e.g. `BoardSettings` types fields as `string | undefined` and `ResolvedSettings` expects literal types), cast at the boundary: `(board.settings?.card_display_mode as ResolvedSettings['card_display_mode']) ?? 'normal'`. Same for any other narrowing failure.

- [ ] **Step 5: Commit**

```bash
git add web/shared/src/adapters/local.ts web/shared/src/adapters/local.settings.test.ts
git commit -m "fix(shared): LocalAdapter.getSettings reads stored board.settings

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: `useBoardSettings` + `useUpdateSettings` hooks

**Files:**
- Create: `web/renderer/default/src/queries/useBoardSettings.ts`

(There's no existing `queries/` directory — verify/create as needed. If the project convention places hooks under `mutations/`, follow that; the path adapts but the file is otherwise self-contained.)

- [ ] **Step 1: Implement**

Create `web/renderer/default/src/queries/useBoardSettings.ts`:
```ts
import { useMutation, useQuery, useQueryClient, type UseMutationResult } from '@tanstack/react-query'
import type { ResolvedSettings, BoardSettings } from '@shared/types.js'
import { ProtocolError } from '@shared/protocol.js'
import { useClient } from '../queries.js'
import { errorToast } from '../toast.js'

export const SETTINGS_DEFAULTS: ResolvedSettings = {
  show_checkbox: true,
  card_position: 'bottom',
  expand_columns: false,
  view_mode: 'board',
  card_display_mode: 'normal',
  week_start: 'monday',
}

export function useBoardSettings(boardId: string | null): ResolvedSettings {
  const client = useClient()
  const q = useQuery({
    queryKey: ['settings', boardId],
    queryFn: () => client.getSettings(boardId!),
    enabled: !!boardId,
  })
  return q.data ?? SETTINGS_DEFAULTS
}

function code(err: unknown): string {
  return err instanceof ProtocolError ? err.code : 'INTERNAL'
}

export function useUpdateSettings(
  boardId: string,
): UseMutationResult<void, Error, Partial<BoardSettings>> {
  const client = useClient()
  const qc = useQueryClient()
  return useMutation<void, Error, Partial<BoardSettings>>({
    mutationFn: (patch) => client.putBoardSettings(boardId, patch),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['settings', boardId] })
    },
    onError: (err) => errorToast(code(err)),
  })
}
```

Verify the import paths:
- `ResolvedSettings` is exported from `@shared/adapter.js` (not `types.js`) per earlier inspection — adjust the import to match. Use the actual location: `web/shared/src/adapter.ts`.
- `BoardSettings` is exported from `@shared/types.js`.

If those exports differ in your tree, fix the imports rather than re-exporting.

- [ ] **Step 2: Typecheck**

```bash
cd web/renderer/default && bun run typecheck
```
Expected: only pre-existing TS6196.

- [ ] **Step 3: Commit**

```bash
git add web/renderer/default/src/queries/useBoardSettings.ts
git commit -m "feat(renderer): add useBoardSettings + useUpdateSettings hooks

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: `<BoardSettingsModal>` component

**Files:**
- Create: `web/renderer/default/src/components/BoardSettingsModal.tsx`

- [ ] **Step 1: Implement**

Create `web/renderer/default/src/components/BoardSettingsModal.tsx`:
```tsx
import { useRef, type FormEvent } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { useBoardSettings, useUpdateSettings } from '../queries/useBoardSettings.js'

export function BoardSettingsModal({
  boardId,
  boardName,
  open,
  onOpenChange,
}: {
  boardId: string
  boardName: string
  open: boolean
  onOpenChange: (next: boolean) => void
}): JSX.Element {
  const settings = useBoardSettings(boardId)
  const mutation = useUpdateSettings(boardId)
  const checkboxRef = useRef<HTMLInputElement>(null)
  const modeRef = useRef<HTMLSelectElement>(null)

  const submit = (e: FormEvent): void => {
    e.preventDefault()
    mutation.mutate(
      {
        show_checkbox: checkboxRef.current?.checked ?? true,
        card_display_mode: (modeRef.current?.value as 'normal' | 'compact') ?? 'normal',
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
          key={String(open)}
          aria-label="Board settings"
          className="fixed left-1/2 top-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-lg bg-white p-6 shadow-xl"
        >
          <Dialog.Title className="text-lg font-semibold text-slate-800">
            Settings: {boardName}
          </Dialog.Title>
          <form onSubmit={submit} className="mt-4 space-y-4">
            <label className="flex items-center gap-2 text-sm text-slate-700">
              <input
                ref={checkboxRef}
                aria-label="show complete checkbox"
                type="checkbox"
                defaultChecked={settings.show_checkbox}
                className="h-4 w-4"
              />
              Show complete checkbox on cards
            </label>
            <label className="block text-sm text-slate-700">
              <span className="block text-xs font-medium text-slate-600">Card display</span>
              <select
                ref={modeRef}
                aria-label="card display mode"
                defaultValue={settings.card_display_mode}
                className="mt-1 w-full rounded border border-slate-300 px-2 py-1 text-sm outline-none focus:border-blue-400"
              >
                <option value="normal">Normal</option>
                <option value="compact">Compact</option>
              </select>
            </label>
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
                disabled={mutation.isPending}
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

- [ ] **Step 3: Commit**

```bash
git add web/renderer/default/src/components/BoardSettingsModal.tsx
git commit -m "feat(renderer): add BoardSettingsModal with show_checkbox + card_display_mode

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 4: `<BoardSettingsModal>` tests

**Files:**
- Create: `web/renderer/default/src/components/BoardSettingsModal.test.tsx`

- [ ] **Step 1: Tests**

Create `web/renderer/default/src/components/BoardSettingsModal.test.tsx`:
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
import { BoardSettingsModal } from './BoardSettingsModal.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  // Seed cache with current settings.
  qc.setQueryData(['settings', 'welcome'], await client.getSettings('welcome'))
  return { client, qc }
}

describe('BoardSettingsModal', () => {
  it('renders form seeded from settings cache', async () => {
    const { client, qc } = await setup()
    const { findByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardSettingsModal
          boardId="welcome"
          boardName="Welcome"
          open={true}
          onOpenChange={() => {}}
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    const checkbox = (await findByLabelText('show complete checkbox')) as HTMLInputElement
    const select = (await findByLabelText('card display mode')) as HTMLSelectElement
    expect(checkbox.checked).toBe(true)
    expect(select.value).toBe('normal')
  })

  it('Save persists the toggled values', async () => {
    const { client, qc } = await setup()
    const calls: boolean[] = []
    const { findByLabelText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardSettingsModal
          boardId="welcome"
          boardName="Welcome"
          open={true}
          onOpenChange={(v) => calls.push(v)}
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    const checkbox = (await findByLabelText('show complete checkbox')) as HTMLInputElement
    const select = (await findByLabelText('card display mode')) as HTMLSelectElement
    fireEvent.click(checkbox) // → unchecked
    fireEvent.change(select, { target: { value: 'compact' } })
    fireEvent.click(await findByText('Save'))

    await waitFor(() => expect(calls).toContain(false))

    const after = await client.getSettings('welcome')
    expect(after.show_checkbox).toBe(false)
    expect(after.card_display_mode).toBe('compact')
  })

  it('Cancel closes without writing', async () => {
    const { client, qc } = await setup()
    const calls: boolean[] = []
    const before = await client.getSettings('welcome')
    const { findByLabelText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardSettingsModal
          boardId="welcome"
          boardName="Welcome"
          open={true}
          onOpenChange={(v) => calls.push(v)}
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    const checkbox = (await findByLabelText('show complete checkbox')) as HTMLInputElement
    fireEvent.click(checkbox)
    fireEvent.click(await findByText('Cancel'))
    expect(calls).toContain(false)
    const after = await client.getSettings('welcome')
    expect(after.show_checkbox).toBe(before.show_checkbox)
  })
})
```

- [ ] **Step 2: Run + commit**

```bash
cd web/renderer/default && bun test src/components/BoardSettingsModal.test.tsx && bun run typecheck
git add web/renderer/default/src/components/BoardSettingsModal.test.tsx
git commit -m "test(renderer): cover BoardSettingsModal seed, save, cancel

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```
Expected: 3 pass.

If a test hangs because Radix Portal mounts on a tick, wrap initial assertions in `await waitFor(() => findByLabelText(...))`. If `findByLabelText` already polls, leave as-is. Don't commit on red.

---

## Task 5: Wire `<BoardRow>` to open settings

**Files:**
- Modify: `web/renderer/default/src/components/BoardRow.tsx`

- [ ] **Step 1: Read current `BoardRow.tsx`**

```bash
cat web/renderer/default/src/components/BoardRow.tsx
```

- [ ] **Step 2: Edit**

Apply these changes:

(a) Add an import for the modal at the top:
```tsx
import { BoardSettingsModal } from './BoardSettingsModal.js'
```

(b) Add a state hook near the existing `useState` for `mode`:
```tsx
const [settingsOpen, setSettingsOpen] = useState(false)
```

(c) In the `<DropdownMenu.Content>`, add a Settings item between the Rename item and the Separator:
```tsx
            <DropdownMenu.Item
              onSelect={() => setSettingsOpen(true)}
              className="cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-slate-100"
            >
              Settings
            </DropdownMenu.Item>
```

(d) Wrap the existing `<li>...</li>` return in a fragment so we can mount the modal sibling, and append the modal:
```tsx
return (
  <>
    <li className="group flex items-center gap-1">
      ...existing children...
    </li>
    <BoardSettingsModal
      boardId={board.id}
      boardName={board.name}
      open={settingsOpen}
      onOpenChange={setSettingsOpen}
    />
  </>
)
```

(The modal portals out of the `<li>`, so DOM validity is fine.)

- [ ] **Step 3: Verify existing BoardRow tests still pass**

```bash
cd web/renderer/default && bun test src/components/BoardRow.test.tsx && bun run typecheck
```
Expected: 2 pass. The existing tests don't open the menu, so adding the Settings item doesn't disturb them.

- [ ] **Step 4: Commit**

```bash
git add web/renderer/default/src/components/BoardRow.tsx
git commit -m "feat(renderer): add Settings entry to BoardRow kebab menu

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 6: `<CardEditable>` honors settings

**Files:**
- Modify: `web/renderer/default/src/components/CardEditable.tsx`

- [ ] **Step 1: Edit**

In `web/renderer/default/src/components/CardEditable.tsx`:

(a) Add the import at the top:
```tsx
import { useBoardSettings } from '../queries/useBoardSettings.js'
```

(b) Inside the function body, after the existing `const mutation = useBoardMutation(boardId)`:
```tsx
  const settings = useBoardSettings(boardId)
  const showCheckbox = settings.show_checkbox
  const compact = settings.card_display_mode === 'compact'
```

(c) In the view-mode return, replace:
```tsx
      <div className="group relative rounded-md bg-white p-3 shadow-sm ring-1 ring-slate-200">
```
with:
```tsx
      <div className={`group relative rounded-md bg-white shadow-sm ring-1 ring-slate-200 ${compact ? 'p-2 text-xs' : 'p-3 text-sm'}`}>
```

(d) Wrap the complete-circle button in a conditional. Replace:
```tsx
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
```
with:
```tsx
          {showCheckbox && (
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
          )}
```

- [ ] **Step 2: Verify CardEditable tests still pass**

```bash
cd web/renderer/default && bun test src/components/CardEditable.test.tsx && bun run typecheck
```

The existing tests assert `getByLabelText('mark complete')`. Since the default settings (`show_checkbox: true`) keep the button rendered, those assertions still hold. If a test fails because the test environment doesn't have a valid welcome cache and `useBoardSettings` returns DEFAULTS (which still has `show_checkbox: true`), it still passes.

- [ ] **Step 3: Commit**

```bash
git add web/renderer/default/src/components/CardEditable.tsx
git commit -m "feat(renderer): CardEditable honors show_checkbox and card_display_mode

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 7: Build + bundle measurement

**Files:** none.

- [ ] **Step 1: Build**

```bash
cd /Users/htruong/code/htruong/liveboard && make renderer
```

- [ ] **Step 2: Measure**

```bash
gzip -c web/renderer/default/dist/assets/*.js | wc -c
```
Expected ~127 KB (no new deps; +~1 KB for hooks + modal).

- [ ] **Step 3: Verify Go embed**

```bash
go test ./internal/api/ -run TestShellRoute
```
Expected: 3 tests pass.

- [ ] **Step 4: No commit.**

---

## Task 8: Manual browser smoke

Not a code change.

- [ ] **Step 1: Build + serve**

```bash
make adapter-test
```

- [ ] **Step 2: At <http://localhost:7070/app/> verify**

1. Hover a board row → kebab `⋮` appears.
2. Open kebab → menu shows Rename / Settings / Delete.
3. Click Settings → modal opens "Settings: <board name>". Show-checkbox checked, Card display = Normal.
4. Uncheck "Show complete checkbox on cards" → click Save → modal closes; complete-circle disappears from every card on that board.
5. Open Settings again → checkbox is unchecked (persisted).
6. Change Card display to "Compact" → Save → cards visibly tighter (smaller padding, smaller text).
7. Open Settings → set both back to defaults → Save → cards return to normal.
8. Refresh the page → settings persist (localStorage).
9. Cancel after toggling something → no change applied.
10. `?renderer=stub` still loads.

- [ ] **Step 3: Report.** Capture failures with step + expected vs actual.

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| `LocalAdapter.getSettings` reads stored `board.settings` | 1 |
| Round-trip get-after-put test | 1 |
| `useBoardSettings` hook (returns DEFAULTS during loading) | 2 |
| `useUpdateSettings` hook + invalidate | 2 |
| `<BoardSettingsModal>` Radix Dialog with two controls | 3 |
| Modal tests (seed, save, cancel) | 4 |
| `BoardRow` kebab Settings entry + modal mount | 5 |
| `CardEditable` hides complete-circle when `show_checkbox` false | 6 |
| `CardEditable` applies compact spacing when mode = compact | 6 |
| Bundle measurement | 7 |
| Manual smoke | 8 |

## Notes for implementer

1. **`ResolvedSettings` import path** — likely `@shared/adapter.js`, not `@shared/types.js`. Verify before commit; fix the import in Task 2's hook file rather than re-exporting.
2. **Type narrowing on stored values** — `BoardSettings` types fields loosely. Cast at the boundary in the adapter fix if TS complains: `(board.settings?.card_display_mode as ResolvedSettings['card_display_mode']) ?? 'normal'`.
3. **Modal reseeds on reopen** — `key={String(open)}` on `Dialog.Content` (same trick as `CardDetailModal`). Don't drop it.
4. **Hook path** — placed under `web/renderer/default/src/queries/`. If repo convention prefers `mutations/`, follow that; the file is otherwise self-contained.
5. **No CardEditable test changes needed** — defaults render the same UI. Compact / hidden-checkbox is verified via manual smoke.
6. **No commit amending** — forward-only commits.
