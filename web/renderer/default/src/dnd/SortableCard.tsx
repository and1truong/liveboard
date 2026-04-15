import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import type { Card as CardModel } from '@shared/types.js'
import { CardEditable } from '../components/CardEditable.js'
import { encodeCardId } from './cardId.js'

export function SortableCard({
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
  const id = encodeCardId(colIdx, cardIdx)
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id,
    data: { type: 'card', col_idx: colIdx, card_idx: cardIdx },
  })
  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }
  return (
    <div ref={setNodeRef} style={style} className="group/sortable relative">
      <button
        type="button"
        aria-label="drag card"
        {...attributes}
        {...listeners}
        className="absolute -left-4 top-3 cursor-grab text-slate-300 opacity-0 group-hover/sortable:opacity-100 active:cursor-grabbing"
      >
        ⋮⋮
      </button>
      <CardEditable card={card} colIdx={colIdx} cardIdx={cardIdx} boardId={boardId} />
    </div>
  )
}
