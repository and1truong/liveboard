import { useSortable, SortableContext, rectSortingStrategy, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { useDroppable } from '@dnd-kit/core'
import { CSS } from '@dnd-kit/utilities'
import type { Column as ColumnModel } from '@shared/types.js'
import { ColumnHeader } from '../components/ColumnHeader.js'
import { AddCardButton } from '../components/AddCardButton.js'
import { SortableCard } from './SortableCard.js'
import { encodeCardId, encodeColumnId, encodeColumnEndId } from './cardId.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { useBoardSettings } from '../queries/useBoardSettings.js'
import { useDragState } from './BoardDndContext.js'
import { useBoardFilter } from '../contexts/BoardFilterContext.js'
import { activeFilterCount, filterCard } from '../utils/cardFilter.js'

export function SortableColumn({
  column,
  colIdx,
  allColumnNames,
  boardId,
  collapsed = false,
  isFocusMode = false,
}: {
  column: ColumnModel
  colIdx: number
  allColumnNames: string[]
  boardId: string
  collapsed?: boolean
  isFocusMode?: boolean
}): JSX.Element {
  const id = encodeColumnId(column.name)
  const cards = column.cards ?? []
  const { filter } = useBoardFilter()
  const visibleCards = cards.filter((card) => filterCard(card, filter))
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

  const filtersActive = activeFilterCount(filter) > 0
  const hasMatches = filtersActive && visibleCards.length > 0

  if (collapsed && !isFocusMode) {
    return (
      <section
        ref={setNodeRef}
        style={style}
        aria-label={`collapsed column ${column.name}${hasMatches ? ` (${visibleCards.length} match)` : ''}`}
        className={`flex w-12 shrink-0 cursor-pointer flex-col items-center rounded-lg bg-[color:var(--color-column-bg)] p-3 ${
          hasMatches ? 'ring-2 ring-[color:var(--accent-500)]/60' : ''
        }`}
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
        <span
          className={`mt-2 text-xs ${
            hasMatches
              ? 'inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-[color:var(--accent-500)] px-1 font-semibold text-white'
              : 'text-slate-500'
          }`}
        >
          {visibleCards.length}
        </span>
      </section>
    )
  }

  const sectionClass = isFocusMode
    ? 'flex w-full flex-1 flex-col rounded-lg bg-[color:var(--color-column-bg)] p-3'
    : `flex ${settings.expand_columns ? 'min-w-[200px] flex-[1_1_0]' : 'w-72 shrink-0'} flex-col rounded-lg bg-[color:var(--color-column-bg)] p-3`

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
          <ColumnEndDropZone columnName={column.name} active={showEndDropLine} />
        </ul>
      </SortableContext>
      <AddCardButton columnName={column.name} boardId={boardId} />
    </section>
  )
}

function ColumnEndDropZone({
  columnName,
  active,
}: {
  columnName: string
  active: boolean
}): JSX.Element {
  const { setNodeRef } = useDroppable({
    id: encodeColumnEndId(columnName),
    data: { type: 'column-end', name: columnName },
  })
  const { isDragActive } = useDragState()
  // Reserve hover area only during a drag, so the column doesn't have permanent dead space below cards.
  const baseClass = isDragActive ? 'min-h-[2rem]' : 'h-0'
  const lineClass = active ? 'h-[3px] min-h-[3px] mt-1 bg-[color:var(--accent-500)] rounded-full' : ''
  return <li ref={setNodeRef} aria-hidden className={`${baseClass} ${lineClass}`} />
}
