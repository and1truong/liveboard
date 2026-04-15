import { describe, expect, it } from 'bun:test'
import { LocalAdapter } from './local.js'
import { MemoryStorage } from './local-storage-driver.js'

describe('LocalAdapter settings round-trip', () => {
  it('getSettings reflects putBoardSettings', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    await a.putBoardSettings('welcome', { show_checkbox: false, card_display_mode: 'compact' })
    const s = await a.getSettings('welcome')
    expect(s.show_checkbox).toBe(false)
    expect(s.card_display_mode).toBe('compact')
    // Other defaults preserved.
    expect(s.expand_columns).toBe(false)
    expect(s.view_mode).toBe('board')
  })

  it('getSettings returns defaults for a fresh board', async () => {
    const a = new LocalAdapter(new MemoryStorage())
    const s = await a.getSettings('welcome')
    expect(s.show_checkbox).toBe(true)
    expect(s.card_display_mode).toBe('normal')
  })
})
