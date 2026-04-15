# P4b.3 — Card Detail Modal — Design

## Goal

Add a click-to-open card detail modal in the `/app/` default renderer that edits all card fields (title, body, tags, priority, due, assignee) in one form. Single click on a card body opens the modal; double-click on the title preserves the existing inline title-edit path.

**Shippable value:** Renderer reaches feature parity with the HTMX UI's "open card to edit everything" flow. After P4b.3, P4b is done — only P4c (board CRUD chrome) and P4d (settings/themes) remain in P4.

## Scope

**In:**
- Radix Dialog modal triggered by single click on card body.
- Form with title, body, tags, priority, due, assignee.
- Save fires one `edit_card` mutation; Cancel/Escape closes without mutation.
- Save disabled when title is empty.
- Optimistic update + rollback flow through existing `useBoardMutation`.
- Modal stays open during request and on error so the user can retry.

**Out (later milestones):**
- Markdown rendering for body (plain textarea ships now; preview deferred to P4d).
- Tag autocomplete / member picker (plain text inputs ship now).
- Dirty-state warning on cancel.
- Inline modal editor (e.g., calendar widget for due) — native `<input type="date">` is enough.
- Modal triggered by keyboard (Enter on focused card) — defer.

## Stack additions

| Concern | Choice | Size (gz) |
|---|---|---|
| Modal primitive | `@radix-ui/react-dialog` | ~3–4 KB |

Bundle delta on top of P4b.2's 116.5 KB → ~120 KB. Gate stays deferred to P4d.

## Architecture

```
CardEditable (existing — extended)
  ├─ inline-edit input (mode='edit', from title double-click — unchanged)
  └─ view mode
       ├─ title region (double-click → inline edit; single click ignored)
       └─ body region (single click → setModalOpen(true))
            └─ <CardDetailModal open onOpenChange card colIdx cardIdx boardId />
                 └─ <Dialog.Root open onOpenChange>
                      └─ <Dialog.Portal>
                           ├─ <Dialog.Overlay>
                           └─ <Dialog.Content>
                                ├─ <Dialog.Title> "Edit card"
                                ├─ form (title, body, tags, priority, due, assignee)
                                └─ Cancel + Save buttons
```

`CardDetailModal` is **controlled** — open/close state lives in `CardEditable`. The dialog itself owns no business state besides per-field local form values seeded from the `card` prop on each open.

## Click-vs-doubleclick disambiguation

The single-click vs. double-click race is sidestepped by **surface partitioning**: title and body are separate DOM regions with non-overlapping handlers.

- Title region: keeps its existing `onDoubleClick` → inline edit. No `onClick` registered.
- Body region (everything else inside the card except action buttons): registers `onClick` → open modal. No `onDoubleClick` registered.
- Action buttons (complete circle, delete ✕, drag grip): retain their own click handlers; click events never bubble to the body click handler (`stopPropagation` if needed).

This makes click and double-click semantically distinct by location, not by timing.

## Component contracts

### `<CardDetailModal>`

Props:
```ts
{
  card: CardModel
  colIdx: number
  cardIdx: number
  boardId: string
  open: boolean
  onOpenChange: (next: boolean) => void
}
```

Behavior:
- On `open=true` transition: seed local form state from `card`. (Use `key={String(open)}` on the form, or a `useEffect` watching `open`, to reseed on every reopen.)
- Form fields:
  - **Title** — uncontrolled `<input>`, `defaultValue={card.title}`. Required.
  - **Body** — uncontrolled `<textarea rows={6}>`, `defaultValue={card.body ?? ''}`.
  - **Tags** — uncontrolled `<input>`, `defaultValue={(card.tags ?? []).join(', ')}`.
  - **Priority** — `<select defaultValue={card.priority ?? ''}>` with `''`, `low`, `medium`, `high`, `critical`.
  - **Due** — `<input type="date" defaultValue={card.due ?? ''}>`.
  - **Assignee** — uncontrolled `<input>`, `defaultValue={card.assignee ?? ''}`.
- Refs collect all six values on Save.
- Save handler:
  - Read all refs.
  - `title.trim()` empty → no-op (Save button is disabled in this state, but defensive).
  - Build `edit_card` op with all six fields. Tags: `tagsRef.value.split(',').map(t => t.trim()).filter(Boolean)`.
  - Call `useBoardMutation(boardId).mutate(op)`.
  - On `mutation.isSuccess`, call `onOpenChange(false)`.
  - On error, modal stays open (errorToast already fires from `useBoardMutation`).
