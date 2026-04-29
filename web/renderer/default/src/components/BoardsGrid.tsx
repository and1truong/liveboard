import { useRef, useState, useEffect } from 'react'
import type { BoardSummary } from '@shared/adapter.js'
import { useBoardList } from '../queries.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { useCreateBoard } from '../mutations/useBoardCrud.js'
import { BoardIcon } from './BoardIcon.js'

function NewBoardInline({ onDone }: { onDone: () => void }): JSX.Element {
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useCreateBoard()
  const committedRef = useRef(false)

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  const commit = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    const name = (inputRef.current?.value ?? '').trim()
    if (name) mutation.mutate(name)
    onDone()
  }

  const cancel = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    onDone()
  }

  return (
    <input
      ref={inputRef}
      aria-label="new board name"
      defaultValue=""
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === 'Enter') { e.preventDefault(); commit() }
        else if (e.key === 'Escape') { e.preventDefault(); cancel() }
      }}
      placeholder="Board name…"
      className="rounded bg-[color:var(--color-surface)] px-3 py-1.5 text-sm font-medium outline-none ring-1 ring-[color:var(--color-border)] focus:ring-[color:var(--accent-500)] dark:text-slate-100 dark:placeholder:text-slate-400"
    />
  )
}

function BoardCard({ board }: { board: BoardSummary }): JSX.Element {
  const { setActive } = useActiveBoard()
  const cardCount = board.cardCount ?? 0
  const doneCount = board.doneCount ?? 0
  const pct = cardCount > 0 ? (doneCount / cardCount) * 100 : 0

  return (
    <button
      type="button"
      onClick={() => setActive(board.id)}
      className="flex flex-col rounded-xl border border-[color:var(--color-border)] bg-[color:var(--color-surface)] p-5 text-left transition-shadow hover:shadow-md"
    >
      <div className="mb-2 flex items-center gap-2">
        {board.icon && <BoardIcon icon={board.icon} color={board.icon_color} size="lg" />}
        <h3 className="text-base font-semibold text-slate-800 dark:text-slate-100">{board.name}</h3>
      </div>

      {board.description && (
        <p className="mb-3 text-sm text-slate-500 dark:text-slate-400 line-clamp-2">{board.description}</p>
      )}

      {cardCount > 0 ? (
        <div className="mb-3 flex items-center gap-2">
          <div className="relative h-1.5 flex-1 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-700">
            <div
              className="absolute inset-y-0 left-0 rounded-full bg-[color:var(--accent-600)]"
              style={{ width: `${pct}%` }}
            />
          </div>
          <span className="shrink-0 text-xs text-slate-500 dark:text-slate-400">{doneCount}/{cardCount} done</span>
        </div>
      ) : null}

      {board.updatedAgo && (
        <div className="mt-auto flex items-center justify-end">
          <span className="shrink-0 text-xs text-slate-400 dark:text-slate-500">{board.updatedAgo}</span>
        </div>
      )}
    </button>
  )
}

export function BoardsGrid(): JSX.Element {
  const boards = useBoardList()
  const [adding, setAdding] = useState(false)

  return (
    <div className="flex h-full flex-col overflow-y-auto bg-[color:var(--color-bg)]">
      <div className="flex items-center justify-between px-8 py-6">
        <h1 className="text-2xl font-bold text-slate-800 dark:text-slate-100">My Boards</h1>
        {adding ? (
          <NewBoardInline onDone={() => setAdding(false)} />
        ) : (
          <button
            type="button"
            onClick={() => setAdding(true)}
            className="rounded-lg bg-[color:var(--accent-600)] px-4 py-2 text-sm font-medium text-white hover:opacity-90"
          >
            + New Board
          </button>
        )}
      </div>

      <div className="px-8 pb-8">
        {boards.isLoading ? (
          <p className="text-sm text-slate-400 dark:text-slate-500">Loading…</p>
        ) : boards.error ? (
          <p className="text-sm text-red-500">Failed to load boards</p>
        ) : !boards.data || boards.data.length === 0 ? (
          <p className="text-sm text-slate-400 dark:text-slate-500">No boards yet. Create one to get started.</p>
        ) : (
          <div className="grid grid-cols-2 gap-4">
            {boards.data.map((b) => (
              <BoardCard key={b.id} board={b} />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
