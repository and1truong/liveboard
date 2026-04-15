import { describe, expect, it } from 'bun:test'
import { render } from '@testing-library/react'
import { EmptyState } from './EmptyState.js'

describe('EmptyState', () => {
  it('renders title', () => {
    const { getByText } = render(<EmptyState title="Nothing here" />)
    expect(getByText('Nothing here')).toBeDefined()
  })
  it('renders detail when provided', () => {
    const { getByText } = render(<EmptyState title="x" detail="some detail" />)
    expect(getByText('some detail')).toBeDefined()
  })
})