- Cancel button: `onOpenChange(false)`. Escape and overlay click route through Radix Dialog's own close path → `onOpenChange(false)`.
- Save button is disabled while `mutation.isPending` and shows "Saving…".

### Modified `<CardEditable>`

- Adds `const [modalOpen, setModalOpen] = useState(false)`.
- View-mode markup splits into:
  - Existing title `<h-something>` element wrapped in a `<div onDoubleClick={enterEdit}>` (no onClick).
  - A separate clickable wrapper around the rest of the card body that calls `setModalOpen(true)` on click.
  - Action buttons (complete, delete, grip) keep their click handlers and add `e.stopPropagation()` so they don't accidentally open the modal.
- Renders `<CardDetailModal>` after the card markup with `open={modalOpen}` and `onOpenChange={setModalOpen}`.

## Data flow

```
click body → setModalOpen(true)
   │
   ▼
<CardDetailModal open=true>  reads `card` prop, seeds defaults
   │
user edits form
   │
clicks Save
   │
   ▼
read refs → build edit_card op → mutation.mutate(op)
   │
useBoardMutation:
   onMutate: optimistic applyOp
   mutationFn: client.mutateBoard → server
   onSuccess: cache write
       └─ effect: onOpenChange(false)
   onError: rollback + errorToast (modal stays open)
```

## Error handling

All errors flow through `useBoardMutation`:
- `VERSION_CONFLICT` → rollback + invalidate + toast; modal stays open with user's current edits.
- `INVALID` → toast; modal stays open.
- Other codes → existing copy.

The modal does NOT special-case any error code. The `errorToast` fires from `useBoardMutation.onError` regardless of where the mutation was triggered.

## Testing

`CardDetailModal.test.tsx` uses the same setup as the other component tests (`Broker` + `LocalAdapter` + `MemoryStorage`, seeded cache). Reuses uncontrolled-input pattern (`defaultValue` + ref reads + blur/click events) to avoid the happy-dom `keyDown` issue documented in P4b.1a.

Cases:
- Modal renders title, body, tags-as-CSV, priority, due, assignee from `card` prop.
- Save fires `edit_card` with the modified values; tags round-trip through CSV split.
- Cancel button closes without firing mutation.
- Empty title disables Save (button has `disabled` attribute).
- Save closes the modal on success (assert `onOpenChange(false)` was called).

`CardEditable.test.tsx` gets one new case:
- Single click on body region opens the modal (modal title "Edit card" appears).
- Double click on title region still enters inline edit (existing test).
- Click on the complete or delete button does NOT open the modal.

Radix Dialog uses Portal + focus trap; happy-dom supports both well enough for these assertions per the P4b.1a Radix experience.

## Visual spec

Minimal, deferred polish to P4d:
- Overlay: dimmed backdrop (`bg-black/40`).
- Content: centered card-style panel, `max-w-lg`, `rounded-lg`, `bg-white`, `p-6`, with `<Dialog.Title>` styled like an `<h2>`.
- Form: vertical stack with `space-y-3`. Each field labeled, native browser controls.
- Buttons row: right-aligned, Cancel (ghost), Save (primary blue).

No animations beyond Radix defaults.

## Risks

- **Bundle**: ~4 KB more on top of 116.5 KB → ~120 KB gz. Within deferred budget.
- **Touch & DnD interaction**: single click on card body now triggers the modal. Touch sensor in P4b.2 has 200 ms / 5 px activation, so a tap completes click before the drag starts — modal opens. This is the intended behavior on touch (no double-click on touch). Drag still works from the grip icon.
- **Click-bubbling from inner action buttons**: complete and delete buttons must `stopPropagation` on click; grip is wrapped in dnd-kit's listeners and doesn't fire normal clicks. Document and verify.
- **Reopening with stale data**: if the card was edited elsewhere between modal opens, the seed is from the *latest* `card` prop (passed from cache) — so the cache invalidation flow already keeps it fresh. No special handling needed.

## Open questions

None blocking. Pre-decided:
- Click-on-body opens; double-click-on-title still inline-edits; partitioned by surface, not by timing.
- Radix Dialog.
- Save/Cancel; no auto-save; no dirty-state confirm.
- Plain textarea for body (no markdown rendering yet).
- Tags as comma-separated text; split on save.

## Dependencies on prior work

- P2: shared `edit_card` MutationOp + `applyOp` semantics.
- P4a: `Card`, query infrastructure.
- P4b.1a: `CardEditable`, `useBoardMutation`, `errorToast`, uncontrolled-input test pattern.
- P4b.2: drag handle pattern (separate grip icon, click never on card body via DnD listeners).
