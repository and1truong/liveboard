import { Suspense, lazy, useState, useRef, useEffect } from 'react'
import type { Card as CardModel } from '@shared/types.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { stageDelete } from '../mutations/undoable.js'
const CardDetailModal = lazy(() =>
  import('./CardDetailModal.js').then((m) => ({ default: m.CardDetailModal })),
)
import { useBoardSettings } from '../queries/useBoardSettings.js'
import { useTagColors } from '../queries.js'
import { tagChipStyle } from '../utils/tagColor.js'

const PRIORITY_DOT: Record<string, string> = {
  critical: 'bg-red-600',
  high: 'bg-orange-500',
  medium: 'bg-yellow-400',
  low: 'bg-slate-300',
}

export function CardEditable({
  card,
  colIdx,
  cardIdx,
  boardId,
  modalOpen,
  onModalOpenChange,
}: {
  card: CardModel
  colIdx: number
  cardIdx: number
  boardId: string
  modalOpen: boolean
  onModalOpenChange: (next: boolean) => void
}): JSX.Element {
  const [mode, setMode] = useState<'view' | 'edit'>('view')
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useBoardMutation(boardId)
  const settings = useBoardSettings(boardId)
  const tagColors = useTagColors(boardId)
  const showCheckbox = settings.show_checkbox
  const compact = settings.card_display_mode === 'compact'
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
        links: card.links ?? [],
        priority: card.priority ?? '',
        due: card.due ?? '',
        assignee: card.assignee ?? '',
      })
    }
    Promise.resolve().then(() => setMode('view'))
  }

  const cancel = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    Promise.resolve().then(() => setMode('view'))
  }

  if (mode === 'edit') {
    return (
      <div className="rounded-md bg-[color:var(--color-surface)] p-3 shadow-sm ring-2 ring-[color:var(--accent-500)]">
        <input
          ref={inputRef}
          aria-label="card title"
          defaultValue={card.title}
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === 'Enter') { e.preventDefault(); commit() }
            else if (e.key === 'Escape') { e.preventDefault(); cancel() }
          }}
          className="w-full bg-transparent text-sm font-semibold outline-none dark:text-slate-100"
        />
      </div>
    )
  }

  return (
    <>
      <div className={`group relative rounded-md bg-[color:var(--color-surface)] shadow-sm ring-1 ring-[color:var(--color-border)] ${compact ? 'p-2 text-xs' : 'p-3 text-sm'}`}>
        <div className="flex items-start gap-2">
          {showCheckbox && (
            <button
              type="button"
              aria-label={card.completed ? 'mark incomplete' : 'mark complete'}
              onClick={(e) => {
                e.stopPropagation()
                mutation.mutate({
                  type: 'complete_card',
                  col_idx: colIdx,
                  card_idx: cardIdx,
                })
              }}
              className={`mt-1 h-4 w-4 shrink-0 rounded-full border ${
                card.completed ? 'bg-slate-400 border-slate-400' : 'border-slate-300'
              }`}
            />
          )}
          <div className="flex-1">
            <div
              onDoubleClick={() => setMode('edit')}
              className="flex items-start gap-2"
            >
              {card.priority && (
                <span
                  aria-label={`priority ${card.priority}`}
                  className={`mt-1.5 h-2 w-2 shrink-0 rounded-full ${PRIORITY_DOT[card.priority] ?? 'bg-slate-300'}`}
                />
              )}
              <h3 className={`text-sm font-semibold dark:text-slate-100 ${card.completed ? 'text-slate-400' : ''}`}>
                {card.title}
              </h3>
            </div>
            <button
              type="button"
              aria-label="open card details"
              onClick={() => onModalOpenChange(true)}
              className="mt-2 flex min-h-6 w-full items-center justify-between gap-2 rounded text-left hover:bg-[color:var(--color-hover)]"
            >
              {card.tags && card.tags.length > 0 ? (
                <ul className="flex flex-wrap gap-1">
                  {card.tags.map((t) => {
                    const style = tagChipStyle(tagColors[t])
                    return (
                      <li
                        key={t}
                        style={style}
                        className={
                          style
                            ? 'rounded px-1.5 py-0.5 text-xs'
                            : 'rounded bg-[color:var(--color-column-bg)] px-1.5 py-0.5 text-xs text-slate-700 dark:text-slate-200'
                        }
                      >
                        {t}
                      </li>
                    )
                  })}
                </ul>
              ) : (
                <span className="text-xs text-slate-500 dark:text-slate-500 group-hover:text-slate-700 dark:group-hover:text-slate-300">Click to edit details</span>
              )}
              <span aria-hidden className="text-xs text-slate-500 dark:text-slate-500 opacity-0 group-hover:opacity-100 group-hover:text-slate-700 dark:group-hover:text-slate-300">↗</span>
            </button>
          </div>
          <button
            type="button"
            aria-label="delete card"
            onClick={(e) => {
              e.stopPropagation()
              stageDelete(
                () => mutation.mutate({ type: 'delete_card', col_idx: colIdx, card_idx: cardIdx }),
                card.title,
              )
            }}
            className="opacity-0 group-hover:opacity-100 mt-1 text-xs text-slate-400 hover:text-red-500"
          >
            ✕
          </button>
        </div>
      </div>
      <Suspense fallback={null}>
        <CardDetailModal
          card={card}
          colIdx={colIdx}
          cardIdx={cardIdx}
          boardId={boardId}
          open={modalOpen}
          onOpenChange={onModalOpenChange}
        />
      </Suspense>
    </>
  )
}
