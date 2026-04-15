import { describe, expect, it } from 'bun:test'
import { act, renderHook } from '@testing-library/react'
import type { Column } from '@shared/types.js'
import { BoardFocusProvider, useBoardFocus } from './BoardFocusContext.js'

const cols3x3: Column[] = [
  { name: 'A', cards: [{ title: 'a0' }, { title: 'a1' }, { title: 'a2' }] },
  { name: 'B', cards: [{ title: 'b0' }] },
  { name: 'C', cards: [{ title: 'c0' }, { title: 'c1' }] },
]

const colsWithEmpty: Column[] = [
  { name: 'A', cards: [{ title: 'a0' }] },
  { name: 'Empty', cards: [] },
  { name: 'C', cards: [{ title: 'c0' }] },
]

function wrapper(columns: Column[]) {
  return function Wrap({ children }: { children: React.ReactNode }) {
    return <BoardFocusProvider columns={columns}>{children}</BoardFocusProvider>
  }
}

describe('BoardFocusContext.move', () => {
  it('starts with focused=null', () => {
    const { result } = renderHook(() => useBoardFocus(), { wrapper: wrapper(cols3x3) })
    expect(result.current.focused).toBeNull()
  })

  it('move from null jumps to (0,0)', () => {
    const { result } = renderHook(() => useBoardFocus(), { wrapper: wrapper(cols3x3) })
    act(() => result.current.move('down'))
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 0 })
  })

  it('move down increments cardIdx and stops at last', () => {
    const { result } = renderHook(() => useBoardFocus(), { wrapper: wrapper(cols3x3) })
    act(() => result.current.setFocused({ colIdx: 0, cardIdx: 0 }))
    act(() => result.current.move('down'))
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 1 })
    act(() => result.current.move('down'))
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 2 })
    act(() => result.current.move('down'))
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 2 })
  })

  it('move up decrements and stops at 0', () => {
    const { result } = renderHook(() => useBoardFocus(), { wrapper: wrapper(cols3x3) })
    act(() => result.current.setFocused({ colIdx: 0, cardIdx: 2 }))
    act(() => result.current.move('up'))
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 1 })
    act(() => result.current.move('up'))
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 0 })
    act(() => result.current.move('up'))
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 0 })
  })

  it('move right clamps cardIdx to new column length', () => {
    const { result } = renderHook(() => useBoardFocus(), { wrapper: wrapper(cols3x3) })
    act(() => result.current.setFocused({ colIdx: 0, cardIdx: 2 }))
    act(() => result.current.move('right'))
    expect(result.current.focused).toEqual({ colIdx: 1, cardIdx: 0 })
  })

  it('move left from colIdx 0 is a no-op', () => {
    const { result } = renderHook(() => useBoardFocus(), { wrapper: wrapper(cols3x3) })
    act(() => result.current.setFocused({ colIdx: 0, cardIdx: 0 }))
    act(() => result.current.move('left'))
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 0 })
  })

  it('move right from last column is a no-op', () => {
    const { result } = renderHook(() => useBoardFocus(), { wrapper: wrapper(cols3x3) })
    act(() => result.current.setFocused({ colIdx: 2, cardIdx: 0 }))
    act(() => result.current.move('right'))
    expect(result.current.focused).toEqual({ colIdx: 2, cardIdx: 0 })
  })

  it('move right skips empty column', () => {
    const { result } = renderHook(() => useBoardFocus(), { wrapper: wrapper(colsWithEmpty) })
    act(() => result.current.setFocused({ colIdx: 0, cardIdx: 0 }))
    act(() => result.current.move('right'))
    expect(result.current.focused).toEqual({ colIdx: 2, cardIdx: 0 })
  })
})

describe('BoardFocusContext clamp effect', () => {
  it('clamps cardIdx down when column shrinks', () => {
    let current: Column[] = cols3x3
    const { result, rerender } = renderHook(() => useBoardFocus(), {
      wrapper: ({ children }: { children: React.ReactNode }) => (
        <BoardFocusProvider columns={current}>{children}</BoardFocusProvider>
      ),
    })
    act(() => result.current.setFocused({ colIdx: 0, cardIdx: 2 }))
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 2 })
    current = [
      { name: 'A', cards: [{ title: 'a0' }] },
      ...cols3x3.slice(1),
    ]
    rerender()
    expect(result.current.focused).toEqual({ colIdx: 0, cardIdx: 0 })
  })
})
