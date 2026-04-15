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
