# P4c.1 — Sidebar Board CRUD UI — Design

## Goal

Wire the P4c.0 protocol into the renderer sidebar: a "+ New board" affordance, a per-board hover-kebab with Rename + Delete actions, and automatic active-board re-routing on rename and on delete-of-active. No protocol additions — pure UI on top of P4c.0.

**Shippable value:** users can fully manage their workspace from `/app/`. After P4c.1, only command-palette (P4c.2) and keyboard nav (P4c.3) remain for P4c.

## Scope

**In:**
- `<AddBoardButton>` at the bottom of the sidebar (idle ghost button → inline input → Enter / blur commits).
- Per-board hover-kebab using Radix DropdownMenu with **Rename** and **Delete** items.
- Inline rename (uncontrolled input, defaultValue + ref pattern).
- Delete with 5s undo toast (reusing the P4b.1a `stageDelete` mechanism, generalized to take `() => void`).
- Active-board state lifted into `<ActiveBoardContext>`.
- Auto-route on rename: if the renamed board was active, switch active to the new id.
- Auto-route on delete: if the deleted board was active, switch active to the first remaining board, or `null` if none.
- Cross-tab `board.list.updated` invalidates `['boards']` query.
- `errorToast` for protocol errors; specific copy override for `ALREADY_EXISTS` on create/rename.

**Out:**
- Drag-reorder of boards in the sidebar (P4d or later).
- Board metadata UI (icon, description) — not yet exposed by `createBoard`/`renameBoard`. Defer to P4d.
- Sidebar collapse / favorites / sections.
- Command palette (P4c.2).
- Keyboard navigation (P4c.3).

## Architecture

```
App
 └─ <ActiveBoardProvider>          # owns active boardId + setActive
     ├─ useBoardListEvents()        # subscribes to board.list.updated → invalidates ['boards']
     ├─ <BoardSidebar>
     │    ├─ <BoardRow board /> × N
     │    │    ├─ row label (click → setActive(id))
     │    │    └─ kebab → Radix menu (Rename inline-input, Delete via stageDelete)
     │    └─ <AddBoardButton>
     └─ <BoardView boardId={active} />
```

Three new mutation hooks (`useCreateBoard`, `useRenameBoard`, `useDeleteBoard`) each read `useClient` and `useActiveBoard` and own their own `onSuccess` cache + active-id logic.

## File structure

**New:**
- `web/renderer/default/src/contexts/ActiveBoardContext.tsx` — provider + `useActiveBoard()` hook returning `{ active, setActive }`.
- `web/renderer/default/src/mutations/useBoardCrud.ts` — three hooks below.
- `web/renderer/default/src/mutations/useBoardCrud.test.tsx`
- `web/renderer/default/src/mutations/useBoardListEvents.ts` — small hook that wires `client.on('board.list.updated', ...)` to `qc.invalidateQueries(['boards'])`.
- `web/renderer/default/src/components/AddBoardButton.tsx`
- `web/renderer/default/src/components/AddBoardButton.test.tsx`
- `web/renderer/default/src/components/BoardRow.tsx`
- `web/renderer/default/src/components/BoardRow.test.tsx`

**Modified:**
- `web/renderer/default/src/mutations/undoable.ts` — generalize `stageDelete(fire: () => void, label: string)`.
- `web/renderer/default/src/mutations/undoable.test.ts` — update for new signature.
- `web/renderer/default/src/components/CardEditable.tsx` — call site of `stageDelete`.
- `web/renderer/default/src/components/ColumnHeader.tsx` — call site of `stageDelete`.
- `web/renderer/default/src/components/BoardSidebar.tsx` — render `BoardRow`s + `AddBoardButton`; remove its current onSelect prop now that ActiveBoardContext owns active state.
- `web/renderer/default/src/App.tsx` — wrap in `<ActiveBoardProvider>`; mount `useBoardListEvents`; remove its own `useState<string|null>` for active.
- `web/renderer/default/src/toast.ts` — extend `errorToast` map to include a friendly `ALREADY_EXISTS` line.

## Component contracts

