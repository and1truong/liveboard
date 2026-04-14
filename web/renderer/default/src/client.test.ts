import { describe, expect, it } from 'bun:test'
import { QueryClient } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'

describe('client event → query invalidation', () => {
  it('invalidates board query on board.updated', async () => {
    const [iframeT, shellT] = createMemoryPair()
    const adapter = new LocalAdapter(new MemoryStorage())
    new Broker(shellT, adapter, { shellVersion: 't' })
    const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })

    const qc = new QueryClient()
    // Seed a query so invalidateQueries has something to mark stale
    qc.setQueryData(['board', 'welcome'], { id: 'welcome' })

    let invalidated = 0
    qc.getQueryCache().subscribe((ev) => {
      if (ev.type === 'updated' && ev.action.type === 'invalidate') invalidated++
    })
    client.on('board.updated', ({ boardId }) => {
      void qc.invalidateQueries({ queryKey: ['board', boardId] })
    })

    await client.ready()
    await client.subscribe('welcome')
    await adapter.mutateBoard('welcome', 1, { type: 'add_card', column: 'Todo', title: 'x' })
    // Allow microtask queue to drain: transport queueMicrotask chain needs a few ticks
    await new Promise((r) => setTimeout(r, 50))
    expect(invalidated).toBeGreaterThan(0)
  })
})
