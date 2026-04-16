# P3 — Shell + LocalAdapter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a headless TypeScript shell + postMessage broker + localStorage-backed `LocalAdapter` so a stub renderer iframe can drive the board model end-to-end without any backend. Sets up the protocol and adapter boundary for P4 (real React renderer) and P5 (RestAdapter).

**Architecture:** Four pieces. (1) `BackendAdapter` interface + `LocalAdapter` implementation using the shared `boardOps.ts` from P2. (2) Transport-agnostic postMessage **Broker** (shell side) and **Client** (iframe side) with JSON-RPC-style request/response + server-push events. (3) A shell host HTML page that wires adapter → broker → iframe. (4) A stub renderer iframe that exercises every protocol method, used as the integration harness. Go serves everything at `/app/*` behind an env flag `LIVEBOARD_APP_SHELL=1`.

**Tech Stack:** TypeScript 5.x (existing `web/shared/`), bun (package manager + test runner + bundler), `//go:embed` for static assets. No new runtime dependencies.

**Spec:** `docs/superpowers/specs/2026-04-15-iframe-renderer-architecture-design.md` §postMessage protocol, §Backend Adapter interface, §Shell, §Iframe renderer (boot sequence only — the renderer itself is P4).
**Plan of plans:** `docs/superpowers/plans/2026-04-15-iframe-renderer-plan-of-plans.md`

**Out of scope for P3:**
- Real React renderer (P4).
- `RestAdapter` (P5).
- Multi-board cross-tab conflict handling beyond BroadcastChannel fan-out.
- Auth. Shell runs same-origin as Go server.
- Bundle size budget enforcement (P4 concern).
- Playwright-style end-to-end tests. The stub iframe is the integration harness; a human loads it in a browser and verifies.

**Conventions:**
- Module paths: TS lives under `web/shared/src/`. Shell host HTML + entrypoint under `web/shell/`.
- Shell build output at `web/shell/dist/` — gitignored, built by `make shell`, embedded by Go.
- Commit prefixes: `feat(shell)`, `feat(adapter)`, `test(shell)`, `chore(build)`, `docs`.
- Protocol version: fixed at `1` for P3. No negotiation beyond a single version.
- Origin validation: same-origin only. Shell validates `event.origin === window.location.origin`. Iframe validates `event.origin === <parent origin injected at build>`.
- Canonical error codes map: reuses P2's set (`NOT_FOUND`, `OUT_OF_RANGE`, `INVALID`, `ALREADY_EXISTS`, `INTERNAL`) plus `VERSION_CONFLICT` (checked inside adapter, not `Apply`) and `PROTOCOL_UNSUPPORTED`.

---

## File structure

New files:
- `web/shared/src/protocol.ts` — message type definitions + error codes
- `web/shared/src/adapter.ts` — `BackendAdapter` interface, `Subscription`, `BoardSummary`, `WorkspaceInfo`, `ResolvedSettings`
- `web/shared/src/adapters/local-storage-driver.ts` — thin abstraction over `localStorage` for testability
- `web/shared/src/adapters/local.ts` — `LocalAdapter` implementation
- `web/shared/src/adapters/local-seed.ts` — welcome board JSON
- `web/shared/src/transport.ts` — `Transport` interface (send/onMessage)
- `web/shared/src/broker.ts` — shell-side `Broker` (request router + event publisher)
- `web/shared/src/client.ts` — iframe-side `Client` (request correlation + event subscription)
- `web/shared/src/transports/post-message.ts` — browser postMessage transports (shell + iframe)
- Test files for each of the above: `*.test.ts` colocated
- `web/shell/index.html` — shell host page
- `web/shell/src/main.ts` — shell entrypoint (wires adapter → broker → iframe)
- `web/shell/build.ts` — `bun build` driver for shell + stub bundles
- `web/shell/stub/index.html` — stub renderer host
- `web/shell/stub/src/main.ts` — stub entrypoint: hello → exercise every method → log results
- `web/shell/dist/.gitignore` — ignore build output

Modified:
- `Makefile` — add `make shell` target; extend `make all` or default to include it
- `.github/workflows/ci.yml` — run `make shell` before Go tests
- `internal/api/server.go` — mount `/app/*` route behind `LIVEBOARD_APP_SHELL=1` env flag
- `web/shell/embed.go` — new file declaring `//go:embed dist`
- `README.md` — link to new shell entry point
- `docs/parity.md` — add note that adapter tests live in `web/shared/src/adapters/`

---

## Task 1: Protocol message types

**Files:**
- Create: `web/shared/src/protocol.ts`
- Create: `web/shared/src/protocol.test.ts`

Define the wire format that every layer below speaks. Types only — no runtime behavior.

- [ ] **Step 1: Write protocol.ts**

```ts
// Wire format for iframe ↔ shell postMessage communication.
// Tagged unions — discriminator `kind`. Requests and responses correlate by `id`.

import type { MutationOp, Board, BoardSettings } from './types.js'

export const PROTOCOL_VERSION = 1 as const

// Iframe → Shell
export type Request =
  | { id: string; kind: 'request'; method: 'board.list'; params?: undefined }
  | { id: string; kind: 'request'; method: 'board.get'; params: { boardId: string } }
  | {
      id: string
      kind: 'request'
      method: 'board.mutate'
      params: { boardId: string; clientVersion: number; op: MutationOp }
    }
  | { id: string; kind: 'request'; method: 'workspace.info'; params?: undefined }
  | { id: string; kind: 'request'; method: 'settings.get'; params: { boardId: string } }
  | {
      id: string
      kind: 'request'
      method: 'settings.put'
      params: { boardId: string; patch: Partial<BoardSettings> }
    }
  | { id: string; kind: 'request'; method: 'subscribe'; params: { boardId: string } }
  | { id: string; kind: 'request'; method: 'unsubscribe'; params: { boardId: string } }

export type ErrorCode =
  | 'NOT_FOUND'
  | 'OUT_OF_RANGE'
  | 'INVALID'
  | 'ALREADY_EXISTS'
  | 'INTERNAL'
  | 'VERSION_CONFLICT'
  | 'PROTOCOL_UNSUPPORTED'

// Shell → Iframe (response)
export type Response =
  | { id: string; kind: 'response'; ok: true; data: unknown }
  | { id: string; kind: 'response'; ok: false; error: { code: ErrorCode; message: string } }

// Shell → Iframe (push)
export type Event =
  | { kind: 'event'; type: 'board.updated'; data: { boardId: string; version: number } }
  | { kind: 'event'; type: 'settings.updated'; data: { boardId: string } }
  | { kind: 'event'; type: 'connection.status'; data: { online: boolean } }

// Handshake — iframe → shell, first message.
export interface Hello {
  kind: 'hello'
  protocols: number[]
  rendererId: string
  rendererVersion: string
}

// Shell → iframe — handshake reply.
export interface Welcome {
  kind: 'welcome'
  protocol: number
  shellVersion: string
  capabilities: string[]
}

export interface HandshakeError {
  kind: 'welcome-error'
  error: { code: 'PROTOCOL_UNSUPPORTED'; minSupported: number; maxSupported: number }
}

export type Message = Request | Response | Event | Hello | Welcome | HandshakeError

export class ProtocolError extends Error {
  constructor(public code: ErrorCode, message: string) {
    super(message)
    this.name = 'ProtocolError'
  }
}

export type { Board, BoardSettings, MutationOp } from './types.js'
```

- [ ] **Step 2: Write protocol.test.ts**

```ts
import { describe, expect, it } from 'bun:test'
import { PROTOCOL_VERSION, ProtocolError } from './protocol.js'
import type { Request, Response, Event } from './protocol.js'

describe('protocol', () => {
  it('exports a stable version integer', () => {
    expect(PROTOCOL_VERSION).toBe(1)
  })

  it('ProtocolError carries a code', () => {
    const e = new ProtocolError('NOT_FOUND', 'no board')
    expect(e.code).toBe('NOT_FOUND')
    expect(e.message).toBe('no board')
  })

  it('Request discriminator narrows via method', () => {
    const r: Request = { id: 'x', kind: 'request', method: 'board.list' }
    expect(r.method).toBe('board.list')
  })

  it('Response ok=true carries data; ok=false carries error', () => {
    const ok: Response = { id: 'x', kind: 'response', ok: true, data: null }
    const err: Response = {
      id: 'x',
      kind: 'response',
      ok: false,
      error: { code: 'INTERNAL', message: 'boom' },
    }
    expect(ok.ok).toBe(true)
    expect(err.ok).toBe(false)
  })

  it('Event types are distinguishable by type field', () => {
    const e: Event = { kind: 'event', type: 'board.updated', data: { boardId: 'x', version: 1 } }
    expect(e.type).toBe('board.updated')
  })
})
```

