import { useCallback, useState } from 'react'
import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
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
          className={`group/sortable relative outline-none rounded-md ${
            isFocused ? 'ring-2 ring-[color:var(--accent-500)] ring-offset-2' : ''
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
            className="absolute -left-4 top-3 cursor-grab text-slate-300 opacity-0 group-hover/sortable:opacity-100 active:cursor-grabbing"
          >
            ⋮⋮
          </button>
          <CardEditable
            card={card}
            colIdx={colIdx}
            cardIdx={cardIdx}
            boardId={boardId}
            modalOpen={modalOpen}
            onModalOpenChange={setModalOpen}
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
