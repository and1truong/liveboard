# P4c.2 — Command Palette (Cmd+K) — Design

## Goal

Add a Cmd+K / Ctrl+K command palette to the `/app/` default renderer that lets users:
- Jump to any board by typing.
- Create a new board.
- Rename the current board.
- Delete the current board (with the existing 5s undo toast).

Built on `cmdk` (Vercel) + Radix Dialog. Reuses existing CRUD hooks from P4c.1. No protocol additions.

**Shippable value:** keyboard-first workflow for board management. After P4c.2, only P4c.3 (board-content keyboard nav) remains in P4c.

## Scope

**In:**
- `<CommandPalette>` component, controlled `open` toggled by global Cmd+K / Ctrl+K listener.
- cmdk-based filterable list of boards + three action items.
- Internal "pages": **list**, **create-input**, **rename-input**.
- Delete fires `stageDelete(...)` immediately — no input page.
- Rename / Delete actions hidden when `active === null`.
- Esc / outside-click closes (via Radix Dialog).

**Out:**
- Recents / pinned boards (defer).
- Card / column commands (out of scope; would conflate workspace and board concerns).
- Customizable shortcuts.
- Multiple keybindings beyond Cmd/Ctrl+K.
- Action discovery via icons (text labels only for P4c.2).

## Stack additions

| Concern | Choice | Size (gz) |
|---|---|---|
| Command primitive | `cmdk` (Vercel) | ~6 KB |

Bundle delta on top of P4c.1's 120 KB → ~126 KB. Bundle gate stays deferred to P4d.

## Architecture

```
App
 └─ <ActiveBoardProvider>
     ├─ <BoardSidebar />
     ├─ <BoardView />
     ├─ <Toaster />
     └─ <CommandPalette />        # NEW

CommandPalette
 ├─ global useEffect: keydown(window, k+meta/ctrl) → toggle open
 └─ <Dialog.Root open onOpenChange>     # Radix dialog wrapper
     └─ <Dialog.Portal>
         ├─ <Dialog.Overlay />
         └─ <Dialog.Content>
             └─ page === 'list'
                 │  <Command>           # cmdk filter primitive
                 │    ├─ <Command.Input placeholder="Type a command or board name…" />
                 │    ├─ <Command.List>
                 │    │   ├─ <Command.Group heading="Boards">
                 │    │   │    └─ <Command.Item value="board:<id>" onSelect>...</Command.Item> × N
                 │    │   └─ <Command.Group heading="Actions">
                 │    │        ├─ Create board → setPage('create')
                 │    │        ├─ Rename current board → setPage('rename')   # only if active
                 │    │        └─ Delete current board → stageDelete(...)    # only if active
                 │    │   </Command.Group>
                 │    └─ </Command.List>
                 │  </Command>
                 ├─ page === 'create'
                 │  <input ref autoFocus onBlur=close onKeyDown(Enter)>
                 │  Enter → useCreateBoard().mutate(name); close
                 └─ page === 'rename'
                    <input ref defaultValue=activeName autoFocus>
                    Enter → useRenameBoard().mutate({ boardId: active, newName }); close
```

Internal state:
```ts
const [open, setOpen] = useState(false)
const [page, setPage] = useState<'list' | 'create' | 'rename'>('list')
```

When `open` flips to `true`, reset `page` to `'list'` via effect (or via Dialog `onOpenChange` callback).

## Keyboard