- [ ] **Step 3: Run tests and typecheck**

```bash
cd web/shared && bun test src/protocol.test.ts && bun run typecheck
```
Expected: 4 pass, typecheck green.

- [ ] **Step 4: Commit**

```bash
git add web/shared/src/protocol.ts web/shared/src/protocol.test.ts
git commit -m "feat(shell): add postMessage protocol types"
```

---

## Task 2: BackendAdapter interface + supporting types

**Files:**
- Create: `web/shared/src/adapter.ts`

Defines the interface that both LocalAdapter (P3) and RestAdapter (P5) implement. No implementation yet.

- [ ] **Step 1: Write adapter.ts**

```ts
import type { Board, BoardSettings, MutationOp } from './types.js'

export interface BoardSummary {
  id: string
  name: string
  icon?: string
  version: number
}

export interface WorkspaceInfo {
  name: string
  boardCount: number
}

// Mirrors internal/web.ResolvedSettings — concrete (non-nullable) values.
export interface ResolvedSettings {
  show_checkbox: boolean
  card_position: string
  expand_columns: boolean
  view_mode: string
  card_display_mode: string
  week_start: string
}

export interface Subscription {
  close(): void
}

export type BoardUpdateHandler = (payload: { boardId: string; version: number }) => void

export interface BackendAdapter {
  listBoards(): Promise<BoardSummary[]>
  getWorkspaceInfo(): Promise<WorkspaceInfo>
  getBoard(boardId: string): Promise<Board>
  mutateBoard(boardId: string, clientVersion: number, op: MutationOp): Promise<Board>
  getSettings(boardId: string): Promise<ResolvedSettings>
  putBoardSettings(boardId: string, patch: Partial<BoardSettings>): Promise<void>
  subscribe(boardId: string, onUpdate: BoardUpdateHandler): Subscription
}
```

- [ ] **Step 2: Typecheck**

```bash
cd web/shared && bun run typecheck
```
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add web/shared/src/adapter.ts
git commit -m "feat(adapter): add BackendAdapter interface"
```

---

## Task 3: LocalAdapter — storage driver abstraction

**Files:**
- Create: `web/shared/src/adapters/local-storage-driver.ts`
- Create: `web/shared/src/adapters/local-storage-driver.test.ts`

Abstract `localStorage` behind a tiny interface so adapter tests don't touch the real DOM storage.

- [ ] **Step 1: Write local-storage-driver.ts**

```ts
// Minimal key/value abstraction so LocalAdapter tests can inject an in-memory store.
export interface StorageDriver {
  get(key: string): string | null
  set(key: string, value: string): void
  remove(key: string): void
  keys(prefix: string): string[]
}

export class MemoryStorage implements StorageDriver {
  private readonly map = new Map<string, string>()

  get(key: string): string | null {
    return this.map.has(key) ? this.map.get(key)! : null
  }

  set(key: string, value: string): void {
    this.map.set(key, value)
  }

  remove(key: string): void {
    this.map.delete(key)
  }

  keys(prefix: string): string[] {
    const out: string[] = []
    for (const k of this.map.keys()) {
      if (k.startsWith(prefix)) out.push(k)
    }
    return out
  }
}

export class BrowserStorage implements StorageDriver {
  constructor(private readonly storage: Storage = globalThis.localStorage) {}

  get(key: string): string | null {
    return this.storage.getItem(key)
  }

  set(key: string, value: string): void {
    this.storage.setItem(key, value)
  }

  remove(key: string): void {
    this.storage.removeItem(key)
  }

  keys(prefix: string): string[] {
    const out: string[] = []
    for (let i = 0; i < this.storage.length; i++) {
      const k = this.storage.key(i)
      if (k !== null && k.startsWith(prefix)) out.push(k)
    }
    return out
  }
}
```

- [ ] **Step 2: Write test**

```ts
import { describe, expect, it } from 'bun:test'
import { MemoryStorage } from './local-storage-driver.js'

describe('MemoryStorage', () => {
  it('stores and retrieves values', () => {
    const s = new MemoryStorage()
    s.set('a', '1')
    expect(s.get('a')).toBe('1')
  })

  it('returns null for missing keys', () => {
    expect(new MemoryStorage().get('x')).toBeNull()
  })

  it('lists keys by prefix', () => {
    const s = new MemoryStorage()
    s.set('lb:board:a', '1')
    s.set('lb:board:b', '2')
    s.set('other:c', '3')
    expect(s.keys('lb:board:').sort()).toEqual(['lb:board:a', 'lb:board:b'])
  })

  it('remove deletes the key', () => {
    const s = new MemoryStorage()
    s.set('a', '1')
    s.remove('a')
    expect(s.get('a')).toBeNull()
  })
})
```

- [ ] **Step 3: Run**

```bash
cd web/shared && bun test src/adapters/local-storage-driver.test.ts
```
Expected: 4 pass.

- [ ] **Step 4: Commit**

```bash
git add web/shared/src/adapters/local-storage-driver.ts web/shared/src/adapters/local-storage-driver.test.ts
git commit -m "feat(adapter): add StorageDriver abstraction"
```

---

## Task 4: LocalAdapter — seed data + listBoards/getBoard

**Files:**
- Create: `web/shared/src/adapters/local-seed.ts`
- Create: `web/shared/src/adapters/local.ts`
- Create: `web/shared/src/adapters/local.test.ts`

First slice: read-only operations. Seeds a welcome board on first load.

- [ ] **Step 1: Write local-seed.ts**

```ts
import type { Board } from '../types.js'

export const WELCOME_BOARD: Board = {
  version: 1,
  name: 'Welcome',
  description: 'This is your demo board. Data stays in this browser.',
  icon: '👋',
  tags: ['demo'],
  columns: [
    {
      name: 'Todo',
      cards: [
        { title: 'Try dragging this card to Done' },
        { title: 'Double-click the board title to rename it' },
      ],
    },
    { name: 'Doing', cards: [{ title: 'Build something awesome' }] },
    { name: 'Done', cards: [{ title: 'Read the intro' }] },
  ],
}

export const WORKSPACE_NAME = 'Demo'
```

- [ ] **Step 2: Write local.ts (partial — read operations only)**

```ts
import type {
  BackendAdapter,
  BoardSummary,
  BoardUpdateHandler,
  ResolvedSettings,
  Subscription,
  WorkspaceInfo,
} from '../adapter.js'
import type { Board, BoardSettings, MutationOp } from '../types.js'
import { ProtocolError } from '../protocol.js'
import type { StorageDriver } from './local-storage-driver.js'
import { WELCOME_BOARD, WORKSPACE_NAME } from './local-seed.js'

const KEY_PREFIX = 'liveboard:v1:'
const boardKey = (id: string): string => `${KEY_PREFIX}board:${id}`
const workspaceKey = (): string => `${KEY_PREFIX}workspace`

interface StoredWorkspace {
  name: string
  boardIds: string[]
}

export class LocalAdapter implements BackendAdapter {
  constructor(private readonly storage: StorageDriver) {
    this.seedIfEmpty()
  }

  private seedIfEmpty(): void {
    if (this.storage.get(workspaceKey()) !== null) return
    const ws: StoredWorkspace = { name: WORKSPACE_NAME, boardIds: ['welcome'] }
    this.storage.set(workspaceKey(), JSON.stringify(ws))
    this.storage.set(boardKey('welcome'), JSON.stringify(WELCOME_BOARD))
  }

  private loadWorkspace(): StoredWorkspace {
    const raw = this.storage.get(workspaceKey())
    if (raw === null) throw new ProtocolError('INTERNAL', 'workspace missing')
    return JSON.parse(raw) as StoredWorkspace
  }

