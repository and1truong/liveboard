import { Suspense, lazy, useCallback, useState } from 'react'
import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import type { Card as CardModel } from '@shared/types.js'
import { CardContextMenu } from '../components/CardContextMenu.js'
import { QuickEditDialog } from '../components/QuickEditDialog.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { stageDelete } from '../mutations/undoable.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { encodeCardId } from './cardId.js'
import { useDragState } from './BoardDndContext.js'

const CardDetailModal = lazy(() =>
  import('../components/CardDetailModal.js').then((m) => ({ default: m.CardDetailModal })),
)

const PRIORITY_CHIP: Record<string, string> = {
  critical: 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300',
  high: 'bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-300',
  medium: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/40 dark:text-yellow-300',
  low: 'bg-[color:var(--color-column-bg)] text-slate-600 dark:text-slate-300',
}

export function SortableListRow({
  card,
  colIdx,
  cardIdx,
  boardId,
  allColumnNames,
}: {
  card: CardModel
  colIdx: number
  cardIdx: number
  boardId: string
  allColumnNames: string[]
}): JSX.Element {
  const id = encodeCardId(colIdx, cardIdx)
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id,
    data: { type: 'card', col_idx: colIdx, card_idx: cardIdx },
  })
  const { activeCard, setActiveCard } = useActiveBoard()
  const modalOpen = activeCard?.colIdx === colIdx && activeCard?.cardIdx === cardIdx
  const setModalOpen = useCallback(
    (open: boolean) => setActiveCard(open ? { colIdx, cardIdx } : null),
    [setActiveCard, colIdx, cardIdx],
  )
  const [quickOpen, setQuickOpen] = useState(false)
  const mutation = useBoardMutation(boardId)
  const { drop } = useDragState()
  const showDropLine =
    !isDragging &&
    drop?.type === 'card' &&
    drop.colIdx === colIdx &&
    drop.cardIdx === cardIdx

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  const onKeyDown = (e: React.KeyboardEvent): void => {
    if (e.defaultPrevented) return
    const tag = (e.target as HTMLElement).tagName
    if (tag === 'INPUT' || tag === 'TEXTAREA') return
    if (e.key === 'Enter') {
      e.preventDefault()
      setModalOpen(true)
    } else if (e.key === 'Delete' || e.key === 'Backspace') {
      e.preventDefault()
      stageDelete(
        () => mutation.mutate({ type: 'delete_card', col_idx: colIdx, card_idx: cardIdx }),
        card.title,
      )
    }
  }

  const hasMeta =
    !!card.priority || !!card.due || !!card.assignee || (card.tags && card.tags.length > 0)

  return (
    <>
      <CardContextMenu
        card={card}
        colIdx={colIdx}
        cardIdx={cardIdx}
        boardId={boardId}
        allColumnNames={allColumnNames}
        onQuickEdit={() => setQuickOpen(true)}
        onOpenDetail={() => setModalOpen(true)}
      >
        <div
          ref={setNodeRef}
          style={style}
          tabIndex={0}
          onKeyDown={onKeyDown}
          className={`group relative flex items-start gap-3 border-b border-[color:var(--color-border-dashed)] px-3 py-2 outline-none last:border-b-0 hover:bg-[color:var(--color-column-bg)] focus:bg-[color:var(--color-column-bg)] ${
            card.completed ? 'opacity-50' : ''
          }`}
        >
          {showDropLine && (
            <div
              aria-hidden
              className="pointer-events-none absolute -top-[1.5px] left-0 right-0 h-[3px] rounded-full bg-[color:var(--accent-500)]"
            />
          )}
          <button
            type="button"
            aria-label="drag card"
            {...attributes}
            {...listeners}
            className="mt-1 cursor-grab text-slate-300 opacity-0 group-hover:opacity-100 active:cursor-grabbing dark:text-slate-600"
          >
            ⋮⋮
          </button>
          <button
            type="button"
            aria-label={card.completed ? 'mark incomplete' : 'mark complete'}
            onClick={(e) => {
              e.stopPropagation()
              mutation.mutate({ type: 'complete_card', col_idx: colIdx, card_idx: cardIdx })
            }}
            className={`mt-1 flex h-4 w-4 shrink-0 items-center justify-center rounded-full border transition-colors ${
              card.completed
                ? 'border-[color:var(--accent-500)] bg-[color:var(--accent-500)] text-white'
                : 'border-[color:var(--color-border)] hover:border-[color:var(--accent-500)]'
            }`}
          >
            {card.completed && (
              <svg width="10" height="10" viewBox="0 0 10 10" fill="none" aria-hidden>
                <path d="M2 5L4.2 7L8 3" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            )}
          </button>
          <button
            type="button"
            aria-label="open card details"
            onClick={() => setModalOpen(true)}
            className="min-w-0 flex-1 text-left"
          >
            <div className="truncate text-sm font-medium text-slate-800 dark:text-slate-100">
              {card.title}
            </div>
            {card.body && (
              <div className="mt-0.5 line-clamp-2 text-xs text-slate-500 dark:text-slate-400">
                {card.body}
              </div>
            )}
            {hasMeta && (
              <div className="mt-1 flex flex-wrap items-center gap-1">
                {card.priority && (
                  <span
                    className={`rounded px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide ${PRIORITY_CHIP[card.priority] ?? PRIORITY_CHIP.low}`}
                  >
                    {card.priority}
                  </span>
                )}
                {card.due && (
                  <span className="rounded bg-[color:var(--color-column-bg)] px-1.5 py-0.5 text-[10px] text-slate-600 dark:text-slate-300">
                    📅 {card.due}
                  </span>
                )}
                {card.assignee && (
                  <span className="rounded bg-[color:var(--color-column-bg)] px-1.5 py-0.5 text-[10px] text-slate-600 dark:text-slate-300">
                    👤 {card.assignee}
                  </span>
                )}
                {card.tags?.map((t) => (
                  <span
                    key={t}
                    className="rounded bg-[color:var(--color-column-bg)] px-1.5 py-0.5 text-[10px] text-slate-700 dark:text-slate-200"
                  >
                    {t}
                  </span>
                ))}
              </div>
            )}
          </button>
        </div>
      </CardContextMenu>
      <Suspense fallback={null}>
        <CardDetailModal
          card={card}
          colIdx={colIdx}
          cardIdx={cardIdx}
          boardId={boardId}
          open={modalOpen}
          onOpenChange={setModalOpen}
        />
      </Suspense>
      <QuickEditDialog
        card={card}
        colIdx={colIdx}
        cardIdx={cardIdx}
        boardId={boardId}
        open={quickOpen}
        onOpenChange={setQuickOpen}
      />
    </>
  )
}
