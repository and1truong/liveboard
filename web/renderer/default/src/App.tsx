import { useState } from 'react'
import type { Client } from '@shared/client.js'
import { BoardSidebar } from './components/BoardSidebar.js'
import { BoardView } from './components/BoardView.js'

export function App({ client }: { client: Client }): JSX.Element {
  const [activeId, setActiveId] = useState<string | null>(null)
  return (
    <div className="flex h-screen w-screen">
      <BoardSidebar activeId={activeId} onSelect={setActiveId} />
      <main className="flex-1 overflow-hidden">
        <BoardView boardId={activeId} client={client} />
      </main>
    </div>
  )
}
