import { describe, expect, it } from 'bun:test'
import { act, renderHook } from '@testing-library/react'
import { ActiveBoardProvider, useActiveBoard } from './ActiveBoardContext.js'

describe('ActiveBoardContext', () => {
  it('starts with active=null', () => {
    const { result } = renderHook(() => useActiveBoard(), { wrapper: ActiveBoardProvider })
    expect(result.current.active).toBeNull()
  })

  it('setActive updates the context', () => {
    const { result } = renderHook(() => useActiveBoard(), { wrapper: ActiveBoardProvider })
    act(() => result.current.setActive('foo'))
    expect(result.current.active).toBe('foo')
    act(() => result.current.setActive(null))
    expect(result.current.active).toBeNull()
  })

  it('throws when used outside provider', () => {
    expect(() => renderHook(() => useActiveBoard())).toThrow()
  })
})
