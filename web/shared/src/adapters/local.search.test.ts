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
    expect(hits[0]!.boardId).toBe('foo')
    expect(hits[0]!.cardTitle).toBe('alpha-token')
    expect(hits[0]!.colIdx).toBe(0)
    expect(hits[0]!.cardIdx).toBe(0)
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

  it('search returns cardId on hits', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    await a.mutateBoard('foo', 1, { type: 'add_card', column: 'Todo', title: 'alpha-token' })
    const hits = await a.search('alpha')
    expect(hits[0]!.cardId).not.toBe('')
  })
})

describe('LocalAdapter.backlinks', () => {
  it('returns cards that link to the given cardId', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Target')
    await a.mutateBoard('target', 1, { type: 'add_card', column: 'Todo', title: 'tgt' })
    const tgtBoard = await a.getBoard('target')
    const tgtId = tgtBoard.columns![0]!.cards![0]!.id!
    expect(tgtId).toBeTruthy()

    await a.createBoard('Source')
    await a.mutateBoard('source', 1, { type: 'add_card', column: 'Todo', title: 'src' })
    await a.mutateBoard('source', 2, {
      type: 'edit_card', col_idx: 0, card_idx: 0,
      title: 'src', body: '', tags: [], links: [`target:${tgtId}`],
      priority: '', due: '', assignee: '',
    })

    const back = await a.backlinks(tgtId)
    expect(back.length).toBe(1)
    expect(back[0]!.boardId).toBe('source')
  })
})
