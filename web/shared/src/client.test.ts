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
    await new Promise((r) => setTimeout(r, 5))
    expect(seen).toEqual([2])
  })
})
