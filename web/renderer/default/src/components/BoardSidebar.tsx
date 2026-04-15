import { useBoardList, useWorkspaceInfo } from '../queries.js'
import { EmptyState } from './EmptyState.js'
import { BoardRow } from './BoardRow.js'
import { AddBoardButton } from './AddBoardButton.js'
import { ThemePicker } from './ThemePicker.js'

export function BoardSidebar(): JSX.Element {
  const boards = useBoardList()
  const ws = useWorkspaceInfo()

  return (
    <aside className="flex h-full w-60 shrink-0 flex-col border-r border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
      <header className="border-b border-slate-200 p-3 dark:border-slate-800">
        <p className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Workspace</p>
        <p className="truncate text-sm font-semibold text-slate-800 dark:text-slate-100">
          {ws.data?.name ?? '—'}
        </p>
      </header>
      <div className="flex-1 overflow-y-auto p-2">
        {boards.isLoading ? (
          <EmptyState title="Loading…" />
        ) : boards.error ? (
          <EmptyState title="Failed to load" detail={String(boards.error)} />
        ) : !boards.data || boards.data.length === 0 ? (
          <EmptyState title="No boards yet" />
        ) : (
          <ul className="flex flex-col gap-1">
            {boards.data.map((b) => (
              <BoardRow key={b.id} board={b} />
            ))}
          </ul>
        )}
      </div>
      <div className="flex items-center justify-end border-t border-slate-200 px-2 py-1 dark:border-slate-800">
        <ThemePicker />
      </div>
      <AddBoardButton />
    </aside>
  )
}
