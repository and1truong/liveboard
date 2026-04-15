import { useEffect } from 'react'
import { SortableContext, horizontalListSortingStrategy } from '@dnd-kit/sortable'
import type { Client } from '@shared/client.js'
import { ProtocolError } from '@shared/protocol.js'
import { useBoard } from '../queries.js'
import { EmptyState } from './EmptyState.js'
import { AddColumnButton } from './AddColumnButton.js'
import { BoardDndContext } from '../dnd/BoardDndContext.js'
import { SortableColumn } from '../dnd/SortableColumn.js'
import { encodeColumnId } from '../dnd/cardId.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'

export function BoardView({ client }: { client: Client }): JSX.Element {
  const { active, setActive } = useActiveBoard()
  const { data, isLoading, error } = useBoard(active)

  useEffect(() => {
    if (!active) return
    void client.subscribe(active)
    return () => {
      void client.unsubscribe(active)
    }
  }, [active, client])

  useEffect(() => {
    if (error instanceof ProtocolError && error.code === 'NOT_FOUND') {
      setActive(null)
    }
  }, [error, setActive])

  if (!active) return <EmptyState title="Select a board" />
  if (isLoading) return <EmptyState title="Loading…" />
  if (error) return <EmptyState title="Failed to load board" detail={String(error)} />
  if (!data) return <EmptyState title="Board not found" />

  const columns = data.columns ?? []
  if (columns.length === 0) {
    return (
      <div className="flex h-full gap-4 overflow-x-auto p-4">
        <AddColumnButton boardId={active} />
      </div>
    )
  }

  const names = columns.map((c) => c.name)
  const columnIds = names.map(encodeColumnId)

  return (
    <BoardDndContext boardId={active}>
      <SortableContext items={columnIds} strategy={horizontalListSortingStrategy}>
        <div className="flex h-full gap-4 overflow-x-auto p-4">
          {columns.map((col, i) => (
            <SortableColumn
              key={`${col.name}-${i}`}
              column={col}
              colIdx={i}
              allColumnNames={names}
              boardId={active}
            />
          ))}
          <AddColumnButton boardId={active} />
        </div>
      </SortableContext>
    </BoardDndContext>
  )
}
