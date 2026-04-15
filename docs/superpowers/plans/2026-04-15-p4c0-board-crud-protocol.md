# P4c.0 — Board CRUD Protocol Extension Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `board.create`, `board.rename`, `board.delete` methods (plus a `board.list.updated` event) end-to-end through `web/shared/*` — protocol, adapter interface, LocalAdapter, Broker routing, and Client surface — with full test coverage. No renderer changes.

**Architecture:** boardId is name-derived (slug). Rename mints a new id and the response carries the new `BoardSummary`. A new global event channel (`onBoardListUpdate`) on `BackendAdapter` lets the Broker forward `board.list.updated` events to renderers without overloading the per-board subscribe channel. A pure `slugify` helper centralizes id derivation.

**Tech Stack:** TypeScript, bun test, BroadcastChannel for cross-tab. No new deps.

**Spec:** `docs/superpowers/specs/2026-04-15-p4c0-board-crud-protocol-design.md`

**Conventions:**
- All work in `web/shared/src/`. Tests colocated.
- Commit prefixes: `feat(shared)`, `test(shared)`.
- Pre-existing `TS6196` in `web/shared/src/protocol.ts` is NOT a blocker.
- Use bun, never npx.

---

## File structure

**New:**
- `web/shared/src/util/slug.ts`
- `web/shared/src/util/slug.test.ts`
- `web/shared/src/adapters/local.create.test.ts`
- `web/shared/src/adapters/local.rename.test.ts`
- `web/shared/src/adapters/local.delete.test.ts`
- `web/shared/src/client.boards.test.ts`

**Modified:**
- `web/shared/src/protocol.ts` — three new `Request` variants, one new `Event` variant.
- `web/shared/src/adapter.ts` — `createBoard`, `renameBoard`, `deleteBoard`, `onBoardListUpdate` on `BackendAdapter`.
- `web/shared/src/adapters/local.ts` — implementations + `boardListHandlers` + cross-tab event broadcast.
- `web/shared/src/broker.ts` — three switch cases + global event subscription forwarding `board.list.updated`.
- `web/shared/src/client.ts` — `createBoard`, `renameBoard`, `deleteBoard` methods.

---

## Task 1: `slugify` helper

**Files:**
- Create: `web/shared/src/util/slug.ts`
- Create: `web/shared/src/util/slug.test.ts`

- [ ] **Step 1: Failing test first**

Create `web/shared/src/util/slug.test.ts`:
```ts
import { describe, expect, it } from 'bun:test'
import { slugify } from './slug.js'

describe('slugify', () => {
  const cases: Array<[string, string]> = [
    ['My Board', 'my-board'],
    ['Hello, World!', 'hello-world'],
    ['  spaces  ', 'spaces'],
    ['!!!', ''],
    ['Foo___Bar', 'foobar'],
    ['a   b', 'a-b'],
    ['--leading', 'leading'],
    ['trailing--', 'trailing'],
    ['a--b', 'a-b'],
    ['MIXEDcase', 'mixedcase'],
    ['', ''],
    ['日本語', ''],
  ]
  for (const [input, expected] of cases) {
    it(`${JSON.stringify(input)} → ${JSON.stringify(expected)}`, () => {
      expect(slugify(input)).toBe(expected)
    })
  }
})
```

- [ ] **Step 2: Run, expect fail (module missing)**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src/util/slug.test.ts
```

- [ ] **Step 3: Implement**

Create `web/shared/src/util/slug.ts`:
```ts
export function slugify(name: string): string {
  return name
    .toLowerCase()
    .replace(/\s+/g, '-')          // whitespace runs → single dash
    .replace(/[^a-z0-9-]/g, '')    // strip everything outside [a-z0-9-]
    .replace(/-+/g, '-')           // collapse dash runs
    .replace(/^-+|-+$/g, '')       // trim leading/trailing dashes
}
```

- [ ] **Step 4: Run, expect 12 pass**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src/util/slug.test.ts
```

- [ ] **Step 5: Commit**

