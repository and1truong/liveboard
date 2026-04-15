# P4d.3 — Bundle Gate + Cleanup — Design

## Goal

Close out P4 with two concerns:
1. **Bundle gate**: measure the current `/app/` renderer gzipped JS, introduce a post-build check (`make bundle-check`) that fails when size exceeds a fixed budget, and wire it into `make renderer` so the gate runs automatically.
2. **Cleanup**: code-split the three heaviest user-triggered components (`CommandPalette`, `CardDetailModal`, `BoardSettingsModal`) via `React.lazy` so they load on demand, remove debug leftovers, drop unused exports, resolve the long-standing `TS6196`, and sweep dark-mode stragglers.

**Shippable value:** shrinks the initial-load JS, prevents future regressions, clears lingering warnings. After P4d.3, P4 is closed.

## Scope

**In:**
- `scripts/check-bundle-size.sh` + `make bundle-check` + hookup from `make renderer`.
- Code-split `CommandPalette`, `CardDetailModal`, `BoardSettingsModal` via `React.lazy` + `<Suspense>`.
- `CommandPalette` split into `CommandPaletteHost` (eager, listener + state) + `CommandPalette` content (lazy).
- Drop dev-only `console.error` from `useBoardCrud`.
- Drop unused `Board` import in `web/shared/src/protocol.ts` (removes pre-existing TS6196).
- Dark-mode straggler sweep: `EmptyState`, `AddCardButton`, `AddColumnButton`, `AddBoardButton`, `CardEditable` body placeholder.
- Budget measurement → set `MAX_BYTES` in the script to `measured + ~10 KB` headroom, rounded to a 5 KB step.

**Out:**
- Bundle splitting beyond these three components (e.g., lazy-load Radix Dialog itself): diminishing returns.
- CI integration beyond local make gate (CI already runs `make renderer`).
- Further dep replacement (cmdk, Radix) — deferred unless the budget needs to shrink.
- Workspace-level settings UI.
- Per-board theme override.

## Architecture

### Bundle gate

`scripts/check-bundle-size.sh`:
```sh
#!/usr/bin/env bash
set -euo pipefail
MAX_BYTES="${MAX_BYTES:-<set-after-measurement>}"
TOTAL=$(gzip -c web/renderer/default/dist/assets/*.js | wc -c | tr -d ' ')
echo "renderer bundle (gz): ${TOTAL} bytes (max ${MAX_BYTES})"
if [ "$TOTAL" -gt "$MAX_BYTES" ]; then
  echo "ERROR: bundle exceeds budget"
  exit 1
fi
```

`Makefile` changes:
- Add `.PHONY: bundle-check` and a recipe that runs `bash scripts/check-bundle-size.sh`.
- The existing `renderer` target appends `@$(MAKE) bundle-check` as its last step.

Raising the budget is a deliberate one-line script change.

### Code-splitting

Each lazy-target becomes a dynamic import:
```ts
const CardDetailModal = React.lazy(() => import('./CardDetailModal.js').then(m => ({ default: m.CardDetailModal })))
```
Rendered inside `<Suspense fallback={null}>`.

`CommandPalette` refactor:
- `CommandPaletteHost` (new file): owns `open` + `setOpen` state and the global `keydown` listener. Renders the lazy `<CommandPalette open onOpenChange>` only when `hasBeenOpen` is true (tracked via `useRef` so we never re-mount after close).
- `CommandPalette` (existing file): unchanged signature; receives `open` + `onOpenChange` as props. Dialog Content mounts only when the host decides.

App.tsx wires `<CommandPaletteHost />` in place of `<CommandPalette />`.

### Cleanup

1. **`useBoardCrud.ts`** — remove the debug `console.error` block added during P4c.1 manual-smoke debugging. Restore each hook's `onError` to `(err) => errorToast(code(err))`.
2. **`web/shared/src/protocol.ts`** — drop the unused `Board` from the import. Protocol still references `BoardSettings` and `MutationOp`; only `Board` is extraneous.
3. **Dark-mode stragglers**:
   - `EmptyState` heading: add `dark:text-slate-300` (or whatever reads against slate-950).
   - `AddCardButton`, `AddColumnButton`, `AddBoardButton` ghost/hover: add `dark:text-slate-400 dark:hover:bg-slate-800`.
   - `CardEditable`'s "Click to edit details" placeholder: add `dark:text-slate-600` (softer than light mode's slate-300 against slate-800 card bg).
