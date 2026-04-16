import { useState, useRef, useEffect } from 'react'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { stageDelete } from '../mutations/undoable.js'
import { moveColumnTarget } from '../mutations/moveColumn.js'

export function ColumnHeader({
  name,
  cardCount,
  colIdx,
  allColumnNames,
  boardId,
  collapsed = false,
}: {
  name: string
  cardCount: number
  colIdx: number
  allColumnNames: string[]
  boardId: string
  collapsed?: boolean
}): JSX.Element {
  const [mode, setMode] = useState<'view' | 'edit'>('view')
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useBoardMutation(boardId)
  const committedRef = useRef(false)

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
    if (next && next !== name) {
      mutation.mutate({ type: 'rename_column', old_name: name, new_name: next })
    }
    Promise.resolve().then(() => setMode('view'))
  }

  const cancelRename = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    Promise.resolve().then(() => setMode('view'))
  }

  const move = (dir: 'left' | 'right'): void => {
    const target = moveColumnTarget(allColumnNames, colIdx, dir)
    if (target === null) return
    mutation.mutate({ type: 'move_column', name, after_col: target })
  }

  const leftDisabled = colIdx === 0
  const rightDisabled = colIdx === allColumnNames.length - 1

  if (mode === 'edit') {
    return (
      <header className="mb-3 flex items-center justify-between">
        <input
          ref={inputRef}
          aria-label={`rename column ${name}`}
          defaultValue={name}
          onBlur={commitRename}
          onKeyDown={(e) => {
            if (e.key === 'Enter') { e.preventDefault(); commitRename() }
            else if (e.key === 'Escape') { e.preventDefault(); cancelRename() }
          }}
          className="w-full bg-white px-1 text-sm font-semibold outline-none ring-1 ring-[color:var(--accent-500)] rounded dark:bg-slate-800 dark:text-slate-100"
        />
      </header>
    )
  }

  return (
    <header className="mb-3 flex items-center justify-between">
      <div className="flex items-center gap-1">
        <button
          type="button"
          aria-label={collapsed ? `expand column ${name}` : `collapse column ${name}`}
          onClick={() => mutation.mutate({ type: 'toggle_column_collapse', col_idx: colIdx })}
          className="flex h-5 w-5 items-center justify-center rounded text-slate-400 hover:bg-slate-200 hover:text-slate-600 dark:text-slate-500 dark:hover:bg-slate-700 dark:hover:text-slate-300"
        >
          <svg
            width="10"
            height="10"
            viewBox="0 0 10 10"
            fill="none"
            className={`transition-transform ${collapsed ? '' : 'rotate-90'}`}
          >
            <path d="M3 1L7 5L3 9" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </button>
        <h2 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{name}</h2>
      </div>
      <div className="flex items-center gap-2">
        <span className="text-xs text-slate-500">{cardCount}</span>
        <DropdownMenu.Root>
          <DropdownMenu.Trigger
            aria-label={`column menu ${name}`}
            className="rounded p-1 text-slate-500 hover:bg-slate-200 dark:text-slate-400 dark:hover:bg-slate-700"
          >
            ⋮
          </DropdownMenu.Trigger>
          <DropdownMenu.Portal>
            <DropdownMenu.Content
              sideOffset={4}
              className="z-50 min-w-40 rounded-md bg-white p-1 shadow-lg ring-1 ring-slate-200 dark:bg-slate-800 dark:ring-slate-700 dark:text-slate-100"
            >
              <DropdownMenu.Item
                onSelect={() => setMode('edit')}
                className="cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-slate-100 dark:hover:bg-slate-700"
              >
                Rename
              </DropdownMenu.Item>
              <DropdownMenu.Item
                disabled={leftDisabled}
                onSelect={() => move('left')}
                className="cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-slate-100 dark:hover:bg-slate-700 data-[disabled]:text-slate-300 dark:data-[disabled]:text-slate-600 data-[disabled]:cursor-not-allowed"
              >
                Move left
              </DropdownMenu.Item>
              <DropdownMenu.Item
                disabled={rightDisabled}
                onSelect={() => move('right')}
                className="cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-slate-100 dark:hover:bg-slate-700 data-[disabled]:text-slate-300 dark:data-[disabled]:text-slate-600 data-[disabled]:cursor-not-allowed"
              >
                Move right
              </DropdownMenu.Item>
              <DropdownMenu.Item
                onSelect={() =>
                  mutation.mutate({ type: 'toggle_column_collapse', col_idx: colIdx })
                }
                className="cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-slate-100 dark:hover:bg-slate-700"
              >
                {collapsed ? 'Expand' : 'Collapse'}
              </DropdownMenu.Item>
              <DropdownMenu.Separator className="my-1 h-px bg-slate-200 dark:bg-slate-700" />
              <DropdownMenu.Item
                onSelect={() =>
                  stageDelete(() => mutation.mutate({ type: 'delete_column', name }), name)
                }
                className="cursor-pointer rounded px-2 py-1 text-sm text-red-600 outline-none hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950"
              >
                Delete
              </DropdownMenu.Item>
            </DropdownMenu.Content>
          </DropdownMenu.Portal>
        </DropdownMenu.Root>
      </div>
    </header>
  )
}
