# P4d.3 — Bundle Gate + Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close out P4 with three deliverables: a bundle-size gate (`scripts/check-bundle-size.sh` + `make bundle-check` + hookup from `make renderer`), code-splitting of `CommandPalette` / `CardDetailModal` / `BoardSettingsModal` via `React.lazy`, and a cleanup sweep (remove debug `console.error`, drop unused `Board` import, fix dark-mode stragglers).

**Architecture:** Three lazy boundaries let modals and the palette load on first use. `CommandPalette` splits into a tiny eager `CommandPaletteHost` (keeps the global Cmd+K listener + open state) and a lazy `CommandPalette` body. The bundle gate is a one-file shell script plus a Make target that runs after every renderer build.

**Tech Stack:** No new deps. Vite handles dynamic-import chunking. `React.lazy` + `<Suspense>` for the lazy boundaries.

**Spec:** `docs/superpowers/specs/2026-04-15-p4d3-bundle-gate-cleanup-design.md`

**Conventions:**
- Renderer code under `web/renderer/default/src/`.
- Script under `scripts/`.
- Commit prefixes: `refactor(renderer)`, `chore(build)`, `fix(shared)`, `style(renderer)`.
- Use bun, never npx.

---

## File structure

**New:**
- `scripts/check-bundle-size.sh`
- `web/renderer/default/src/components/CommandPaletteHost.tsx`

**Modified:**
- `Makefile` — `bundle-check` target + hookup from `renderer`.
- `web/renderer/default/src/components/CommandPalette.tsx` — drop internal `open` state + keydown listener; accept controlled props.
- `web/renderer/default/src/App.tsx` — swap `<CommandPalette />` for `<CommandPaletteHost />`.
- `web/renderer/default/src/components/CardEditable.tsx` — lazy-import `CardDetailModal` + `<Suspense>`.
- `web/renderer/default/src/components/BoardRow.tsx` — lazy-import `BoardSettingsModal` + `<Suspense>`.
- `web/renderer/default/src/mutations/useBoardCrud.ts` — drop debug `console.error`.
- `web/shared/src/protocol.ts` — drop unused `Board` import.
- Dark-mode straggler sweep: `web/renderer/default/src/components/EmptyState.tsx`, `AddCardButton.tsx`, `AddColumnButton.tsx`, `AddBoardButton.tsx`, `CardEditable.tsx` (placeholder text).

---

## Task 1: Remove debug `console.error` in `useBoardCrud`

**Files:**
- Modify: `web/renderer/default/src/mutations/useBoardCrud.ts`

- [ ] **Step 1: Edit**

Read `web/renderer/default/src/mutations/useBoardCrud.ts`. Every `onError` currently looks like:
```ts
    onError: (err) => {
      // eslint-disable-next-line no-console
      console.error('[useBoardCrud] mutation failed:', err)
      errorToast(code(err))
    },
```
Replace each occurrence with:
```ts
    onError: (err) => errorToast(code(err)),
```

- [ ] **Step 2: Verify**

```bash
cd /Users/htruong/code/htruong/liveboard && bun --cwd web/renderer/default test && bun --cwd web/renderer/default run typecheck
```
Expected: all green.

- [ ] **Step 3: Commit**

```bash
git add web/renderer/default/src/mutations/useBoardCrud.ts
git commit -m "refactor(renderer): drop debug console.error from useBoardCrud

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: Drop unused `Board` import from shared protocol

**Files:**
- Modify: `web/shared/src/protocol.ts`

- [ ] **Step 1: Edit imports**

Read `web/shared/src/protocol.ts`. The top-of-file import currently reads:
```ts
import type { MutationOp, Board, BoardSettings } from './types.js'
```
Drop `Board`:
```ts
import type { MutationOp, BoardSettings } from './types.js'
```

Also check the re-export line near the bottom:
```ts
export type { Board, BoardSettings, MutationOp } from './types.js'
```
Leave that alone — it's a public re-export, not the same as the top import.

- [ ] **Step 2: Typecheck**

```bash
cd web/renderer/default && bun run typecheck
```
Expected: clean — pre-existing TS6196 is gone.

- [ ] **Step 3: Commit**

```bash
git add web/shared/src/protocol.ts
git commit -m "fix(shared): drop unused Board import from protocol

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: Split `CommandPalette` into host + content

