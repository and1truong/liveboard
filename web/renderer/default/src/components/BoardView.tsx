import { useEffect } from 'react'
import { SortableContext, horizontalListSortingStrategy } from '@dnd-kit/sortable'
import type { Client } from '@shared/client.js'
import { ProtocolError } from '@shared/protocol.js'
import type { Column } from '@shared/types.js'
import { useBoard } from '../queries.js'
import { EmptyState } from './EmptyState.js'
import { AddColumnButton } from './AddColumnButton.js'
import { BoardDndContext } from '../dnd/BoardDndContext.js'
import { SortableColumn } from '../dnd/SortableColumn.js'
import { encodeColumnId } from '../dnd/cardId.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { BoardFocusProvider } from '../contexts/BoardFocusContext.js'
import { BoardFilterProvider } from '../contexts/BoardFilterContext.js'
import { BoardsGrid } from './BoardsGrid.js'
import { BoardListView } from './BoardListView.js'
import { BoardCalendarView } from './BoardCalendarView.js'
import { BoardHeader } from './BoardHeader.js'
import { FocusedColumnProvider, useFocusedColumn } from '../contexts/FocusedColumnContext.js'
import { FocusExitBar } from './FocusExitBar.js'
import { useBoardSettings } from '../queries/useBoardSettings.js'
import { useAvailableTags } from '../queries/useAvailableTags.js'
import { useBoardSettingsContext } from '../contexts/BoardSettingsContext.js'

function BoardColumns({
  data,
  active,
  columns,
}: {
  data: NonNullable<ReturnType<typeof useBoard>['data']>
  active: string
  columns: Column[]
}): JSX.Element {
  const { focused } = useFocusedColumn()
  const names = columns.map((c) => c.name)
  const columnIds = names.map(encodeColumnId)

  const visibleColumns = focused !== null ? columns.filter((c) => c.name === focused) : columns

  const columnsRow = (
    <div className="flex flex-1 gap-4 overflow-x-auto p-4">
      {visibleColumns.map((col) => {
        const i = columns.indexOf(col)
        return (
          <SortableColumn
            key={`${col.name}-${i}`}
            column={col}
            colIdx={i}
            allColumnNames={names}
            boardId={active}
            collapsed={data.list_collapse?.[i] ?? false}
            isFocusMode={focused !== null}
          />
        )
      })}
      {focused === null && <AddColumnButton boardId={active} />}
    </div>
  )

  return (
    <div className="flex flex-1 flex-col">
      {focused !== null && (
        <div className="px-4 pt-4">
          <FocusExitBar />
        </div>
      )}
      {focused !== null ? (
        columnsRow
      ) : (
        <SortableContext items={columnIds} strategy={horizontalListSortingStrategy}>
          {columnsRow}
        </SortableContext>
      )}
    </div>
  )
}

export function BoardView({ client, onToggleSidebar }: { client: Client; onToggleSidebar: () => void }): JSX.Element {
  const { active, setActive } = useActiveBoard()
  const { data, isLoading, error } = useBoard(active)
  const settings = useBoardSettings(active)
  const { openSettings } = useBoardSettingsContext()
  const availableTags = useAvailableTags(data)

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

  useEffect(() => {
    const title = data?.name ? `${data.name} — LiveBoard` : 'LiveBoard'
    client.emit('title.changed', { title, icon: data?.icon ?? null })
  }, [client, data?.name, data?.icon])

  if (!active) return <BoardsGrid />
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

  return (
    <BoardFilterProvider boardId={active} availableTags={availableTags}>
      <BoardFocusProvider columns={columns}>
        <FocusedColumnProvider columns={columns}>
          <BoardDndContext boardId={active}>
            <div className="flex h-full flex-col">
              <BoardHeader
                data={data}
                availableTags={availableTags}
                onToggleSidebar={onToggleSidebar}
                onOpenSettings={openSettings}
              />
              {settings.view_mode === 'calendar' ? (
                <BoardCalendarView data={data} active={active} columns={columns} />
              ) : settings.view_mode === 'list' ? (
                <BoardListView data={data} active={active} columns={columns} />
              ) : (
                <BoardColumns data={data} active={active} columns={columns} />
              )}
            </div>
          </BoardDndContext>
        </FocusedColumnProvider>
      </BoardFocusProvider>
    </BoardFilterProvider>
  )
}
