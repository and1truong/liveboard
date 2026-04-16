import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  type ReactNode,
} from 'react'
import type { Column } from '@shared/types.js'
import { useActiveBoard } from './ActiveBoardContext.js'

export interface FocusedColumnCtx {
  focused: string | null
  setFocused: (name: string | null) => void
}

export const FocusedColumnContext = createContext<FocusedColumnCtx | null>(null)

export function FocusedColumnProvider({
  columns,
  children,
}: {
  columns: Column[]
  children: ReactNode
}): JSX.Element {
  const { focusedColumn, setFocusedColumn } = useActiveBoard()

  // Exit focus mode if the focused column no longer exists.
  useEffect(() => {
    if (focusedColumn === null) return
    const exists = columns.some((c) => c.name === focusedColumn)
    if (!exists) setFocusedColumn(null)
  }, [columns, focusedColumn, setFocusedColumn])

  // Escape key exits focus mode.
  useEffect(() => {
    if (focusedColumn === null) return
    function onKey(e: KeyboardEvent): void {
      if (e.key !== 'Escape') return
      const el = document.activeElement as HTMLElement | null
      if (el) {
        const tag = el.tagName
        if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || el.isContentEditable) return
      }
      if (document.querySelector('[role="dialog"][data-state="open"]')) return
      setFocusedColumn(null)
    }
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('keydown', onKey)
    }
  }, [focusedColumn, setFocusedColumn])

  const value = useMemo<FocusedColumnCtx>(
    () => ({ focused: focusedColumn, setFocused: setFocusedColumn }),
    [focusedColumn, setFocusedColumn],
  )

  return <FocusedColumnContext.Provider value={value}>{children}</FocusedColumnContext.Provider>
}

export function useFocusedColumn(): FocusedColumnCtx {
  const v = useContext(FocusedColumnContext)
  if (!v) throw new Error('useFocusedColumn must be used within FocusedColumnProvider')
  return v
}