- **Cmd+K** (Mac) / **Ctrl+K** (Win/Linux) → toggle `open`. `event.preventDefault()` to suppress browser default (Chrome's URL bar focus).
- **Escape** → Radix Dialog handles close.
- **Up/Down** → cmdk arrow nav.
- **Enter** → activates current item; on input pages, commits.
- The keydown listener mounts once at App; survives the session.

## Component contract

`<CommandPalette />` — no props.

Reads:
- `useActiveBoard` → `{ active, setActive }`
- `useBoardList()` → `BoardSummary[]`
- `useCreateBoard()`, `useRenameBoard()`, `useDeleteBoard()` (existing P4c.1 hooks)

Inputs (uncontrolled, refs collect values on commit) — same `committedRef` + `Promise.resolve().then(...)` pattern from prior milestones for happy-dom safety.

Action handlers:

```ts
// Boards
onSelect: (b) => { setActive(b.id); setOpen(false) }

// Create
onSelect: () => setPage('create')
// Then in create page Enter:
const name = inputRef.current?.value.trim()
if (name) createBoard.mutate(name)
setOpen(false)

// Rename (only if active)
onSelect: () => setPage('rename')
// Then in rename page Enter:
const next = inputRef.current?.value.trim()
if (next && next !== activeName) renameBoard.mutate({ boardId: active, newName: next })
setOpen(false)

// Delete (only if active)
onSelect: () => {
  stageDelete(() => deleteBoard.mutate(active), activeName)
  setOpen(false)
}
```

## Data flow

```
Cmd+K
  │
  ▼
setOpen(true) → <Dialog> opens, page='list'
  │
user types → cmdk filters
  │
selects board OR action
  │
  ▼
boards: setActive + close
actions: page change OR mutation + close
```

Mutations use the existing CRUD hooks; their toast/error paths apply unchanged. `errorToast` from P4b.1a fires on `ProtocolError`. Palette closes optimistically; failures show only in the toast.

## Testing

`CommandPalette.test.tsx` (bun test + happy-dom + @testing-library/react):

1. **Closed by default** — palette content not in document.
2. **Cmd+K opens it** — `fireEvent.keyDown(window, { key: 'k', metaKey: true })` then assert input visible. (If cmdk's portal needs a tick under happy-dom, wrap in `waitFor`.)
3. **Lists boards from cache** — after open, board names render.
4. **Type to filter** — type "wel" → only "Welcome" remains visible.
5. **Selecting a board fires setActive + closes** — wrap render in `<ActiveBoardProvider>` plus a `<Probe />` that exposes `useActiveBoard()` so we can assert `active` after click.
6. **"Create board" → input page → Enter creates** — palette switches to input; `fireEvent.input` + `fireEvent.blur` (or simulated Enter via blur fallback per uncontrolled pattern).
7. **Rename action hidden when no active board** — query for "Rename current board" returns null when active === null.
8. **Delete action triggers undo toast** — sonner mounted in test scaffolding (existing `Toaster` test setup from P4b.3).

cmdk's filter uses `requestAnimationFrame` indirectly; happy-dom polyfills RAF, but a `waitFor` wrap on the filter assertions is safer.

## Risks

- **Browser default Cmd+K** in Chrome focuses the address bar. `event.preventDefault()` on our captured keydown suppresses this only if focus is in the iframe. Acceptable: most users will already be focused inside the app. Documented limitation.
- **Iframe focus**: the renderer runs in an iframe under the shell. The key event captures correctly when the iframe has focus. If focus is in the shell topbar, Cmd+K does nothing — fine.
- **cmdk + happy-dom flakiness** — covered by `waitFor` wraps. Manual smoke catches real-world correctness.
- **Page reset on close** — must reset `page` to `'list'` on every reopen, not on every render. Use the `onOpenChange` callback or a `useEffect` keyed on `open`.
- **Bundle**: ~6 KB; within deferred budget.

## Open questions

None blocking. Pre-decided:
- `cmdk` for the primitive.
- Wrapped in Radix Dialog for focus trap / Esc / overlay.
- Three internal pages (list, create-input, rename-input).
- Delete fires immediately; relies on undo toast.
- Cmd+K and Ctrl+K both bound.
- Rename / Delete hidden when no active board.

## Dependencies on prior work

- P4c.0: `Client.createBoard / renameBoard / deleteBoard`.
- P4c.1: `useCreateBoard`, `useRenameBoard`, `useDeleteBoard`, `<ActiveBoardProvider>`, generalized `stageDelete(fire, label)`, `errorToast` `ALREADY_EXISTS` copy.
- P4b.3: `@radix-ui/react-dialog` (already installed).
- P4b.1a: `errorToast` mechanism, sonner Toaster mount, uncontrolled-input + committedRef pattern.
