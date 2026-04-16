import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import type { Column } from '@shared/types.js'

export interface FocusedColumnCtx {
  focused: string | null
  setFocused: (name: string | null) => void
}

const Ctx = createContext<FocusedColumnCtx | null>(null)

export function FocusedColumnProvider({
  columns,
  active,
  children,
}: {
  columns: Column[]
  active: string | null
  children: ReactNode
}): JSX.Element {
  const [focused, setFocused] = useState<string | null>(null)

  useEffect(() => {
    setFocused(null)
  }, [active])

  useEffect(() => {
    if (focused === null) return
    const exists = columns.some((c) => c.name === focused)
    if (!exists) setFocused(null)
  }, [columns, focused])

  useEffect(() => {
    if (focused === null) return
    function onKey(e: KeyboardEvent): void {
      if (e.key !== 'Escape') return
      // Ignore when typing in an input/textarea/contenteditable.
      const el = document.activeElement as HTMLElement | null
      if (el) {
        const tag = el.tagName
        if (tag === 'INPUT' || tag === 'TEXTAREA' || el.isContentEditable) return
      }
      // Ignore when a Radix (or compatible) dialog is open.
      if (document.querySelector('[role="dialog"][data-state="open"]')) return
      setFocused(null)
    }
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('keydown', onKey)
    }
  }, [focused])

  const value = useMemo<FocusedColumnCtx>(
    () => ({ focused, setFocused }),
    [focused],
  )

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useFocusedColumn(): FocusedColumnCtx {
  const v = useContext(Ctx)
  if (!v) throw new Error('useFocusedColumn must be used within FocusedColumnProvider')
  return v
}
