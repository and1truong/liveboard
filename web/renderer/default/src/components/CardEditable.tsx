import { Suspense, lazy, useState, useRef, useEffect } from 'react'
import type { Card as CardModel } from '@shared/types.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { stageDelete } from '../mutations/undoable.js'
const CardDetailModal = lazy(() =>
  import('./CardDetailModal.js').then((m) => ({ default: m.CardDetailModal })),
)
import { useBoardSettings } from '../queries/useBoardSettings.js'
import { useTagColors } from '../queries.js'
import { AttachmentBadge } from './AttachmentBadge.js'

function truncateWords(text: string, maxWords: number): string {
  const words = text.split(/\s+/)
  if (words.length <= maxWords) return text
  return words.slice(0, maxWords).join(' ') + '...'
}

const PRIORITY_STRIPE: Record<string, string> = {
  critical: 'bg-red-500',
  high: 'bg-orange-500',
  medium: 'bg-yellow-400',
  low: 'bg-slate-300 dark:bg-slate-600',
}

export function CardEditable({
  card,
  colIdx,
  cardIdx,
  boardId,
  modalOpen,
  onModalOpenChange,
  isActive = false,
}: {
  card: CardModel
  colIdx: number
  cardIdx: number
  boardId: string
  modalOpen: boolean
  onModalOpenChange: (next: boolean) => void
  isActive?: boolean
}): JSX.Element {
  const [mode, setMode] = useState<'view' | 'edit'>('view')
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useBoardMutation(boardId)
  const settings = useBoardSettings(boardId)
  const tagColors = useTagColors()
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
      <div className="rounded-md bg-[color:var(--color-column-bg)] p-3 ring-2 ring-[color:var(--accent-500)]">
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
      <div
        className={`group relative overflow-hidden rounded-md transition-[background-color,box-shadow] duration-150 hover:shadow-[0_4px_14px_rgba(15,23,42,0.14),0_1px_4px_rgba(15,23,42,0.06)] ${
          isActive
            ? 'bg-[color-mix(in_srgb,#000_2%,color-mix(in_srgb,var(--color-surface)_50%,var(--color-column-bg)))] dark:bg-[color-mix(in_srgb,#fff_5%,color-mix(in_srgb,var(--color-surface)_50%,var(--color-column-bg)))]'
            : 'bg-[color-mix(in_srgb,var(--color-surface)_50%,var(--color-column-bg))]'
        } ${compact ? 'py-2 pl-3 pr-2 text-xs' : 'py-3 pl-3.5 pr-3 text-sm'}`}
      >
        <span
          aria-hidden
          className={`pointer-events-none absolute left-0 top-0 bottom-0 w-[3px] ${
            card.priority
              ? (PRIORITY_STRIPE[card.priority] ?? PRIORITY_STRIPE.low)
              : 'bg-[color:var(--color-border-dashed)]'
          }`}
        />
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
            <div onDoubleClick={() => setMode('edit')}>
              <h3 className={`inline text-sm font-semibold dark:text-slate-100 ${card.completed ? 'text-slate-400' : ''}`}>
                {card.title}
              </h3>
              <button
                type="button"
                aria-label="open card details"
                onClick={() => onModalOpenChange(true)}
                className="ml-1.5 opacity-0 group-hover:opacity-100 inline-flex items-center rounded bg-slate-100 dark:bg-slate-700 px-1.5 py-0.5 text-[10px] font-medium text-slate-500 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-600 transition-opacity"
              >
                ↗
              </button>
              <AttachmentBadge attachments={card.attachments} />
            </div>
            {!compact && card.body && (
              <p className="mt-1.5 text-xs text-slate-500 dark:text-slate-400">
                {truncateWords(card.body, 120)}
              </p>
            )}
            {card.tags && card.tags.length > 0 && (
              <ul className="mt-1.5 flex flex-wrap gap-x-2 gap-y-0.5">
                {card.tags.map((t) => {
                  const color = tagColors[t]
                  return (
                    <li
                      key={t}
                      style={color ? { color } : undefined}
                      className={
                        color
                          ? 'text-[11px] font-medium uppercase tracking-wide'
                          : 'text-[11px] font-medium uppercase tracking-wide text-slate-600 dark:text-slate-300'
                      }
                    >
                      {t}
                    </li>
                  )
                })}
              </ul>
            )}
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