**Files:**
- Modify: `web/renderer/default/src/components/CommandPalette.tsx`
- Create: `web/renderer/default/src/components/CommandPaletteHost.tsx`
- Modify: `web/renderer/default/src/App.tsx`

Strip the keydown listener + `open` state out of `CommandPalette` and move them into a new `CommandPaletteHost`. The host decides when to mount the lazy palette body. App switches to `<CommandPaletteHost />`.

- [ ] **Step 1: Edit `CommandPalette.tsx` — accept controlled props**

Read the current file. Change the function signature + drop the internal `open` state and keydown effect. Replace these lines at the top of the component:
```tsx
export function CommandPalette(): JSX.Element {
  const [open, setOpen] = useState(false)
  const [page, setPage] = useState<Page>('list')
  ...

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
```
with:
```tsx
interface CommandPaletteProps {
  open: boolean
  onOpenChange: (next: boolean) => void
}

export function CommandPalette({ open, onOpenChange }: CommandPaletteProps): JSX.Element {
  const [page, setPage] = useState<Page>('list')
```

Then wherever the file called `setOpen(false)` (close actions) and `setOpen(true)`, replace with `onOpenChange(false)` / `onOpenChange(true)`. The existing `close` helper is `const close = () => setOpen(false)` — update it to `const close = () => onOpenChange(false)`. The `<Dialog.Root open={open} onOpenChange={setOpen}>` line becomes `<Dialog.Root open={open} onOpenChange={onOpenChange}>`. Also delete the now-unused `useState` import if it's no longer needed by the file — inspect and trim.

Re-run typecheck to confirm the prop wiring is consistent.

- [ ] **Step 2: Create `CommandPaletteHost.tsx`**

Create `web/renderer/default/src/components/CommandPaletteHost.tsx`:
```tsx
import { Suspense, lazy, useEffect, useRef, useState } from 'react'

const CommandPalette = lazy(() =>
  import('./CommandPalette.js').then((m) => ({ default: m.CommandPalette })),
)

export function CommandPaletteHost(): JSX.Element | null {
  const [open, setOpen] = useState(false)
  const hasBeenOpen = useRef(false)
  if (open) hasBeenOpen.current = true

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

  if (!hasBeenOpen.current) return null

  return (
    <Suspense fallback={null}>
      <CommandPalette open={open} onOpenChange={setOpen} />
    </Suspense>
  )
}
```

- [ ] **Step 3: Swap in `App.tsx`**

Read `web/renderer/default/src/App.tsx`. Replace the `CommandPalette` import and its JSX use:

Import line:
```tsx
import { CommandPalette } from './components/CommandPalette.js'
```
→
```tsx
import { CommandPaletteHost } from './components/CommandPaletteHost.js'
```

JSX:
```tsx
<CommandPalette />
```
→
```tsx
<CommandPaletteHost />
```

- [ ] **Step 4: Run tests + typecheck**

```bash
cd /Users/htruong/code/htruong/liveboard && bun --cwd web/renderer/default test && bun --cwd web/renderer/default run typecheck
```
Expected: all green. The existing `CommandPalette.test.tsx` renders `<CommandPalette />` directly with props — it needs the new prop shape. Either:
  (a) Update the test to supply `open={true} onOpenChange={() => {}}` if any assertion relies on an open palette, or
  (b) Switch tests to render `<CommandPaletteHost />` and drive open via the keydown event (the existing test already fires `fireEvent.keyDown(window, { key: 'k', metaKey: true })`, so render the Host instead).

