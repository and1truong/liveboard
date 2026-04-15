import type { Client } from '@shared/client.js'
import { BoardSidebar } from './components/BoardSidebar.js'
import { BoardView } from './components/BoardView.js'
import { Toaster } from './toast.js'
import { ActiveBoardProvider } from './contexts/ActiveBoardContext.js'
import { useBoardListEvents } from './mutations/useBoardListEvents.js'

function ListEventsBridge(): null {
  useBoardListEvents()
  return null
}

export function App({ client }: { client: Client }): JSX.Element {
  return (
    <ActiveBoardProvider>
      <ListEventsBridge />
      <div className="flex h-screen w-screen">
        <BoardSidebar />
        <main className="flex-1 overflow-hidden">
          <BoardView client={client} />
        </main>
        <Toaster position="bottom-right" richColors closeButton />
      </div>
    </ActiveBoardProvider>
  )
}
