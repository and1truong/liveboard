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
