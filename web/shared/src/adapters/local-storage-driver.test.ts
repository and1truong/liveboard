import { describe, expect, it } from 'bun:test'
import { MemoryStorage } from './local-storage-driver.js'

describe('MemoryStorage', () => {
  it('stores and retrieves values', () => {
    const s = new MemoryStorage()
    s.set('a', '1')
    expect(s.get('a')).toBe('1')
  })

  it('returns null for missing keys', () => {
    expect(new MemoryStorage().get('x')).toBeNull()
  })

  it('lists keys by prefix', () => {
    const s = new MemoryStorage()
    s.set('lb:board:a', '1')
    s.set('lb:board:b', '2')
    s.set('other:c', '3')
    expect(s.keys('lb:board:').sort()).toEqual(['lb:board:a', 'lb:board:b'])
  })

  it('remove deletes the key', () => {
    const s = new MemoryStorage()
    s.set('a', '1')
    s.remove('a')
    expect(s.get('a')).toBeNull()
  })
})
