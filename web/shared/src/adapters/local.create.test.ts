import { describe, expect, it } from 'bun:test'
import { LocalAdapter } from './local.js'
import { MemoryStorage } from './local-storage-driver.js'
import { ProtocolError } from '../protocol.js'

describe('LocalAdapter.createBoard', () => {
  it('returns BoardSummary with slugified id', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const summary = await a.createBoard('My Board')
    expect(summary.id).toBe('my-board')
    expect(summary.name).toBe('My Board')
    expect(summary.version).toBe(1)
  })

  it('persists the board with a default Todo column', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    const board = await a.getBoard('foo')
    expect(board.name).toBe('Foo')
    expect(board.version).toBe(1)
    expect(board.columns?.[0]?.name).toBe('Todo')
    expect(board.columns?.[0]?.cards).toEqual([])
  })

  it('appends the new board to listBoards', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    const list = await a.listBoards()
    const ids = list.map((s) => s.id)
    expect(ids).toContain('welcome')
    expect(ids).toContain('foo')
  })

  it('rejects empty name as INVALID', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await expect(a.createBoard('   ')).rejects.toMatchObject({ code: 'INVALID' })
  })

  it('rejects name that slugifies to empty as INVALID', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await expect(a.createBoard('!!!')).rejects.toMatchObject({ code: 'INVALID' })
  })

  it('rejects collision as ALREADY_EXISTS', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Foo')
    await expect(a.createBoard('Foo')).rejects.toMatchObject({ code: 'ALREADY_EXISTS' })
  })

  it('invokes onBoardListUpdate handler', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    let calls = 0
    a.onBoardListUpdate(() => { calls++ })
    await a.createBoard('Foo')
    expect(calls).toBe(1)
  })

  it('errors are ProtocolError instances', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    try {
      await a.createBoard('')
      throw new Error('should have thrown')
    } catch (e) {
      expect(e).toBeInstanceOf(ProtocolError)
    }
  })
})
