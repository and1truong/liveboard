import { describe, expect, it } from 'bun:test'
import { LocalAdapter } from './local.js'
import { MemoryStorage } from './local-storage-driver.js'

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
