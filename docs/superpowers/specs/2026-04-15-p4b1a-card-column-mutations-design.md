# P4b.1a — Card + Column Mutations — Design

## Goal

Make the read-only P4a renderer interactive for card and column mutations, using optimistic UI and an undo-toast pattern for destructive ops. No drag-drop, no card modal, no board CRUD — those are P4b.2, P4b.3, and P4c respectively.

**Shippable value:** `/app/` becomes a usable kanban. Users can add/edit/complete/delete cards, and add/rename/reorder/delete columns, with instant feedback and markdown-file truth.

## Scope

**In:**
- Card: add, inline-edit title (double-click), toggle complete, delete (with undo toast)
- Column: add, rename, reorder (move left/right), delete (with undo toast)
- Optimistic UI for non-destructive ops; staged delete with 5s undo window for destructive ops
- VERSION_CONFLICT rollback + refetch
- Error toasts for all protocol error codes

**Out (tracked for later milestones):**
- Drag-drop (P4b.2)
- Card detail modal — body, tags, priority, due, assignee (P4b.3)
- Board create/rename/delete — requires new protocol methods (P4c or later)
- Keyboard navigation beyond Enter/Escape in active inputs
- Undo stacking; focus restoration after edit

## Stack additions

| Concern | Choice | Size (gz) |
|---|---|---|
| Toasts + undo action | `sonner` | ~3 KB |
| Column menu primitive | `@radix-ui/react-dropdown-menu` | ~5 KB |

**No changes to `web/shared/*`.** The Client's `mutateBoard(boardId, clientVersion, op)` and the MutationOp union already cover every op in scope. Shared `applyOp` from P2 drives optimistic updates.

**Bundle target:** ≤ 75 KB gz (P4a: 59.5 KB; headroom for P4b.2/.3).

## Data flow — every mutation

```
user action (click / Enter)
    │
    ▼
useBoardMutation(boardId).mutate(op)   ── TanStack useMutation
    │
    ├─ onMutate:
    │     cancelQueries(['board', boardId])
    │     snapshot prev board
    │     setQueryData(applyOptimistic(prev, op))
    │     return { prev }
    │
    ├─ mutationFn:
    │     client.mutateBoard(boardId, prev.version, op)
    │     ── server applies op, version++, returns Board
    │
    ├─ onSuccess(newBoard):
    │     setQueryData(newBoard)         # authoritative
    │
    └─ onError(err, _op, { prev }):
          setQueryData(prev)             # rollback
          if err.code === 'VERSION_CONFLICT':
            invalidateQueries (forces refetch)
            toast.error("Board changed elsewhere — refreshed")
          else:
            toast.error(err.code)
```

**Why the cached `prev.version`**: the optimistic write deliberately does NOT bump version. The server is the only source of version truth. When `mutationFn` runs, it reads `prev.version` from cache — that's what the server compares against.

**Echo protection**: our P4a wiring calls `invalidateQueries` on every `board.updated` event. Our own successful mutation triggers that event. `cancelQueries` in `onMutate` + authoritative `setQueryData` in `onSuccess` makes the trailing invalidation a no-op — the cache is already current.

## Undo mechanism (delete only)

Delete ops are NOT fired optimistically. They stage behind a 5-second toast. If the user clicks "Undo", the mutation never fires and the cache is never touched.

```
delete click
    │
    ▼
stageDelete(mutation, op, label)
    │
    ├─ setTimeout 5s → mutation.mutate(op)  (fires real delete)
    │
    └─ toast("Deleted X", { action: 'Undo' → clearTimeout })
```

This avoids the UX hazard of "undo restored the card but the server already deleted it" — markdown truth and UI stay consistent. Tradeoff: a 5-second latency before the file on disk reflects the delete. Acceptable for local-first workflow.

**Single undo at a time**: if a second delete starts while a toast is still visible, sonner dismisses the prior toast. The prior delete then fires immediately (timer still runs). Out of scope to stack undos; document as a known limitation.

## File structure

**New files:**
```
web/renderer/default/src/
  mutations/
    useBoardMutation.ts
    useBoardMutation.test.tsx
    applyOptimistic.ts
    applyOptimistic.test.ts
    undoable.ts
    undoable.test.ts
  components/
    CardEditable.tsx
    CardEditable.test.tsx
    ColumnHeader.tsx
    ColumnHeader.test.tsx
    AddCardButton.tsx
    AddCardButton.test.tsx
    AddColumnButton.tsx
    AddColumnButton.test.tsx
    ToastHost.tsx
  toast.ts
```

**Modified:**
- `components/Card.tsx` — no behavioral change; kept pure/presentational. `CardEditable` wraps it.
- `components/Column.tsx` — uses `ColumnHeader` instead of inline header; maps `CardEditable` not `Card`; renders `AddCardButton` at bottom.
- `components/BoardView.tsx` — renders `AddColumnButton` after the column list.
- `App.tsx` — mounts `<ToastHost />`.

## Component contracts

### `applyOptimistic(board: Board, op: MutationOp): Board`
Pure. Clones the input via `structuredClone`, calls shared `applyOp` on the clone, returns the clone. Any op that throws from `applyOp` propagates.

### `useBoardMutation(boardId: string): UseMutationResult<Board, ProtocolError, MutationOp, { prev?: Board }>`
See data flow above. Reads Client from `ClientContext`. No side effects beyond cache writes + toast calls.

### `stageDelete(mutation, op, label): void`
Fires `setTimeout` + `toast` with Undo action. Undo clears the timeout. Second call before the first fires dismisses the first toast (sonner default); the first timer still fires.

