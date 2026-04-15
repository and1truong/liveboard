import { describe, expect, it } from 'bun:test'
import { moveColumnTarget } from './moveColumn.js'

describe('moveColumnTarget', () => {
  const names = ['A', 'B', 'C', 'D']

  it('move left from index 1 → first position', () => {
    expect(moveColumnTarget(names, 1, 'left')).toBe('')
  })
  it('move left from index 2 → after col 0', () => {
    expect(moveColumnTarget(names, 2, 'left')).toBe('A')
  })
  it('move left from index 3 → after col 1', () => {
    expect(moveColumnTarget(names, 3, 'left')).toBe('B')
  })
  it('move right from index 0 → after col 1', () => {
    expect(moveColumnTarget(names, 0, 'right')).toBe('B')
  })
  it('move right from index 2 → after col 3', () => {
    expect(moveColumnTarget(names, 2, 'right')).toBe('D')
  })
  it('returns null for move-left at index 0 (disabled edge)', () => {
    expect(moveColumnTarget(names, 0, 'left')).toBeNull()
  })
  it('returns null for move-right at last index (disabled edge)', () => {
    expect(moveColumnTarget(names, 3, 'right')).toBeNull()
  })
})
