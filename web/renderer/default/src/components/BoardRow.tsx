import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import type { BoardSummary } from '@shared/adapter.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { useDeleteBoard } from '../mutations/useBoardCrud.js'
import { stageDelete } from '../mutations/undoable.js'

export function BoardRow({ board }: { board: BoardSummary }): JSX.Element {
  const { active, setActive } = useActiveBoard()
  const deleteMut = useDeleteBoard()
  const isActive = active === board.id

  return (
    <li className="group flex items-center gap-1">
      <button
        type="button"
        onClick={() => setActive(board.id)}
        className={`flex flex-1 items-center gap-2 rounded px-2 py-1.5 text-left text-sm ${
          isActive ? 'bg-slate-200 text-slate-900 dark:bg-slate-700 dark:text-slate-100' : 'text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-800'
        }`}
      >
        {board.icon && <span aria-hidden>{board.icon}</span>}
        <span className="flex min-w-0 flex-1 flex-col">
          <span className="truncate">{board.name}</span>
          {board.updatedAgo && (
            <span className="mt-0.5 text-[11px] leading-none text-slate-400 dark:text-slate-500">
              {board.updatedAgo}
            </span>
          )}
          {board.tags?.length ? (
            <span className="mt-0.5 flex flex-wrap items-center gap-1">
              {board.tags.map(t => (
                <span key={t} className="rounded bg-slate-100 px-1 py-px text-[10px] leading-none text-slate-500 dark:bg-slate-700 dark:text-slate-400">
                  {t}
                </span>
              ))}
            </span>
          ) : null}
        </span>
      </button>
      <DropdownMenu.Root>
        <DropdownMenu.Trigger
          aria-label={`board menu ${board.name}`}
          className="rounded p-1 text-slate-400 opacity-0 group-hover:opacity-100 hover:bg-slate-200 dark:text-slate-500 dark:hover:bg-slate-700"
        >
          ⋮
        </DropdownMenu.Trigger>
        <DropdownMenu.Portal>
          <DropdownMenu.Content
            sideOffset={4}
            className="z-50 min-w-32 rounded-md bg-white p-1 shadow-lg ring-1 ring-slate-200 dark:bg-slate-800 dark:ring-slate-700 dark:text-slate-100"
          >
            <DropdownMenu.Item
              onSelect={() =>
                stageDelete(() => deleteMut.mutate(board.id), board.name)
              }
              className="cursor-pointer rounded px-2 py-1 text-sm text-red-600 outline-none hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950"
            >
              Delete
            </DropdownMenu.Item>
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
    </li>
  )
}
