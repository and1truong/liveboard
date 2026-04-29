import { Suspense, lazy, useState, useCallback } from 'react'
import type { Client } from '@shared/client.js'
import { BoardSidebar } from './components/BoardSidebar.js'
import { BoardView } from './components/BoardView.js'
import { CommandPaletteHost } from './components/CommandPaletteHost.js'
import { Toaster } from './toast.js'
import { ActiveBoardProvider } from './contexts/ActiveBoardContext.js'
import { useActiveBoard } from './contexts/ActiveBoardContext.js'
import { ThemeProvider } from './contexts/ThemeContext.js'
import { BoardSettingsContext } from './contexts/BoardSettingsContext.js'
import { GlobalSettingsContext, type GlobalSettingsSection } from './contexts/GlobalSettingsContext.js'
import { useBoardListEvents } from './mutations/useBoardListEvents.js'
import { useBoard } from './queries.js'

const BoardSettingsModal = lazy(() =>
  import('./components/BoardSettingsModal.js').then((m) => ({ default: m.BoardSettingsModal })),
)
const GlobalSettingsModal = lazy(() =>
  import('./components/GlobalSettingsModal.js').then((m) => ({ default: m.GlobalSettingsModal })),
)

function ListEventsBridge(): null {
  useBoardListEvents()
  return null
}

function BoardSettingsHost({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
}): JSX.Element | null {
  const { active } = useActiveBoard()
  const boardQuery = useBoard(active)
  if (!active) return null
  return (
    <Suspense fallback={null}>
      <BoardSettingsModal
        boardId={active}
        boardName={boardQuery.data?.name ?? active}
        open={open}
        onOpenChange={onOpenChange}
      />
    </Suspense>
  )
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
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [globalSettingsOpen, setGlobalSettingsOpen] = useState(false)
  const [globalSettingsSection, setGlobalSettingsSection] = useState<GlobalSettingsSection | null>(null)

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
        <BoardSettingsContext.Provider value={{ openSettings: () => setSettingsOpen(true) }}>
          <GlobalSettingsContext.Provider value={{ openSettings: (section) => { setGlobalSettingsSection(section ?? null); setGlobalSettingsOpen(true) } }}>
          <ListEventsBridge />
          <div className="lb-app-shell flex h-screen w-screen flex-col md:flex-row">
            <BoardSidebar collapsed={sidebarCollapsed} />
            <main className="flex-1 min-w-80 overflow-hidden bg-[color:var(--color-bg)]">
              <BoardView client={client} onToggleSidebar={toggleSidebar} />
            </main>
            <CommandPaletteHost />
            <Toaster position="bottom-right" richColors closeButton />
          </div>
          <BoardSettingsHost open={settingsOpen} onOpenChange={setSettingsOpen} />
          <Suspense fallback={null}>
            <GlobalSettingsModal
              open={globalSettingsOpen}
              onOpenChange={(v) => { if (!v) setGlobalSettingsSection(null); setGlobalSettingsOpen(v) }}
              initialSection={globalSettingsSection}
            />
          </Suspense>
          </GlobalSettingsContext.Provider>
        </BoardSettingsContext.Provider>
      </ActiveBoardProvider>
    </ThemeProvider>
  )
}
