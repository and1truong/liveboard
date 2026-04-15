import { useBoardList, useWorkspaceInfo } from '../queries.js'
import { EmptyState } from './EmptyState.js'

export function BoardSidebar({
  activeId,
  onSelect,
}: {
  activeId: string | null
  onSelect: (boardId: string) => void
}): JSX.Element {
  const boards = useBoardList()
  const ws = useWorkspaceInfo()

  return (
    <aside className="flex h-full w-60 shrink-0 flex-col border-r border-slate-200 bg-white">
      <header className="border-b border-slate-200 p-3">
        <p className="text-xs uppercase tracking-wide text-slate-500">Workspace</p>
        <p className="truncate text-sm font-semibold text-slate-800">
          {ws.data?.name ?? '—'}
        </p>
      </header>
      {boards.isLoading ? (
        <EmptyState title="Loading…" />
      ) : boards.error ? (
        <EmptyState title="Failed to load" detail={String(boards.error)} />
      ) : !boards.data || boards.data.length === 0 ? (
        <EmptyState title="No boards yet" />
      ) : (
        <ul className="flex-1 overflow-y-auto p-2">
          {boards.data.map((b) => {
            const active = b.id === activeId
            return (
              <li key={b.id}>
                <button
                  type="button"
                  onClick={() => onSelect(b.id)}
                  className={`flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-sm ${
                    active
                      ? 'bg-slate-200 text-slate-900'
                      : 'text-slate-700 hover:bg-slate-100'
                  }`}
                >
                  {b.icon && <span aria-hidden>{b.icon}</span>}
                  <span className="truncate">{b.name}</span>
                </button>
              </li>
            )
          })}
        </ul>
      )}
    </aside>
  )
}
