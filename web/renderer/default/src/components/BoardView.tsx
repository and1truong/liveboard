import { useEffect } from 'react'
import type { Client } from '@shared/client.js'
import { useBoard } from '../queries.js'
import { Column } from './Column.js'
import { EmptyState } from './EmptyState.js'
import { AddColumnButton } from './AddColumnButton.js'

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

  return (
    <div className="flex h-full gap-4 overflow-x-auto p-4">
      {columns.map((col, i) => (
        <Column
          key={`${col.name}-${i}`}
          column={col}
          colIdx={i}
          allColumnNames={names}
          boardId={boardId}
        />
      ))}
      <AddColumnButton boardId={boardId} />
    </div>
  )
}
