import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import type { Column } from '@shared/types.js'

export interface FocusedCard {
  colIdx: number
  cardIdx: number
}

export interface BoardFocusCtx {
  focused: FocusedCard | null
  setFocused: (next: FocusedCard | null) => void
  move: (dir: 'up' | 'down' | 'left' | 'right') => void
  registerCard: (colIdx: number, cardIdx: number, el: HTMLElement | null) => void
}

const Ctx = createContext<BoardFocusCtx | null>(null)

function nearestNonEmpty(columns: Column[], fromIdx: number): FocusedCard | null {
  for (let step = 1; step <= columns.length; step++) {
    for (const dir of [-1, 1]) {
      const idx = fromIdx + dir * step
      if (idx >= 0 && idx < columns.length) {
        const len = columns[idx]?.cards?.length ?? 0
        if (len > 0) return { colIdx: idx, cardIdx: 0 }
      }
    }
  }
  return null
}

export function BoardFocusProvider({
  columns,
  children,
}: {
  columns: Column[]
  children: ReactNode
}): JSX.Element {
  const [focused, _setFocused] = useState<FocusedCard | null>(null)
  const refs = useRef(new Map<string, HTMLElement>())
  const pendingFocusRef = useRef(false)

  const setFocused = useCallback((next: FocusedCard | null) => {
    _setFocused((prev) => {
      if (prev === next) return prev
      if (prev && next && prev.colIdx === next.colIdx && prev.cardIdx === next.cardIdx) return prev
      return next
    })
  }, [])

  const registerCard = useCallback((colIdx: number, cardIdx: number, el: HTMLElement | null) => {
    const key = `${colIdx}:${cardIdx}`
    if (el) refs.current.set(key, el)
    else refs.current.delete(key)
  }, [])

  // Programmatic focus only when requested (move/arrow-key), not on passive
  // onFocus sync — otherwise we fight focus traps like Radix Dialog.
  useEffect(() => {
    if (!pendingFocusRef.current) return
    pendingFocusRef.current = false
    if (!focused) return
    const el = refs.current.get(`${focused.colIdx}:${focused.cardIdx}`)
    if (el && document.activeElement !== el) el.focus()
  }, [focused])

  // Clamp focused to a still-valid position whenever columns mutate.
  useEffect(() => {
    if (!focused) return
    const col = columns[focused.colIdx]
    const len = col?.cards?.length ?? 0
    if (!col) {
      const lastCol = Math.max(0, columns.length - 1)
      const lastLen = columns[lastCol]?.cards?.length ?? 0
      if (columns.length === 0 || lastLen === 0) {
        setFocused(null)
      } else {
        setFocused({ colIdx: lastCol, cardIdx: lastLen - 1 })
      }
      return
    }
    if (focused.cardIdx >= len) {
      if (len === 0) {
        setFocused(nearestNonEmpty(columns, focused.colIdx))
      } else {
        setFocused({ colIdx: focused.colIdx, cardIdx: len - 1 })
      }
    }
  }, [columns, focused])

  const move = useCallback(
    (dir: 'up' | 'down' | 'left' | 'right') => {
      if (!focused) {
        if (columns[0] && (columns[0].cards?.length ?? 0) > 0) {
          setFocused({ colIdx: 0, cardIdx: 0 })
        }
        return
      }
      const { colIdx, cardIdx } = focused
      switch (dir) {
        case 'up':
          if (cardIdx > 0) setFocused({ colIdx, cardIdx: cardIdx - 1 })
          return
        case 'down': {
          const len = columns[colIdx]?.cards?.length ?? 0
          if (cardIdx < len - 1) setFocused({ colIdx, cardIdx: cardIdx + 1 })
          return
        }
        case 'left':
        case 'right': {
          const step = dir === 'left' ? -1 : 1
          let next = colIdx + step
          while (next >= 0 && next < columns.length && (columns[next]?.cards?.length ?? 0) === 0) {
            next += step
          }
          if (next < 0 || next >= columns.length) return
          const newLen = columns[next]?.cards?.length ?? 0
          setFocused({ colIdx: next, cardIdx: Math.min(cardIdx, newLen - 1) })
          return
        }
      }
    },
    [columns, focused],
  )

  const value = useMemo<BoardFocusCtx>(
    () => ({ focused, setFocused, move, registerCard }),
    [focused, move, registerCard],
  )

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useBoardFocus(): BoardFocusCtx {
  const v = useContext(Ctx)
  if (!v) throw new Error('useBoardFocus must be used within BoardFocusProvider')
  return v
}

export function useCardFocus(
  colIdx: number,
  cardIdx: number,
): { isFocused: boolean; ref: (el: HTMLElement | null) => void } {
  const { focused, registerCard } = useBoardFocus()
  const isFocused = focused?.colIdx === colIdx && focused?.cardIdx === cardIdx
  const ref = useCallback(
    (el: HTMLElement | null) => registerCard(colIdx, cardIdx, el),
    [colIdx, cardIdx, registerCard],
  )
  return { isFocused, ref }
}
