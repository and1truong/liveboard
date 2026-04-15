# P4d.1 — Per-Board Settings UI — Design

## Goal

Surface two per-board settings (`show_checkbox`, `card_display_mode`) in `/app/`'s default renderer:
- A **Settings** item appears in `BoardRow`'s kebab menu, opening a Radix Dialog modal.
- The modal saves via the existing `client.putBoardSettings` round-trip.
- `CardEditable` consults the resolved settings and renders accordingly: hides the complete-circle when `show_checkbox === false`, applies tighter spacing when `card_display_mode === 'compact'`.
- Fix the `LocalAdapter.getSettings` bug where stored `board.settings` were ignored.

**Shippable value:** users can toggle two cosmetic board options that have visible effect. Unblocks P4d.2 (themes) and P4d.3 (bundle gate / cleanup).

## Scope

**In:**
- `<BoardSettingsModal>` Radix Dialog with two form controls.
- `useBoardSettings(boardId)` + `useUpdateSettings(boardId)` hooks.
- `BoardRow` kebab menu Settings item that opens the modal.
- `CardEditable` reads settings: hide complete-circle, apply compact layout class.
- `LocalAdapter.getSettings` reads `board.settings` and merges with defaults.
- Tests for the LocalAdapter round-trip and the modal.

**Out:**
- The other 4 settings (`card_position`, `expand_columns`, `view_mode`, `week_start`).
- Cross-tab `settings.updated` event subscription in the renderer.
- Global (workspace-level) settings.
- Theme switcher (P4d.2).
- Settings palette integration (could land in P4d.2 if Cmd+K wants a "Board settings" action).
- Storage migration of pre-existing localStorage entries (no migration needed — the bug fix is read-side only).

## Architecture

```
BoardRow
 └─ DropdownMenu.Item "Settings" → setSettingsOpen(true)
 └─ <BoardSettingsModal boardId={board.id} boardName={board.name}
                        open={settingsOpen} onOpenChange={setSettingsOpen} />

BoardSettingsModal (Radix Dialog)
 └─ form
     ├─ checkbox: show_checkbox
     ├─ select:   card_display_mode (normal | compact)
     └─ Cancel | Save
        Save → useUpdateSettings.mutate({ show_checkbox, card_display_mode })
              onSuccess: invalidate ['settings', boardId], close modal

CardEditable (existing)
 └─ const settings = useBoardSettings(boardId)
    - Hide complete-circle button when !settings.show_checkbox
    - Apply compact spacing when settings.card_display_mode === 'compact'
```

## Wire-shape: no protocol changes

`Client.getSettings` and `Client.putBoardSettings` already exist (P3). The protocol response/request shapes are unchanged.

## LocalAdapter fix

Current `LocalAdapter.getSettings`:
```ts
async getSettings(boardId: string): Promise<ResolvedSettings> {
  this.loadBoard(boardId)
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
**Bug:** ignores `board.settings`. After `putBoardSettings({ show_checkbox: false })`, `getSettings` still returns `true`.

Fix:
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

`putBoardSettings` already merges into `board.settings` correctly — no change there.

## Component contracts

### `useBoardSettings(boardId: string | null): ResolvedSettings`

```ts
const DEFAULTS: ResolvedSettings = {
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
  return q.data ?? DEFAULTS
}
```

Synchronous return of defaults during loading keeps consumers simple — no `isLoading` ternary at every call site.

### `useUpdateSettings(boardId: string)`

`useMutation` wrapping `client.putBoardSettings(boardId, patch)`. `onSuccess` invalidates `['settings', boardId]` and refetches. `onError` calls `errorToast(code)`.

### `<BoardSettingsModal>`

Props:
```ts
{
  boardId: string
  boardName: string
  open: boolean
  onOpenChange: (next: boolean) => void
}
```

Uses uncontrolled inputs (`defaultValue` + ref reads, `committedRef` not strictly needed since the form has explicit Save). Reseeds via `key={String(open)}` on `Dialog.Content`.

```tsx
<Dialog.Root open={open} onOpenChange={onOpenChange}>
  <Dialog.Portal>
    <Dialog.Overlay className="fixed inset-0 z-40 bg-black/40" />
    <Dialog.Content key={String(open)} className="..." aria-label="Board settings">
      <Dialog.Title>Settings: {boardName}</Dialog.Title>
      <form onSubmit={submit}>
        <label>
          <input type="checkbox" ref={checkboxRef} defaultChecked={settings.show_checkbox} />
          Show complete checkbox on cards
        </label>
        <label>
          Card display
          <select ref={modeRef} defaultValue={settings.card_display_mode}>
            <option value="normal">Normal</option>
            <option value="compact">Compact</option>
          </select>
        </label>
        <button type="button" onClick={() => onOpenChange(false)}>Cancel</button>
        <button type="submit">Save</button>
      </form>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
