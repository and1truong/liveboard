import { describe, expect, it } from 'bun:test'
import { LocalAdapter } from './local.js'
import { MemoryStorage } from './local-storage-driver.js'

describe('LocalAdapter.renameBoard', () => {
  it('moves board to new id and returns new BoardSummary', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    const summary = await a.renameBoard('foo', 'Bar')
    expect(summary.id).toBe('bar')
    expect(summary.name).toBe('Bar')
    expect(summary.version).toBeGreaterThanOrEqual(2)
    const list = (await a.listBoards()).map((s) => s.id)
    expect(list).toContain('bar')
    expect(list).not.toContain('foo')
  })

  it('preserves position in workspace boardIds', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    await a.createBoard('Baz')
    const beforeIds = (await a.listBoards()).map((s) => s.id)
    const fooIdx = beforeIds.indexOf('foo')
    await a.renameBoard('foo', 'Quux')
    const afterIds = (await a.listBoards()).map((s) => s.id)
    expect(afterIds[fooIdx]).toBe('quux')
  })

  it('in-place name change keeps id when slug unchanged', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('foo')
    const summary = await a.renameBoard('foo', 'FOO')
    expect(summary.id).toBe('foo')
    expect(summary.name).toBe('FOO')
  })

  it('rejects missing source as NOT_FOUND', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await expect(a.renameBoard('nope', 'X')).rejects.toMatchObject({ code: 'NOT_FOUND' })
  })

  it('rejects new id collision as ALREADY_EXISTS', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    await a.createBoard('Bar')
    await expect(a.renameBoard('foo', 'Bar')).rejects.toMatchObject({ code: 'ALREADY_EXISTS' })
  })

  it('rejects empty new name as INVALID', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    await expect(a.renameBoard('foo', '   ')).rejects.toMatchObject({ code: 'INVALID' })
  })

  it('invokes onBoardListUpdate handler', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    let calls = 0
    a.onBoardListUpdate(() => { calls++ })
    await a.renameBoard('foo', 'Bar')
    expect(calls).toBe(1)
  })
})
