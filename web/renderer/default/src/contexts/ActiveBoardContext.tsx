import { createContext, useContext, useState, type ReactNode } from 'react'

interface ActiveBoardCtx {
  active: string | null
  setActive: (next: string | null) => void
}

const Ctx = createContext<ActiveBoardCtx | null>(null)

export function ActiveBoardProvider({ children }: { children: ReactNode }): JSX.Element {
  const [active, setActive] = useState<string | null>(null)
  return <Ctx.Provider value={{ active, setActive }}>{children}</Ctx.Provider>
}

export function useActiveBoard(): ActiveBoardCtx {
  const v = useContext(Ctx)
  if (!v) throw new Error('useActiveBoard must be used within ActiveBoardProvider')
  return v
}

export function useOptionalActiveBoard(): ActiveBoardCtx | null {
  return useContext(Ctx)
}