Prefer (b) — one-line swap at each render site. Update the file as needed. The tests should continue to pass after the swap since the keydown behavior is preserved.

- [ ] **Step 5: Commit**

```bash
git add web/renderer/default/src/components/CommandPalette.tsx \
        web/renderer/default/src/components/CommandPaletteHost.tsx \
        web/renderer/default/src/App.tsx \
        web/renderer/default/src/components/CommandPalette.test.tsx
git commit -m "refactor(renderer): split CommandPalette into host + lazy content

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 4: Lazy-load `CardDetailModal`

**Files:**
- Modify: `web/renderer/default/src/components/CardEditable.tsx`

- [ ] **Step 1: Edit imports and JSX**

Read the current `CardEditable.tsx`. Replace the top import:
```tsx
import { CardDetailModal } from './CardDetailModal.js'
```
with:
```tsx
import { Suspense, lazy } from 'react'

const CardDetailModal = lazy(() =>
  import('./CardDetailModal.js').then((m) => ({ default: m.CardDetailModal })),
)
```
Also, there are already `useState`, `useRef`, `useEffect` imported from `'react'` — merge the new `Suspense, lazy` into that line rather than adding a second import: e.g.
```tsx
import { Suspense, lazy, useState, useRef, useEffect } from 'react'
```

Wrap the existing `<CardDetailModal ... />` element in `<Suspense fallback={null}>`:
```tsx
<Suspense fallback={null}>
  <CardDetailModal
    card={card}
    colIdx={colIdx}
    cardIdx={cardIdx}
    boardId={boardId}
    open={modalOpen}
    onOpenChange={onModalOpenChange}
  />
</Suspense>
```

- [ ] **Step 2: Run tests + typecheck**

```bash
cd web/renderer/default && bun test src/components/CardEditable.test.tsx src/components/CardDetailModal.test.tsx && bun run typecheck
```
Expected: all green. Dynamic imports in bun+happy-dom resolve synchronously in practice. If a CardEditable test flakes because the modal hasn't rendered on the first tick, wrap the assertion in `await waitFor(() => getByText('Edit card'))`.

- [ ] **Step 3: Commit**

```bash
git add web/renderer/default/src/components/CardEditable.tsx
git commit -m "refactor(renderer): lazy-load CardDetailModal

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 5: Lazy-load `BoardSettingsModal`

**Files:**
- Modify: `web/renderer/default/src/components/BoardRow.tsx`

- [ ] **Step 1: Edit imports and JSX**

Read `web/renderer/default/src/components/BoardRow.tsx`. Replace the import:
```tsx
import { BoardSettingsModal } from './BoardSettingsModal.js'
```
with:
```tsx
const BoardSettingsModal = lazy(() =>
  import('./BoardSettingsModal.js').then((m) => ({ default: m.BoardSettingsModal })),
)
```
and merge `Suspense, lazy` into the existing `react` import line (adjacent to `useState`, `useRef`, `useEffect`).

Wrap the existing `<BoardSettingsModal ... />` in `<Suspense fallback={null}>`.

- [ ] **Step 2: Run tests + typecheck**

```bash
cd web/renderer/default && bun test src/components/BoardRow.test.tsx src/components/BoardSettingsModal.test.tsx && bun run typecheck
```
Expected: green.

- [ ] **Step 3: Commit**