  private loadBoard(id: string): Board {
    const raw = this.storage.get(boardKey(id))
    if (raw === null) throw new ProtocolError('NOT_FOUND', `board ${id}`)
    return JSON.parse(raw) as Board
  }

  async listBoards(): Promise<BoardSummary[]> {
    const ws = this.loadWorkspace()
    return ws.boardIds.map((id) => {
      const b = this.loadBoard(id)
      return {
        id,
        name: b.name ?? id,
        icon: b.icon,
        version: b.version ?? 0,
      }
    })
  }

  async getWorkspaceInfo(): Promise<WorkspaceInfo> {
    const ws = this.loadWorkspace()
    return { name: ws.name, boardCount: ws.boardIds.length }
  }

  async getBoard(boardId: string): Promise<Board> {
    return this.loadBoard(boardId)
  }

  async mutateBoard(_boardId: string, _clientVersion: number, _op: MutationOp): Promise<Board> {
    throw new ProtocolError('INTERNAL', 'mutateBoard not yet implemented')
  }

  async getSettings(_boardId: string): Promise<ResolvedSettings> {
    throw new ProtocolError('INTERNAL', 'getSettings not yet implemented')
  }

  async putBoardSettings(_boardId: string, _patch: Partial<BoardSettings>): Promise<void> {
    throw new ProtocolError('INTERNAL', 'putBoardSettings not yet implemented')
  }

  subscribe(_boardId: string, _onUpdate: BoardUpdateHandler): Subscription {
    return { close: () => {} }
  }
}
```

- [ ] **Step 3: Write local.test.ts**

```ts
import { describe, expect, it } from 'bun:test'
import { LocalAdapter } from './local.js'
import { MemoryStorage } from './local-storage-driver.js'

describe('LocalAdapter seed + reads', () => {
  it('seeds workspace on first construction', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const ws = await a.getWorkspaceInfo()
    expect(ws.name).toBe('Demo')
    expect(ws.boardCount).toBe(1)
  })

  it('listBoards returns the welcome board summary', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const list = await a.listBoards()
    expect(list).toHaveLength(1)
    expect(list[0]?.id).toBe('welcome')
    expect(list[0]?.name).toBe('Welcome')
  })

  it('getBoard returns full board', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const b = await a.getBoard('welcome')
    expect(b.name).toBe('Welcome')
    expect(b.columns?.length).toBe(3)
  })

  it('getBoard on missing id throws NOT_FOUND', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await expect(a.getBoard('nope')).rejects.toMatchObject({ code: 'NOT_FOUND' })
  })

  it('second construction on same storage does not reseed', async () => {
    const storage = new MemoryStorage()
    new LocalAdapter(storage)
    // Mutate welcome name; a re-seed would overwrite it.
    const raw = storage.get('liveboard:v1:board:welcome')!
    const b = JSON.parse(raw)
    b.name = 'Changed'
    storage.set('liveboard:v1:board:welcome', JSON.stringify(b))
    const a2 = new LocalAdapter(storage)
    expect((await a2.getBoard('welcome')).name).toBe('Changed')
  })
})
```

- [ ] **Step 4: Run**

```bash
cd web/shared && bun test src/adapters/local.test.ts && bun run typecheck
```
Expected: 5 pass, typecheck green.

- [ ] **Step 5: Commit**

```bash
git add web/shared/src/adapters/local-seed.ts web/shared/src/adapters/local.ts web/shared/src/adapters/local.test.ts
git commit -m "feat(adapter): add LocalAdapter seed and read operations"
```

---

## Task 5: LocalAdapter — mutateBoard with version check

**Files:**
- Modify: `web/shared/src/adapters/local.ts`
- Modify: `web/shared/src/adapters/local.test.ts`

Apply the mutation via the shared `boardOps.ts` from P2, bump the version, persist. Mirror the Go `MutateBoard` semantics: if `clientVersion >= 0` and mismatches, throw `VERSION_CONFLICT`; `clientVersion < 0` bypasses the check.

- [ ] **Step 1: Replace mutateBoard in local.ts**

Find the `async mutateBoard` stub and replace with:

```ts
  async mutateBoard(boardId: string, clientVersion: number, op: MutationOp): Promise<Board> {
    const board = this.loadBoard(boardId)
    const currentVersion = board.version ?? 0
    if (clientVersion >= 0 && clientVersion !== currentVersion) {
      throw new ProtocolError('VERSION_CONFLICT', `expected version ${clientVersion}, have ${currentVersion}`)
    }
    try {
      const next = applyOp(board, op)
      next.version = currentVersion + 1
      this.storage.set(boardKey(boardId), JSON.stringify(next))
      this.publishUpdate(boardId, next.version)
      return next
    } catch (e) {
      if (e instanceof OpError) throw new ProtocolError(e.code, e.message)
      throw e
    }
  }

  private publishUpdate(_boardId: string, _version: number): void {
    // BroadcastChannel wiring lands in Task 6.
  }
```

Add imports at the top of `local.ts`:
```ts
import { applyOp } from '../boardOps.js'
import { OpError } from '../types.js'
```

- [ ] **Step 2: Add mutateBoard tests**

Append to `local.test.ts`:

```ts
import type { MutationOp } from '../types.js'

describe('LocalAdapter mutateBoard', () => {
  it('applies op, bumps version, persists', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const op: MutationOp = { type: 'add_card', column: 'Todo', title: 'x' }
    const next = await a.mutateBoard('welcome', 1, op)
    expect(next.version).toBe(2)
    const again = await a.getBoard('welcome')
    expect(again.version).toBe(2)
    expect(again.columns?.[0]?.cards.some((c) => c.title === 'x')).toBe(true)
  })

  it('throws VERSION_CONFLICT on stale clientVersion', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const op: MutationOp = { type: 'add_card', column: 'Todo', title: 'x' }
    await expect(a.mutateBoard('welcome', 42, op)).rejects.toMatchObject({
      code: 'VERSION_CONFLICT',
    })
  })

  it('clientVersion < 0 bypasses conflict check', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const op: MutationOp = { type: 'add_card', column: 'Todo', title: 'x' }
    const r = await a.mutateBoard('welcome', -1, op)
    expect(r.version).toBe(2)
  })

  it('propagates applyOp errors with mapped code', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const op: MutationOp = { type: 'add_card', column: 'Missing', title: 'x' }
    await expect(a.mutateBoard('welcome', 1, op)).rejects.toMatchObject({ code: 'NOT_FOUND' })
  })
})
```

- [ ] **Step 3: Run**

```bash
cd web/shared && bun test src/adapters/local.test.ts
```
Expected: all 9 pass.

- [ ] **Step 4: Commit**

```bash
git add web/shared/src/adapters/local.ts web/shared/src/adapters/local.test.ts
git commit -m "feat(adapter): implement LocalAdapter mutateBoard with version check"
```

---

## Task 6: LocalAdapter — BroadcastChannel subscribe + settings

**Files:**
- Modify: `web/shared/src/adapters/local.ts`
- Modify: `web/shared/src/adapters/local.test.ts`

Wire BroadcastChannel so same-browser multi-tab sync works, and implement `getSettings` / `putBoardSettings`. Resolved settings use a fixed default (the demo doesn't need global settings).

- [ ] **Step 1: Extend local.ts**

Add channel setup inside the class. Full updated class body should look like:

```ts
export class LocalAdapter implements BackendAdapter {
  private readonly channel: BroadcastChannel | null
  private readonly handlers = new Map<string, Set<BoardUpdateHandler>>()

  constructor(private readonly storage: StorageDriver, channelName = 'liveboard') {
    this.seedIfEmpty()
    this.channel =
      typeof BroadcastChannel !== 'undefined' ? new BroadcastChannel(channelName) : null
    if (this.channel) {
      this.channel.onmessage = (ev: MessageEvent) => {
        const data = ev.data as { type?: string; boardId?: string; version?: number }
        if (data?.type === 'board.updated' && data.boardId) {
          this.fanOut(data.boardId, data.version ?? 0)
        }
      }
    }
  }

  subscribe(boardId: string, onUpdate: BoardUpdateHandler): Subscription {
    let set = this.handlers.get(boardId)
    if (!set) {
      set = new Set()
      this.handlers.set(boardId, set)
    }
    set.add(onUpdate)
    return {
      close: () => {
        this.handlers.get(boardId)?.delete(onUpdate)
      },
    }
  }

