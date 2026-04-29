import { describe, it, expect } from 'bun:test'
import 'fake-indexeddb/auto'
import { putBlob, getBlob, deleteBlob } from './local-attachments.js'

describe('local-attachments', () => {
  it('hashes content and stores by hash', async () => {
    const blob = new Blob(['hello'], { type: 'text/plain' })
    const desc = await putBlob(blob, 'h.txt')
    // sha256("hello") = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
    expect(desc.h).toBe('2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824.txt')
    expect(desc.n).toBe('h.txt')
    expect(desc.s).toBe(5)
    expect(desc.m).toBe('text/plain')

    const got = await getBlob(desc.h)
    expect(got).not.toBeNull()
    expect(await got!.text()).toBe('hello')
  })

  it('returns null for missing hash', async () => {
    const got = await getBlob('does-not-exist.bin')
    expect(got).toBeNull()
  })

  it('falls back to application/octet-stream when blob has no type', async () => {
    const blob = new Blob(['bytes']) // no type
    const desc = await putBlob(blob, 'mystery')
    expect(desc.m).toBe('application/octet-stream')
  })

  it('deleteBlob removes the entry; subsequent get returns null', async () => {
    const blob = new Blob(['gone'], { type: 'text/plain' })
    const desc = await putBlob(blob, 'gone.txt')
    await deleteBlob(desc.h)
    expect(await getBlob(desc.h)).toBeNull()
  })
})
