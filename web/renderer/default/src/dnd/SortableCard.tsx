import { useCallback, useState } from 'react'
import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { GripVertical } from 'lucide-react'
import type { Card as CardModel } from '@shared/types.js'
import { CardEditable } from '../components/CardEditable.js'
import { CardContextMenu } from '../components/CardContextMenu.js'
import { QuickEditDialog } from '../components/QuickEditDialog.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { stageDelete } from '../mutations/undoable.js'
import { useBoardFocus, useCardFocus } from '../contexts/BoardFocusContext.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { encodeCardId } from './cardId.js'
import { useDragState } from './BoardDndContext.js'

export function SortableCard({
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
  const { isFocused, ref: focusRef } = useCardFocus(colIdx, cardIdx)
  const { focused, setFocused, move } = useBoardFocus()
  const { activeCard, setActiveCard } = useActiveBoard()
  const modalOpen = activeCard?.colIdx === colIdx && activeCard?.cardIdx === cardIdx
  const setModalOpen = useCallback((open: boolean) => {
    setActiveCard(open ? { colIdx, cardIdx } : null)
  }, [setActiveCard, colIdx, cardIdx])
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

  const showFocusedTabStop =
    isFocused || (focused === null && colIdx === 0 && cardIdx === 0)

  const onKeyDown = (e: React.KeyboardEvent): void => {
    if (e.defaultPrevented) return
    const tag = (e.target as HTMLElement).tagName
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'BUTTON') return
    switch (e.key) {
      case 'ArrowUp':    e.preventDefault(); move('up'); break
      case 'ArrowDown':  e.preventDefault(); move('down'); break
      case 'ArrowLeft':  e.preventDefault(); move('left'); break
      case 'ArrowRight': e.preventDefault(); move('right'); break
      case 'Enter':      e.preventDefault(); setModalOpen(true); break
      case 'Delete':
      case 'Backspace':
        e.preventDefault()
        stageDelete(
          () => mutation.mutate({ type: 'delete_card', col_idx: colIdx, card_idx: cardIdx }),
          card.title,
        )
        break
    }
  }

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
          ref={(el) => {
            setNodeRef(el)
            focusRef(el)
          }}
          style={style}
          tabIndex={showFocusedTabStop ? 0 : -1}
          onFocus={() => setFocused({ colIdx, cardIdx })}
          onKeyDown={onKeyDown}
          className={`group/sortable relative outline-none rounded-md transition-[box-shadow] duration-150 ${
            isFocused
              ? 'shadow-[0_14px_32px_-12px_rgba(15,23,42,0.28),0_4px_10px_-4px_rgba(15,23,42,0.1)]'
              : ''
          }`}
        >
          {showDropLine && (
            <div
              aria-hidden
              className="pointer-events-none absolute -top-[5.5px] left-0 right-0 h-[3px] rounded-full bg-[color:var(--accent-500)]"
            />
          )}
          <button
            type="button"
            aria-label="drag card"
            {...attributes}
            {...listeners}
            className="absolute -left-5 top-3 flex h-5 w-4 items-center justify-center cursor-grab text-slate-500 opacity-0 group-hover/sortable:opacity-100 hover:text-slate-700 active:cursor-grabbing dark:text-slate-400 dark:hover:text-slate-200"
          >
            <GripVertical size={14} aria-hidden strokeWidth={2.25} />
          </button>
          <CardEditable
            card={card}
            colIdx={colIdx}
            cardIdx={cardIdx}
            boardId={boardId}
            modalOpen={modalOpen}
            onModalOpenChange={setModalOpen}
            isActive={isFocused}
          />
        </div>
      </CardContextMenu>
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
