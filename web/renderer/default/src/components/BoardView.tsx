import { useEffect } from 'react'
import { SortableContext, horizontalListSortingStrategy } from '@dnd-kit/sortable'
import type { Client } from '@shared/client.js'
import { useBoard } from '../queries.js'
import { EmptyState } from './EmptyState.js'
import { AddColumnButton } from './AddColumnButton.js'
import { BoardDndContext } from '../dnd/BoardDndContext.js'
import { SortableColumn } from '../dnd/SortableColumn.js'
import { encodeColumnId } from '../dnd/cardId.js'

export function BoardView({
  boardId,
  client,
}: {
  boardId: string | null
  client: Client
}): JSX.Element {
  const { data, isLoading, error } = useBoard(boardId)

  useEffect(() => {
    if (!boardId) return
    void client.subscribe(boardId)
    return () => {
      void client.unsubscribe(boardId)
    }
  }, [boardId, client])

  if (!boardId) return <EmptyState title="Select a board" />
  if (isLoading) return <EmptyState title="Loading…" />
  if (error) return <EmptyState title="Failed to load board" detail={String(error)} />
  if (!data) return <EmptyState title="Board not found" />

  const columns = data.columns ?? []
  if (columns.length === 0) {
    return (
      <div className="flex h-full gap-4 overflow-x-auto p-4">
        <AddColumnButton boardId={boardId} />
      </div>
    )
  }

  const names = columns.map((c) => c.name)
  const columnIds = names.map(encodeColumnId)

  return (
    <BoardDndContext boardId={boardId}>
      <SortableContext items={columnIds} strategy={horizontalListSortingStrategy}>
        <div className="flex h-full gap-4 overflow-x-auto p-4">
          {columns.map((col, i) => (
            <SortableColumn
              key={`${col.name}-${i}`}
              column={col}
              colIdx={i}
              allColumnNames={names}
              boardId={boardId}
            />
          ))}
          <AddColumnButton boardId={boardId} />
        </div>
      </SortableContext>
    </BoardDndContext>
  )
}
