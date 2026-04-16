import { useEffect, useRef, useState, useContext } from 'react'
import { useSortable, SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { useDroppable } from '@dnd-kit/core'
import { CSS } from '@dnd-kit/utilities'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import type { Column as ColumnModel } from '@shared/types.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { stageDelete } from '../mutations/undoable.js'
import { moveColumnTarget } from '../mutations/moveColumn.js'
import { FocusedColumnContext } from '../contexts/FocusedColumnContext.js'
import { SortableListRow } from './SortableListRow.js'
import { encodeCardId, encodeColumnId, encodeColumnEndId } from './cardId.js'
import { useDragState } from './BoardDndContext.js'
import { useBoardFilter } from '../contexts/BoardFilterContext.js'
import { filterCard } from '../utils/cardFilter.js'

export function SortableListSection({
  column,
  colIdx,
  allColumnNames,
  boardId,
  collapsed = false,
}: {
  column: ColumnModel
  colIdx: number
  allColumnNames: string[]
  boardId: string
  collapsed?: boolean
}): JSX.Element {
  const id = encodeColumnId(column.name)
  const cards = column.cards ?? []
  const { filter } = useBoardFilter()
  const visibleCards = cards
    .map((card, i) => ({ card, i }))
    .filter(({ card }) => filterCard(card, filter))
  const cardIds = visibleCards.map(({ i }) => encodeCardId(colIdx, i))
  const mutation = useBoardMutation(boardId)
  const focusCtx = useContext(FocusedColumnContext)
  const isFocused = focusCtx?.focused === column.name

  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id,
    data: { type: 'column', name: column.name, col_idx: colIdx },
  })
  const { drop } = useDragState()
  const showEndDropLine = !isDragging && drop?.type === 'column' && drop.colIdx === colIdx

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  const leftDisabled = colIdx === 0
  const rightDisabled = colIdx === allColumnNames.length - 1

  const [renaming, setRenaming] = useState(false)
  const renameRef = useRef<HTMLInputElement>(null)
  const renameCommittedRef = useRef(false)

  useEffect(() => {
    if (renaming) {
      renameCommittedRef.current = false
      renameRef.current?.focus()
      renameRef.current?.select()
    }
  }, [renaming])

  const commitRename = (): void => {
    if (renameCommittedRef.current) return
    renameCommittedRef.current = true
    const next = (renameRef.current?.value ?? '').trim()
    if (next && next !== column.name) {
      mutation.mutate({ type: 'rename_column', old_name: column.name, new_name: next })
    }
    Promise.resolve().then(() => setRenaming(false))
  }

  const cancelRename = (): void => {
    if (renameCommittedRef.current) return
    renameCommittedRef.current = true
    Promise.resolve().then(() => setRenaming(false))
  }

  const toggleCollapse = (): void => {
    mutation.mutate({ type: 'toggle_column_collapse', col_idx: colIdx })
  }

  const move = (dir: 'left' | 'right'): void => {
    const target = moveColumnTarget(allColumnNames, colIdx, dir)
    if (target === null) return
    mutation.mutate({ type: 'move_column', name: column.name, after_col: target })
  }

  return (
    <section
      ref={setNodeRef}
      style={style}
      aria-label={`section ${column.name}`}
      className="overflow-hidden rounded-xl bg-[color:var(--color-surface)] border border-[color:var(--color-border)] shadow-[var(--shadow-card)]"
    >
      <header className="group flex items-center gap-2 px-3 py-2">
        <button
          type="button"
          aria-label={`drag section ${column.name}`}
          {...attributes}
          {...listeners}
          onClick={(e) => e.stopPropagation()}
          className="cursor-grab text-slate-300 opacity-0 group-hover:opacity-100 active:cursor-grabbing dark:text-slate-600"
        >
          ⋮⋮
        </button>
        <button
          type="button"
          aria-label={collapsed ? `expand section ${column.name}` : `collapse section ${column.name}`}
          onClick={toggleCollapse}
          className="flex h-5 w-5 shrink-0 items-center justify-center rounded text-slate-500 hover:bg-[color:var(--color-column-bg)] dark:text-slate-400"
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
        {renaming ? (
          <input
            ref={renameRef}
            aria-label={`rename column ${column.name}`}
            defaultValue={column.name}
            onBlur={commitRename}
            onKeyDown={(e) => {
              if (e.key === 'Enter') { e.preventDefault(); commitRename() }
              else if (e.key === 'Escape') { e.preventDefault(); cancelRename() }
            }}
            className="flex-1 rounded bg-[color:var(--color-column-bg)] px-1 text-sm font-semibold outline-none ring-1 ring-[color:var(--accent-500)] dark:text-slate-100"
          />
        ) : (
          <button
            type="button"
            onClick={toggleCollapse}
            className="flex min-w-0 flex-1 items-center gap-2 text-left"
          >
            <h2 className="truncate text-sm font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">
              {column.name}
            </h2>
            <span className="rounded-full bg-[color:var(--color-column-bg)] px-2 py-0.5 text-xs text-slate-500 dark:text-slate-400">
              {visibleCards.length}
            </span>
          </button>
        )}
        <DropdownMenu.Root>
          <DropdownMenu.Trigger
            aria-label={`section menu ${column.name}`}
            className="rounded p-1 text-slate-400 opacity-0 hover:bg-[color:var(--color-column-bg)] hover:text-slate-600 group-hover:opacity-100"
          >
            ⋮
          </DropdownMenu.Trigger>
          <DropdownMenu.Portal>
            <DropdownMenu.Content
              sideOffset={4}
              className="z-50 min-w-40 rounded-md bg-[color:var(--color-surface)] p-1 shadow-[var(--shadow-raised)] border border-[color:var(--color-border)] dark:text-slate-100"
            >
              {focusCtx && !isFocused && (
                <DropdownMenu.Item
                  onSelect={() => focusCtx.setFocused(column.name)}
                  className="cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-[color:var(--color-column-bg)]"
                >
                  Focus
                </DropdownMenu.Item>
              )}
              <DropdownMenu.Item
                onSelect={() => setRenaming(true)}
                className="cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-[color:var(--color-column-bg)]"
              >
                Rename
              </DropdownMenu.Item>
              <DropdownMenu.Item
                disabled={leftDisabled}
                onSelect={() => move('left')}
                className="cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-[color:var(--color-column-bg)] data-[disabled]:cursor-not-allowed data-[disabled]:text-slate-300 dark:data-[disabled]:text-slate-600"
              >
                Move up
              </DropdownMenu.Item>
              <DropdownMenu.Item
                disabled={rightDisabled}
                onSelect={() => move('right')}
                className="cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-[color:var(--color-column-bg)] data-[disabled]:cursor-not-allowed data-[disabled]:text-slate-300 dark:data-[disabled]:text-slate-600"
              >
                Move down
              </DropdownMenu.Item>
              <DropdownMenu.Item
                onSelect={toggleCollapse}
                className="cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-[color:var(--color-column-bg)]"
              >
                {collapsed ? 'Expand' : 'Collapse'}
              </DropdownMenu.Item>
              <DropdownMenu.Separator className="my-1 h-px bg-[color:var(--color-border)]" />
              <DropdownMenu.Item
                onSelect={() =>
                  stageDelete(
                    () => mutation.mutate({ type: 'delete_column', name: column.name }),
                    column.name,
                  )
                }
                className="cursor-pointer rounded px-2 py-1 text-sm text-red-600 outline-none hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950"
              >
                Delete
              </DropdownMenu.Item>
            </DropdownMenu.Content>
          </DropdownMenu.Portal>
        </DropdownMenu.Root>
      </header>
      {!collapsed && (
        <>
          <SortableContext items={cardIds} strategy={verticalListSortingStrategy}>
            <ul className="flex min-h-[1rem] flex-col">
              {visibleCards.map(({ card, i }) => (
                <li key={`${column.name}-${i}`}>
                  <SortableListRow
                    card={card}
                    colIdx={colIdx}
                    cardIdx={i}
                    boardId={boardId}
                    allColumnNames={allColumnNames}
                  />
                </li>
              ))}
              <SectionEndDropZone columnName={column.name} active={showEndDropLine} />
            </ul>
          </SortableContext>
          <ListQuickAdd columnName={column.name} boardId={boardId} />
        </>
      )}
    </section>
  )
}