### `<ActiveBoardProvider>` + `useActiveBoard()`

```tsx
interface ActiveBoardCtx {
  active: string | null
  setActive: (next: string | null) => void
}
```

Provider holds `useState<string | null>(null)` internally. Single context, single hook. App.tsx wraps `<BoardSidebar>` and `<BoardView>` with this provider. `BoardView` reads `active` instead of taking it as a prop.

### `useCreateBoard()`

Returns `UseMutationResult<BoardSummary, Error, string, ...>` where the variable is the new name.
- `mutationFn: (name) => client.createBoard(name)`
- `onSuccess: (summary) => { qc.invalidateQueries(['boards']); setActive(summary.id) }`
- `onError: (err) => errorToast(code(err))`

### `useRenameBoard()`

Variables: `{ boardId: string; newName: string }`.
- `mutationFn: ({ boardId, newName }) => client.renameBoard(boardId, newName)`
- `onSuccess: (summary, { boardId }) => { qc.invalidateQueries(['boards']); if (active === boardId) setActive(summary.id) }`
- `onError: errorToast`

### `useDeleteBoard()`

Variable: `boardId: string`.
- `mutationFn: (boardId) => client.deleteBoard(boardId)`
- `onMutate: (boardId)` — read current `['boards']` snapshot from cache; compute fallbackActive = first id !== boardId, or null. Stash in context.
- `onSuccess: (_void, boardId, ctx) => { qc.invalidateQueries(['boards']); if (active === boardId) setActive(ctx.fallbackActive) }`
- `onError: errorToast`

### `useBoardListEvents()`

Hook that mounts a `useEffect` on `client.on('board.list.updated', () => qc.invalidateQueries({ queryKey: ['boards'] }))`. Returns the unsubscribe via cleanup.

### `<AddBoardButton>`

Mirrors `AddColumnButton`:
- Idle state: ghost `<button>` "+ New board" at the bottom of the sidebar.
- Click → inline `<input aria-label="new board name">` (uncontrolled, defaultValue=''), focus + select.
- Enter / blur with text → `useCreateBoard().mutate(value)`.
- Escape / blur empty → revert to idle.
- After successful creation, the `useCreateBoard.onSuccess` switches active board automatically.

### `<BoardRow board />`

Props: `{ board: BoardSummary }`.

Local state: `mode: 'view' | 'edit'`.

View mode:
- Whole row is a `<button>` (or `<div role="button">`) that calls `setActive(board.id)` on click.
- Active styling when `active === board.id`.
- Hover-kebab on the right edge: Radix `<DropdownMenu>` with:
  - **Rename** → `setMode('edit')`.
  - **Delete** → `stageDelete(() => useDeleteBoard().mutate(board.id), board.name)`.

Edit mode:
- Inline `<input aria-label="rename board ${board.name}">` with defaultValue=board.name, autofocus + select.
- Enter / blur → `useRenameBoard().mutate({ boardId: board.id, newName: <ref value> })` (via committedRef pattern), then `setMode('view')` (deferred via Promise.resolve().then).
- Escape → revert to view mode.

### `stageDelete` (generalized)

Old signature:
```ts
function stageDelete(
  mutation: UseMutationResult<unknown, unknown, MutationOp, unknown>,
  op: MutationOp,
  label: string,
): void
```
New signature:
```ts
function stageDelete(fire: () => void, label: string): void
```

Internally uses the existing `scheduleDelete(fire, ms)` (already split out in P4b.1a).

Call sites:
- `CardEditable`: `stageDelete(() => mutation.mutate({ type: 'delete_card', col_idx, card_idx }), card.title)`
- `ColumnHeader`: `stageDelete(() => mutation.mutate({ type: 'delete_column', name }), name)`
- `BoardRow`: `stageDelete(() => deleteBoard.mutate(board.id), board.name)`

## Cross-tab refresh

`useBoardListEvents` registers a `client.on('board.list.updated', ...)` listener in App. Any cross-tab create/rename/delete arrives here and invalidates the sidebar's list query. The same listener also fires for *this tab*'s own mutations (the `LocalAdapter` broadcasts to itself too), but the explicit `qc.invalidateQueries` in each hook's `onSuccess` keeps things responsive without waiting for the BroadcastChannel round-trip.

