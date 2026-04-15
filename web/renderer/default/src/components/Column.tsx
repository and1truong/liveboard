import type { Column as ColumnModel } from '@shared/types.js'
import { CardEditable } from './CardEditable.js'
import { ColumnHeader } from './ColumnHeader.js'
import { AddCardButton } from './AddCardButton.js'

export function Column({
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
  const cards = column.cards ?? []
  return (
    <section className="flex w-72 shrink-0 flex-col rounded-lg bg-slate-100 p-3">
      <ColumnHeader
        name={column.name}
        cardCount={cards.length}
        colIdx={colIdx}
        allColumnNames={allColumnNames}
        boardId={boardId}
      />
      <ul className="flex flex-col gap-2">
        {cards.map((card, i) => (
          <li key={`${column.name}-${i}`}>
            <CardEditable card={card} colIdx={colIdx} cardIdx={i} boardId={boardId} />
          </li>
        ))}
      </ul>
      <AddCardButton columnName={column.name} boardId={boardId} />
    </section>
  )
}
