import { useState, useRef, useEffect } from 'react'
import type { Card as CardModel } from '@shared/types.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { stageDelete } from '../mutations/undoable.js'
import { Card } from './Card.js'

export function CardEditable({
  card,
  colIdx,
  cardIdx,
  boardId,
}: {
  card: CardModel
  colIdx: number
  cardIdx: number
  boardId: string
}): JSX.Element {
  const [mode, setMode] = useState<'view' | 'edit'>('view')
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useBoardMutation(boardId)
  // Guards against double-commit from concurrent blur+keydown.
  const committedRef = useRef(false)

  useEffect(() => {
    if (mode === 'edit') {
      committedRef.current = false
      inputRef.current?.focus()
      inputRef.current?.select()
    }
  }, [mode])

  const commit = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    const title = (inputRef.current?.value ?? '').trim()
    if (title && title !== card.title) {
      mutation.mutate({
        type: 'edit_card',
        col_idx: colIdx,
        card_idx: cardIdx,
        title,
        body: card.body ?? '',
        tags: card.tags ?? [],
        priority: card.priority ?? '',
        due: card.due ?? '',
        assignee: card.assignee ?? '',
      })
    }
    // Defer mode switch so the current event cycle completes before unmounting the input.
    Promise.resolve().then(() => setMode('view'))
  }

  const cancel = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    // Defer mode switch so the current event cycle completes before unmounting the input.
    Promise.resolve().then(() => setMode('view'))
  }

  if (mode === 'edit') {
    return (
      <div className="rounded-md bg-white p-3 shadow-sm ring-2 ring-blue-400">
        <input
          ref={inputRef}
          aria-label="card title"
          defaultValue={card.title}
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              commit()
            } else if (e.key === 'Escape') {
              e.preventDefault()
              cancel()
            }
          }}
          className="w-full bg-transparent text-sm font-semibold outline-none"
        />
      </div>
    )
  }

  return (
    <div
      className="group relative"
      onDoubleClick={() => {
        setMode('edit')
      }}
    >
      <div className="flex items-start gap-2">
        <button
          type="button"
          aria-label={card.completed ? 'mark incomplete' : 'mark complete'}
          onClick={() =>
            mutation.mutate({
              type: 'complete_card',
              col_idx: colIdx,
              card_idx: cardIdx,
            })
          }
          className={`mt-3 h-4 w-4 shrink-0 rounded-full border ${
            card.completed ? 'bg-slate-400 border-slate-400' : 'border-slate-300'
          }`}
        />
        <div className="flex-1">
          <Card card={card} />
        </div>
        <button
          type="button"
          aria-label="delete card"
          onClick={() =>
            stageDelete(
              mutation,
              { type: 'delete_card', col_idx: colIdx, card_idx: cardIdx },
              card.title,
            )
          }
          className="opacity-0 group-hover:opacity-100 mt-1 text-xs text-slate-400 hover:text-red-500"
        >
          ✕
        </button>
      </div>
    </div>
  )
}
