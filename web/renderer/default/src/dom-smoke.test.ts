import { describe, expect, it } from 'bun:test'

describe('happy-dom', () => {
  it('provides document global', () => {
    const el = document.createElement('div')
    el.textContent = 'hi'
    expect(el.textContent).toBe('hi')
  })
})
