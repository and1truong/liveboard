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

  it('clears focused when the focused column is removed', () => {
    let columns: Column[] = cols
    const { result, rerender } = renderHook(() => useFocusedColumn(), {
      wrapper: ({ children }: { children: React.ReactNode }) => (
        <FocusedColumnProvider columns={columns} active="b1">
          {children}
        </FocusedColumnProvider>
      ),
    })
    act(() => result.current.setFocused('Doing'))
    expect(result.current.focused).toBe('Doing')
    columns = [cols[0]!, cols[2]!]
    rerender()
    expect(result.current.focused).toBeNull()
  })

  it('clears focused when the focused column is renamed', () => {
    let columns: Column[] = cols
    const { result, rerender } = renderHook(() => useFocusedColumn(), {
      wrapper: ({ children }: { children: React.ReactNode }) => (
        <FocusedColumnProvider columns={columns} active="b1">
          {children}
        </FocusedColumnProvider>
      ),
    })
    act(() => result.current.setFocused('Doing'))
    columns = [cols[0]!, { name: 'In Progress', cards: [] }, cols[2]!]
    rerender()
    expect(result.current.focused).toBeNull()
  })

  it('Escape keydown clears focused', () => {
    const { result } = renderHook(() => useFocusedColumn(), {
      wrapper: wrapper(cols),
    })
    act(() => result.current.setFocused('Todo'))
    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    })
    expect(result.current.focused).toBeNull()
  })

  it('Escape is ignored while an input is focused', () => {
    const input = document.createElement('input')
    document.body.appendChild(input)
    input.focus()
    try {
      const { result } = renderHook(() => useFocusedColumn(), {
        wrapper: wrapper(cols),
      })
      act(() => result.current.setFocused('Todo'))
      act(() => {
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
      })
      expect(result.current.focused).toBe('Todo')
    } finally {
      input.remove()
    }
  })

  it('Escape is ignored while a select is focused', () => {
    const select = document.createElement('select')
    document.body.appendChild(select)
    select.focus()
    try {
      const { result } = renderHook(() => useFocusedColumn(), {
        wrapper: wrapper(cols),
      })
      act(() => result.current.setFocused('Todo'))
      act(() => {
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
      })
      expect(result.current.focused).toBe('Todo')
    } finally {
      document.body.removeChild(select)
    }
  })

  it('Escape is ignored while a Radix dialog is open', () => {
    const dialog = document.createElement('div')
    dialog.setAttribute('role', 'dialog')
    dialog.setAttribute('data-state', 'open')
    document.body.appendChild(dialog)
    try {
      const { result } = renderHook(() => useFocusedColumn(), {
        wrapper: wrapper(cols),
      })
      act(() => result.current.setFocused('Todo'))
      act(() => {
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
      })
      expect(result.current.focused).toBe('Todo')
    } finally {
      dialog.remove()
    }
  })

  it('does not handle Escape when no column is focused', () => {
    const { result } = renderHook(() => useFocusedColumn(), {
      wrapper: wrapper(cols),
    })
    // No-op should not throw.
    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    })
    expect(result.current.focused).toBeNull()
  })
})
