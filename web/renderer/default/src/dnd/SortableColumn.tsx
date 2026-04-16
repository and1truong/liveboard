import { useSortable, SortableContext, rectSortingStrategy, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import type { Column as ColumnModel } from '@shared/types.js'
import { ColumnHeader } from '../components/ColumnHeader.js'
import { AddCardButton } from '../components/AddCardButton.js'
import { SortableCard } from './SortableCard.js'
import { encodeCardId, encodeColumnId } from './cardId.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { useBoardSettings } from '../queries/useBoardSettings.js'
import { useDragState } from './BoardDndContext.js'

export function SortableColumn({
  column,
  colIdx,
  allColumnNames,
  boardId,
  collapsed = false,
  filterQuery = '',
  hideCompleted = false,
  isFocusMode = false,
}: {
  column: ColumnModel
  colIdx: number
  allColumnNames: string[]
  boardId: string
  collapsed?: boolean
  filterQuery?: string
  hideCompleted?: boolean
  isFocusMode?: boolean
}): JSX.Element {
  const id = encodeColumnId(column.name)
  const cards = column.cards ?? []
  const q = filterQuery.toLowerCase()
  const visibleCards = cards.filter((card) => {
    if (hideCompleted && card.completed) return false
    if (q) {
      const hay = [card.title, card.body ?? '', ...(card.tags ?? []), card.assignee ?? ''].join(' ').toLowerCase()
      return hay.includes(q)
    }
    return true
  })
  const cardIds = visibleCards.map((_, i) => encodeCardId(colIdx, i))
  const mutation = useBoardMutation(boardId)
  const settings = useBoardSettings(boardId)

  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id,
    data: { type: 'column', name: column.name, col_idx: colIdx },
  })
  const { drop } = useDragState()
  const showEndDropLine = !isDragging && drop?.type === 'column' && drop.colIdx === colIdx
  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  const toggleCollapse = (): void => {
    mutation.mutate({ type: 'toggle_column_collapse', col_idx: colIdx })
  }

  if (collapsed && !isFocusMode) {
    return (
      <section
        ref={setNodeRef}
        style={style}
        aria-label={`collapsed column ${column.name}`}
        className="flex w-12 shrink-0 cursor-pointer flex-col items-center rounded-lg bg-slate-100 p-3 dark:bg-slate-900"
        onClick={toggleCollapse}
      >
        <button
          type="button"
          aria-label={`drag column ${column.name}`}
          {...attributes}
          {...listeners}
          onClick={(e) => e.stopPropagation()}
          className="cursor-grab text-slate-400 hover:text-slate-600 active:cursor-grabbing"
        >
          ⋮⋮
        </button>
        <h2
          className="mt-2 text-sm font-semibold text-slate-800 dark:text-slate-100"
          style={{ writingMode: 'vertical-rl' }}
        >
          {column.name}
        </h2>
        <span className="mt-2 text-xs text-slate-500">{visibleCards.length}</span>
      </section>
    )
  }

  const sectionClass = isFocusMode
    ? 'flex w-full flex-1 flex-col rounded-lg bg-slate-100 p-3 dark:bg-slate-900'
    : `flex ${settings.expand_columns ? 'min-w-[200px] flex-[1_1_0]' : 'w-72 shrink-0'} flex-col rounded-lg bg-slate-100 p-3 dark:bg-slate-900`

  return (
    <section
      ref={setNodeRef}
      style={style}
      className={sectionClass}
    >
      <div className="mb-3 flex items-center gap-2">
        {!isFocusMode && (
          <button
            type="button"
            aria-label={`drag column ${column.name}`}
            {...attributes}
            {...listeners}
            className="cursor-grab text-slate-400 hover:text-slate-600 active:cursor-grabbing"
          >
            ⋮⋮
          </button>
        )}
        <div className="flex-1">
          <ColumnHeader
            name={column.name}
            cardCount={visibleCards.length}
            colIdx={colIdx}
            allColumnNames={allColumnNames}
            boardId={boardId}
            collapsed={collapsed}
          />
        </div>
      </div>
      <SortableContext items={cardIds} strategy={isFocusMode ? rectSortingStrategy : verticalListSortingStrategy}>
        <ul
          className={
            isFocusMode
              ? 'min-h-[3rem] grid grid-cols-[repeat(auto-fill,minmax(280px,1fr))] gap-2.5 overflow-y-auto'
              : 'min-h-[3rem] flex flex-col gap-2'
          }
        >
          {visibleCards.map((card, i) => (
            <li key={`${column.name}-${i}`}>
              <SortableCard
                card={card}
                colIdx={colIdx}
                cardIdx={i}
                boardId={boardId}
                allColumnNames={allColumnNames}
              />
            </li>
          ))}
          {showEndDropLine && (
            <li
              aria-hidden
              className="pointer-events-none h-[3px] rounded-full bg-[color:var(--accent-500)]"
            />
          )}
        </ul>
      </SortableContext>
      <AddCardButton columnName={column.name} boardId={boardId} />
    </section>
  )
}
