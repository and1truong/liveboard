import { describe, expect, it } from 'bun:test'
import { render } from '@testing-library/react'
import { Column } from './Column.js'

describe('Column', () => {
  it('renders column name and card count', () => {
    const { getByText } = render(
      <Column column={{ name: 'Todo', cards: [{ title: 'a' }, { title: 'b' }] }} />,
    )
    expect(getByText('Todo')).toBeDefined()
    expect(getByText('2')).toBeDefined()
  })
  it('renders all cards', () => {
    const { getByText } = render(
      <Column column={{ name: 'x', cards: [{ title: 'A' }, { title: 'B' }] }} />,
    )
    expect(getByText('A')).toBeDefined()
    expect(getByText('B')).toBeDefined()
  })
  it('handles empty cards array', () => {
    const { getByText } = render(<Column column={{ name: 'Empty', cards: [] }} />)
    expect(getByText('0')).toBeDefined()
  })
})
