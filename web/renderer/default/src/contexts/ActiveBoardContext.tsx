import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react'
import { useClient } from '../queries.js'

export interface CardPos {
  colIdx: number
  cardIdx: number
}

interface ActiveBoardCtx {
  active: string | null
  setActive: (next: string | null) => void
  activeCard: CardPos | null
  setActiveCard: (next: CardPos | null) => void
  focusedColumn: string | null
  setFocusedColumn: (col: string | null) => void
}

const Ctx = createContext<ActiveBoardCtx | null>(null)

export function ActiveBoardProvider({
  children,
  initialBoardId,
  initialCardPos,
  initialFocusedColumn,
}: {
  children: ReactNode
  initialBoardId?: string | null
  initialCardPos?: CardPos | null
  initialFocusedColumn?: string | null
}): JSX.Element {
  const client = useClient()
  const [active, setActiveRaw] = useState<string | null>(initialBoardId ?? null)
  const [activeCard, setActiveCardRaw] = useState<CardPos | null>(initialCardPos ?? null)
  const [focusedColumn, setFocusedColumnRaw] = useState<string | null>(initialFocusedColumn ?? null)
  const remoteRef = useRef(false)

  const setActive = useCallback((next: string | null) => {
    setActiveRaw(next)
    setActiveCardRaw(null)
    setFocusedColumnRaw(null)
    if (!remoteRef.current) {
      client.emit('active.changed', { boardId: next, cardPos: null, focusedColumn: null })
    }
  }, [client])

  const setActiveCard = useCallback((next: CardPos | null) => {
    setActiveCardRaw(next)
    if (!remoteRef.current) {
      client.emit('active.changed', { boardId: active, cardPos: next })
    }
  }, [client, active])

  const setFocusedColumn = useCallback((col: string | null) => {
    setFocusedColumnRaw(col)
    if (!remoteRef.current) {
      client.emit('active.changed', { boardId: active, cardPos: null, focusedColumn: col })
    }
  }, [client, active])

  useEffect(() => {
    return client.on('active.set', ({ boardId, cardPos, focusedColumn: col }) => {
      remoteRef.current = true
      setActiveRaw(boardId)
      setActiveCardRaw(cardPos ?? null)
      setFocusedColumnRaw(col ?? null)
      remoteRef.current = false
    })
  }, [client])

  return <Ctx.Provider value={{ active, setActive, activeCard, setActiveCard, focusedColumn, setFocusedColumn }}>{children}</Ctx.Provider>
}

export function useActiveBoard(): ActiveBoardCtx {
  const v = useContext(Ctx)
  if (!v) throw new Error('useActiveBoard must be used within ActiveBoardProvider')
  return v
}

export function useOptionalActiveBoard(): ActiveBoardCtx | null {
  return useContext(Ctx)
}