function SectionEndDropZone({
  columnName,
  active,
}: {
  columnName: string
  active: boolean
}): JSX.Element {
  const { setNodeRef } = useDroppable({
    id: encodeColumnEndId(columnName),
    data: { type: 'column-end', name: columnName },
  })
  const { isDragActive } = useDragState()
  const baseClass = isDragActive ? 'min-h-[1.5rem]' : 'h-0'
  const lineClass = active ? 'h-[3px] min-h-[3px] bg-[color:var(--accent-500)] rounded-full' : ''
  return <li ref={setNodeRef} aria-hidden className={`${baseClass} ${lineClass}`} />
}

function ListQuickAdd({
  columnName,
  boardId,
}: {
  columnName: string
  boardId: string
}): JSX.Element {
  const mutation = useBoardMutation(boardId)
  const inputRef = useRef<HTMLInputElement>(null)

  const commit = (): void => {
    const title = (inputRef.current?.value ?? '').trim()
    if (!title) return
    mutation.mutate({ type: 'add_card', column: columnName, title })
    if (inputRef.current) inputRef.current.value = ''
  }

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault()
        commit()
      }}
      className="flex items-center gap-2 border-t border-[color:var(--color-border-dashed)] px-3 py-2"
    >
      <span aria-hidden className="text-slate-300 dark:text-slate-600">+</span>
      <input
        ref={inputRef}
        type="text"
        aria-label={`new item in ${columnName}`}
        placeholder="New item"
        autoComplete="off"
        className="flex-1 bg-transparent text-sm text-slate-700 outline-none placeholder:text-slate-400 dark:text-slate-200"
      />
    </form>
  )
}