## Error handling

`errorToast` map in `toast.ts` extends with:
```ts
ALREADY_EXISTS: 'A board with that name already exists',
```
Placed alongside the existing entries for VERSION_CONFLICT etc. All other codes fall through to the existing entries or the `code` verbatim.

Inline rename input stays open on error so the user can fix the name; modal-less create input is hidden by `AddBoardButton`'s own success path — on error it simply leaves the user in the idle state (the toast carries the message). Re-clicking "+ New board" re-opens.

## Testing

**Unit / component (bun test + happy-dom + @testing-library/react):**

- `useBoardCrud.test.tsx` — round-trip each hook against in-memory Broker + LocalAdapter:
  - Create → returns BoardSummary; sidebar list grows; active switches to new id.
  - Rename → BoardSummary returned with new id; if was active, active follows.
  - Delete → list shrinks; if was active, active falls back to first remaining (or null).
- `BoardRow.test.tsx`:
  - Click row → setActive called with row's id.
  - Kebab → Rename → input appears; blur with text fires renameBoard with new name.
  - Kebab → Delete → undo toast appears (sonner mounted via test-utils).
- `AddBoardButton.test.tsx`:
  - Click reveals input; blur with text fires createBoard; sidebar list contains the new board.
- `undoable.test.ts` — updated for new `stageDelete(fire, label)` signature; `scheduleDelete` tests unchanged.

Reuses the uncontrolled-input pattern + `committedRef` to dodge happy-dom keyDown.

## Visual spec

- AddBoardButton: full-width ghost row at the bottom of the 240px sidebar.
- BoardRow: 32px tall, padding-x 12, name flex-1, kebab fixed 24px right; active row has `bg-slate-200`.
- Inline rename input: replaces the name span in-place, `bg-white ring-1 ring-blue-400`.
- Kebab menu: same Radix style as `ColumnHeader`'s.

## Risks

- **`stageDelete` signature churn** ripples through CardEditable + ColumnHeader. Mechanical, covered by existing tests.
- **Active-id race on rename**: if a `useBoard(active)` request is in flight when rename completes, the in-flight response writes to the *old* query key (since the query was issued under the old key). Switching `active` causes a fresh fetch under the new key. The old key's data lingers harmlessly until GC. Documented as benign.
- **Rename or delete from another tab while the local renderer has the doomed board active**: the sidebar invalidates its list, but `useBoard(active)` is still pointed at a now-missing id and will get a `NOT_FOUND`. Mitigation: in `BoardView`, on `error?.code === 'NOT_FOUND'`, call `setActive(null)` (or first board). Add this as a small effect inside `BoardView`. Out-of-scope as a polish item if it complicates P4c.1; otherwise include.
  **Decision:** include the effect in scope. Bare-minimum: `useEffect` on error → setActive(null).
- **`ALREADY_EXISTS` copy override**: extending `errorToast` mid-renderer is fine; just keep the map a single source of truth.

## Open questions

None blocking. Pre-decided:
- Bottom-anchored "+ New board".
- Per-row hover-kebab Radix menu with Rename + Delete.
- Inline rename, undo-toast delete.
- Active-board state in `<ActiveBoardContext>`; auto-routes on rename + delete-of-active.
- `stageDelete` generalized to `(fire, label)`.
- Cross-tab refresh via `client.on('board.list.updated', ...)`.
- `BoardView` self-recovers on `NOT_FOUND` (cross-tab delete of active board).

## Dependencies on prior work

- P4c.0: protocol methods + `Client.createBoard / renameBoard / deleteBoard` + `board.list.updated` event.
- P4b.1a: `useBoardMutation` (referenced for shape), `errorToast`, `stageDelete` (to be generalized), Radix DropdownMenu pattern from `ColumnHeader`, uncontrolled-input + committedRef pattern.
- P4a: `useBoardList`, `BoardSidebar`, `App`.