```bash
git add web/renderer/default/src/components/BoardRow.tsx
git commit -m "refactor(renderer): lazy-load BoardSettingsModal

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 6: Dark-mode straggler sweep

**Files:**
- Modify: `web/renderer/default/src/components/EmptyState.tsx`
- Modify: `web/renderer/default/src/components/AddCardButton.tsx`
- Modify: `web/renderer/default/src/components/AddColumnButton.tsx`
- Modify: `web/renderer/default/src/components/AddBoardButton.tsx`
- Modify: `web/renderer/default/src/components/CardEditable.tsx`

Append `dark:` variants. The specific classes vary slightly per file, but the pattern is the same: wherever a text-slate-<n> or bg-slate-<n> appears and doesn't already have a `dark:` sibling, add one with the inverse slate step.

- [ ] **Step 1: `EmptyState.tsx`**

Read the file. Any `text-slate-500` / `text-slate-600` on the heading → append `dark:text-slate-300`. Any `text-slate-400` detail → append `dark:text-slate-500`. Leave layout alone.

- [ ] **Step 2: `AddCardButton.tsx` / `AddColumnButton.tsx` / `AddBoardButton.tsx`**

Each has a ghost/idle `<button>` with classes like `text-slate-500 hover:bg-slate-200`. Append `dark:text-slate-400 dark:hover:bg-slate-800` to each ghost button. Input `<input>` elements should already pick up dark via browser defaults; only touch if visual check in Task 8 surfaces an issue.

- [ ] **Step 3: `CardEditable.tsx` — placeholder**

The "Click to edit details" span:
```tsx
<span className="text-xs text-slate-300">Click to edit details</span>
```
→
```tsx
<span className="text-xs text-slate-300 dark:text-slate-600">Click to edit details</span>
```

(The dark:text-slate-600 is intentionally dim against slate-800 card bg; it's a hint, not a CTA.)

- [ ] **Step 4: Verify**

```bash
cd web/renderer/default && bun test && bun run typecheck
```
Expected: green.

- [ ] **Step 5: Commit**

```bash
git add web/renderer/default/src/components/EmptyState.tsx \
        web/renderer/default/src/components/AddCardButton.tsx \
        web/renderer/default/src/components/AddColumnButton.tsx \
        web/renderer/default/src/components/AddBoardButton.tsx \
        web/renderer/default/src/components/CardEditable.tsx
git commit -m "style(renderer): dark-mode classes on EmptyState + ghost buttons

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 7: Bundle gate script + Makefile hookup

**Files:**
- Create: `scripts/check-bundle-size.sh`
- Modify: `Makefile`

- [ ] **Step 1: Measure current bundle**

```bash
cd /Users/htruong/code/htruong/liveboard && make renderer && gzip -c web/renderer/default/dist/assets/*.js | wc -c
```
Record the number. Example output: `129000` bytes (~126 KB gz).

- [ ] **Step 2: Pick `MAX_BYTES`**

Take the measured value, add 10 KB (10240), round up to the next 5 KB (5120) boundary. Example: measured 129000 → +10240 = 139240 → round to 143360 (140 KB).

- [ ] **Step 3: Create the script**

Create `scripts/check-bundle-size.sh`. Insert the rounded `MAX_BYTES` default from Step 2 (example uses 143360 / 140 KB — adjust to your measurement):
```sh
#!/usr/bin/env bash
# Renderer bundle size gate.
# Measured <YYYY-MM-DD>: <measured> bytes gzipped.
# Budget = measured + ~10 KB headroom, rounded to next 5 KB.
set -euo pipefail
MAX_BYTES="${MAX_BYTES:-143360}"  # 140 KB
if ! compgen -G "web/renderer/default/dist/assets/*.js" > /dev/null; then
  echo "ERROR: no built JS in web/renderer/default/dist/assets/. Run 'make renderer' first."
  exit 1
fi
TOTAL=$(gzip -c web/renderer/default/dist/assets/*.js | wc -c | tr -d ' ')
echo "renderer bundle (gz): ${TOTAL} bytes (max ${MAX_BYTES})"
if [ "$TOTAL" -gt "$MAX_BYTES" ]; then
  echo "ERROR: bundle exceeds budget"
  exit 1
fi
```
Make executable: `chmod +x scripts/check-bundle-size.sh`.

- [ ] **Step 4: Add Makefile target**

