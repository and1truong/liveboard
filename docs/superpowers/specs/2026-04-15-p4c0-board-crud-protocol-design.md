# P4c.0 — Board CRUD Protocol Extension — Design

## Goal

Add `board.create`, `board.rename`, `board.delete` to the shared protocol, the `BackendAdapter` interface, the `LocalAdapter` implementation, and the `Client` surface. Add a `board.list.updated` event so subscribers (the sidebar in P4c.1) can refresh on cross-tab board-list changes. No renderer changes — this is the pure-protocol prerequisite for P4c.1.

**Shippable value:** Tests prove the new methods round-trip end-to-end. P4c.1 is unblocked.

## Scope

**In:**
- Three new `Request` variants in `web/shared/src/protocol.ts`.
- One new `Event` variant: `board.list.updated`.
- Three new methods on `BackendAdapter`.
- Implementations on `LocalAdapter`.
- Three new methods on `Client`.
- Routing in `Broker`.
- A `slugify` helper in `web/shared/src/util/slug.ts`.
- Tests for adapter logic, slug helper, and round-trip via in-memory pair.

**Out:**
- UI for board CRUD (P4c.1).
- Server-backed adapter (out of scope for P4 entirely; LocalAdapter is the only one today).
- Stable random IDs separate from name slugs (rejected: see Architecture).
- Granular create/rename/delete events (rejected: A only).
- Multi-tab atomic create-conflict handling (documented limitation).

## Architecture

Three new methods, one new event. `boardId` is name-derived (slug), so rename mints a new id and the response carries the new `BoardSummary`. The single `board.list.updated` event tells subscribers "your cached board list is stale" — they refetch.

```
Client.createBoard(name)
   │
   ▼ postMessage Request { method: 'board.create', params: { name } }
Broker → adapter.createBoard(name) → BoardSummary
   │
   ├─ Response { ok: true, data: BoardSummary }
   └─ Event { type: 'board.list.updated' } (broadcast)
```

`renameBoard` and `deleteBoard` follow the same shape. All three emit the event; only `createBoard` and `renameBoard` return data.

### Why slug-derived ids

The existing LocalAdapter already keys storage by id-equals-slug-equals-name. Going to stable random ids would require:
- Storage layout migration.
- Workspace metadata changes (id→name map).
- Renderer URL state changes (cache key by id, but human-readable name in URL).

Out of scope for P4c.0. Rename-changes-id is the cost; renderer subscribers re-subscribe using the BoardSummary returned by the rename call.

## Wire shapes

### Protocol additions (`web/shared/src/protocol.ts`)

```ts
export type Request =
  | ...existing
  | { id: string; kind: 'request'; method: 'board.create'; params: { name: string } }
  | { id: string; kind: 'request'; method: 'board.rename'; params: { boardId: string; newName: string } }
  | { id: string; kind: 'request'; method: 'board.delete'; params: { boardId: string } }

export type Event =
  | ...existing
  | { kind: 'event'; type: 'board.list.updated' }
```

Response data per method:

| Method | `data` on success |
|---|---|
| `board.create` | `BoardSummary` (the new board) |
| `board.rename` | `BoardSummary` (with new id) |
| `board.delete` | `null` |

`ErrorCode` already includes `INVALID`, `ALREADY_EXISTS`, `NOT_FOUND`, `INTERNAL` — no additions needed.

### Adapter interface additions (`web/shared/src/adapter.ts`)

```ts
export interface BackendAdapter {
  // ...existing...
  createBoard(name: string): Promise<BoardSummary>
  renameBoard(boardId: string, newName: string): Promise<BoardSummary>
  deleteBoard(boardId: string): Promise<void>
}
```

### Client additions (`web/shared/src/client.ts`)

```ts
async createBoard(name: string): Promise<BoardSummary>
async renameBoard(boardId: string, newName: string): Promise<BoardSummary>
async deleteBoard(boardId: string): Promise<void>
```

