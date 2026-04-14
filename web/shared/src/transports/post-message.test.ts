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
