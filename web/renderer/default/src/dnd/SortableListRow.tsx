import { Suspense, lazy, useCallback, useEffect, useRef, useState } from 'react'
import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import type { Card as CardModel } from '@shared/types.js'
import { CardContextMenu } from '../components/CardContextMenu.js'
import { QuickEditDialog } from '../components/QuickEditDialog.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { stageDelete } from '../mutations/undoable.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { useTagColors } from '../queries.js'
import { tagChipStyle } from '../utils/tagColor.js'
import { formatDueBadge } from '../utils/dueDate.js'
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

const menuItemCls =
  'cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-[color:var(--color-column-bg)]'

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
  const tagColors = useTagColors(boardId)
  const { drop } = useDragState()
  const showDropLine =
    !isDragging &&
    drop?.type === 'card' &&
    drop.colIdx === colIdx &&
    drop.cardIdx === cardIdx

  const [popKey, setPopKey] = useState(0)
  const firstRenderRef = useRef(true)
  useEffect(() => {
    if (firstRenderRef.current) {
      firstRenderRef.current = false
      return
    }
    setPopKey((k) => k + 1)
  }, [card.completed])

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

  const due = card.due ? formatDueBadge(card.due) : null
  const hasMeta =
    !!card.priority || !!card.assignee || (card.tags && card.tags.length > 0)

  const dueClass = due?.overdue
    ? 'text-[color:var(--color-due-overdue)] bg-[color:var(--color-due-overdue-soft)]'
    : due?.soon
      ? 'text-[color:var(--accent-600)] bg-[color:var(--color-column-bg)] dark:text-[color:var(--accent-500)]'
      : 'text-slate-500 dark:text-slate-400'

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
          className={`group relative grid grid-cols-[auto_auto_1fr_auto] items-start gap-3 border-b border-[color:var(--color-border-dashed)] py-2 outline-none last:border-b-0 hover:bg-[color:var(--color-column-bg)]/60 focus:bg-[color:var(--color-column-bg)]/60 ${
            card.completed ? 'opacity-60' : ''
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
            className="mt-[3px] cursor-grab text-slate-300 opacity-0 group-hover:opacity-100 active:cursor-grabbing dark:text-slate-600"
          >
            ⋮⋮
          </button>
          <button
            key={popKey}
            type="button"
            aria-label={card.completed ? 'mark incomplete' : 'mark complete'}
            onClick={(e) => {
              e.stopPropagation()
              mutation.mutate({ type: 'complete_card', col_idx: colIdx, card_idx: cardIdx })
            }}
            className={`mt-[2px] flex h-[18px] w-[18px] shrink-0 items-center justify-center rounded-full border-[1.5px] transition-colors lb-check-pop ${
              card.completed
                ? 'border-[color:var(--accent-500)] bg-[color:var(--accent-500)] text-white'
                : 'border-[color:var(--color-border-dashed)] hover:border-[color:var(--accent-500)]'
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
            className="min-w-0 text-left"
          >
            <div className="flex min-w-0 flex-wrap items-baseline gap-x-2 gap-y-0.5">
              <span
                className={`truncate text-sm font-medium text-slate-800 dark:text-slate-100 ${
                  card.completed ? 'line-through decoration-[color:var(--color-text-subtle)]' : ''
                }`}
              >
                {card.title}
              </span>
              {due && (
                <span
                  className={`inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[11px] font-medium leading-none ${dueClass}`}
                >
                  <svg width="11" height="11" viewBox="0 0 12 12" fill="none" aria-hidden>
                    <rect x="1.5" y="2.5" width="9" height="8" rx="1.5" stroke="currentColor" strokeWidth="1.3" />
                    <path d="M1.5 5H10.5" stroke="currentColor" strokeWidth="1.3" />
                    <path d="M4 1.5V3M8 1.5V3" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
                  </svg>
                  {due.label}
                </span>
              )}
            </div>
            {card.body && (
              <div
                className={`mt-0.5 line-clamp-2 text-xs text-slate-500 dark:text-slate-400 ${
                  card.completed ? 'line-through decoration-[color:var(--color-text-subtle)]' : ''
                }`}
              >
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
                {card.assignee && (
                  <span className="rounded bg-[color:var(--color-column-bg)] px-1.5 py-0.5 text-[10px] text-slate-600 dark:text-slate-300">
                    👤 {card.assignee}
                  </span>
                )}
                {card.tags?.map((t) => {
                  const style = tagChipStyle(tagColors[t])
                  return (
                    <span
                      key={t}
                      style={style}
                      className={
                        style
                          ? 'rounded px-1.5 py-0.5 text-[10px]'
                          : 'rounded bg-[color:var(--color-column-bg)] px-1.5 py-0.5 text-[10px] text-slate-700 dark:text-slate-200'
                      }
                    >
                      {t}
                    </span>
                  )
                })}
              </div>
            )}
          </button>
          <div className="flex items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100">
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
              className="flex h-6 w-6 items-center justify-center rounded text-slate-400 hover:bg-[color:var(--color-column-bg)] hover:text-slate-700 dark:hover:text-slate-200"
            >
              <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden>
                <path d="M2.5 2.5L9.5 9.5M9.5 2.5L2.5 9.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
              </svg>
            </button>
            <DropdownMenu.Root>
              <DropdownMenu.Trigger
                aria-label="card menu"
                onClick={(e) => e.stopPropagation()}
                className="flex h-6 w-6 items-center justify-center rounded text-slate-400 hover:bg-[color:var(--color-column-bg)] hover:text-slate-700 data-[state=open]:opacity-100 dark:hover:text-slate-200"
              >
                <svg width="14" height="14" viewBox="0 0 14 14" fill="currentColor" aria-hidden>
                  <circle cx="3" cy="7" r="1.2" />
                  <circle cx="7" cy="7" r="1.2" />
                  <circle cx="11" cy="7" r="1.2" />
                </svg>
              </DropdownMenu.Trigger>
              <DropdownMenu.Portal>
                <DropdownMenu.Content
                  sideOffset={4}
                  align="end"
                  className="z-50 min-w-40 rounded-md border border-[color:var(--color-border)] bg-[color:var(--color-surface)] p-1 shadow-[var(--shadow-raised)] dark:text-slate-100"
                >
                  <DropdownMenu.Item className={menuItemCls} onSelect={() => setQuickOpen(true)}>
                    Quick edit
                  </DropdownMenu.Item>
                  <DropdownMenu.Item className={menuItemCls} onSelect={() => setModalOpen(true)}>
                    Open details
                  </DropdownMenu.Item>
                  <DropdownMenu.Item
                    className={menuItemCls}
                    onSelect={() =>
                      mutation.mutate({ type: 'complete_card', col_idx: colIdx, card_idx: cardIdx })
                    }
                  >
                    {card.completed ? 'Mark incomplete' : 'Mark complete'}
                  </DropdownMenu.Item>
                  <DropdownMenu.Separator className="my-1 h-px bg-[color:var(--color-border)]" />
                  <DropdownMenu.Item
                    className={menuItemCls + ' text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950'}
                    onSelect={() =>
                      stageDelete(
                        () => mutation.mutate({ type: 'delete_card', col_idx: colIdx, card_idx: cardIdx }),
                        card.title,
                      )
                    }
                  >
                    Delete
                  </DropdownMenu.Item>
                </DropdownMenu.Content>
              </DropdownMenu.Portal>
            </DropdownMenu.Root>
          </div>
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
