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
