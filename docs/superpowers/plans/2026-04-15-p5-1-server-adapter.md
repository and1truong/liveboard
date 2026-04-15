# P5.1 — TypeScript ServerAdapter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `class ServerAdapter implements BackendAdapter` against the P5.0 `/api/v1/*` HTTP+SSE surface, with constructor-injected `baseUrl` + `fetch`. Lazy `EventSource` lifecycle. HTTP-only unit tests; SSE deferred to P5.2 manual smoke.

**Architecture:** Single class file. Eleven public methods (1:1 with `BackendAdapter`). All HTTP routes through small wrappers (`getJSON`, `postJSON`, etc.) that handle JSON encode/decode + error envelope → `ProtocolError`. SSE handler maintains per-board + workspace-list handler maps; opens one `EventSource` lazily, closes when idle.

**Tech Stack:** Browser-native `fetch` + `EventSource`. No new deps.

**Spec:** `docs/superpowers/specs/2026-04-15-p5-1-server-adapter-design.md`

**Conventions:**
- Code under `web/shared/src/adapters/`.
- Tests colocated.
- Commit prefixes: `feat(shared)`, `test(shared)`.
- Use bun, never npx.
- The `BackendAdapter` interface (from `web/shared/src/adapter.ts`) is the source of truth — adapter `implements` it for compile-time enforcement.

---

## File structure

**New:**
- `web/shared/src/adapters/server.ts`
- `web/shared/src/adapters/server.test.ts`

**No modifications.** P5.2 wires the adapter into the shell.

---

## Task 1: HTTP wrappers + error decoder + class skeleton

**Files:**
- Create: `web/shared/src/adapters/server.ts`

Establish the class scaffold + HTTP plumbing. Methods are stubs that return `throw new Error('not implemented')` — Tasks 2–5 fill them.

- [ ] **Step 1: Skeleton**

Create `web/shared/src/adapters/server.ts`:
```ts
import type { Board, BoardSettings, MutationOp } from '../types.js'
import type {
  BackendAdapter,
  BoardSummary,
  BoardUpdateHandler,
  ResolvedSettings,
  Subscription,
  WorkspaceInfo,
} from '../adapter.js'
import { ProtocolError, type ErrorCode } from '../protocol.js'

export interface ServerAdapterOptions {
  baseUrl: string
  fetch?: typeof globalThis.fetch
}

export class ServerAdapter implements BackendAdapter {
  private readonly baseUrl: string
  private readonly fetchFn: typeof globalThis.fetch
  private es: EventSource | null = null
  private readonly perBoard = new Map<string, Set<BoardUpdateHandler>>()
  private readonly listHandlers = new Set<() => void>()

  constructor(opts: ServerAdapterOptions) {
    this.baseUrl = opts.baseUrl.replace(/\/$/, '')
    this.fetchFn = opts.fetch ?? globalThis.fetch.bind(globalThis)
  }

  private async request(
    method: string,
    path: string,
    body?: unknown,
  ): Promise<Response> {
    let res: Response
    try {
      res = await this.fetchFn(`${this.baseUrl}${path}`, {
        method,
        headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
        body: body !== undefined ? JSON.stringify(body) : undefined,
      })
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      throw new ProtocolError('INTERNAL', msg)
    }
    if (!res.ok) throw await this.decodeError(res)
    return res
  }

  private async decodeError(res: Response): Promise<ProtocolError> {
    let code: ErrorCode = 'INTERNAL'
    let message = `${res.status} ${res.statusText}`
    try {
      const body = (await res.json()) as { error?: { code?: string; message?: string } }
      if (body.error?.code) code = body.error.code as ErrorCode
      if (body.error?.message) message = body.error.message
    } catch {
      // non-JSON body — keep defaults
    }
    return new ProtocolError(code, message)
  }

  private async getJSON<T>(path: string): Promise<T> {
    const res = await this.request('GET', path)
    return (await res.json()) as T
  }

  private async postJSON<T>(path: string, body: unknown): Promise<T> {
    const res = await this.request('POST', path, body)
    return (await res.json()) as T
  }

  private async patchJSON<T>(path: string, body: unknown): Promise<T> {
    const res = await this.request('PATCH', path, body)
    return (await res.json()) as T
  }

  private async putEmpty(path: string, body: unknown): Promise<void> {
    await this.request('PUT', path, body)
  }

  private async deleteEmpty(path: string): Promise<void> {
    await this.request('DELETE', path)
  }

  // === BackendAdapter — stubbed; filled in Tasks 2–5 ===
  listBoards(): Promise<BoardSummary[]> { throw new Error('not implemented') }
  createBoard(_name: string): Promise<BoardSummary> { throw new Error('not implemented') }
  renameBoard(_boardId: string, _newName: string): Promise<BoardSummary> { throw new Error('not implemented') }
  deleteBoard(_boardId: string): Promise<void> { throw new Error('not implemented') }
  getBoard(_boardId: string): Promise<Board> { throw new Error('not implemented') }
  mutateBoard(_boardId: string, _clientVersion: number, _op: MutationOp): Promise<Board> { throw new Error('not implemented') }
  getSettings(_boardId: string): Promise<ResolvedSettings> { throw new Error('not implemented') }
  putBoardSettings(_boardId: string, _patch: Partial<BoardSettings>): Promise<void> { throw new Error('not implemented') }
  getWorkspaceInfo(): Promise<WorkspaceInfo> { throw new Error('not implemented') }
  subscribe(_boardId: string, _onUpdate: BoardUpdateHandler): Subscription { throw new Error('not implemented') }
  onBoardListUpdate(_handler: () => void): Subscription { throw new Error('not implemented') }
}
```