```bash
git add web/shared/src/util/slug.ts web/shared/src/util/slug.test.ts
git commit -m "feat(shared): add slugify helper for board ids

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: Protocol additions

**Files:**
- Modify: `web/shared/src/protocol.ts`

- [ ] **Step 1: Extend `Request` and `Event` unions**

Read current `web/shared/src/protocol.ts`. In the `Request` union, after the existing `unsubscribe` entry, add:
```ts
  | { id: string; kind: 'request'; method: 'board.create'; params: { name: string } }
  | { id: string; kind: 'request'; method: 'board.rename'; params: { boardId: string; newName: string } }
  | { id: string; kind: 'request'; method: 'board.delete'; params: { boardId: string } }
```

In the `Event` union, after the existing entries, add:
```ts
  | { kind: 'event'; type: 'board.list.updated' }
```

- [ ] **Step 2: Typecheck**

```bash
cd /Users/htruong/code/htruong/liveboard && bun --cwd web/shared run typecheck 2>/dev/null || cd web/renderer/default && bun run typecheck
```
Expected: only the pre-existing TS6196 in protocol.ts. (The shared package has no separate typecheck script; the renderer's typecheck transitively checks shared. If your repo provides a shared typecheck script, prefer that.)

- [ ] **Step 3: Commit**

```bash
git add web/shared/src/protocol.ts
git commit -m "feat(shared): extend protocol with board.create/rename/delete + list.updated

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: `BackendAdapter` interface additions

**Files:**
- Modify: `web/shared/src/adapter.ts`

- [ ] **Step 1: Add four methods to `BackendAdapter`**

Read current `web/shared/src/adapter.ts`. Inside `interface BackendAdapter`, after the existing `subscribe` line, append:
```ts
  createBoard(name: string): Promise<BoardSummary>
  renameBoard(boardId: string, newName: string): Promise<BoardSummary>
  deleteBoard(boardId: string): Promise<void>
  onBoardListUpdate(handler: () => void): Subscription
```

- [ ] **Step 2: Typecheck**

```bash
cd web/renderer/default && bun run typecheck
```
Expected: errors in `web/shared/src/adapters/local.ts` ("Class 'LocalAdapter' incorrectly implements interface 'BackendAdapter'..." for the four missing methods). That's expected — we'll fix in Tasks 4–7. Leave the file uncommitted until then.

- [ ] **Step 3: Don't commit yet**

We commit `adapter.ts` together with the LocalAdapter implementations to keep the tree green per commit. Skip to Task 4.

---

## Task 4: `LocalAdapter.createBoard` + `onBoardListUpdate` infrastructure

**Files:**
- Modify: `web/shared/src/adapters/local.ts`
- Create: `web/shared/src/adapters/local.create.test.ts`

This task adds the global event channel infrastructure (`onBoardListUpdate`, `boardListHandlers`, `publishBoardListUpdate`, BroadcastChannel forwarding) AND the `createBoard` method that uses it.

- [ ] **Step 1: Failing test**

Create `web/shared/src/adapters/local.create.test.ts`:
```ts
import { describe, expect, it } from 'bun:test'
import { LocalAdapter } from './local.js'
import { MemoryStorage } from './local-storage-driver.js'
import { ProtocolError } from '../protocol.js'

describe('LocalAdapter.createBoard', () => {
  it('returns BoardSummary with slugified id', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const summary = await a.createBoard('My Board')
    expect(summary.id).toBe('my-board')
    expect(summary.name).toBe('My Board')
    expect(summary.version).toBe(1)
  })

  it('persists the board with a default Todo column', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    const board = await a.getBoard('foo')
    expect(board.name).toBe('Foo')
    expect(board.version).toBe(1)
    expect(board.columns?.[0]?.name).toBe('Todo')
    expect(board.columns?.[0]?.cards).toEqual([])
  })

  it('appends the new board to listBoards', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    const list = await a.listBoards()
    const ids = list.map((s) => s.id)
    expect(ids).toContain('welcome')
    expect(ids).toContain('foo')
  })

  it('rejects empty name as INVALID', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await expect(a.createBoard('   ')).rejects.toMatchObject({
      code: 'INVALID',
    })
  })

  it('rejects name that slugifies to empty as INVALID', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await expect(a.createBoard('!!!')).rejects.toMatchObject({
      code: 'INVALID',
    })
  })

  it('rejects collision as ALREADY_EXISTS', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    await expect(a.createBoard('Foo')).rejects.toMatchObject({
      code: 'ALREADY_EXISTS',
    })
  })

  it('invokes onBoardListUpdate handler', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    let calls = 0
    a.onBoardListUpdate(() => {
      calls++
    })
    await a.createBoard('Foo')
    expect(calls).toBe(1)
  })

  it('errors are ProtocolError instances', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    try {
      await a.createBoard('')
      throw new Error('should have thrown')
    } catch (e) {
      expect(e).toBeInstanceOf(ProtocolError)
    }
  })
})
```

