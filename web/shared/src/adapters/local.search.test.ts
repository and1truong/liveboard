import { describe, expect, it } from 'bun:test'
import { LocalAdapter } from './local.js'
import { MemoryStorage } from './local-storage-driver.js'

describe('LocalAdapter.search', () => {
  it('returns empty for empty query', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    expect(await a.search('')).toEqual([])
    expect(await a.search('   ')).toEqual([])
  })

  it('substring matches across title/body/tags', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    await a.mutateBoard('foo', 1, { type: 'add_card', column: 'Todo', title: 'alpha-token' })
    const hits = await a.search('alpha')
    expect(hits.length).toBe(1)
    expect(hits[0].boardId).toBe('foo')
    expect(hits[0].cardTitle).toBe('alpha-token')
    expect(hits[0].colIdx).toBe(0)
    expect(hits[0].cardIdx).toBe(0)
  })

  it('respects limit', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    for (let i = 0; i < 5; i++) {
      await a.mutateBoard('foo', i + 1, { type: 'add_card', column: 'Todo', title: `xx ${i}` })
    }
    const hits = await a.search('xx', 2)
    expect(hits.length).toBe(2)
  })
})
