import { describe, expect, it, afterEach } from 'bun:test'
import { ALPHABET, newCardId, _setGenerator, _resetGenerator } from './cardid.js'

describe('newCardId', () => {
  afterEach(() => _resetGenerator())

  it('returns 10 chars from the alphabet', () => {
    const id = newCardId()
    expect(id).toHaveLength(10)
    for (const ch of id) expect(ALPHABET).toContain(ch)
  })

  it('produces no duplicates in 10k draws', () => {
    const seen = new Set<string>()
    for (let i = 0; i < 10000; i++) {
      const id = newCardId()
      expect(seen.has(id)).toBe(false)
      seen.add(id)
    }
  })

  it('can be overridden for tests', () => {
    _setGenerator(() => 'FIXED00001')
    expect(newCardId()).toBe('FIXED00001')
    _resetGenerator()
    expect(newCardId()).not.toBe('FIXED00001')
  })
})
