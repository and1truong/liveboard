import type { Client } from '@shared/client.js'
import { BoardSidebar } from './components/BoardSidebar.js'
import { BoardView } from './components/BoardView.js'
import { CommandPalette } from './components/CommandPalette.js'
import { Toaster } from './toast.js'
import { ActiveBoardProvider } from './contexts/ActiveBoardContext.js'
import { ThemeProvider } from './contexts/ThemeContext.js'
import { useBoardListEvents } from './mutations/useBoardListEvents.js'

function ListEventsBridge(): null {
  useBoardListEvents()
  return null
}

export function App({ client }: { client: Client }): JSX.Element {
  return (
    <ThemeProvider>
      <ActiveBoardProvider>
        <ListEventsBridge />
        <div className="flex h-screen w-screen">
          <BoardSidebar />
          <main className="flex-1 overflow-hidden dark:bg-slate-950">
            <BoardView client={client} />
          </main>
          <CommandPalette />
          <Toaster position="bottom-right" richColors closeButton />
        </div>
      </ActiveBoardProvider>
    </ThemeProvider>
  )
}
