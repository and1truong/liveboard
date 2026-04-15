import { useState, useRef, useEffect } from 'react'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import type { BoardSummary } from '@shared/adapter.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { useRenameBoard, useDeleteBoard } from '../mutations/useBoardCrud.js'
import { stageDelete } from '../mutations/undoable.js'

export function BoardRow({ board }: { board: BoardSummary }): JSX.Element {
  const { active, setActive } = useActiveBoard()
  const [mode, setMode] = useState<'view' | 'edit'>('view')
  const inputRef = useRef<HTMLInputElement>(null)
  const renameMut = useRenameBoard()
  const deleteMut = useDeleteBoard()
  const committedRef = useRef(false)
  const isActive = active === board.id

  useEffect(() => {
    if (mode === 'edit') {
      committedRef.current = false
      inputRef.current?.focus()
      inputRef.current?.select()
    }
  }, [mode])

  const commitRename = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    const next = (inputRef.current?.value ?? '').trim()
    if (next && next !== board.name) {
      renameMut.mutate({ boardId: board.id, newName: next })
    }
    Promise.resolve().then(() => setMode('view'))
  }

  const cancelRename = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    Promise.resolve().then(() => setMode('view'))
  }

  if (mode === 'edit') {
    return (
      <li>
        <input
          ref={inputRef}
          aria-label={`rename board ${board.name}`}
          defaultValue={board.name}
          onBlur={commitRename}
          onKeyDown={(e) => {
            if (e.key === 'Enter') { e.preventDefault(); commitRename() }
            else if (e.key === 'Escape') { e.preventDefault(); cancelRename() }
          }}
          className="block w-full rounded bg-white px-2 py-1 text-sm outline-none ring-1 ring-blue-400"
        />
      </li>
    )
  }

  return (
    <li className="group flex items-center gap-1">
      <button
        type="button"
        onClick={() => setActive(board.id)}
        className={`flex flex-1 items-center gap-2 rounded px-2 py-1.5 text-left text-sm ${
          isActive ? 'bg-slate-200 text-slate-900' : 'text-slate-700 hover:bg-slate-100'
        }`}
      >
        {board.icon && <span aria-hidden>{board.icon}</span>}
        <span className="truncate">{board.name}</span>
      </button>
      <DropdownMenu.Root>
        <DropdownMenu.Trigger
          aria-label={`board menu ${board.name}`}
          className="rounded p-1 text-slate-400 opacity-0 group-hover:opacity-100 hover:bg-slate-200"
        >
          ⋮
        </DropdownMenu.Trigger>
        <DropdownMenu.Portal>
          <DropdownMenu.Content
            sideOffset={4}
            className="z-50 min-w-32 rounded-md bg-white p-1 shadow-lg ring-1 ring-slate-200"
          >
            <DropdownMenu.Item
              onSelect={() => setMode('edit')}
              className="cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-slate-100"
            >
              Rename
            </DropdownMenu.Item>
            <DropdownMenu.Separator className="my-1 h-px bg-slate-200" />
            <DropdownMenu.Item
              onSelect={() =>
                stageDelete(() => deleteMut.mutate(board.id), board.name)
              }
              className="cursor-pointer rounded px-2 py-1 text-sm text-red-600 outline-none hover:bg-red-50"
            >
              Delete
            </DropdownMenu.Item>
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
    </li>
  )
}
