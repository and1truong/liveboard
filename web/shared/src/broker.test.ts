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
