import { useState, useCallback } from 'react'
import type { Client } from '@shared/client.js'
import { BoardSidebar } from './components/BoardSidebar.js'
import { BoardView } from './components/BoardView.js'
import { CommandPaletteHost } from './components/CommandPaletteHost.js'
import { Toaster } from './toast.js'
import { ActiveBoardProvider } from './contexts/ActiveBoardContext.js'
import { ThemeProvider } from './contexts/ThemeContext.js'
import { useBoardListEvents } from './mutations/useBoardListEvents.js'

function ListEventsBridge(): null {
  useBoardListEvents()
  return null
}

export function App({ client, initialBoardId, initialCardPos, initialFocusedColumn }: {
  client: Client
  initialBoardId?: string | null
  initialCardPos?: { colIdx: number; cardIdx: number } | null
  initialFocusedColumn?: string | null
}): JSX.Element {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(
    () => localStorage.getItem('lb_sidebarCollapsed') === 'true'
  )

  const toggleSidebar = useCallback(() => {
    setSidebarCollapsed((prev) => {
      const next = !prev
      localStorage.setItem('lb_sidebarCollapsed', String(next))
      return next
    })
  }, [])

  return (
    <ThemeProvider>
      <ActiveBoardProvider initialBoardId={initialBoardId ?? null} initialCardPos={initialCardPos ?? null} initialFocusedColumn={initialFocusedColumn ?? null}>
        <ListEventsBridge />
        <div className="flex h-screen w-screen">
          <BoardSidebar collapsed={sidebarCollapsed} />
          <main className="flex-1 overflow-hidden dark:bg-slate-950">
            <BoardView client={client} onToggleSidebar={toggleSidebar} />
          </main>
          <CommandPaletteHost />
          <Toaster position="bottom-right" richColors closeButton />
        </div>
      </ActiveBoardProvider>
    </ThemeProvider>
  )
}
