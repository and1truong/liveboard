import { describe, expect, it } from 'bun:test'
import type { Card } from '@shared/types.js'
import { activeFilterCount, EMPTY_FILTER, filterCard } from './cardFilter.js'

const card = (overrides: Partial<Card> = {}): Card => ({
  title: 'Buy milk',
  body: '',
  tags: [],
  completed: false,
  ...overrides,
})

describe('filterCard', () => {
  it('passes everything through with empty filter', () => {
    expect(filterCard(card(), EMPTY_FILTER)).toBe(true)
    expect(filterCard(card({ completed: true }), EMPTY_FILTER)).toBe(true)
  })

  it('hides completed cards when hideCompleted is on', () => {
    const f = { ...EMPTY_FILTER, hideCompleted: true }
    expect(filterCard(card({ completed: true }), f)).toBe(false)
    expect(filterCard(card({ completed: false }), f)).toBe(true)
  })

  it('matches text query against title, body, tags, assignee', () => {
    const f = { ...EMPTY_FILTER, query: 'foo' }
    expect(filterCard(card({ title: 'foo bar' }), f)).toBe(true)
    expect(filterCard(card({ body: 'lorem foo' }), f)).toBe(true)
    expect(filterCard(card({ tags: ['foo'] }), f)).toBe(true)
    expect(filterCard(card({ assignee: 'foo' }), f)).toBe(true)
    expect(filterCard(card({ title: 'unrelated' }), f)).toBe(false)
  })

  it('text query is case-insensitive and trimmed', () => {
    const f = { ...EMPTY_FILTER, query: '  FOO  ' }
    expect(filterCard(card({ title: 'foo' }), f)).toBe(true)
  })

  it('requires every selected tag (AND semantics)', () => {
    const f = { ...EMPTY_FILTER, tags: ['backend', 'urgent'] }
    expect(filterCard(card({ tags: ['backend', 'urgent', 'misc'] }), f)).toBe(true)
    expect(filterCard(card({ tags: ['backend'] }), f)).toBe(false)
    expect(filterCard(card({ tags: [] }), f)).toBe(false)
  })

  it('combines all three predicates', () => {
    const f = { query: 'milk', tags: ['groceries'], hideCompleted: true }
    expect(filterCard(card({ title: 'milk', tags: ['groceries'] }), f)).toBe(true)
    expect(filterCard(card({ title: 'milk', tags: ['groceries'], completed: true }), f)).toBe(false)
    expect(filterCard(card({ title: 'milk', tags: [] }), f)).toBe(false)
    expect(filterCard(card({ title: 'beer', tags: ['groceries'] }), f)).toBe(false)
  })
})

describe('activeFilterCount', () => {
  it('counts each active dimension', () => {
    expect(activeFilterCount(EMPTY_FILTER)).toBe(0)
    expect(activeFilterCount({ ...EMPTY_FILTER, query: 'x' })).toBe(1)
    expect(activeFilterCount({ ...EMPTY_FILTER, tags: ['a', 'b'] })).toBe(2)
    expect(activeFilterCount({ ...EMPTY_FILTER, hideCompleted: true })).toBe(1)
    expect(activeFilterCount({ query: 'x', tags: ['a'], hideCompleted: true })).toBe(3)
  })

  it('ignores whitespace-only query', () => {
    expect(activeFilterCount({ ...EMPTY_FILTER, query: '   ' })).toBe(0)
  })
})