- [ ] **Step 2: Typecheck**

```bash
cd /Users/htruong/code/htruong/liveboard && bun --cwd web/renderer/default run typecheck
```
Expected: clean. `ServerAdapter` satisfies `BackendAdapter` even though methods throw — interface only checks signatures.

- [ ] **Step 3: Commit**

```bash
git add web/shared/src/adapters/server.ts
git commit -m "feat(shared): add ServerAdapter skeleton + HTTP wrappers

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: HTTP test scaffold + `listBoards` + `getWorkspaceInfo`

**Files:**
- Create: `web/shared/src/adapters/server.test.ts`
- Modify: `web/shared/src/adapters/server.ts`

Establish the test pattern (mock fetch + assertions on URL/method/body), implement two read-only methods to validate it.

- [ ] **Step 1: Test scaffold**

Create `web/shared/src/adapters/server.test.ts`:
```ts
import { describe, expect, it } from 'bun:test'
import { ProtocolError } from '../protocol.js'
import { ServerAdapter } from './server.js'

interface RequestRecord {
  method: string
  url: string
  body: string | null
}

function mockFetch(
  handler: (req: RequestRecord) => Response | Promise<Response>,
  log?: RequestRecord[],
): typeof fetch {
  return (async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.toString() : input.url
    const body = init?.body ? String(init.body) : null
    const rec: RequestRecord = { method: init?.method ?? 'GET', url, body }
    log?.push(rec)
    return await handler(rec)
  }) as typeof fetch
}