4. **`ts-prune` audit (advisory only)**: run `bunx ts-prune` from `web/renderer/default` as part of Task 7 (Cleanup sweep). Delete confirmed-unused exports; do not break public API surface.

## Component contracts

### `CommandPaletteHost` (new)

```ts
// web/renderer/default/src/components/CommandPaletteHost.tsx
export function CommandPaletteHost(): JSX.Element | null
```

- `const [open, setOpen] = useState(false)`
- `const hasBeenOpenRef = useRef(false)` — flips to `true` the first time `open` transitions true.
- Effect: global Cmd+K / Ctrl+K listener that toggles `open`. Copied verbatim from current `CommandPalette` body.
- Returns:
  - If `!hasBeenOpenRef.current`: `null` (no lazy load triggered).
  - Else: `<Suspense fallback={null}><LazyCommandPalette open={open} onOpenChange={setOpen} /></Suspense>`.

### `CommandPalette` (existing, signature extended)

Add controlled props:
```ts
interface CommandPaletteProps {
  open: boolean
  onOpenChange: (next: boolean) => void
}
```
Remove the internal `open` state + keydown effect (now lives in the Host). Everything else (page state, Dialog, cmdk) stays.

### `CardEditable` + `CardDetailModal`

```ts
const CardDetailModal = React.lazy(() =>
  import('./CardDetailModal.js').then((m) => ({ default: m.CardDetailModal })),
)
```
Wrap `<CardDetailModal ... />` in `<Suspense fallback={null}>`.

### `BoardRow` + `BoardSettingsModal`

Same pattern.

## Bundle budget selection

Procedure:
1. Land all code-splitting + cleanup (Tasks 1–7).
2. Build: `make renderer` (without bundle-check).
3. Measure: `gzip -c web/renderer/default/dist/assets/*.js | wc -c`.
4. `MAX_BYTES = round_up_to_next_5KB(measured + 10 KB)`.
5. Commit `check-bundle-size.sh` with that `MAX_BYTES` default and a comment recording the measurement date + raw value.

## Testing

- Existing tests still pass. Lazy imports resolve synchronously in bun+happy-dom; if a race appears in a specific test, wrap its render tree in `<Suspense fallback={null}>`.
- `bundle-check` tested by running `make bundle-check` after a clean build — exits 0 on pass; exits 1 if the measured size exceeds the budget. No dedicated unit test.
- `useBoardCrud` tests unchanged (removing `console.error` is invisible to assertions).
- No new tests for `CommandPaletteHost` — its keydown logic is the same as before, and the existing `CommandPalette.test.tsx` already covers Cmd+K open via the same listener code.

## Risks

- **`React.lazy` + SSR / first paint**: we don't SSR, so no regression. The first-open latency is a single-digit-ms in-memory module fetch — imperceptible.
- **Suspense fallback briefly flashing**: `fallback={null}` keeps the page quiet. Acceptable for a modal that wasn't visible a moment ago.
- **Test breakage**: dynamic imports under bun+happy-dom are usually synchronous. If a modal test fails, the fix is either adding a `<Suspense>` around the test subject or waiting an extra tick via `await new Promise(r => setTimeout(r, 0))`.
- **Bundle gate too tight**: a future feature that legitimately needs more KB fails the gate and must bump `MAX_BYTES`. That's the point — it's explicit.
- **Vite chunk naming**: the `dist/assets/*.js` glob captures both the main chunk and any dynamic-import chunks. The sum accurately represents user-facing JS; first-load shrinks while measured total may be similar or slightly larger due to code-split chunk boundary overhead. Spec accepts "measured" as the gate input, not "first-load".

## Open questions

None blocking. Pre-decided:
- Lazy via `React.lazy`; no custom loader.
- `CommandPalette` gets a thin host; others lazy directly at the call site.
- `MAX_BYTES` set from post-cleanup measurement with 10 KB headroom.
- `ts-prune` advisory only, not a gate.

## Dependencies on prior work

- P4c.2: `CommandPalette` exists and is eagerly mounted.
- P4b.3: `CardDetailModal` exists and is imported by `CardEditable`.
- P4d.1: `BoardSettingsModal` exists and is imported by `BoardRow`.
- P4d.2: dark-mode classes on top-level surfaces — this milestone finishes stragglers.
- P4c.1 manual-smoke debug: `console.error` in `useBoardCrud` that this milestone removes.