  private fanOut(boardId: string, version: number): void {
    const set = this.handlers.get(boardId)
    if (!set) return
    for (const h of set) h({ boardId, version })
  }

  private publishUpdate(boardId: string, version: number): void {
    this.fanOut(boardId, version)
    this.channel?.postMessage({ type: 'board.updated', boardId, version })
  }

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

  async putBoardSettings(boardId: string, patch: Partial<BoardSettings>): Promise<void> {
    const board = this.loadBoard(boardId)
    board.settings = { ...(board.settings ?? {}), ...patch }
    board.version = (board.version ?? 0) + 1
    this.storage.set(boardKey(boardId), JSON.stringify(board))
    this.publishUpdate(boardId, board.version)
  }
```

(Keep all other methods; this replaces `subscribe`, `publishUpdate`, `getSettings`, and `putBoardSettings`.)

- [ ] **Step 2: Add tests**

Append:

```ts
describe('LocalAdapter subscribe', () => {
  it('fires handler on mutateBoard', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const seen: Array<{ boardId: string; version: number }> = []
    a.subscribe('welcome', (p) => seen.push(p))
    await a.mutateBoard('welcome', 1, { type: 'add_card', column: 'Todo', title: 'x' })
    expect(seen).toEqual([{ boardId: 'welcome', version: 2 }])
  })

  it('close() stops delivery', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const seen: number[] = []
    const sub = a.subscribe('welcome', (p) => seen.push(p.version))
    sub.close()
    await a.mutateBoard('welcome', 1, { type: 'add_card', column: 'Todo', title: 'x' })
    expect(seen).toHaveLength(0)
  })
})

describe('LocalAdapter settings', () => {
  it('getSettings returns defaults for an existing board', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const s = await a.getSettings('welcome')
    expect(s.view_mode).toBe('board')
  })

  it('getSettings on missing board throws NOT_FOUND', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await expect(a.getSettings('nope')).rejects.toMatchObject({ code: 'NOT_FOUND' })
  })

  it('putBoardSettings merges patch and bumps version', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.putBoardSettings('welcome', { card_display_mode: 'compact' })
    const b = await a.getBoard('welcome')
    expect(b.version).toBe(2)
    expect(b.settings?.card_display_mode).toBe('compact')
  })
})
```

- [ ] **Step 3: Run**

```bash
cd web/shared && bun test src/adapters/local.test.ts && bun run typecheck
```
Expected: 14 pass, typecheck green.

- [ ] **Step 4: Commit**

```bash
git add web/shared/src/adapters/local.ts web/shared/src/adapters/local.test.ts
git commit -m "feat(adapter): add subscribe, settings, BroadcastChannel fan-out"
```

---

## Task 7: Transport abstraction + in-memory transport

**Files:**
- Create: `web/shared/src/transport.ts`
- Create: `web/shared/src/transport.test.ts`

Abstract postMessage behind `Transport` so Broker/Client are testable with a pure in-memory channel.

- [ ] **Step 1: Write transport.ts**

```ts
import type { Message } from './protocol.js'

export type MessageHandler = (msg: Message) => void

export interface Transport {
  send(msg: Message): void
  onMessage(handler: MessageHandler): void
  close(): void
}

// Two transports wired together in-memory. Useful for tests.
export function createMemoryPair(): [Transport, Transport] {
  const handlersA: MessageHandler[] = []
  const handlersB: MessageHandler[] = []
  let closed = false

  const a: Transport = {
    send(msg) {
      if (closed) return
      queueMicrotask(() => {
        for (const h of handlersB) h(msg)
      })
    },
    onMessage(h) {
      handlersA.push(h)
    },
    close() {
      closed = true
    },
  }

  const b: Transport = {
    send(msg) {
      if (closed) return
      queueMicrotask(() => {
        for (const h of handlersA) h(msg)
      })
    },
    onMessage(h) {
      handlersB.push(h)
    },
    close() {
      closed = true
    },
  }

  return [a, b]
}
```

- [ ] **Step 2: Write transport.test.ts**

```ts
import { describe, expect, it } from 'bun:test'
import { createMemoryPair } from './transport.js'
import type { Message } from './protocol.js'

describe('createMemoryPair', () => {
  it('delivers from a to b', async () => {
    const [a, b] = createMemoryPair()
    const seen: Message[] = []
    b.onMessage((m) => seen.push(m))
    a.send({ id: '1', kind: 'request', method: 'board.list' })
    await Promise.resolve()
    expect(seen).toHaveLength(1)
    expect((seen[0] as { method?: string }).method).toBe('board.list')
  })

  it('delivers both directions independently', async () => {
    const [a, b] = createMemoryPair()
    const seenA: Message[] = []
    const seenB: Message[] = []
    a.onMessage((m) => seenA.push(m))
    b.onMessage((m) => seenB.push(m))
    a.send({ id: '1', kind: 'request', method: 'board.list' })
    b.send({ id: '1', kind: 'response', ok: true, data: [] })
    await Promise.resolve()
    expect(seenB).toHaveLength(1)
    expect(seenA).toHaveLength(1)
  })

  it('close() stops further delivery', async () => {
    const [a, b] = createMemoryPair()
    const seen: Message[] = []
    b.onMessage((m) => seen.push(m))
    a.close()
    a.send({ id: '1', kind: 'request', method: 'board.list' })
    await Promise.resolve()
    expect(seen).toHaveLength(0)
  })
})
```

- [ ] **Step 3: Run**

```bash
cd web/shared && bun test src/transport.test.ts && bun run typecheck
```
Expected: 3 pass, typecheck green.

- [ ] **Step 4: Commit**

```bash
git add web/shared/src/transport.ts web/shared/src/transport.test.ts
git commit -m "feat(shell): add Transport abstraction with memory pair for tests"
```

---

## Task 8: Broker — routes requests to adapter, fans out events

**Files:**
- Create: `web/shared/src/broker.ts`
- Create: `web/shared/src/broker.test.ts`

Shell-side. Receives `Request`s, dispatches to the adapter, sends `Response`s. On `subscribe`, registers a handler on the adapter that pushes `Event`s back through the transport. Also handles `hello` and replies with `welcome` or `welcome-error`.

- [ ] **Step 1: Write broker.ts**

```ts
import type { BackendAdapter, Subscription } from './adapter.js'
import type { Message, Request, Response } from './protocol.js'
import { PROTOCOL_VERSION } from './protocol.js'
import type { Transport } from './transport.js'

export interface BrokerOptions {
  shellVersion: string
  capabilities?: string[]
}

export class Broker {
  private readonly subs = new Map<string, Subscription>()

  constructor(
    private readonly transport: Transport,
    private readonly adapter: BackendAdapter,
    private readonly opts: BrokerOptions,
  ) {
    this.transport.onMessage((m) => {
      void this.route(m)
    })
  }

  private async route(msg: Message): Promise<void> {
    if (msg.kind === 'hello') {
      if (!msg.protocols.includes(PROTOCOL_VERSION)) {
        this.transport.send({
          kind: 'welcome-error',
          error: {
            code: 'PROTOCOL_UNSUPPORTED',
            minSupported: PROTOCOL_VERSION,
            maxSupported: PROTOCOL_VERSION,
          },
        })
        return
      }
      this.transport.send({
        kind: 'welcome',
        protocol: PROTOCOL_VERSION,
        shellVersion: this.opts.shellVersion,
        capabilities: this.opts.capabilities ?? ['local-storage', 'realtime'],
      })
      return
    }
    if (msg.kind !== 'request') return

    try {
      const data = await this.handle(msg)
      const resp: Response = { id: msg.id, kind: 'response', ok: true, data }
      this.transport.send(resp)
    } catch (e) {
      const code =
        e && typeof e === 'object' && 'code' in e && typeof (e as { code?: unknown }).code === 'string'
          ? (e as { code: string }).code
          : 'INTERNAL'
      const message = e instanceof Error ? e.message : String(e)
      this.transport.send({
        id: msg.id,
        kind: 'response',
        ok: false,
        error: { code: code as Response extends { error: { code: infer C } } ? C : never, message },
      })
    }
  }

