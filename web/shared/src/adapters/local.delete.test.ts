import { describe, expect, it } from 'bun:test'
import { LocalAdapter } from './local.js'
import { MemoryStorage } from './local-storage-driver.js'

describe('LocalAdapter.deleteBoard', () => {
  it('removes the board from listBoards', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    await a.deleteBoard('foo')
    const ids = (await a.listBoards()).map((s) => s.id)
    expect(ids).not.toContain('foo')
  })

  it('subsequent getBoard throws NOT_FOUND', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    await a.deleteBoard('foo')
    await expect(a.getBoard('foo')).rejects.toMatchObject({ code: 'NOT_FOUND' })
  })

  it('rejects missing source as NOT_FOUND', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await expect(a.deleteBoard('nope')).rejects.toMatchObject({ code: 'NOT_FOUND' })
  })

  it('invokes onBoardListUpdate handler', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    let calls = 0
    a.onBoardListUpdate(() => { calls++ })
    await a.deleteBoard('foo')
    expect(calls).toBe(1)
  })

  it('returns void', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    const result = await a.deleteBoard('foo')
    expect(result).toBeUndefined()
  })
})
