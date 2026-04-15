# P5.1 — TypeScript ServerAdapter — Design

## Goal

Implement `BackendAdapter` against the P5.0 `/api/v1/*` HTTP + SSE surface so the React renderer can be backed by the Go server's filesystem-stored workspace. No shell wiring yet — that lands in P5.2.

**Shippable value:** `web/shared/src/adapters/server.ts` exists, fully unit-tested for the HTTP path, ready for P5.2 to swap in. Closes the gap between the LocalAdapter (browser localStorage) and a server-backed adapter (Markdown files on disk via the Go engine).

## Scope

**In:**
- `class ServerAdapter implements BackendAdapter` in `web/shared/src/adapters/server.ts`.
- HTTP for: `listBoards`, `createBoard`, `getBoard`, `renameBoard`, `deleteBoard`, `mutateBoard`, `getSettings`, `putBoardSettings`, `getWorkspaceInfo`.
- SSE multiplex for: `subscribe(boardId, handler)`, `onBoardListUpdate(handler)`. (`settings.updated` plumbed into the SSE handler but not surfaced — no consumer in the renderer yet.)
- Constructor-injected `baseUrl` and optional `fetch` for testability.
- Lazy SSE: open `EventSource` on first subscribe; close when last handler unsubscribes.
- Error envelope decoding → `ProtocolError(code, message)`.
- Unit tests for every HTTP method (happy + error paths).

**Out:**
- SSE unit tests (no EventSource in bun test env; covered by P5.2 manual smoke).
- Auth headers / cookies (P5.0 has no auth).
- Retry / timeout / circuit-breaker policy (browser fetch defaults are fine for local-first).
- Streaming uploads / downloads.
- WebSocket fallback.

## Architecture

```
ServerAdapter (web/shared/src/adapters/server.ts)
 ├─ baseUrl          string                           # from constructor
 ├─ fetch            typeof fetch                     # constructor or globalThis.fetch
 ├─ es               EventSource | null               # lazy
 ├─ perBoard         Map<string, Set<BoardUpdateHandler>>
 ├─ listHandlers     Set<() => void>
 ├─ HTTP wrappers    getJSON / postJSON / patchJSON / putJSON / deleteEmpty
 ├─ Public methods   1:1 with BackendAdapter
 └─ ensureEventSource() / closeIfIdle() / onSseMessage()
```

The class is a thin transport: every public method is HTTP I/O + JSON (de)serialization + error mapping. The `BackendAdapter` interface drives the surface.

## Wire shapes

Mirrors the P5.0 contract verbatim. Field naming uses `snake_case` on the wire and `camelCase` in TS — adapter does the keysmith at the boundary.

### HTTP error decoder

```ts
async function decodeError(res: Response): Promise<ProtocolError> {
  let code: ErrorCode = 'INTERNAL'
  let message = `${res.status} ${res.statusText}`
  try {
    const body = await res.json() as { error?: { code?: string; message?: string } }
    if (body.error?.code) code = body.error.code as ErrorCode
    if (body.error?.message) message = body.error.message
  } catch { /* non-JSON body — leave defaults */ }
  return new ProtocolError(code, message)
}
```

Network errors (`fetch` itself throws) → `new ProtocolError('INTERNAL', err.message)`.

### Method map

| `BackendAdapter` method | HTTP |
|---|---|
| `listBoards()` | `GET /boards` |
| `createBoard(name)` | `POST /boards` body `{ name }` |
| `getBoard(boardId)` | `GET /boards/{id}` |
| `renameBoard(boardId, newName)` | `PATCH /boards/{id}` body `{ new_name }` |
| `deleteBoard(boardId)` | `DELETE /boards/{id}` |
| `mutateBoard(boardId, clientVersion, op)` | `POST /boards/{id}/mutations` body `{ client_version, op }` |
| `getSettings(boardId)` | `GET /boards/{id}/settings` |
| `putBoardSettings(boardId, patch)` | `PUT /boards/{id}/settings` body = patch |
| `getWorkspaceInfo()` | `GET /workspace` |

`mutateBoard` returns the post-mutation `Board`; `applyOp` is unused server-side. The optimistic flow in `useBoardMutation` (P4b.1a) still owns local optimistic apply.

### SSE multiplex

`/events` stream emits three event types. Adapter dispatches:

| SSE `event:` | Adapter delivery |
|---|---|
| `board.updated` | `perBoard.get(data.board_id)` → call each handler with `{ boardId, version }` |
| `board.list.updated` | `listHandlers` → call each |
| `settings.updated` | parsed and ignored (no subscriber); reserved for future |

## Component contract

```ts
export interface ServerAdapterOptions {
  baseUrl: string                 // e.g. '/api/v1' or 'http://localhost:7070/api/v1'
  fetch?: typeof globalThis.fetch // testable injection; defaults to globalThis.fetch
}

export class ServerAdapter implements BackendAdapter {
  constructor(opts: ServerAdapterOptions)

  // BackendAdapter
  listBoards(): Promise<BoardSummary[]>
  createBoard(name: string): Promise<BoardSummary>
  renameBoard(boardId: string, newName: string): Promise<BoardSummary>
  deleteBoard(boardId: string): Promise<void>
  getBoard(boardId: string): Promise<Board>
  mutateBoard(boardId: string, clientVersion: number, op: MutationOp): Promise<Board>
  getSettings(boardId: string): Promise<ResolvedSettings>
  putBoardSettings(boardId: string, patch: Partial<BoardSettings>): Promise<void>
  getWorkspaceInfo(): Promise<WorkspaceInfo>
  subscribe(boardId: string, onUpdate: BoardUpdateHandler): Subscription
  onBoardListUpdate(handler: () => void): Subscription
}
```