  private async handle(req: Request): Promise<unknown> {
    switch (req.method) {
      case 'board.list':
        return this.adapter.listBoards()
      case 'board.get':
        return this.adapter.getBoard(req.params.boardId)
      case 'board.mutate':
        return this.adapter.mutateBoard(
          req.params.boardId,
          req.params.clientVersion,
          req.params.op,
        )
      case 'workspace.info':
        return this.adapter.getWorkspaceInfo()
      case 'settings.get':
        return this.adapter.getSettings(req.params.boardId)
      case 'settings.put':
        await this.adapter.putBoardSettings(req.params.boardId, req.params.patch)
        return null
      case 'subscribe': {
        const { boardId } = req.params
        this.subs.get(boardId)?.close()
        const sub = this.adapter.subscribe(boardId, ({ version }) => {
          this.transport.send({
            kind: 'event',
            type: 'board.updated',
            data: { boardId, version },
          })
        })
        this.subs.set(boardId, sub)
        return null
      }
      case 'unsubscribe':
        this.subs.get(req.params.boardId)?.close()
        this.subs.delete(req.params.boardId)
        return null
    }
  }

  close(): void {
    for (const s of this.subs.values()) s.close()
    this.subs.clear()
    this.transport.close()
  }
}
```

- [ ] **Step 2: Write broker.test.ts**

```ts
import { describe, expect, it } from 'bun:test'
import { Broker } from './broker.js'
import { LocalAdapter } from './adapters/local.js'
import { MemoryStorage } from './adapters/local-storage-driver.js'
import { createMemoryPair } from './transport.js'
import type { Message } from './protocol.js'

function collect(t: { onMessage: (h: (m: Message) => void) => void }): Message[] {
  const out: Message[] = []
  t.onMessage((m) => out.push(m))
  return out
}

async function flush(): Promise<void> {
  await new Promise((r) => queueMicrotask(() => r(null)))
  await new Promise((r) => queueMicrotask(() => r(null)))
  await new Promise((r) => queueMicrotask(() => r(null)))
}

describe('Broker handshake', () => {
  it('replies with welcome on matching protocol', async () => {
    const [iframe, shell] = createMemoryPair()
    const adapter = new LocalAdapter(new MemoryStorage())
    new Broker(shell, adapter, { shellVersion: '0.0.0' })
    const seen = collect(iframe)
    iframe.send({ kind: 'hello', protocols: [1], rendererId: 'stub', rendererVersion: '0' })
    await flush()
    expect(seen[0]?.kind).toBe('welcome')
  })

  it('replies with welcome-error on unsupported protocol', async () => {
    const [iframe, shell] = createMemoryPair()
    new Broker(shell, new LocalAdapter(new MemoryStorage()), { shellVersion: '0.0.0' })
    const seen = collect(iframe)
    iframe.send({ kind: 'hello', protocols: [99], rendererId: 'x', rendererVersion: '0' })
    await flush()
    expect(seen[0]?.kind).toBe('welcome-error')
  })
})

describe('Broker requests', () => {
  it('routes board.list to adapter', async () => {
    const [iframe, shell] = createMemoryPair()
    new Broker(shell, new LocalAdapter(new MemoryStorage()), { shellVersion: '0.0.0' })
    const seen = collect(iframe)
    iframe.send({ id: 'r1', kind: 'request', method: 'board.list' })
    await flush()
    const resp = seen[0] as { id: string; ok: boolean; data: unknown }
    expect(resp.id).toBe('r1')
    expect(resp.ok).toBe(true)
    expect(Array.isArray(resp.data)).toBe(true)
  })

  it('maps adapter errors to response.error.code', async () => {
    const [iframe, shell] = createMemoryPair()
    new Broker(shell, new LocalAdapter(new MemoryStorage()), { shellVersion: '0.0.0' })
    const seen = collect(iframe)
    iframe.send({ id: 'r2', kind: 'request', method: 'board.get', params: { boardId: 'nope' } })
    await flush()
    const resp = seen[0] as { ok: boolean; error: { code: string } }
    expect(resp.ok).toBe(false)
    expect(resp.error.code).toBe('NOT_FOUND')
  })

  it('subscribe pushes board.updated events after a mutation', async () => {
    const [iframe, shell] = createMemoryPair()
    const adapter = new LocalAdapter(new MemoryStorage())
    new Broker(shell, adapter, { shellVersion: '0.0.0' })
    const seen = collect(iframe)
    iframe.send({ id: 's', kind: 'request', method: 'subscribe', params: { boardId: 'welcome' } })
    await flush()
    await adapter.mutateBoard('welcome', 1, { type: 'add_card', column: 'Todo', title: 'x' })
    await flush()
    const ev = seen.find((m) => m.kind === 'event') as
      | { type: string; data: { boardId: string; version: number } }
      | undefined
    expect(ev?.type).toBe('board.updated')
    expect(ev?.data.boardId).toBe('welcome')
    expect(ev?.data.version).toBe(2)
  })
})
```

- [ ] **Step 3: Run**

```bash
cd web/shared && bun test src/broker.test.ts && bun run typecheck
```
Expected: 5 pass, typecheck green.

- [ ] **Step 4: Commit**

```bash
git add web/shared/src/broker.ts web/shared/src/broker.test.ts
git commit -m "feat(shell): add Broker that routes requests and forwards events"
```

---

## Task 9: Client — iframe-side request/response + event subscription

**Files:**
- Create: `web/shared/src/client.ts`
- Create: `web/shared/src/client.test.ts`

Iframe-side. Sends `hello`, resolves promises for each `Request`, and forwards `Event`s to handler callbacks.

- [ ] **Step 1: Write client.ts**

```ts
import type { Board, BoardSettings } from './types.js'
import type { BoardSummary, ResolvedSettings, WorkspaceInfo } from './adapter.js'
import type { MutationOp } from './types.js'
import type { Event as ProtoEvent, Message, Request, Welcome } from './protocol.js'
import { ProtocolError, PROTOCOL_VERSION } from './protocol.js'
import type { Transport } from './transport.js'

type EventType = ProtoEvent['type']
type EventHandler<T extends EventType> = (
  data: Extract<ProtoEvent, { type: T }>['data'],
) => void

interface Pending {
  resolve: (data: unknown) => void
  reject: (err: Error) => void
}

export interface ClientOptions {
  rendererId: string
  rendererVersion: string
}

export class Client {
  private readonly pending = new Map<string, Pending>()
  private readonly handlers = new Map<EventType, Set<EventHandler<EventType>>>()
  private nextId = 1
  private welcome: Welcome | null = null
  private welcomePromise: Promise<Welcome>

  constructor(private readonly transport: Transport, opts: ClientOptions) {
    this.transport.onMessage((m) => this.handle(m))
    this.welcomePromise = new Promise((resolve, reject) => {
      const timer = setTimeout(() => reject(new ProtocolError('INTERNAL', 'welcome timeout')), 5000)
      const check = (): void => {
        if (this.welcome) {
          clearTimeout(timer)
          resolve(this.welcome)
        }
      }
      this.welcomeResolvers = { check, reject: (e) => { clearTimeout(timer); reject(e) } }
    })
    this.transport.send({
      kind: 'hello',
      protocols: [PROTOCOL_VERSION],
      rendererId: opts.rendererId,
      rendererVersion: opts.rendererVersion,
    })
  }

  private welcomeResolvers!: { check: () => void; reject: (e: Error) => void }

  ready(): Promise<Welcome> {
    return this.welcomePromise
  }

  private handle(msg: Message): void {
    switch (msg.kind) {
      case 'welcome':
        this.welcome = msg
        this.welcomeResolvers.check()
        return
      case 'welcome-error':
        this.welcomeResolvers.reject(
          new ProtocolError('PROTOCOL_UNSUPPORTED', `server supports ${msg.error.minSupported}..${msg.error.maxSupported}`),
        )
        return
      case 'response': {
        const p = this.pending.get(msg.id)
        if (!p) return
        this.pending.delete(msg.id)
        if (msg.ok) p.resolve(msg.data)
        else p.reject(new ProtocolError(msg.error.code, msg.error.message))
        return
      }
      case 'event': {
        const set = this.handlers.get(msg.type) as Set<EventHandler<typeof msg.type>> | undefined
        if (!set) return
        for (const h of set) h(msg.data)
        return
      }
    }
  }

