import { describe, expect, it } from 'bun:test'
import { LocalAdapter } from './local.js'
import { MemoryStorage } from './local-storage-driver.js'
import type { MutationOp } from '../types.js'

describe('LocalAdapter seed + reads', () => {
  it('seeds workspace on first construction', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const ws = await a.getWorkspaceInfo()
    expect(ws.name).toBe('Demo')
    expect(ws.boardCount).toBe(1)
  })

  it('listBoards returns the welcome board summary', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const list = await a.listBoards()
    expect(list).toHaveLength(1)
    expect(list[0]?.id).toBe('welcome')
    expect(list[0]?.name).toBe('Welcome')
  })

  it('getBoard returns full board', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const b = await a.getBoard('welcome')
    expect(b.name).toBe('Welcome')
    expect(b.columns?.length).toBe(3)
  })

  it('getBoard on missing id throws NOT_FOUND', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await expect(a.getBoard('nope')).rejects.toMatchObject({ code: 'NOT_FOUND' })
  })

  it('second construction on same storage does not reseed', async () => {
    const storage = new MemoryStorage()
    new LocalAdapter(storage)
    const raw = storage.get('liveboard:v1:board:welcome')!
    const b = JSON.parse(raw)
    b.name = 'Changed'
    storage.set('liveboard:v1:board:welcome', JSON.stringify(b))
    const a2 = new LocalAdapter(storage)
    expect((await a2.getBoard('welcome')).name).toBe('Changed')
  })
})

describe('LocalAdapter mutateBoard', () => {
  it('applies op, bumps version, persists', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const op: MutationOp = { type: 'add_card', column: 'Todo', title: 'x' }
    const next = await a.mutateBoard('welcome', 1, op)
    expect(next.version).toBe(2)
    const again = await a.getBoard('welcome')
    expect(again.version).toBe(2)
    expect(again.columns?.[0]?.cards.some((c) => c.title === 'x')).toBe(true)
  })

  it('throws VERSION_CONFLICT on stale clientVersion', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const op: MutationOp = { type: 'add_card', column: 'Todo', title: 'x' }
    await expect(a.mutateBoard('welcome', 42, op)).rejects.toMatchObject({
      code: 'VERSION_CONFLICT',
    })
  })

  it('clientVersion < 0 bypasses conflict check', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const op: MutationOp = { type: 'add_card', column: 'Todo', title: 'x' }
    const r = await a.mutateBoard('welcome', -1, op)
    expect(r.version).toBe(2)
  })

  it('propagates applyOp errors with mapped code', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const op: MutationOp = { type: 'add_card', column: 'Missing', title: 'x' }
    await expect(a.mutateBoard('welcome', 1, op)).rejects.toMatchObject({ code: 'NOT_FOUND' })
  })

  it('mutateBoard add_card → getBoard returns card with non-empty id', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.mutateBoard('welcome', -1, { type: 'add_card', column: 'Todo', title: 'x' })
    const b = await a.getBoard('welcome')
    const card = b.columns?.[0]?.cards.find((c) => c.title === 'x')
    expect(card).toBeDefined()
    expect(card?.id).toBeDefined()
    expect(card?.id?.length).toBe(10)
  })
})

describe('LocalAdapter subscribe', () => {
  it('fires handler on mutateBoard', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const seen: Array<{ boardId: string; version: number }> = []
    a.subscribe('welcome', (p) => seen.push(p))
    await a.mutateBoard('welcome', 1, { type: 'add_card', column: 'Todo', title: 'x' })
    expect(seen).toEqual([{ boardId: 'welcome', version: 2 }])
  })

  it('close() stops delivery', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const seen: number[] = []
    const sub = a.subscribe('welcome', (p) => seen.push(p.version))
    sub.close()
    await a.mutateBoard('welcome', 1, { type: 'add_card', column: 'Todo', title: 'x' })
    expect(seen).toHaveLength(0)
  })
})

describe('LocalAdapter settings', () => {
  it('getSettings returns defaults for an existing board', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const s = await a.getSettings('welcome')
    expect(s.view_mode).toBe('board')
  })

  it('getSettings on missing board throws NOT_FOUND', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await expect(a.getSettings('nope')).rejects.toMatchObject({ code: 'NOT_FOUND' })
  })

  it('putBoardSettings merges patch and bumps version', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.putBoardSettings('welcome', { card_display_mode: 'compact' })
    const b = await a.getBoard('welcome')
    expect(b.version).toBe(2)
    expect(b.settings?.card_display_mode).toBe('compact')
  })
})
