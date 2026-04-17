import { describe, expect, it } from 'bun:test'
import { LocalAdapter } from './local.js'
import { MemoryStorage } from './local-storage-driver.js'

describe('LocalAdapter folders', () => {
  it('creates a board inside a folder and exposes it via listBoards', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const s = await a.createBoard('Ideas', 'Work')
    expect(s.id).toBe('Work/ideas')
    expect(s.folder).toBe('Work')

    const list = await a.listBoards()
    const row = list.find((b) => b.id === 'Work/ideas')
    expect(row).toBeDefined()
    expect(row!.folder).toBe('Work')
  })

  it('listFolders returns unique folders derived from board ids and registered empty folders', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Ideas', 'Work')
    await a.createFolder('Personal')
    const folders = await a.listFolders()
    expect(folders).toEqual(['Personal', 'Work'])
  })

  it('renameFolder moves every nested board and rewrites pins', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Ideas', 'Work')
    await a.togglePin('Work/ideas')
    await a.renameFolder('Work', 'Job')

    const list = await a.listBoards()
    const ids = list.map((b) => b.id)
    expect(ids).toContain('Job/ideas')
    expect(ids).not.toContain('Work/ideas')

    const settings = await a.getAppSettings()
    expect(settings.pinned_boards).toEqual(['Job/ideas'])
  })

  it('deleteFolder rejects non-empty folders', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Ideas', 'Work')
    await expect(a.deleteFolder('Work')).rejects.toMatchObject({ code: 'INVALID' })
  })

  it('deleteFolder removes empty folders from the registry', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createFolder('Tmp')
    expect(await a.listFolders()).toContain('Tmp')
    await a.deleteFolder('Tmp')
    expect(await a.listFolders()).not.toContain('Tmp')
  })

  it('renameBoard with folder arg relocates the board', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.createBoard('Ideas')
    const s = await a.renameBoard('ideas', 'Ideas', 'Work')
    expect(s.id).toBe('Work/ideas')

    const list = await a.listBoards()
    const ids = list.map((b) => b.id)
    expect(ids).toContain('Work/ideas')
    expect(ids).not.toContain('ideas')
  })
})
