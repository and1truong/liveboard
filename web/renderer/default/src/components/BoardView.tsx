import { Suspense, lazy, useEffect, useState } from 'react'
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
import { BoardsGrid } from './BoardsGrid.js'
import { BoardListView } from './BoardListView.js'
import { FocusedColumnProvider, useFocusedColumn } from '../contexts/FocusedColumnContext.js'
import { FocusExitBar } from './FocusExitBar.js'
import { useBoardSettings } from '../queries/useBoardSettings.js'
const BoardSettingsModal = lazy(() =>
  import('./BoardSettingsModal.js').then((m) => ({ default: m.BoardSettingsModal })),
)

function BoardColumns({
  data,
  active,
  columns,
  filterQuery,
  hideCompleted,
}: {
  data: NonNullable<ReturnType<typeof useBoard>['data']>
  active: string
  columns: Column[]
  filterQuery: string
  hideCompleted: boolean
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
            filterQuery={filterQuery}
            hideCompleted={hideCompleted}
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
  const [filterQuery, setFilterQuery] = useState('')
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [hideCompleted, setHideCompleted] = useState(
    () => localStorage.getItem('lb_hideCompleted') === 'true'
  )

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

  const toggleHideCompleted = (): void => {
    const next = !hideCompleted
    setHideCompleted(next)
    localStorage.setItem('lb_hideCompleted', String(next))
  }

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
    <BoardFocusProvider columns={columns}>
      <FocusedColumnProvider columns={columns}>
        <BoardDndContext boardId={active}>
          <div className="flex h-full flex-col">
            <div className="flex h-12 shrink-0 items-center gap-3 border-b border-[color:var(--color-border)] px-4">
              <button
                type="button"
                onClick={onToggleSidebar}
                title="Toggle sidebar"
                className="hidden md:flex h-6 w-6 shrink-0 items-center justify-center rounded text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600 dark:hover:bg-slate-800 dark:hover:text-slate-300"
              >
                <svg width="16" height="16" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
                  <path d="M16.5 4A1.5 1.5 0 0 1 18 5.5v9a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 2 14.5v-9A1.5 1.5 0 0 1 3.5 4zM7 15h9.5a.5.5 0 0 0 .5-.5v-9a.5.5 0 0 0-.5-.5H7zM3.5 5a.5.5 0 0 0-.5.5v9a.5.5 0 0 0 .5.5H6V5z" />
                </svg>
              </button>
              <h1 className="text-base font-semibold text-slate-800 dark:text-slate-100">{data.name}</h1>
              <button
                type="button"
                onClick={() => setSettingsOpen(true)}
                title="Board settings"
                className="flex h-6 w-6 items-center justify-center rounded text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600 dark:hover:bg-slate-800"
              >
                <svg width="14" height="14" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
                  <path fillRule="evenodd" d="M11.49 3.17c-.38-1.56-2.6-1.56-2.98 0a1.532 1.532 0 01-2.286.948c-1.372-.836-2.942.734-2.106 2.106.54.886.061 2.042-.947 2.287-1.561.379-1.561 2.6 0 2.978a1.532 1.532 0 01.947 2.287c-.836 1.372.734 2.942 2.106 2.106a1.532 1.532 0 012.287.947c.379 1.561 2.6 1.561 2.978 0a1.533 1.533 0 012.287-.947c1.372.836 2.942-.734 2.106-2.106a1.533 1.533 0 01.947-2.287c1.561-.379 1.561-2.6 0-2.978a1.532 1.532 0 01-.947-2.287c.836-1.372-.734-2.942-2.106-2.106a1.532 1.532 0 01-2.287-.947zM10 13a3 3 0 100-6 3 3 0 000 6z" clipRule="evenodd" />
                </svg>
              </button>
              <div className="ml-auto flex items-center gap-2">
                <div className="relative flex items-center">
                  <svg className="pointer-events-none absolute left-2 text-[color:var(--color-text-muted)]" width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
                    <circle cx="6.5" cy="6.5" r="5"/><line x1="10" y1="10" x2="14.5" y2="14.5"/>
                  </svg>
                  <input
                    type="text"
                    placeholder="Filter…"
                    value={filterQuery}
                    onChange={(e) => setFilterQuery(e.target.value)}
                    onKeyDown={(e) => e.key === 'Escape' && setFilterQuery('')}
                    className="h-7 w-40 rounded border border-[color:var(--color-border)] bg-[color:var(--color-surface)] py-1 pl-7 pr-2 text-sm text-[color:var(--color-text-primary)] placeholder-[color:var(--color-text-muted)] focus:border-[color:var(--accent-500)] focus:outline-none"
                  />
                  {filterQuery && (
                    <button
                      type="button"
                      onClick={() => setFilterQuery('')}
                      className="absolute right-1.5 text-[color:var(--color-text-muted)] hover:text-[color:var(--color-text-secondary)]"
                      aria-label="Clear filter"
                    >
                      ×
                    </button>
                  )}
                </div>
                <button
                  type="button"
                  onClick={toggleHideCompleted}
                  title="Hide completed items"
                  className={`flex h-7 w-7 items-center justify-center rounded border text-sm transition-colors ${
                    hideCompleted
                      ? 'border-green-300 bg-green-100 text-green-700 dark:border-green-700 dark:bg-green-900/40 dark:text-green-400'
                      : 'border-[color:var(--color-border)] bg-[color:var(--color-surface)] text-[color:var(--color-text-muted)] hover:text-[color:var(--color-text-secondary)]'
                  }`}
                >
                  ✓
                </button>
              </div>
            </div>
            {settings.view_mode === 'list' ? (
              <BoardListView
                data={data}
                active={active}
                columns={columns}
                filterQuery={filterQuery}
                hideCompleted={hideCompleted}
              />
            ) : (
              <BoardColumns
                data={data}
                active={active}
                columns={columns}
                filterQuery={filterQuery}
                hideCompleted={hideCompleted}
              />
            )}
          </div>
        </BoardDndContext>
        <Suspense fallback={null}>
          <BoardSettingsModal
            boardId={active}
            boardName={data.name}
            open={settingsOpen}
            onOpenChange={setSettingsOpen}
          />
        </Suspense>
      </FocusedColumnProvider>
    </BoardFocusProvider>
  )
}
