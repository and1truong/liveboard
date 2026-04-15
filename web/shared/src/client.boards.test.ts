import { describe, expect, it } from 'bun:test'
import { Broker } from './broker.js'
import { Client } from './client.js'
import { LocalAdapter } from './adapters/local.js'
import { MemoryStorage } from './adapters/local-storage-driver.js'
import { createMemoryPair } from './transport.js'

async function setup(): Promise<{ client: Client }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  return { client }
}

describe('Client board CRUD', () => {
  it('createBoard round-trips and returns BoardSummary', async () => {
    const { client } = await setup()
    const summary = await client.createBoard('My Board')
    expect(summary.id).toBe('my-board')
    expect(summary.name).toBe('My Board')
  })

  it('createBoard surfaces ALREADY_EXISTS as ProtocolError', async () => {
    const { client } = await setup()
    await client.createBoard('Foo')
    try {
      await client.createBoard('Foo')
      throw new Error('should have thrown')
    } catch (e) {
      expect((e as { code: string }).code).toBe('ALREADY_EXISTS')
    }
  })

  it('renameBoard returns new BoardSummary with new id', async () => {
    const { client } = await setup()
    await client.createBoard('Foo')
    const renamed = await client.renameBoard('foo', 'Bar')
    expect(renamed.id).toBe('bar')
    expect(renamed.name).toBe('Bar')
  })

  it('renameBoard surfaces NOT_FOUND', async () => {
    const { client } = await setup()
    try {
      await client.renameBoard('nope', 'X')
      throw new Error('should have thrown')
    } catch (e) {
      expect((e as { code: string }).code).toBe('NOT_FOUND')
    }
  })

  it('deleteBoard removes from listBoards', async () => {
    const { client } = await setup()
    await client.createBoard('Foo')
    await client.deleteBoard('foo')
    const list = await client.listBoards()
    expect(list.map((s) => s.id)).not.toContain('foo')
  })

  it('emits board.list.updated event to subscribers', async () => {
    const { client } = await setup()
    let count = 0
    client.on('board.list.updated', () => { count++ })
    await client.createBoard('Foo')
    await client.renameBoard('foo', 'Bar')
    await client.deleteBoard('bar')
    await new Promise((r) => setTimeout(r, 10))
    expect(count).toBeGreaterThanOrEqual(3)
  })
})
