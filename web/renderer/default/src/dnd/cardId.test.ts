import { describe, expect, it } from 'bun:test'
import {
  encodeCardId,
  decodeCardId,
  encodeColumnId,
  decodeColumnId,
  encodeColumnEndId,
  decodeColumnEndId,
} from './cardId.js'

describe('cardId', () => {
  it('encodes card with col_idx + card_idx', () => {
    expect(encodeCardId(0, 0)).toBe('card:0:0')
    expect(encodeCardId(2, 7)).toBe('card:2:7')
  })

  it('decodes card id', () => {
    expect(decodeCardId('card:2:7')).toEqual({ colIdx: 2, cardIdx: 7 })
  })

  it('returns null for non-card id', () => {
    expect(decodeCardId('column:Todo')).toBeNull()
    expect(decodeCardId('garbage')).toBeNull()
    expect(decodeCardId('card:2')).toBeNull()
  })

  it('encodes column with name', () => {
    expect(encodeColumnId('Todo')).toBe('column:Todo')
    expect(encodeColumnId('In Progress')).toBe('column:In Progress')
  })

  it('decodes column id', () => {
    expect(decodeColumnId('column:Todo')).toBe('Todo')
    expect(decodeColumnId('column:In Progress')).toBe('In Progress')
  })

  it('returns null for non-column id', () => {
    expect(decodeColumnId('card:0:0')).toBeNull()
    expect(decodeColumnId('garbage')).toBeNull()
  })

  it('encodes/decodes column-end with name', () => {
    expect(encodeColumnEndId('Todo')).toBe('colend:Todo')
    expect(decodeColumnEndId('colend:Todo')).toBe('Todo')
    expect(decodeColumnEndId('column:Todo')).toBeNull()
    expect(decodeColumnEndId('garbage')).toBeNull()
  })
})
