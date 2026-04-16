import { describe, expect, it } from 'bun:test'
import { act, renderHook } from '@testing-library/react'
import type { Column } from '@shared/types.js'
import {
  FocusedColumnProvider,
  useFocusedColumn,
} from './FocusedColumnContext.js'

const cols: Column[] = [
  { name: 'Todo', cards: [] },
  { name: 'Doing', cards: [] },
  { name: 'Done', cards: [] },
]

function wrapper(columns: Column[], active: string | null = 'b1') {
  return function Wrap({ children }: { children: React.ReactNode }) {
    return (
      <FocusedColumnProvider columns={columns} active={active}>
        {children}
      </FocusedColumnProvider>
    )
  }
}

describe('FocusedColumnContext', () => {
  it('starts with focused=null', () => {
    const { result } = renderHook(() => useFocusedColumn(), {
      wrapper: wrapper(cols),
    })
    expect(result.current.focused).toBeNull()
  })

  it('setFocused updates the value', () => {
    const { result } = renderHook(() => useFocusedColumn(), {
      wrapper: wrapper(cols),
    })
    act(() => result.current.setFocused('Todo'))
    expect(result.current.focused).toBe('Todo')
    act(() => result.current.setFocused(null))
    expect(result.current.focused).toBeNull()
  })

  it('clears focused when active board changes', () => {
    let active: string | null = 'b1'
    const { result, rerender } = renderHook(() => useFocusedColumn(), {
      wrapper: ({ children }: { children: React.ReactNode }) => (
        <FocusedColumnProvider columns={cols} active={active}>
          {children}
        </FocusedColumnProvider>
      ),
    })
    act(() => result.current.setFocused('Todo'))
    expect(result.current.focused).toBe('Todo')
    active = 'b2'
    rerender()
    expect(result.current.focused).toBeNull()
  })
})