  on<T extends EventType>(type: T, handler: EventHandler<T>): () => void {
    let set = this.handlers.get(type) as Set<EventHandler<T>> | undefined
    if (!set) {
      set = new Set()
      this.handlers.set(type, set as unknown as Set<EventHandler<EventType>>)
    }
    set.add(handler)
    return () => set!.delete(handler)
  }

  private request<T>(req: Omit<Request, 'id'>): Promise<T> {
    const id = `r${this.nextId++}`
    return new Promise<T>((resolve, reject) => {
      this.pending.set(id, { resolve: resolve as (d: unknown) => void, reject })
      this.transport.send({ ...req, id } as Request)
    })
  }

  listBoards(): Promise<BoardSummary[]> {
    return this.request({ kind: 'request', method: 'board.list' })
  }
  getBoard(boardId: string): Promise<Board> {
    return this.request({ kind: 'request', method: 'board.get', params: { boardId } })
  }
  mutateBoard(boardId: string, clientVersion: number, op: MutationOp): Promise<Board> {
    return this.request({
      kind: 'request',
      method: 'board.mutate',
      params: { boardId, clientVersion, op },
    })
  }
  workspaceInfo(): Promise<WorkspaceInfo> {
    return this.request({ kind: 'request', method: 'workspace.info' })
  }
  getSettings(boardId: string): Promise<ResolvedSettings> {
    return this.request({ kind: 'request', method: 'settings.get', params: { boardId } })
  }
  putBoardSettings(boardId: string, patch: Partial<BoardSettings>): Promise<void> {
    return this.request({
      kind: 'request',
      method: 'settings.put',
      params: { boardId, patch },
    })
  }
  subscribe(boardId: string): Promise<void> {
    return this.request({ kind: 'request', method: 'subscribe', params: { boardId } })
  }
  unsubscribe(boardId: string): Promise<void> {
    return this.request({ kind: 'request', method: 'unsubscribe', params: { boardId } })
  }
}
```

- [ ] **Step 2: Write client.test.ts**

```ts
import { describe, expect, it } from 'bun:test'
import { Broker } from './broker.js'
import { Client } from './client.js'
import { LocalAdapter } from './adapters/local.js'
import { MemoryStorage } from './adapters/local-storage-driver.js'
import { createMemoryPair } from './transport.js'

function setup(): { client: Client; broker: Broker; adapter: LocalAdapter } {
  const [iframe, shell] = createMemoryPair()
  const adapter = new LocalAdapter(new MemoryStorage())
  const broker = new Broker(shell, adapter, { shellVersion: '0.0.0' })
  const client = new Client(iframe, { rendererId: 'test', rendererVersion: '0' })
  return { client, broker, adapter }
}

describe('Client', () => {
  it('ready() resolves after welcome', async () => {
    const { client } = setup()
    const w = await client.ready()
    expect(w.protocol).toBe(1)
  })

  it('listBoards round-trips through broker + adapter', async () => {
    const { client } = setup()
    await client.ready()
    const list = await client.listBoards()
    expect(list).toHaveLength(1)
    expect(list[0]?.id).toBe('welcome')
  })

  it('mutateBoard returns a new board', async () => {
    const { client } = setup()
    await client.ready()
    const b = await client.mutateBoard('welcome', 1, { type: 'add_card', column: 'Todo', title: 'x' })
    expect(b.version).toBe(2)
  })

  it('server errors surface as ProtocolError', async () => {
    const { client } = setup()
    await client.ready()
    await expect(client.getBoard('nope')).rejects.toMatchObject({ code: 'NOT_FOUND' })
  })

  it('subscribe + on("board.updated") delivers events after a mutation', async () => {
    const { client, adapter } = setup()
    await client.ready()
    const seen: number[] = []
    client.on('board.updated', (d) => seen.push(d.version))
    await client.subscribe('welcome')
    await adapter.mutateBoard('welcome', 1, { type: 'add_card', column: 'Todo', title: 'x' })
    // Wait for event microtasks to drain.
    await new Promise((r) => setTimeout(r, 5))
    expect(seen).toEqual([2])
  })
})
```

- [ ] **Step 3: Run**

```bash
cd web/shared && bun test src/client.test.ts && bun run typecheck
```
Expected: 5 pass, typecheck green.

- [ ] **Step 4: Commit**

```bash
git add web/shared/src/client.ts web/shared/src/client.test.ts
git commit -m "feat(shell): add iframe-side Client with handshake and events"
```

---

## Task 10: postMessage transports (browser DOM bindings)

**Files:**
- Create: `web/shared/src/transports/post-message.ts`
- Create: `web/shared/src/transports/post-message.test.ts`

Browser-specific transports. Wrap `window.postMessage` / `MessageEvent`. Testable via node's built-in `MessageChannel` (bun supports it natively).

- [ ] **Step 1: Write transports/post-message.ts**

```ts
import type { Message } from '../protocol.js'
import type { MessageHandler, Transport } from '../transport.js'

// Transport used by the shell to talk to a specific iframe.
export function shellTransport(iframe: HTMLIFrameElement, expectedOrigin: string): Transport {
  const handlers: MessageHandler[] = []
  const listener = (ev: MessageEvent): void => {
    if (ev.source !== iframe.contentWindow) return
    if (ev.origin !== expectedOrigin) return
    for (const h of handlers) h(ev.data as Message)
  }
  window.addEventListener('message', listener)
  return {
    send(msg) {
      iframe.contentWindow?.postMessage(msg, expectedOrigin)
    },
    onMessage(h) {
      handlers.push(h)
    },
    close() {
      window.removeEventListener('message', listener)
    },
  }
}

// Transport used inside the iframe to talk back to the parent shell.
export function iframeTransport(allowedParentOrigin: string): Transport {
  const handlers: MessageHandler[] = []
  const listener = (ev: MessageEvent): void => {
    if (ev.source !== window.parent) return
    if (ev.origin !== allowedParentOrigin) return
    for (const h of handlers) h(ev.data as Message)
  }
  window.addEventListener('message', listener)
  return {
    send(msg) {
      window.parent.postMessage(msg, allowedParentOrigin)
    },
    onMessage(h) {
      handlers.push(h)
    },
    close() {
      window.removeEventListener('message', listener)
    },
  }
}

// MessagePort-based transport — useful for tests, also usable with real
// MessageChannels if we ever need a hidden channel instead of window.postMessage.
export function messagePortTransport(port: MessagePort): Transport {
  const handlers: MessageHandler[] = []
  port.onmessage = (ev: MessageEvent) => {
    for (const h of handlers) h(ev.data as Message)
  }
  port.start()
  return {
    send(msg) {
      port.postMessage(msg)
    },
    onMessage(h) {
      handlers.push(h)
    },
    close() {
      port.close()
    },
  }
}
```

- [ ] **Step 2: Write test using MessageChannel**

```ts
import { describe, expect, it } from 'bun:test'
import { messagePortTransport } from './post-message.js'

describe('messagePortTransport', () => {
  it('pairs with another port through a MessageChannel', async () => {
    const channel = new MessageChannel()
    const a = messagePortTransport(channel.port1)
    const b = messagePortTransport(channel.port2)
    const seen: unknown[] = []
    b.onMessage((m) => seen.push(m))
    a.send({ id: '1', kind: 'request', method: 'board.list' })
    await new Promise((r) => setTimeout(r, 5))
    expect(seen).toHaveLength(1)
    expect((seen[0] as { method?: string }).method).toBe('board.list')
  })
})
```

The DOM-coupled `shellTransport` / `iframeTransport` are not unit-tested — they're thin wrappers around `window.addEventListener('message', ...)` that get exercised by the stub renderer in Task 13 (browser smoke).

- [ ] **Step 3: Run**

```bash
cd web/shared && bun test src/transports/post-message.test.ts && bun run typecheck
```
Expected: 1 pass, typecheck green.

If typecheck fails on `window`/`HTMLIFrameElement` references, add `"dom"` to `tsconfig.json` `lib`:

```json
"lib": ["ES2022", "DOM"]
```

- [ ] **Step 4: Commit**

```bash
git add web/shared/tsconfig.json web/shared/src/transports/post-message.ts web/shared/src/transports/post-message.test.ts
git commit -m "feat(shell): add browser postMessage transports + MessagePort test"
```

---

## Task 11: Shell host HTML + entrypoint

**Files:**
- Create: `web/shell/index.html`
- Create: `web/shell/src/main.ts`

The shell is the outer page served at `/app/`. It instantiates `LocalAdapter`, creates the stub iframe (for P3), and wires them together with a `Broker`.

- [ ] **Step 1: Write index.html**

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>LiveBoard Shell</title>
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <style>
    body { margin: 0; font-family: system-ui, sans-serif; }
    #renderer { width: 100vw; height: 100vh; border: 0; display: block; }
  </style>
</head>
<body>
  <iframe id="renderer" src="/app/renderer-stub/" title="LiveBoard renderer"></iframe>
  <script type="module" src="/app/main.js"></script>
</body>
</html>
```