```

Save handler:
```tsx
const submit = (e) => {
  e.preventDefault()
  mutation.mutate(
    {
      show_checkbox: checkboxRef.current?.checked ?? true,
      card_display_mode: modeRef.current?.value ?? 'normal',
    },
    { onSuccess: () => onOpenChange(false) },
  )
}
```

### `<BoardRow>` kebab gains "Settings"

After Rename, before the Separator/Delete:
```tsx
<DropdownMenu.Item onSelect={() => setSettingsOpen(true)}>
  Settings
</DropdownMenu.Item>
```

`const [settingsOpen, setSettingsOpen] = useState(false)`. Mount `<BoardSettingsModal>` after the menu.

### `<CardEditable>` settings consumption

Add at the top:
```tsx
const settings = useBoardSettings(boardId)
const showCheckbox = settings.show_checkbox
const compact = settings.card_display_mode === 'compact'
```

In view-mode markup:
- Render the complete-circle button only if `showCheckbox`.
- Apply tighter padding/text classes when `compact` (e.g., outer wrapper `p-2 text-xs` vs `p-3 text-sm`).

Concrete class diff for the outer `<div>`:
```tsx
className={`group relative rounded-md bg-white shadow-sm ring-1 ring-slate-200 ${
  compact ? 'p-2 text-xs' : 'p-3 text-sm'
}`}
```

## Data flow

```
Open settings modal → useBoardSettings(boardId) hits cache (or fetches)
   │
form seeded with current values
   │
user toggles + clicks Save
   │
   ▼
useUpdateSettings.mutate(patch)
   client.putBoardSettings → broker → adapter.putBoardSettings → board.settings updated, version++
   │
onSuccess: invalidate ['settings', boardId] → refetch
   │
useBoardSettings re-emits → CardEditable re-renders with new layout
```

## Error handling

`errorToast` from P4b.1a covers all `ProtocolError` codes. Modal stays open on error. No new error paths.

## Testing

### `web/shared/src/adapters/local.settings.test.ts` (new)

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

### `web/renderer/default/src/components/BoardSettingsModal.test.tsx` (new)

Cases:
- Renders form seeded from settings (toggle off, mode compact) when stored settings differ from defaults.
- Save fires `putBoardSettings` with the toggled values; `getSettings` reflects.
- Cancel closes without mutation.

Reuses the Broker + LocalAdapter setup pattern from prior modal tests.

## Visual

- Modal: same Radix Dialog visual treatment as `CardDetailModal` (centered card, `max-w-md`).
- Form: vertical stack, label-on-left layout for the checkbox, label-above for the select.
- Compact card: noticeably tighter padding and slightly smaller font; tags row remains.

## Risks

- **`useBoardSettings` called from `CardEditable` adds a query per board** — TanStack dedupes by key, so one query per active board, refetch on invalidate. Fine.
- **Re-render cascade on settings change** — every `CardEditable` for the board re-renders because the query key invalidates. Acceptable for a settings save (rare event).
- **Compact mode and existing layout tests** — `Card.test.tsx` / `CardEditable.test.tsx` assertions don't depend on padding classes. Should pass unchanged.
- **`enabled: !!boardId`** keeps the query off when there's no active board. Hook still returns DEFAULTS, no error.

## Open questions

None blocking. Pre-decided:
- Two wired settings, four left exposed in protocol but not surfaced in UI.
- Trigger via BoardRow kebab.
- Modal pattern mirrors CardDetailModal.
- LocalAdapter bug fix lands in this milestone.
- No cross-tab settings.updated subscription.

## Dependencies on prior work

- P3: `Client.getSettings`, `Client.putBoardSettings`, `BackendAdapter`, LocalAdapter primitives.
- P4b.1a: `errorToast`, `useBoardMutation` shape (referenced for hook structure).
- P4b.3: `@radix-ui/react-dialog` (already installed).
- P4c.1: `BoardRow`, `useActiveBoard` (not strictly required here; settings are scoped to the row's board id).