- [ ] **Step 2: Run, expect fail**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src/adapters/local.create.test.ts
```

- [ ] **Step 3: Implement**

Read `web/shared/src/adapters/local.ts`. Make these edits:

(a) Add import for `slugify` near the top:
```ts
import { slugify } from '../util/slug.js'
```

(b) Inside the `LocalAdapter` class, after the existing `private readonly handlers = ...` line, add:
```ts
  private readonly boardListHandlers = new Set<() => void>()
```

(c) In the constructor's `this.channel.onmessage` body, extend it to also handle the new event. Replace the existing onmessage block with:
```ts
      this.channel.onmessage = (ev: MessageEvent) => {
        const data = ev.data as { type?: string; boardId?: string; version?: number }
        if (data?.type === 'board.updated' && data.boardId) {
          this.fanOut(data.boardId, data.version ?? 0)
        } else if (data?.type === 'board.list.updated') {
          this.fanOutBoardList()
        }
      }
```

(d) Add these methods to the class (before the closing brace):
```ts
  onBoardListUpdate(handler: () => void): Subscription {
    this.boardListHandlers.add(handler)
    return {
      close: () => {
        this.boardListHandlers.delete(handler)
      },
    }
  }

  private fanOutBoardList(): void {
    for (const h of this.boardListHandlers) h()
  }

  private publishBoardListUpdate(): void {
    this.fanOutBoardList()
    this.channel?.postMessage({ type: 'board.list.updated' })
  }

  async createBoard(name: string): Promise<BoardSummary> {
    const trimmed = name.trim()
    if (!trimmed) throw new ProtocolError('INVALID', 'name required')
    const id = slugify(trimmed)
    if (!id) throw new ProtocolError('INVALID', 'name has no usable characters')
    const ws = this.loadWorkspace()
    if (ws.boardIds.includes(id)) {
      throw new ProtocolError('ALREADY_EXISTS', `board ${id} exists`)
    }
    const board: Board = {
      name: trimmed,
      version: 1,
      columns: [{ name: 'Todo', cards: [] }],
    }
    this.storage.set(boardKey(id), JSON.stringify(board))
    ws.boardIds.push(id)
    this.storage.set(workspaceKey(), JSON.stringify(ws))
    this.publishBoardListUpdate()
    return { id, name: trimmed, version: 1 }
  }
```

- [ ] **Step 4: Run, expect 8 pass**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src/adapters/local.create.test.ts
```