- [ ] **Step 2: Write main.ts**

```ts
import { Broker } from '../../shared/src/broker.js'
import { LocalAdapter } from '../../shared/src/adapters/local.js'
import { BrowserStorage } from '../../shared/src/adapters/local-storage-driver.js'
import { shellTransport } from '../../shared/src/transports/post-message.js'

const SHELL_VERSION = '0.0.1'

function bootstrap(): void {
  const iframe = document.getElementById('renderer') as HTMLIFrameElement | null
  if (!iframe) throw new Error('renderer iframe not found')

  const adapter = new LocalAdapter(new BrowserStorage())
  const transport = shellTransport(iframe, window.location.origin)
  const broker = new Broker(transport, adapter, { shellVersion: SHELL_VERSION })

  window.addEventListener('beforeunload', () => broker.close())
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', bootstrap)
} else {
  bootstrap()
}
```

- [ ] **Step 3: Typecheck only (no unit tests — pure wiring)**

```bash
cd web/shared && bun run typecheck
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add web/shell/index.html web/shell/src/main.ts
git commit -m "feat(shell): add host HTML and entrypoint"
```

---

## Task 12: Stub renderer iframe (integration harness)

**Files:**
- Create: `web/shell/stub/index.html`
- Create: `web/shell/stub/src/main.ts`

A minimal iframe app that exercises every Client method and logs the results visibly. When you load `/app/` in a browser, this is what you see. If it reports green across the board, P3 is working.

