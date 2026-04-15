import { describe, expect, it } from 'bun:test'
import { render } from '@testing-library/react'
import { Card } from './Card.js'

describe('Card', () => {
  it('renders title', () => {
    const { getByText } = render(<Card card={{ title: 'Hello' }} />)
    expect(getByText('Hello')).toBeDefined()
  })
  it('renders tags as pills', () => {
    const { getByText } = render(<Card card={{ title: 'x', tags: ['a', 'b'] }} />)
    expect(getByText('a')).toBeDefined()
    expect(getByText('b')).toBeDefined()
  })
  it('shows priority dot when priority set', () => {
    const { getByLabelText } = render(<Card card={{ title: 'x', priority: 'high' }} />)
    expect(getByLabelText('priority high')).toBeDefined()
  })
  it('strikes through completed cards', () => {
    const { getByText } = render(<Card card={{ title: 'done', completed: true }} />)
    expect(getByText('done').className).toContain('line-through')
  })
})