Each thin-wraps `request<BoardSummary | null>(method, params)`, throws `ProtocolError` on failure. Same shape as existing `mutateBoard`.

## `slugify` helper

`web/shared/src/util/slug.ts`:

```ts
export function slugify(name: string): string
```

Rules:
- Lowercase the input.
- Replace runs of whitespace with single dashes.
- Strip any character not in `[a-z0-9-]` (after the lowercase + whitespace pass).
- Trim leading/trailing dashes.
- Collapse multiple consecutive dashes to one.

Examples (drive the test):
- `"My Board"` → `"my-board"`
- `"Hello, World!"` → `"hello-world"`
- `"  spaces  "` → `"spaces"`
- `"!!!"` → `""` (caller treats empty as INVALID)
- `"Foo___Bar"` → `"foo-bar"` (underscores stripped, runs collapsed — actually `"foobar"` since underscores are stripped without leaving a separator; pin behavior in test)

The exact treatment of underscores and other separators is fixed by the test table — implementer follows the tests.

## `LocalAdapter` implementations

Source: `web/shared/src/adapters/local.ts`. The existing `seedIfEmpty` and storage helpers (`boardKey`, `workspaceKey`) are reused.

### `createBoard(name)`

1. Trim `name`. If empty → throw `ProtocolError('INVALID', 'name required')`.
2. `id = slugify(name)`. If empty → throw `ProtocolError('INVALID', 'name has no usable characters')`.
3. Read workspace; if `boardIds.includes(id)` → throw `ProtocolError('ALREADY_EXISTS', 'board exists')`.
4. Write a new `Board { name, version: 1, columns: [{ name: 'Todo', cards: [] }] }` under `boardKey(id)`.
5. Update workspace: append `id` to `boardIds`, write back.
6. Broadcast `board.list.updated` via the BroadcastChannel and emit it through any registered handler path used for events.
7. Return `BoardSummary { id, name, version: 1 }` (no icon).

### `renameBoard(boardId, newName)`

1. Trim `newName`. If empty → `INVALID`.
2. `newId = slugify(newName)`. If empty → `INVALID`.
3. Read source board; if missing → `NOT_FOUND`.
4. If `newId === boardId`: in-place name change (board content identical, board.name updated). Skip step 6.
5. Read workspace. If `newId !== boardId && boardIds.includes(newId)` → `ALREADY_EXISTS`.
6. Atomic move:
   - Read source board, rewrite under `boardKey(newId)` with `name = newName.trim()`.
   - Delete `boardKey(boardId)`.
   - Read settings under the source key (if any); rewrite under the new id; delete the source settings.
   - In workspace `boardIds`, replace `boardId` with `newId` (preserving order).
7. Broadcast `board.list.updated`.
8. Return `BoardSummary { id: newId, name: newName.trim(), version: <bumped> }`. Version increments by 1 to invalidate any optimistic caches still keyed off the old id.

### `deleteBoard(boardId)`

1. Read source board; if missing → `NOT_FOUND`.
2. Delete `boardKey(boardId)`.
3. Delete settings key for this board if present (no error on miss).
4. Read workspace, remove `boardId` from `boardIds`, write back.
5. Broadcast `board.list.updated`.
6. Return `void`.

## `Broker` routing

`web/shared/src/broker.ts` already routes by `req.method`. Add three switch cases:
```ts
case 'board.create':  return adapter.createBoard(req.params.name)
case 'board.rename':  return adapter.renameBoard(req.params.boardId, req.params.newName)
case 'board.delete':  return adapter.deleteBoard(req.params.boardId).then(() => null)
```

(Current Broker shape may differ slightly; follow the existing pattern.)

The Broker also needs to forward `board.list.updated` from the adapter. The current adapter→broker event hookup is via `subscribe(boardId, ...)` which is per-board. The list event is global. Two implementation choices for the adapter→broker bridge:

A. Add a second adapter event channel: `onBoardListUpdate(handler): Subscription`. Broker subscribes once at construction and forwards as event.
B. Reuse the existing per-board `BoardUpdateHandler` shape with a sentinel boardId like `'*'`. Hacky.