### `<CardEditable card col_idx card_idx boardId />`
Local state: `mode: 'view' | 'edit'`. Double-click title → edit mode, `<input>` autofocused + selected.
- Enter: commit `edit_card` with existing fields, only `title` changed.
- Escape: revert to view mode, no mutation.
- Blur: commit (matches HTMX convention).
- Checkbox click: `complete_card`.
- Kebab → Delete: `stageDelete`.

### `<ColumnHeader column col_idx total_cols boardId />`
Radix DropdownMenu triggered by kebab. Items:
- Rename → name swaps to `<input>`; Enter commits `rename_column` `{ old_name, new_name }`; Escape cancels.
- Move left → `move_column` `{ name, after_col: <prev-prev-col-name or ''> }`. Disabled when `col_idx === 0`.
- Move right → `move_column` `{ name, after_col: <next-col-name> }`. Disabled when `col_idx === total_cols - 1`.
- Delete → `stageDelete` with `delete_column` `{ name }`.

### `<AddCardButton column_name boardId />`
Idle: ghost `<button>` "+ Add card". Click → inline `<input>`. Enter commits `add_card` `{ column: column_name, title }`. Escape / blur-empty reverts to idle. Blur with text commits.

### `<AddColumnButton boardId />`
Same pattern as `AddCardButton`. Dispatches `add_column` `{ name }`.

### `<ToastHost />`
`<Toaster position="bottom-right" richColors closeButton />` — mounted once.

## `move_column` semantics

The shared `move_column` op is `{ name, after_col: string }`. `after_col: ''` means "move to first position." The header needs to compute the target:

- **Move left from col_idx 1+**: `after_col = columns[col_idx - 2]?.name ?? ''`
- **Move right from col_idx < last**: `after_col = columns[col_idx + 1].name`

This is non-obvious and easy to get wrong. Implementation pulls the target computation into a named helper with its own tests.

## Error handling reference

| ProtocolError.code | Toast copy | Action |
|---|---|---|
| `VERSION_CONFLICT` | "Board changed elsewhere — refreshed" | rollback + invalidate |
| `NOT_FOUND` | "Card or column not found" | rollback |
| `OUT_OF_RANGE` | "Index out of range" | rollback |
| `INVALID` | "Invalid input" | rollback |
| `INTERNAL` | "Server error — try again" | rollback |
| (other) | `err.code` verbatim | rollback |

Copy is intentionally minimal. P4d's polish milestone can refine it.

## Testing

**Unit / component (bun test + happy-dom + @testing-library/react):**
- `applyOptimistic.test.ts` — every MutationOp variant; input not mutated; invalid ops throw.
- `undoable.test.ts` — `bun:test`'s mock timers: fires after 5s by default; `clearTimeout` stops it.
- `useBoardMutation.test.tsx` — against in-memory Broker+LocalAdapter (matches P4a `queries.test.tsx` pattern):
  - optimistic paint before mutationFn resolves
  - `onSuccess` replaces cache with server board (version incremented)
  - error rollback restores `prev`
  - VERSION_CONFLICT → rollback + invalidate
- Per-component tests: happy path + error rollback + keyboard (Enter/Escape).
- `CardEditable`: double-click → edit; Enter commits; Escape cancels; checkbox toggles.
- `ColumnHeader`: menu items fire correct ops; move-left disabled at 0; move-right disabled at last.
- `AddCardButton` / `AddColumnButton`: idle↔active; Enter commits; empty input cancels.

**Manual browser smoke** (same gate as P4a):
- Add a card → appears instantly, persists across refresh.
- Double-click title → edit → Enter → text updates.
- Toggle checkbox → strikethrough, persists.
- Delete card → undo toast; click Undo → card stays. Don't click Undo → card gone after 5s.
- Column add/rename/reorder/delete.
- Open a second tab on the same board; mutation in tab A should appear in tab B (already proven in P3, re-verify with real UI).
- Force a version conflict (devtools: `setQueryData(['board','welcome'], b => ({...b, version: 0}))`, then mutate) → toast + refetch.

## Risks

- **`structuredClone` on large boards**: profiling P4a's seed board is trivial (~1ms). For 1000+ card boards this could matter. Mitigation deferred — revisit if we hit real boards that lag.
- **Sonner + Radix pulled in dev builds vs prod**: Vite tree-shakes both cleanly in prod. Verify during the build task by inspecting `dist/` gzipped size against the 75 KB budget.
- **Editor focus management**: browser focus behavior inside a React 18 StrictMode dev build double-renders. Tests must not rely on effect ordering; use `userEvent` / `waitFor`. Document if we hit flakiness.
- **Move-column edge cases**: `after_col` contract is easy to reason about wrong. Named helper + exhaustive test table mitigates.

## Open questions

None blocking. The following are design-time choices already made:

- Blur on inline edit **commits** (matches HTMX). If users complain, swap to Escape-to-cancel + click-outside-commits with no behavioral diff.
- Undo stacking is out of scope. One active undo toast at a time.
- No auto-retry on VERSION_CONFLICT. User reissues if they still want the change.

## Dependencies on prior work

- P2: `web/shared/src/boardOps.ts::applyOp` — already imported in tests; now used in renderer runtime.
- P3: Client surface (mutateBoard, subscribe, event channel) + ProtocolError.
- P4a: `ClientProvider`, `queryClient`, `Card`, `Column`, `BoardView`, `App`, test-utils.