- [ ] **Step 5: Verify typecheck (renameBoard / deleteBoard still missing — that's expected)**

```bash
cd web/renderer/default && bun run typecheck
```
Expected: errors only for `renameBoard` and `deleteBoard` not yet implemented. `createBoard` and `onBoardListUpdate` are now satisfied.

- [ ] **Step 6: Don't commit yet** — wait for Tasks 5 & 6 to make the file fully implement the interface, then commit all three together. (This avoids landing intermediate states where the tree has typecheck errors.)

---

## Task 5: `LocalAdapter.renameBoard`

**Files:**
- Modify: `web/shared/src/adapters/local.ts`
- Create: `web/shared/src/adapters/local.rename.test.ts`

- [ ] **Step 1: Failing test**

Create `web/shared/src/adapters/local.rename.test.ts`:
```ts
import { describe, expect, it } from 'bun:test'
import { LocalAdapter } from './local.js'
import { MemoryStorage } from './local-storage-driver.js'

describe('LocalAdapter.renameBoard', () => {
  it('moves board to new id and returns new BoardSummary', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    const summary = await a.renameBoard('foo', 'Bar')
    expect(summary.id).toBe('bar')
    expect(summary.name).toBe('Bar')
    expect(summary.version).toBeGreaterThanOrEqual(2)
    const list = (await a.listBoards()).map((s) => s.id)
    expect(list).toContain('bar')
    expect(list).not.toContain('foo')
  })

  it('preserves position in workspace boardIds', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    await a.createBoard('Baz')
    const beforeIds = (await a.listBoards()).map((s) => s.id)
    const fooIdx = beforeIds.indexOf('foo')
    await a.renameBoard('foo', 'Quux')
    const afterIds = (await a.listBoards()).map((s) => s.id)
    expect(afterIds[fooIdx]).toBe('quux')
  })

  it('in-place name change keeps id when slug unchanged', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('foo')
    const summary = await a.renameBoard('foo', 'FOO')
    expect(summary.id).toBe('foo')
    expect(summary.name).toBe('FOO')
  })

  it('rejects missing source as NOT_FOUND', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await expect(a.renameBoard('nope', 'X')).rejects.toMatchObject({ code: 'NOT_FOUND' })
  })

  it('rejects new id collision as ALREADY_EXISTS', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    await a.createBoard('Bar')
    await expect(a.renameBoard('foo', 'Bar')).rejects.toMatchObject({ code: 'ALREADY_EXISTS' })
  })

  it('rejects empty new name as INVALID', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    await expect(a.renameBoard('foo', '   ')).rejects.toMatchObject({ code: 'INVALID' })
  })

  it('invokes onBoardListUpdate handler', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    let calls = 0
    a.onBoardListUpdate(() => {
      calls++
    })
    await a.renameBoard('foo', 'Bar')
    expect(calls).toBe(1)
  })
})
```

- [ ] **Step 2: Run, expect fail**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src/adapters/local.rename.test.ts
```

- [ ] **Step 3: Implement**

Add the `renameBoard` method to the `LocalAdapter` class in `web/shared/src/adapters/local.ts` (alongside `createBoard`):
```ts
  async renameBoard(boardId: string, newName: string): Promise<BoardSummary> {
    const trimmed = newName.trim()
    if (!trimmed) throw new ProtocolError('INVALID', 'name required')
    const newId = slugify(trimmed)
    if (!newId) throw new ProtocolError('INVALID', 'name has no usable characters')
    const board = this.loadBoard(boardId) // throws NOT_FOUND if missing
    const ws = this.loadWorkspace()
    if (newId !== boardId && ws.boardIds.includes(newId)) {
      throw new ProtocolError('ALREADY_EXISTS', `board ${newId} exists`)
    }
    board.name = trimmed
    board.version = (board.version ?? 0) + 1
    if (newId === boardId) {
      // In-place name change, no key move.
      this.storage.set(boardKey(boardId), JSON.stringify(board))
    } else {
      this.storage.set(boardKey(newId), JSON.stringify(board))
      this.storage.delete(boardKey(boardId))
      const idx = ws.boardIds.indexOf(boardId)
      if (idx >= 0) ws.boardIds[idx] = newId
      this.storage.set(workspaceKey(), JSON.stringify(ws))
    }
    this.publishBoardListUpdate()
    return { id: newId, name: trimmed, version: board.version }
  }
```

Note: `LocalAdapter` doesn't currently move settings — settings are stored on the Board itself in `getSettings`/`putBoardSettings` (see existing impl: settings live in `board.settings`). So no separate settings key migration is needed. The board move carries the settings with it.

`StorageDriver.delete(key)` is assumed available. If the driver doesn't have it, check `web/shared/src/adapters/local-storage-driver.ts` and add it (next step covers).

- [ ] **Step 4: Verify `StorageDriver.delete` exists**

```bash
grep -n "delete(" web/shared/src/adapters/local-storage-driver.ts
```
If `delete` is not on the `StorageDriver` interface or `MemoryStorage`, add it:
```ts
// In the interface:
delete(key: string): void

// In MemoryStorage:
delete(key: string): void {
  this.data.delete(key)
}

// In LocalStorageDriver:
delete(key: string): void {
  this.storage.removeItem(key)
}
```
Re-run typecheck after adding.

- [ ] **Step 5: Run, expect 7 pass**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src/adapters/local.rename.test.ts
```

- [ ] **Step 6: Don't commit yet** — Task 6 finishes the interface.

---

## Task 6: `LocalAdapter.deleteBoard`

**Files:**
- Modify: `web/shared/src/adapters/local.ts`
- Create: `web/shared/src/adapters/local.delete.test.ts`

- [ ] **Step 1: Failing test**

Create `web/shared/src/adapters/local.delete.test.ts`:
```ts
import { describe, expect, it } from 'bun:test'
import { LocalAdapter } from './local.js'
import { MemoryStorage } from './local-storage-driver.js'

describe('LocalAdapter.deleteBoard', () => {
  it('removes the board from listBoards', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    await a.deleteBoard('foo')
    const ids = (await a.listBoards()).map((s) => s.id)
    expect(ids).not.toContain('foo')
  })

  it('subsequent getBoard throws NOT_FOUND', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    await a.deleteBoard('foo')
    await expect(a.getBoard('foo')).rejects.toMatchObject({ code: 'NOT_FOUND' })
  })

  it('rejects missing source as NOT_FOUND', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await expect(a.deleteBoard('nope')).rejects.toMatchObject({ code: 'NOT_FOUND' })
  })

  it('invokes onBoardListUpdate handler', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    let calls = 0
    a.onBoardListUpdate(() => {
      calls++
    })
    await a.deleteBoard('foo')
    expect(calls).toBe(1)
  })

  it('returns void', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    const result = await a.deleteBoard('foo')
    expect(result).toBeUndefined()
  })
})
```

- [ ] **Step 2: Run, expect fail**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src/adapters/local.delete.test.ts
```

- [ ] **Step 3: Implement**

Add `deleteBoard` to `LocalAdapter`:
```ts
  async deleteBoard(boardId: string): Promise<void> {
    this.loadBoard(boardId) // throws NOT_FOUND if missing
    this.storage.delete(boardKey(boardId))
    const ws = this.loadWorkspace()
    ws.boardIds = ws.boardIds.filter((x) => x !== boardId)
    this.storage.set(workspaceKey(), JSON.stringify(ws))
    this.publishBoardListUpdate()
  }
```

- [ ] **Step 4: Run, expect 5 pass + full typecheck clean**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src/adapters/ && cd web/renderer/default && bun run typecheck
```
Expected: only the pre-existing TS6196 in protocol.ts remains. LocalAdapter now fully implements the extended interface.

- [ ] **Step 5: Commit all three impls + interface change + storage delete (if added)**

```bash
cd /Users/htruong/code/htruong/liveboard
git add web/shared/src/adapter.ts web/shared/src/adapters/local.ts web/shared/src/adapters/local-storage-driver.ts \
        web/shared/src/adapters/local.create.test.ts \
        web/shared/src/adapters/local.rename.test.ts \
        web/shared/src/adapters/local.delete.test.ts
git commit -m "feat(shared): add board create/rename/delete to LocalAdapter

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 7: Broker routing + global event forwarding

**Files:**
- Modify: `web/shared/src/broker.ts`

- [ ] **Step 1: Add three switch cases + global subscription on construction**

Read `web/shared/src/broker.ts`. Make these edits:

(a) In the constructor, after the `this.transport.onMessage(...)` line, append a global event subscription:
```ts
    this.boardListSub = this.adapter.onBoardListUpdate(() => {
      this.transport.send({ kind: 'event', type: 'board.list.updated' })
    })
```

(b) Add `private boardListSub: Subscription` field next to `subs`:
```ts
  private boardListSub: Subscription
```

(c) Inside `handle`, add three new cases (after `unsubscribe`):
```ts
      case 'board.create':
        return this.adapter.createBoard(req.params.name)
      case 'board.rename':
        return this.adapter.renameBoard(req.params.boardId, req.params.newName)
      case 'board.delete':
        await this.adapter.deleteBoard(req.params.boardId)
        return null
```

(d) In `close()`, also close the global subscription:
```ts
  close(): void {
    this.boardListSub.close()
    for (const s of this.subs.values()) s.close()
    this.subs.clear()
    this.transport.close()
  }
```

- [ ] **Step 2: Typecheck**

```bash
cd web/renderer/default && bun run typecheck
```
Expected: only pre-existing TS6196.

- [ ] **Step 3: Run existing broker tests**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src/broker.test.ts
```
Expected: still pass. (The broker now subscribes to `onBoardListUpdate` on construction, which means existing tests that mock `BackendAdapter` may need to provide a stub. If a test uses an inline mock that doesn't have `onBoardListUpdate`, fix it minimally by adding a stub method that returns `{ close() {} }`.)

- [ ] **Step 4: Commit**

```bash
git add web/shared/src/broker.ts
# include any test file you had to update with stub onBoardListUpdate
git commit -m "feat(shared): route board.create/rename/delete + forward list.updated event

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 8: `Client` method additions + round-trip tests

**Files:**
- Modify: `web/shared/src/client.ts`
- Create: `web/shared/src/client.boards.test.ts`

- [ ] **Step 1: Failing tests first**

Create `web/shared/src/client.boards.test.ts`:
```ts
import { describe, expect, it } from 'bun:test'
import { Broker } from './broker.js'
import { Client } from './client.js'
import { LocalAdapter } from './adapters/local.js'
import { MemoryStorage } from './adapters/local-storage-driver.js'
import { createMemoryPair } from './transport.js'

async function setup(): Promise<{ client: Client }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  return { client }
}

describe('Client board CRUD', () => {
  it('createBoard round-trips and returns BoardSummary', async () => {
    const { client } = await setup()
    const summary = await client.createBoard('My Board')
    expect(summary.id).toBe('my-board')
    expect(summary.name).toBe('My Board')
  })

  it('createBoard surfaces ALREADY_EXISTS as ProtocolError', async () => {
    const { client } = await setup()
    await client.createBoard('Foo')
    try {
      await client.createBoard('Foo')
      throw new Error('should have thrown')
    } catch (e) {
      expect((e as { code: string }).code).toBe('ALREADY_EXISTS')
    }
  })

  it('renameBoard returns new BoardSummary with new id', async () => {
    const { client } = await setup()
    await client.createBoard('Foo')
    const renamed = await client.renameBoard('foo', 'Bar')
    expect(renamed.id).toBe('bar')
    expect(renamed.name).toBe('Bar')
  })

  it('renameBoard surfaces NOT_FOUND', async () => {
    const { client } = await setup()
    try {
      await client.renameBoard('nope', 'X')
      throw new Error('should have thrown')
    } catch (e) {
      expect((e as { code: string }).code).toBe('NOT_FOUND')
    }
  })

  it('deleteBoard removes from listBoards', async () => {
    const { client } = await setup()
    await client.createBoard('Foo')
    await client.deleteBoard('foo')
    const list = await client.listBoards()
    expect(list.map((s) => s.id)).not.toContain('foo')
  })

  it('emits board.list.updated event to subscribers', async () => {
    const { client } = await setup()
    let count = 0
    client.on('board.list.updated', () => {
      count++
    })
    await client.createBoard('Foo')
    await client.renameBoard('foo', 'Bar')
    await client.deleteBoard('bar')
    // Allow the event microtasks to flush.
    await new Promise((r) => setTimeout(r, 10))
    expect(count).toBeGreaterThanOrEqual(3)
  })
})
```

- [ ] **Step 2: Run, expect fail**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src/client.boards.test.ts
```

- [ ] **Step 3: Add Client methods**

Read `web/shared/src/client.ts`. After the `unsubscribe` method (the last existing method), add:
```ts
  createBoard(name: string): Promise<BoardSummary> {
    return this.request({ kind: 'request', method: 'board.create', params: { name } })
  }
  renameBoard(boardId: string, newName: string): Promise<BoardSummary> {
    return this.request({
      kind: 'request',
      method: 'board.rename',
      params: { boardId, newName },
    })
  }
  deleteBoard(boardId: string): Promise<void> {
    return this.request({ kind: 'request', method: 'board.delete', params: { boardId } })
  }
```

- [ ] **Step 4: Verify event handler type covers `board.list.updated`**

The existing `EventType = ProtoEvent['type']` derivation in client.ts already picks up the new `'board.list.updated'` variant from the protocol union — no change needed. Confirm by adding `client.on('board.list.updated', ...)` in the test (already present).

- [ ] **Step 5: Run, expect 6 pass**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src/client.boards.test.ts && cd web/renderer/default && bun run typecheck
```
Expected: 6 pass; only pre-existing TS6196.

- [ ] **Step 6: Commit**

```bash
cd /Users/htruong/code/htruong/liveboard
git add web/shared/src/client.ts web/shared/src/client.boards.test.ts
git commit -m "feat(shared): add Client.createBoard / renameBoard / deleteBoard

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 9: Full suite check

**Files:** none.

- [ ] **Step 1: Run shared + renderer suite**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src && cd web/renderer/default && bun test && bun run typecheck
```
Expected: all green; only pre-existing TS6196 in typecheck.

- [ ] **Step 2: Verify Go embed unaffected**

```bash
cd /Users/htruong/code/htruong/liveboard && go test ./internal/api/ -run TestShellRoute
```
Expected: 3 tests pass. (Renderer bundle unchanged — P4c.0 doesn't touch the renderer.)

- [ ] **Step 3: No commit** — verification only.

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| Three new `Request` variants | 2 |
| New `Event` variant `board.list.updated` | 2 |
| `BackendAdapter.createBoard`/`renameBoard`/`deleteBoard` | 3, 4, 5, 6 |
| `BackendAdapter.onBoardListUpdate` | 3, 4 |
| `slugify` helper + tests | 1 |
| LocalAdapter create with INVALID/ALREADY_EXISTS | 4 |
| LocalAdapter rename with INVALID/NOT_FOUND/ALREADY_EXISTS + in-place | 5 |
| LocalAdapter delete with NOT_FOUND | 6 |
| BroadcastChannel cross-tab event for list | 4 (publish + onmessage handling) |
| Broker routes new methods | 7 |
| Broker forwards `board.list.updated` | 7 |
| `Client.createBoard`/`renameBoard`/`deleteBoard` | 8 |
| Round-trip test for all three | 8 |
| Event delivery to client subscribers | 8 |

## Notes for implementer

1. **`adapter.ts` change in Task 3 is intentionally uncommitted** — committing it alone would land a tree where `LocalAdapter` doesn't satisfy the interface and typecheck is red. Tasks 4–6 land it together with the implementations.
2. **`StorageDriver.delete`** may not exist yet; Task 5 step 4 adds it if missing. Check first; don't duplicate.
3. **`renameBoard` carries settings automatically** because settings live on the Board (`board.settings`) and the rewrite copies the whole Board. Don't add a separate settings-key migration — it would be a no-op.
4. **`Client.on('board.list.updated', ...)` works automatically** — `EventType = ProtoEvent['type']` derives the union from the protocol; new variants are picked up by type inference.
5. **No commit amending** — forward-only commits.