**Choice: A.** Adds one method to the BackendAdapter interface (`onBoardListUpdate`), pure-typed, no overloading.

`BackendAdapter` interface gains:
```ts
onBoardListUpdate(handler: () => void): Subscription
```

LocalAdapter implements it by tracking handlers in a Set, calling them whenever `board.list.updated` is broadcast (locally and via BroadcastChannel from another tab).

## Testing

All tests under `web/shared/src/`. Run via `bun test`.

### `util/slug.test.ts`

Table:
```
"My Board"        → "my-board"
"Hello, World!"   → "hello-world"
"  spaces  "      → "spaces"
"!!!"             → ""
"Foo___Bar"       → "foobar"
"a   b"           → "a-b"
"--leading"       → "leading"
"trailing--"      → "trailing"
"a--b"            → "a-b"
"MIXEDcase"       → "mixedcase"
```

### `adapters/local.create.test.ts`

- happy path: returns `BoardSummary` with slugified id, name preserved
- empty name → `INVALID`
- name slugifies to empty → `INVALID`
- collision with existing id → `ALREADY_EXISTS`
- subsequent `listBoards()` includes the new board
- subsequent `getBoard(id)` returns the new board with version 1 and a default Todo column
- `onBoardListUpdate` handler is invoked

### `adapters/local.rename.test.ts`

- happy path: returns BoardSummary with new id; old id no longer in list; new id is in list
- in-place rename (same slug, e.g. case-only change in name): id unchanged, name updated
- source missing → `NOT_FOUND`
- new id collides → `ALREADY_EXISTS`
- empty new name → `INVALID`
- settings for the old id (if seeded) move to the new id
- `onBoardListUpdate` handler invoked
- post-rename, `listBoards()` shows new entry, version bumped

### `adapters/local.delete.test.ts`

- happy path: returns void; `listBoards()` no longer includes the id; `getBoard(id)` throws `NOT_FOUND`
- source missing → `NOT_FOUND`
- settings (if seeded) cleaned up
- `onBoardListUpdate` handler invoked

### `client.boards.test.ts` (round-trip)

In-memory `Broker` + `LocalAdapter` + `MemoryStorage` + `createMemoryPair`. For each of `createBoard`, `renameBoard`, `deleteBoard`:
- Method round-trips successfully.
- An `error` thrown by the adapter surfaces as a `ProtocolError` with the right `code`.
- The `board.list.updated` event arrives at a subscriber registered via `client.on('board.list.updated', ...)`.

## Risks

- **Rename-changes-id**: documented in Architecture. P4c.1 must handle re-subscribing using the response's new id.
- **Multi-tab race on create**: tab A and tab B simultaneously create "Foo" → both succeed locally with the same id; second BroadcastChannel write wins. Not handling this in P4c.0; documented as a known LocalAdapter limitation. A future server-backed adapter handles it via real concurrency control.
- **Underscore / non-ASCII handling in `slugify`**: pinned by the test table. Non-ASCII falls outside `[a-z0-9-]` after lowercase, so it's stripped — board names with only emoji or CJK characters slugify to `""` and fail with `INVALID`. Document in spec.
- **`board.list.updated` storm**: every create/rename/delete fires the event. Subscribers refetch. For P4c.0 there's no UI subscriber; for P4c.1 the sidebar will be the only consumer, with a single in-flight refetch via TanStack Query. Acceptable.

## Open questions

None blocking. Pre-decided:
- Three methods, one event.
- boardId is name-slug-derived; rename mints new id; response carries new BoardSummary.
- `slugify` rules pinned by test table.
- Adapter exposes `onBoardListUpdate` for the global event channel (no `'*'` sentinel hack).

## Dependencies on prior work

- P3: existing `Request`/`Response`/`Event` infrastructure, `Client.request`, `Broker` switch, `LocalAdapter` storage primitives.