`Subscription` is the existing `{ close: () => void }` shape used throughout `web/shared/src/adapter.ts`.

## SSE lifecycle

```ts
private ensureEventSource(): void {
  if (this.es) return
  const es = new EventSource(`${this.baseUrl}/events`)
  es.addEventListener('board.updated', (ev) => {
    try {
      const data = JSON.parse((ev as MessageEvent).data) as { board_id: string; version: number }
      const set = this.perBoard.get(data.board_id)
      if (set) for (const h of set) h({ boardId: data.board_id, version: data.version })
    } catch { /* ignore malformed payload */ }
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

subscribe(boardId, onUpdate): Subscription {
  let set = this.perBoard.get(boardId)
  if (!set) { set = new Set(); this.perBoard.set(boardId, set) }
  set.add(onUpdate)
  this.ensureEventSource()
  return {
    close: () => {
      set!.delete(onUpdate)
      if (set!.size === 0) this.perBoard.delete(boardId)
      this.closeIfIdle()
    },
  }
}

onBoardListUpdate(handler): Subscription {
  this.listHandlers.add(handler)
  this.ensureEventSource()
  return {
    close: () => {
      this.listHandlers.delete(handler)
      this.closeIfIdle()
    },
  }
}
```

EventSource handles auto-reconnect natively; no application logic needed.

## Testing

`web/shared/src/adapters/server.test.ts` — bun test against an injected mock fetch. SSE deferred to manual smoke per the design.

Cases:

```ts
function mockFetch(handler: (req: Request) => Response | Promise<Response>): typeof fetch {
  return ((url, init) => Promise.resolve(handler(new Request(url, init)))) as typeof fetch
}

describe('ServerAdapter HTTP', () => {
  it('listBoards GETs /boards and returns parsed JSON', async () => {
    const a = new ServerAdapter({
      baseUrl: '/api/v1',
      fetch: mockFetch(() => new Response(JSON.stringify([{ id: 'welcome', name: 'Welcome', version: 1 }]), { status: 200 })),
    })
    const out = await a.listBoards()
    expect(out[0].id).toBe('welcome')
  })

  it('createBoard POSTs name and returns BoardSummary', async () => { /* … */ })
  it('createBoard surfaces ALREADY_EXISTS', async () => { /* … */ })
  it('renameBoard PATCHes new_name and returns new summary', async () => { /* … */ })
  it('deleteBoard returns void on 204', async () => { /* … */ })
  it('mutateBoard POSTs body with client_version + op', async () => { /* … */ })
  it('mutateBoard surfaces VERSION_CONFLICT', async () => { /* … */ })
  it('getSettings + putBoardSettings round-trip via mock', async () => { /* … */ })
  it('getWorkspaceInfo GETs /workspace', async () => { /* … */ })
  it('network error becomes ProtocolError INTERNAL', async () => { /* … */ })
  it('non-JSON 500 body becomes ProtocolError INTERNAL', async () => { /* … */ })
})
```

Helpers:
- `mockFetch(handler)` factory.
- Tests inspect `Request.method`, `Request.url`, body via `await req.text()` to verify wire format.

## Risks

- **EventSource missing in bun test** — explicitly out of scope per design; verified by P5.2 manual smoke.
- **Wire-shape drift between Go and TS** — single source of truth is the P5.0 spec. Adapter and server tests both pin the JSON shape; if either drifts, the contract test catches it. P5.2 manual smoke is the integration check.
- **Fetch in workers vs main thread** — adapter only runs in the iframe (browser main thread); fetch is fine.
- **ProtocolError code typing** — current `ErrorCode` union covers `NOT_FOUND`, `OUT_OF_RANGE`, `INVALID`, `ALREADY_EXISTS`, `INTERNAL`, `VERSION_CONFLICT`, `PROTOCOL_UNSUPPORTED`. Server may return any string in the error envelope; adapter casts to `ErrorCode` and lets `errorToast`'s default-case handle unknowns.
- **Subscription churn opening/closing EventSource rapidly** — opening EventSource isn't free. Acceptable for now: subscribe is called once per active board view in the renderer; not a hot path.

## Open questions

None blocking. Pre-decided:
- `fetch` + `EventSource`, both browser-native.
- Constructor: `{ baseUrl, fetch? }`.
- Single shared EventSource, lazy open + idle close.
- HTTP-only unit tests; SSE deferred to P5.2 smoke.
- `settings.updated` plumbed but unsubscribed.

## Dependencies on prior work

- P5.0: `/api/v1/*` JSON + SSE surface.
- Existing `BackendAdapter` interface (P3) — adapter implements it verbatim.
- `ProtocolError` from `web/shared/src/protocol.ts`.
- Existing types: `Board`, `BoardSummary`, `BoardSettings`, `MutationOp`, `ResolvedSettings`, `WorkspaceInfo`, `Subscription`, `BoardUpdateHandler`.

## Dependencies on later work

- P5.2 swaps `LocalAdapter` for `ServerAdapter` in the shell when a flag is set.