function jsonResponse(data: unknown, status = 200): Response {
  return new Response(JSON.stringify(data), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function errorResponse(code: string, message: string, status = 400): Response {
  return new Response(JSON.stringify({ error: { code, message } }), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('ServerAdapter HTTP', () => {
  it('listBoards GETs /boards and returns parsed JSON', async () => {
    const log: RequestRecord[] = []
    const a = new ServerAdapter({
      baseUrl: '/api/v1',
      fetch: mockFetch(
        () => jsonResponse([{ id: 'welcome', name: 'Welcome', version: 1 }]),
        log,
      ),
    })
    const out = await a.listBoards()
    expect(out).toEqual([{ id: 'welcome', name: 'Welcome', version: 1 }])
    expect(log[0]).toEqual({ method: 'GET', url: '/api/v1/boards', body: null })
  })

  it('getWorkspaceInfo maps board_count → boardCount', async () => {
    const a = new ServerAdapter({
      baseUrl: '/api/v1',
      fetch: mockFetch(() => jsonResponse({ name: 'Demo', board_count: 3 })),
    })
    const ws = await a.getWorkspaceInfo()
    expect(ws).toEqual({ name: 'Demo', boardCount: 3 })
  })

  it('non-2xx with error envelope throws ProtocolError', async () => {
    const a = new ServerAdapter({
      baseUrl: '/api/v1',
      fetch: mockFetch(() => errorResponse('NOT_FOUND', 'gone', 404)),
    })
    try {
      await a.listBoards()
      throw new Error('expected throw')
    } catch (e) {
      expect(e).toBeInstanceOf(ProtocolError)
      expect((e as ProtocolError).code).toBe('NOT_FOUND')
    }
  })

  it('network failure throws ProtocolError INTERNAL', async () => {
    const a = new ServerAdapter({
      baseUrl: '/api/v1',
      fetch: () => { throw new Error('boom') },
    })
    try {
      await a.listBoards()
      throw new Error('expected throw')
    } catch (e) {
      expect(e).toBeInstanceOf(ProtocolError)
      expect((e as ProtocolError).code).toBe('INTERNAL')
    }
  })

  it('non-JSON 500 body becomes ProtocolError INTERNAL', async () => {
    const a = new ServerAdapter({
      baseUrl: '/api/v1',
      fetch: mockFetch(() => new Response('panic', { status: 500 })),
    })
    try {
      await a.listBoards()
      throw new Error('expected throw')
    } catch (e) {
      expect(e).toBeInstanceOf(ProtocolError)
      expect((e as ProtocolError).code).toBe('INTERNAL')
    }
  })
})
```

- [ ] **Step 2: Run, expect fail**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src/adapters/server.test.ts
```
Expected: 5 fail (`not implemented`).

- [ ] **Step 3: Implement two methods**

In `web/shared/src/adapters/server.ts`, replace the stubs:
```ts
  listBoards(): Promise<BoardSummary[]> {
    return this.getJSON<BoardSummary[]>('/boards')
  }

  async getWorkspaceInfo(): Promise<WorkspaceInfo> {
    const raw = await this.getJSON<{ name: string; board_count: number }>('/workspace')
    return { name: raw.name, boardCount: raw.board_count }
  }
```

- [ ] **Step 4: Run, expect 5 pass**

```bash
bun test web/shared/src/adapters/server.test.ts
```

- [ ] **Step 5: Commit**

```bash
git add web/shared/src/adapters/server.ts web/shared/src/adapters/server.test.ts
git commit -m "feat(shared): ServerAdapter listBoards + getWorkspaceInfo + error mapping

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: Board CRUD (`createBoard`, `getBoard`, `renameBoard`, `deleteBoard`)

**Files:**
- Modify: `web/shared/src/adapters/server.ts`
- Modify: `web/shared/src/adapters/server.test.ts`

- [ ] **Step 1: Tests**

Append to `server.test.ts`:
```ts
describe('ServerAdapter CRUD', () => {
  it('createBoard POSTs name and returns BoardSummary', async () => {
    const log: RequestRecord[] = []
    const a = new ServerAdapter({
      baseUrl: '/api/v1',
      fetch: mockFetch(
        () => jsonResponse({ id: 'foo', name: 'Foo', version: 1 }, 201),
        log,
      ),
    })
    const s = await a.createBoard('Foo')
    expect(s).toEqual({ id: 'foo', name: 'Foo', version: 1 })
    expect(log[0]).toEqual({
      method: 'POST',
      url: '/api/v1/boards',
      body: JSON.stringify({ name: 'Foo' }),
    })
  })

  it('createBoard collision surfaces ALREADY_EXISTS', async () => {
    const a = new ServerAdapter({
      baseUrl: '/api/v1',
      fetch: mockFetch(() => errorResponse('ALREADY_EXISTS', 'exists', 409)),
    })
    try {
      await a.createBoard('Foo')
      throw new Error('expected throw')
    } catch (e) {
      expect((e as ProtocolError).code).toBe('ALREADY_EXISTS')
    }
  })

  it('getBoard GETs /boards/{id}', async () => {
    const log: RequestRecord[] = []
    const a = new ServerAdapter({
      baseUrl: '/api/v1',
      fetch: mockFetch(
        () => jsonResponse({ name: 'Welcome', version: 1, columns: [] }),
        log,
      ),
    })
    const b = await a.getBoard('welcome')
    expect(b.name).toBe('Welcome')
    expect(log[0].url).toBe('/api/v1/boards/welcome')
  })

  it('renameBoard PATCHes new_name and returns new summary', async () => {
    const log: RequestRecord[] = []
    const a = new ServerAdapter({
      baseUrl: '/api/v1',
      fetch: mockFetch(
        () => jsonResponse({ id: 'bar', name: 'Bar', version: 2 }),
        log,
      ),
    })
    const s = await a.renameBoard('foo', 'Bar')
    expect(s).toEqual({ id: 'bar', name: 'Bar', version: 2 })
    expect(log[0]).toEqual({
      method: 'PATCH',
      url: '/api/v1/boards/foo',
      body: JSON.stringify({ new_name: 'Bar' }),
    })
  })

  it('deleteBoard DELETEs and resolves void on 204', async () => {
    const log: RequestRecord[] = []
    const a = new ServerAdapter({
      baseUrl: '/api/v1',
      fetch: mockFetch(() => new Response(null, { status: 204 }), log),
    })
    await a.deleteBoard('foo')
    expect(log[0]).toEqual({ method: 'DELETE', url: '/api/v1/boards/foo', body: null })
  })

  it('deleteBoard NOT_FOUND throws ProtocolError', async () => {
    const a = new ServerAdapter({
      baseUrl: '/api/v1',
      fetch: mockFetch(() => errorResponse('NOT_FOUND', 'gone', 404)),
    })
    try {
      await a.deleteBoard('nope')
      throw new Error('expected throw')
    } catch (e) {
      expect((e as ProtocolError).code).toBe('NOT_FOUND')
    }
  })
})
```

- [ ] **Step 2: Run, expect fail**

```bash
bun test web/shared/src/adapters/server.test.ts
```

- [ ] **Step 3: Implement**

In `server.ts`, replace the stubs:
```ts
  createBoard(name: string): Promise<BoardSummary> {
    return this.postJSON<BoardSummary>('/boards', { name })
  }

  renameBoard(boardId: string, newName: string): Promise<BoardSummary> {
    return this.patchJSON<BoardSummary>(`/boards/${encodeURIComponent(boardId)}`, { new_name: newName })
  }

  deleteBoard(boardId: string): Promise<void> {
    return this.deleteEmpty(`/boards/${encodeURIComponent(boardId)}`)
  }

  getBoard(boardId: string): Promise<Board> {
    return this.getJSON<Board>(`/boards/${encodeURIComponent(boardId)}`)
  }
```

- [ ] **Step 4: Run, expect all pass**

```bash
bun test web/shared/src/adapters/server.test.ts
```

- [ ] **Step 5: Commit**

```bash
git add web/shared/src/adapters/server.ts web/shared/src/adapters/server.test.ts
git commit -m "feat(shared): ServerAdapter board CRUD methods

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 4: `mutateBoard`, `getSettings`, `putBoardSettings`

**Files:**
- Modify: `web/shared/src/adapters/server.ts`
- Modify: `web/shared/src/adapters/server.test.ts`

- [ ] **Step 1: Tests**

Append:
```ts
describe('ServerAdapter mutate + settings', () => {
  it('mutateBoard POSTs client_version + op and returns Board', async () => {
    const log: RequestRecord[] = []
    const board = { name: 'Foo', version: 2, columns: [] }
    const a = new ServerAdapter({
      baseUrl: '/api/v1',
      fetch: mockFetch(() => jsonResponse(board), log),
    })
    const out = await a.mutateBoard('foo', 1, { type: 'add_card', column: 'Todo', title: 'x' })
    expect(out).toEqual(board)
    expect(log[0].method).toBe('POST')
    expect(log[0].url).toBe('/api/v1/boards/foo/mutations')
    expect(JSON.parse(log[0].body!)).toEqual({
      client_version: 1,
      op: { type: 'add_card', column: 'Todo', title: 'x' },
    })
  })

  it('mutateBoard surfaces VERSION_CONFLICT', async () => {
    const a = new ServerAdapter({
      baseUrl: '/api/v1',
      fetch: mockFetch(() => errorResponse('VERSION_CONFLICT', 'stale', 409)),
    })
    try {
      await a.mutateBoard('foo', 0, { type: 'add_card', column: 'Todo', title: 'x' })
      throw new Error('expected throw')
    } catch (e) {
      expect((e as ProtocolError).code).toBe('VERSION_CONFLICT')
    }
  })

  it('getSettings GETs and returns ResolvedSettings shape', async () => {
    const a = new ServerAdapter({
      baseUrl: '/api/v1',
      fetch: mockFetch(() =>
        jsonResponse({
          show_checkbox: false,
          card_position: 'bottom',
          expand_columns: false,
          view_mode: 'board',
          card_display_mode: 'compact',
          week_start: 'monday',
        }),
      ),
    })
    const s = await a.getSettings('foo')
    expect(s.show_checkbox).toBe(false)
    expect(s.card_display_mode).toBe('compact')
  })

  it('putBoardSettings PUTs partial body and resolves void on 204', async () => {
    const log: RequestRecord[] = []
    const a = new ServerAdapter({
      baseUrl: '/api/v1',
      fetch: mockFetch(() => new Response(null, { status: 204 }), log),
    })
    await a.putBoardSettings('foo', { show_checkbox: false })
    expect(log[0]).toEqual({
      method: 'PUT',
      url: '/api/v1/boards/foo/settings',
      body: JSON.stringify({ show_checkbox: false }),
    })
  })
})
```

- [ ] **Step 2: Run, expect fail**

```bash
bun test web/shared/src/adapters/server.test.ts
```

- [ ] **Step 3: Implement**

In `server.ts`:
```ts
  mutateBoard(boardId: string, clientVersion: number, op: MutationOp): Promise<Board> {
    return this.postJSON<Board>(
      `/boards/${encodeURIComponent(boardId)}/mutations`,
      { client_version: clientVersion, op },
    )
  }

  getSettings(boardId: string): Promise<ResolvedSettings> {
    return this.getJSON<ResolvedSettings>(`/boards/${encodeURIComponent(boardId)}/settings`)
  }

  putBoardSettings(boardId: string, patch: Partial<BoardSettings>): Promise<void> {
    return this.putEmpty(`/boards/${encodeURIComponent(boardId)}/settings`, patch)
  }
```

- [ ] **Step 4: Run, expect all pass**

```bash
bun test web/shared/src/adapters/server.test.ts
```

- [ ] **Step 5: Commit**

```bash
git add web/shared/src/adapters/server.ts web/shared/src/adapters/server.test.ts
git commit -m "feat(shared): ServerAdapter mutateBoard + settings methods

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 5: SSE multiplex (`subscribe`, `onBoardListUpdate`)

**Files:**
- Modify: `web/shared/src/adapters/server.ts`

No unit tests — `EventSource` isn't in bun's test env per the spec. SSE is verified by P5.2 manual smoke. Implementation is small and predictable.

- [ ] **Step 1: Implement**

Replace the SSE stubs in `server.ts`:
```ts
  subscribe(boardId: string, onUpdate: BoardUpdateHandler): Subscription {
    let set = this.perBoard.get(boardId)
    if (!set) {
      set = new Set()
      this.perBoard.set(boardId, set)
    }
    set.add(onUpdate)
    this.ensureEventSource()
    return {
      close: () => {
        const s = this.perBoard.get(boardId)
        if (!s) return
        s.delete(onUpdate)
        if (s.size === 0) this.perBoard.delete(boardId)
        this.closeIfIdle()
      },
    }
  }

  onBoardListUpdate(handler: () => void): Subscription {
    this.listHandlers.add(handler)
    this.ensureEventSource()
    return {
      close: () => {
        this.listHandlers.delete(handler)
        this.closeIfIdle()
      },
    }
  }

  private ensureEventSource(): void {
    if (this.es) return
    if (typeof EventSource === 'undefined') return // Test env / SSR — handlers stay registered but never fire.
    const es = new EventSource(`${this.baseUrl}/events`)
    es.addEventListener('board.updated', (ev) => {
      try {
        const data = JSON.parse((ev as MessageEvent).data) as { board_id: string; version: number }
        const set = this.perBoard.get(data.board_id)
        if (set) for (const h of set) h({ boardId: data.board_id, version: data.version })
      } catch {
        // ignore malformed payload
      }
    })
    es.addEventListener('board.list.updated', () => {
      for (const h of this.listHandlers) h()
    })
    this.es = es
  }

  private closeIfIdle(): void {
    if (this.perBoard.size === 0 && this.listHandlers.size === 0 && this.es) {
      this.es.close()
      this.es = null
    }
  }
```

- [ ] **Step 2: Typecheck + full test pass**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src/adapters/server.test.ts && bun --cwd web/renderer/default run typecheck
```
Expected: previous tests still pass, typecheck clean.

- [ ] **Step 3: Commit**

```bash
git add web/shared/src/adapters/server.ts
git commit -m "feat(shared): ServerAdapter SSE subscribe + onBoardListUpdate

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 6: Full suite check

**Files:** none.

- [ ] **Step 1: Run shared + renderer suites**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/shared/src && cd web/renderer/default && bun test && bun run typecheck
```
Expected: all green. The pre-existing `boardOps` vector test failure (missing testdata dir) is a known baseline; not introduced here.

- [ ] **Step 2: No commit.**

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| `ServerAdapter` skeleton + ctor + HTTP wrappers + error decoder | 1 |
| `listBoards`, `getWorkspaceInfo` (camelCase mapping) | 2 |
| `createBoard` / `getBoard` / `renameBoard` / `deleteBoard` | 3 |
| `mutateBoard` / `getSettings` / `putBoardSettings` | 4 |
| `subscribe` / `onBoardListUpdate` SSE multiplex | 5 |
| Lazy EventSource open / idle close | 5 |
| Error envelope → ProtocolError | 1 (decoder) + 2 (test) |
| Network failure → ProtocolError INTERNAL | 1 (decoder) + 2 (test) |
| HTTP-only unit tests; SSE deferred to P5.2 smoke | All test tasks |

## Notes for implementer

1. **`encodeURIComponent` on slugs** — the slugify rules in P4c.0 produce strings safe for URL paths, but better safe than sorry: every interpolated slug uses `encodeURIComponent`. Server side receives the decoded slug via chi.
2. **`putEmpty` + `deleteEmpty` don't parse JSON** — they return after `request()` succeeds. 204-with-no-body is the happy case.
3. **`fetch.bind(globalThis)`** in the constructor is required — `fetch` references `this` inside, and passing the bare reference triggers "Illegal invocation" in strict environments.
4. **`EventSource === undefined` early-return** keeps SSE-using methods callable in test environments. Subscription handles still register; they just never fire. P5.2 smoke catches the real behavior.
5. **`BackendAdapter` is the source of truth** — if a method signature drifts (e.g. parameter rename), the `implements` clause forces ServerAdapter to update. Don't loosen the `implements` constraint.
6. **No commit amending** — forward-only commits.
