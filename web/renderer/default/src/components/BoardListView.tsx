import { useEffect, useRef, useState } from 'react'
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable'
import type { Board, Column } from '@shared/types.js'
import { SortableListSection } from '../dnd/SortableListSection.js'
import { encodeColumnId } from '../dnd/cardId.js'
import { useFocusedColumn } from '../contexts/FocusedColumnContext.js'
import { FocusExitBar } from './FocusExitBar.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'

export function BoardListView({
  data,
  active,
  columns,
}: {
  data: Board
  active: string
  columns: Column[]
}): JSX.Element {
  const { focused } = useFocusedColumn()
  const names = columns.map((c) => c.name)
  const columnIds = names.map(encodeColumnId)
  const visibleColumns = focused !== null ? columns.filter((c) => c.name === focused) : columns

  const body = (
    <div className="flex w-full flex-col gap-3 p-4">
      {visibleColumns.map((col) => {
        const i = columns.indexOf(col)
        return (
          <SortableListSection
            key={`${col.name}-${i}`}
            column={col}
            colIdx={i}
            allColumnNames={names}
            boardId={active}
            collapsed={data.list_collapse?.[i] ?? false}
          />
        )
      })}
      {focused === null && <AddListInline boardId={active} />}
    </div>
  )

  return (
    <div className="flex flex-1 flex-col overflow-y-auto">
      {focused !== null && (
        <div className="px-4 pt-4">
          <FocusExitBar />
        </div>
      )}
      {focused !== null ? (
        body
      ) : (
        <SortableContext items={columnIds} strategy={verticalListSortingStrategy}>
          {body}
        </SortableContext>
      )}
    </div>
  )
}

function AddListInline({ boardId }: { boardId: string }): JSX.Element {
  const [open, setOpen] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useBoardMutation(boardId)
  const committedRef = useRef(false)

  useEffect(() => {
    if (open) {
      committedRef.current = false
      inputRef.current?.focus()
    }
  }, [open])

  const commit = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    const name = (inputRef.current?.value ?? '').trim()
    if (name) mutation.mutate({ type: 'add_column', name })
    Promise.resolve().then(() => setOpen(false))
  }

  const cancel = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    Promise.resolve().then(() => setOpen(false))
  }

  if (open) {
    return (
      <input
        ref={inputRef}
        aria-label="new column name"
        defaultValue=""
        placeholder="List name"
        onBlur={commit}
        onKeyDown={(e) => {
          if (e.key === 'Enter') { e.preventDefault(); commit() }
          else if (e.key === 'Escape') { e.preventDefault(); cancel() }
        }}
        className="rounded bg-[color:var(--color-surface)] px-3 py-2 text-sm outline-none border border-[color:var(--color-border)] focus:border-[color:var(--accent-500)] dark:text-slate-100"
      />
    )
  }

  return (
    <button
      type="button"
      onClick={() => setOpen(true)}
      className="self-start rounded px-2 py-1 text-sm text-slate-500 hover:bg-[color:var(--color-column-bg)] dark:text-slate-400"
    >
      + Add list
    </button>
  )
}
