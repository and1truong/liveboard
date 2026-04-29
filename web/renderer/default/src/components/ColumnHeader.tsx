import { useState, useRef, useEffect, useContext } from 'react'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { Crosshair, Pencil, ArrowLeft, ArrowRight, ChevronDown, ChevronRight, Trash2 } from 'lucide-react'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { stageDelete } from '../mutations/undoable.js'
import { moveColumnTarget } from '../mutations/moveColumn.js'
import { FocusedColumnContext } from '../contexts/FocusedColumnContext.js'

const MENU_ITEM_CLS =
  'flex cursor-pointer items-center gap-2 rounded px-2 py-1 text-sm outline-none hover:bg-[color:var(--color-column-bg)] data-[disabled]:cursor-not-allowed data-[disabled]:text-slate-300 dark:data-[disabled]:text-slate-600'
const MENU_ICON_CLS = 'shrink-0 text-[color:var(--color-text-muted)]'

export function ColumnHeader({
  name,
  colIdx,
  allColumnNames,
  boardId,
  collapsed = false,
}: {
  name: string
  colIdx: number
  allColumnNames: string[]
  boardId: string
  collapsed?: boolean
}): JSX.Element {
  const [mode, setMode] = useState<'view' | 'edit'>('view')
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useBoardMutation(boardId)
  const committedRef = useRef(false)
  const focusCtx = useContext(FocusedColumnContext)
  const isFocused = focusCtx?.focused === name

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
    <header className="mb-3 flex items-center justify-between gap-2">
      <div className="flex min-w-0 flex-1 items-center gap-1.5">
        <h2 className="min-w-0 truncate text-[22px] font-extrabold lowercase leading-[1.15] tracking-[-0.4px] text-slate-900 dark:text-slate-100">
          {name}
        </h2>
        <button
          type="button"
          aria-label={collapsed ? `expand column ${name}` : `collapse column ${name}`}
          onClick={() => mutation.mutate({ type: 'toggle_column_collapse', col_idx: colIdx })}
          className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-slate-200 text-slate-600 hover:bg-slate-300 hover:text-slate-800 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-slate-600 dark:hover:text-slate-100"
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
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <DropdownMenu.Root>
          <DropdownMenu.Trigger
            aria-label={`column menu ${name}`}
            className="rounded p-1 text-slate-500 hover:bg-[color:var(--color-column-bg)] dark:text-slate-400"
          >
            ⋮
          </DropdownMenu.Trigger>
          <DropdownMenu.Portal>
            <DropdownMenu.Content
              sideOffset={4}
              className="z-50 min-w-40 rounded-md border border-[color:var(--color-border)] bg-[color:var(--color-surface)] p-1 shadow-[var(--shadow-raised)] dark:text-slate-100"
            >
              {focusCtx && !isFocused && (
                <DropdownMenu.Item
                  onSelect={() => focusCtx.setFocused(name)}
                  className={MENU_ITEM_CLS}
                >
                  <Crosshair size={14} className={MENU_ICON_CLS} aria-hidden />
                  Focus
                </DropdownMenu.Item>
              )}
              <DropdownMenu.Item onSelect={() => setMode('edit')} className={MENU_ITEM_CLS}>
                <Pencil size={14} className={MENU_ICON_CLS} aria-hidden />
                Rename
              </DropdownMenu.Item>
              <DropdownMenu.Item
                disabled={leftDisabled}
                onSelect={() => move('left')}
                className={MENU_ITEM_CLS}
              >
                <ArrowLeft size={14} className={MENU_ICON_CLS} aria-hidden />
                Move left
              </DropdownMenu.Item>
              <DropdownMenu.Item
                disabled={rightDisabled}
                onSelect={() => move('right')}
                className={MENU_ITEM_CLS}
              >
                <ArrowRight size={14} className={MENU_ICON_CLS} aria-hidden />
                Move right
              </DropdownMenu.Item>
              <DropdownMenu.Item
                onSelect={() =>
                  mutation.mutate({ type: 'toggle_column_collapse', col_idx: colIdx })
                }
                className={MENU_ITEM_CLS}
              >
                {collapsed ? (
                  <ChevronDown size={14} className={MENU_ICON_CLS} aria-hidden />
                ) : (
                  <ChevronRight size={14} className={MENU_ICON_CLS} aria-hidden />
                )}
                {collapsed ? 'Expand' : 'Collapse'}
              </DropdownMenu.Item>
              <DropdownMenu.Separator className="my-1 h-px bg-[color:var(--color-border)]" />
              <DropdownMenu.Item
                onSelect={() =>
                  stageDelete(() => mutation.mutate({ type: 'delete_column', name }), name)
                }
                className="flex cursor-pointer items-center gap-2 rounded px-2 py-1 text-sm text-red-600 outline-none hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950"
              >
                <Trash2 size={14} className="shrink-0" aria-hidden />
                Delete
              </DropdownMenu.Item>
            </DropdownMenu.Content>
          </DropdownMenu.Portal>
        </DropdownMenu.Root>
      </div>
    </header>
  )
}
