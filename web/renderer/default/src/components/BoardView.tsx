import { Suspense, lazy, useEffect, useState } from 'react'
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
import { BoardFocusProvider } from '../contexts/BoardFocusContext.js'
const BoardSettingsModal = lazy(() =>
  import('./BoardSettingsModal.js').then((m) => ({ default: m.BoardSettingsModal })),
)

export function BoardView({ client }: { client: Client }): JSX.Element {
  const { active, setActive } = useActiveBoard()
  const { data, isLoading, error } = useBoard(active)
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

  const toggleHideCompleted = (): void => {
    const next = !hideCompleted
    setHideCompleted(next)
    localStorage.setItem('lb_hideCompleted', String(next))
  }

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
    <BoardFocusProvider columns={columns}>
      <BoardDndContext boardId={active}>
        <div className="flex h-full flex-col">
          <div className="flex h-12 shrink-0 items-center gap-3 border-b border-slate-200 px-4 dark:border-slate-800">
            {data.icon && <span className="text-xl leading-none">{data.icon}</span>}
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
                <svg className="pointer-events-none absolute left-2 text-slate-400" width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <circle cx="6.5" cy="6.5" r="5"/><line x1="10" y1="10" x2="14.5" y2="14.5"/>
                </svg>
                <input
                  type="text"
                  placeholder="Filter…"
                  value={filterQuery}
                  onChange={(e) => setFilterQuery(e.target.value)}
                  onKeyDown={(e) => e.key === 'Escape' && setFilterQuery('')}
                  className="h-7 w-40 rounded border border-slate-200 bg-white py-1 pl-7 pr-2 text-sm text-slate-700 placeholder-slate-400 focus:border-blue-400 focus:outline-none dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:placeholder-slate-500"
                />
                {filterQuery && (
                  <button
                    type="button"
                    onClick={() => setFilterQuery('')}
                    className="absolute right-1.5 text-slate-400 hover:text-slate-600"
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
                    : 'border-slate-200 bg-white text-slate-400 hover:text-slate-600 dark:border-slate-700 dark:bg-slate-900'
                }`}
              >
                ✓
              </button>
            </div>
          </div>
          <SortableContext items={columnIds} strategy={horizontalListSortingStrategy}>
            <div className="flex flex-1 gap-4 overflow-x-auto p-4">
              {columns.map((col, i) => (
                <SortableColumn
                  key={`${col.name}-${i}`}
                  column={col}
                  colIdx={i}
                  allColumnNames={names}
                  boardId={active}
                  collapsed={data.list_collapse?.[i] ?? false}
                  filterQuery={filterQuery}
                  hideCompleted={hideCompleted}
                />
              ))}
              <AddColumnButton boardId={active} />
            </div>
          </SortableContext>
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
    </BoardFocusProvider>
  )
}
