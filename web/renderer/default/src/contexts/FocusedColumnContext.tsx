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