- [ ] **Step 1: Write stub/index.html**

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>LiveBoard stub renderer</title>
  <style>
    body { margin: 0; padding: 1rem; font-family: ui-monospace, monospace; }
    .ok { color: #0a7d28; }
    .fail { color: #b00020; }
    .line { padding: 2px 0; }
  </style>
</head>
<body>
  <h1>Stub renderer — P3 integration harness</h1>
  <p>Each line below is a protocol round-trip. All should be <span class="ok">OK</span>.</p>
  <div id="log"></div>
  <script type="module" src="/app/renderer-stub/main.js"></script>
</body>
</html>
```

- [ ] **Step 2: Write stub/src/main.ts**

```ts
import { Client } from '../../../shared/src/client.js'
import { iframeTransport } from '../../../shared/src/transports/post-message.js'

const logEl = document.getElementById('log')!

function line(label: string, ok: boolean, detail = ''): void {
  const div = document.createElement('div')
  div.className = 'line'
  div.innerHTML = `<span class="${ok ? 'ok' : 'fail'}">${ok ? 'OK  ' : 'FAIL'}</span> — ${label} ${detail}`
  logEl.appendChild(div)
}

async function run(): Promise<void> {
  const transport = iframeTransport(window.location.origin)
  const client = new Client(transport, { rendererId: 'stub', rendererVersion: '0.0.1' })

  try {
    const w = await client.ready()
    line('handshake', true, `protocol=${w.protocol} caps=[${w.capabilities.join(',')}]`)
  } catch (e) {
    line('handshake', false, String(e))
    return
  }

  try {
    const list = await client.listBoards()
    line('board.list', list.length > 0, `${list.length} boards`)
  } catch (e) {
    line('board.list', false, String(e))
  }

  try {
    const ws = await client.workspaceInfo()
    line('workspace.info', true, ws.name)
  } catch (e) {
    line('workspace.info', false, String(e))
  }

  try {
    const b = await client.getBoard('welcome')
    line('board.get', (b.columns?.length ?? 0) > 0, `name=${b.name} v=${b.version}`)
  } catch (e) {
    line('board.get', false, String(e))
  }

  try {
    const s = await client.getSettings('welcome')
    line('settings.get', true, s.view_mode)
  } catch (e) {
    line('settings.get', false, String(e))
  }

  try {
    await client.putBoardSettings('welcome', { card_display_mode: 'compact' })
    line('settings.put', true)
  } catch (e) {
    line('settings.put', false, String(e))
  }

  // Subscribe then mutate — expect to observe the event before the response
  // (or at least not long after).
  let observedVersion = -1
  client.on('board.updated', (d) => {
    observedVersion = d.version
  })
  try {
    await client.subscribe('welcome')
    line('subscribe', true)
  } catch (e) {
    line('subscribe', false, String(e))
  }

  try {
    const before = await client.getBoard('welcome')
    const after = await client.mutateBoard(
      'welcome',
      before.version ?? 0,
      { type: 'add_card', column: 'Todo', title: 'stub inserted' },
    )
    line('board.mutate', (after.version ?? 0) > (before.version ?? 0), `v=${after.version}`)
  } catch (e) {
    line('board.mutate', false, String(e))
  }

  await new Promise((r) => setTimeout(r, 50))
  line('event: board.updated received', observedVersion > 0, `v=${observedVersion}`)

  // VERSION_CONFLICT path.
  try {
    await client.mutateBoard('welcome', 0, { type: 'add_card', column: 'Todo', title: 'stale' })
    line('board.mutate stale→error', false, 'expected rejection')
  } catch (e) {
    const code = (e as { code?: string }).code
    line('board.mutate stale→error', code === 'VERSION_CONFLICT', `code=${code}`)
  }

  try {
    await client.unsubscribe('welcome')
    line('unsubscribe', true)
  } catch (e) {
    line('unsubscribe', false, String(e))
  }
}

void run()
```

- [ ] **Step 3: Typecheck**

```bash
cd web/shared && bun run typecheck
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add web/shell/stub/index.html web/shell/stub/src/main.ts
git commit -m "feat(shell): add stub renderer iframe integration harness"
```

---

## Task 13: bun build script + Makefile target

**Files:**
- Create: `web/shell/build.ts`
- Modify: `Makefile`
- Create: `web/shell/dist/.gitignore`

Build the two TS entrypoints (shell main and stub main) to plain JS for the browser. Bun bundles natively — no bundler config needed.

- [ ] **Step 1: Write build.ts**

```ts
// Bundles the shell and stub renderer entrypoints for the browser.
// Run via: bun run web/shell/build.ts

import { mkdir, copyFile, rm } from 'node:fs/promises'
import { join } from 'node:path'

const root = import.meta.dir
const dist = join(root, 'dist')
const stubDist = join(dist, 'renderer-stub')

await rm(dist, { recursive: true, force: true })
await mkdir(stubDist, { recursive: true })

const results = await Bun.build({
  entrypoints: [join(root, 'src/main.ts'), join(root, 'stub/src/main.ts')],
  outdir: dist,
  target: 'browser',
  format: 'esm',
  naming: {
    entry: '[dir]/[name].[ext]',
  },
  minify: false,
  sourcemap: 'linked',
})

if (!results.success) {
  console.error('build failed:')
  for (const log of results.logs) console.error(log)
  process.exit(1)
}

// Bun puts src/main.js under src/ and stub/src/main.js under stub/src/ — move
// them to the URLs the HTML expects.
await Bun.write(Bun.file(join(dist, 'main.js')), Bun.file(join(dist, 'src/main.js')))
await Bun.write(
  Bun.file(join(stubDist, 'main.js')),
  Bun.file(join(dist, 'stub/src/main.js')),
)
await rm(join(dist, 'src'), { recursive: true, force: true })
await rm(join(dist, 'stub'), { recursive: true, force: true })

await copyFile(join(root, 'index.html'), join(dist, 'index.html'))
await copyFile(join(root, 'stub/index.html'), join(stubDist, 'index.html'))

console.log('shell build → web/shell/dist/')
```

- [ ] **Step 2: Add .gitignore for dist**

Create `web/shell/dist/.gitignore`:
```
*
!.gitignore
```

Wait — the dist folder doesn't exist until build. Use a top-level ignore instead. Create or append to `web/shell/.gitignore`:
```
dist/
```

Delete the previous `dist/.gitignore` line. Replace Step 2 with creating `web/shell/.gitignore`:

```
dist/
```

- [ ] **Step 3: Extend Makefile**

Find the `css:` target (or similar) in `Makefile` and add alongside:

```make
.PHONY: shell
shell:
	cd web/shared && bun install --frozen-lockfile
	bun run web/shell/build.ts
```

If there's an aggregated target (e.g. `build` or `all`), add `shell` to its dependencies.

- [ ] **Step 4: Run the build**

```bash
make shell
```

Expected output: `shell build → web/shell/dist/`, and `web/shell/dist/{index.html, main.js, renderer-stub/index.html, renderer-stub/main.js}` exist.

- [ ] **Step 5: Commit**

```bash
git add web/shell/build.ts web/shell/.gitignore Makefile
git commit -m "chore(build): add bun build script and make target for shell"
```

---

## Task 14: Go `//go:embed` + `/app/*` route behind env flag

**Files:**
- Create: `web/shell/embed.go`
- Modify: `internal/api/server.go`
- Modify: `internal/api/server_test.go` (or new test file)

Serve the built bundle from Go. Off by default; set `LIVEBOARD_APP_SHELL=1` to enable.

- [ ] **Step 1: Write embed.go**

```go
// Package shell exposes the built TS shell bundle for embedding in the Go server.
package shell

import "embed"

//go:embed all:dist
var FS embed.FS
```

- [ ] **Step 2: Modify server.go — add route**

In `internal/api/server.go`, find `buildRouter` and add near where `/static/*` is mounted:

```go
	if os.Getenv("LIVEBOARD_APP_SHELL") == "1" {
		s.mountShellRoutes(r)
		log.Println("shell mounted at /app/")
	}
```

Add the method:

```go
func (s *Server) mountShellRoutes(r chi.Router) {
	sub, err := fs.Sub(shell.FS, "dist")
	if err != nil {
		log.Printf("shell embed: %v", err)
		return
	}
	handler := http.StripPrefix("/app/", http.FileServer(http.FS(sub)))
	r.Get("/app", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/app/", http.StatusMovedPermanently)
	})
	r.Get("/app/*", func(w http.ResponseWriter, req *http.Request) {
		if s.noCache {
			w.Header().Set("Cache-Control", "no-cache, no-store")
		}
		handler.ServeHTTP(w, req)
	})
}
```

Add imports at top of the file:

```go
	"io/fs"

	shell "github.com/and1truong/liveboard/web/shell"
```

- [ ] **Step 3: Write server_test.go case**

Create or append to `internal/api/server_shell_test.go`:

```go
package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/workspace"
)

func TestShellRoute_Disabled(t *testing.T) {
	t.Setenv("LIVEBOARD_APP_SHELL", "")
	dir := t.TempDir()
	ws := workspace.New(dir)
	eng := board.NewEngine()
	s := NewServer(ws, eng, false, false, false, "test", "", "")

	req := httptest.NewRequest(http.MethodGet, "/app/", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("shell route should be 404 when flag disabled; got %d", rec.Code)
	}
}

func TestShellRoute_Enabled(t *testing.T) {
	t.Setenv("LIVEBOARD_APP_SHELL", "1")
	dir := t.TempDir()
	ws := workspace.New(dir)
	eng := board.NewEngine()
	s := NewServer(ws, eng, false, false, false, "test", "", "")

	req := httptest.NewRequest(http.MethodGet, "/app/", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "LiveBoard Shell") {
		t.Fatalf("response did not contain expected shell HTML")
	}
}
```

Note: if `workspace.New` has a different signature in this codebase, match the surrounding tests' setup pattern. Search `git grep "workspace.New\|NewServer(" internal/api/` for the pattern.

- [ ] **Step 4: Build shell, then run Go tests**

```bash
make shell
go test ./internal/api/ -race -run 'TestShell' -v
```
Expected: both tests pass. If disabled test still gets 200, `mountShellRoutes` is being called unconditionally — re-check the env guard.

- [ ] **Step 5: Commit**

```bash
git add web/shell/embed.go internal/api/server.go internal/api/server_shell_test.go
git commit -m "feat(shell): serve /app/* from Go behind LIVEBOARD_APP_SHELL flag"
```

---

## Task 15: CI + docs + manual browser smoke

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `README.md`
- Modify: `docs/parity.md` (brief note)

- [ ] **Step 1: Add shell build to CI**

In `.github/workflows/ci.yml`, in the `test` job, before the `Run tests` step, add:

```yaml
      - name: Set up Bun
        uses: oven-sh/setup-bun@v2
        with:
          bun-version: "1.3.10"

      - name: Build shell bundle
        run: make shell
```

The `ts-parity` job (added in P2) already installs bun; leaving that separate is fine — different working directory.

- [ ] **Step 2: Add a README note**

In `README.md`, under the `## REST API` section or a new `## Shell (preview)` section:

```markdown
## Shell (preview)

Experimental postMessage shell + stub renderer. Enable with:

    LIVEBOARD_APP_SHELL=1 liveboard serve

Then open <http://localhost:7070/app/> — the stub iframe exercises every protocol method and logs results.
```

- [ ] **Step 3: Link adapter tests in parity doc**

In `docs/parity.md`, at the bottom, add:

```markdown
## Adapters

The `LocalAdapter` at `web/shared/src/adapters/local.ts` consumes `applyOp` from the parity module. Its tests (`local.test.ts`) verify adapter-specific behavior (version conflicts, seed, BroadcastChannel); the underlying mutation correctness is covered by the vector suite above.
```

- [ ] **Step 4: Manual browser smoke**

```bash
make shell
LIVEBOARD_APP_SHELL=1 go run ./cmd/liveboard serve --port 7070
```

Open http://localhost:7070/app/ in a browser. Expected:
- Page loads, stub iframe visible.
- All lines show `OK` in green.
- Open a second tab to the same URL. In one tab, open devtools and run a mutation (or just reload). The other tab's stub should re-render on board-update (you'll see a new OK line for `event: board.updated received` if you reload the stub).

If any line shows FAIL, copy the detail to the terminal and debug.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/ci.yml README.md docs/parity.md
git commit -m "docs(shell): document /app/ preview and CI shell build"
```

---

## Spec coverage checklist

| Spec requirement | Covered by |
|---|---|
| `BackendAdapter` interface | Task 2 |
| `LocalAdapter` implementation | Tasks 3–6 |
| Seed workspace on first load | Task 4 |
| Version-conflict handling | Task 5 |
| BroadcastChannel multi-tab sync | Task 6 |
| postMessage protocol v1 (JSON-RPC-style + events) | Tasks 1, 8, 9 |
| Handshake with version negotiation | Tasks 8, 9, 12 |
| Origin validation (shell + iframe) | Task 10 |
| Shell at `/app/` (flag-gated) | Task 14 |
| Stub renderer exercising every method | Task 12 |
| Request/response correlation | Task 9 |
| Event push (`board.updated`) | Tasks 8, 9, 12 |
| `VERSION_CONFLICT` reachable | Tasks 5, 12 |

## Open questions / notes for the implementer

1. The Go workspace constructor in the shell-route tests must match the real signature. If `workspace.New(dir)` does not compile, grep for an existing test setup in `internal/api/` and copy it.
2. If the `tsconfig.json` change in Task 10 (add `"DOM"` to `lib`) causes the parity module's types to surface browser globals we didn't want, an alternative is a separate `tsconfig.browser.json` scoped to shell sources. Start with the simpler unified config and only split if type pollution becomes noisy.
3. Bun's `Bun.build` does not currently support HTML entrypoint rewriting; that's why Task 13 does the copy by hand. Keep the build script plain — don't reach for Vite or Rollup unless bun's output proves inadequate.
