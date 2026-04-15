import { useSortable, SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import type { Column as ColumnModel } from '@shared/types.js'
import { ColumnHeader } from '../components/ColumnHeader.js'
import { AddCardButton } from '../components/AddCardButton.js'
import { SortableCard } from './SortableCard.js'
import { encodeCardId, encodeColumnId } from './cardId.js'

export function SortableColumn({
  column,
  colIdx,
  allColumnNames,
  boardId,
}: {
  column: ColumnModel
  colIdx: number
  allColumnNames: string[]
  boardId: string
}): JSX.Element {
  const id = encodeColumnId(column.name)
  const cards = column.cards ?? []
  const cardIds = cards.map((_, i) => encodeCardId(colIdx, i))

  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id,
    data: { type: 'column', name: column.name, col_idx: colIdx },
  })
  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <section
      ref={setNodeRef}
      style={style}
      className="flex w-72 shrink-0 flex-col rounded-lg bg-slate-100 p-3 dark:bg-slate-900"
    >
      <div className="mb-3 flex items-center gap-2">
        <button
          type="button"
          aria-label={`drag column ${column.name}`}
          {...attributes}
          {...listeners}
          className="cursor-grab text-slate-400 hover:text-slate-600 active:cursor-grabbing"
        >
          ⋮⋮
        </button>
        <div className="flex-1">
          <ColumnHeader
            name={column.name}
            cardCount={cards.length}
            colIdx={colIdx}
            allColumnNames={allColumnNames}
            boardId={boardId}
          />
        </div>
      </div>
      <SortableContext items={cardIds} strategy={verticalListSortingStrategy}>
        <ul className="flex flex-col gap-2">
          {cards.map((card, i) => (
            <li key={`${column.name}-${i}`}>
              <SortableCard card={card} colIdx={colIdx} cardIdx={i} boardId={boardId} />
            </li>
          ))}
        </ul>
      </SortableContext>
      <AddCardButton columnName={column.name} boardId={boardId} />
    </section>
  )
}