Read current `Makefile`. In the `.PHONY` line near the top, append `bundle-check`. Add this target (after the `renderer` target block):
```make
.PHONY: bundle-check
bundle-check:
	bash scripts/check-bundle-size.sh
```

Extend the existing `renderer` target so it runs `bundle-check` last. Replace:
```make
renderer:
	cd web/renderer/default && bun install --frozen-lockfile
	cd web/renderer/default && bunx --bun vite build
```
with:
```make
renderer:
	cd web/renderer/default && bun install --frozen-lockfile
	cd web/renderer/default && bunx --bun vite build
	@$(MAKE) bundle-check
```

- [ ] **Step 5: Verify**

```bash
cd /Users/htruong/code/htruong/liveboard && make renderer
```
Expected: build succeeds, `bundle-check` runs and reports `renderer bundle (gz): <TOTAL> bytes (max <MAX>)`, exit 0.

Then check the gate's failure path manually:
```bash
MAX_BYTES=1 bash scripts/check-bundle-size.sh
```
Expected: prints ERROR, exits non-zero.

- [ ] **Step 6: Commit**

```bash
chmod +x scripts/check-bundle-size.sh
git add scripts/check-bundle-size.sh Makefile
git commit -m "chore(build): add renderer bundle size gate

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 8: Manual browser smoke

Not a code change.

- [ ] **Step 1: Build + serve**

```bash
make adapter-test
```

- [ ] **Step 2: At <http://localhost:7070/app/> verify**

1. App loads; sidebar + board render.
2. Press Cmd+K → palette opens on first press (lazy-loaded on demand); close + reopen should feel identical.
3. Click a card's tag strip → CardDetailModal opens; save path works.
4. Hover a board row kebab → Settings → BoardSettingsModal opens; save path works.
5. Toggle Dark mode via 🎨 picker → verify:
   - Sidebar workspace name, board row names readable.
   - Empty state text readable.
   - "+ Add card", "+ Add column", "+ New board" ghost buttons readable.
   - "Click to edit details" placeholder dim but visible.
   - Modal content readable.
6. Toggle back to Light.
7. `?renderer=stub` still loads.

- [ ] **Step 3: Report.** Capture any contrast or lazy-load regressions with the step + expected vs actual.

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| `check-bundle-size.sh` with default `MAX_BYTES` | 7 |
| `make bundle-check` target | 7 |
| `make renderer` invokes `bundle-check` | 7 |
| Lazy `CommandPalette` via Host | 3 |
| Lazy `CardDetailModal` | 4 |
| Lazy `BoardSettingsModal` | 5 |
| Drop `console.error` in `useBoardCrud` | 1 |
| Drop unused `Board` import in `protocol.ts` | 2 |
| Dark-mode stragglers on EmptyState + ghost buttons + placeholder | 6 |
| Budget set from measurement | 7 |
| Manual smoke | 8 |

## Notes for implementer

1. **Task 3's test update** is the trickiest part: existing `CommandPalette.test.tsx` mounts `<CommandPalette />` directly. Swap each render to mount `<CommandPaletteHost />`; the keydown events still open it via the eager listener. Assertions for the list / items / "Create board" etc. remain unchanged — they run against the lazy-mounted content, which resolves synchronously under bun+happy-dom.
2. **`MAX_BYTES` measurement** is the only step that depends on the current state. Do the measurement AFTER Tasks 1–6 land (so splitting effects are included), but BEFORE committing `check-bundle-size.sh`. Task 7 explicitly sequences this.
3. **Lazy fallback** is intentionally `null` — modals/palettes have no natural loading state and flash would be worse than blank for <100 ms.
4. **If Vite emits multiple JS chunks**, the `gzip -c dist/assets/*.js` glob sums them. Budget captures total user-facing JS, not first-load. Acceptable per spec.
5. **`ts-prune`** is advisory per the spec — run it ad hoc if you want, don't add it to CI.
6. **No commit amending** — forward-only commits.
