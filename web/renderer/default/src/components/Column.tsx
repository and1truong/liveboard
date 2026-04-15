import type { Column as ColumnModel } from '@shared/types.js'
import { Card } from './Card.js'

export function Column({ column }: { column: ColumnModel }): JSX.Element {
  const cards = column.cards ?? []
  return (
    <section className="flex w-72 shrink-0 flex-col rounded-lg bg-slate-100 p-3">
      <header className="mb-3 flex items-center justify-between">
        <h2 className="text-sm font-semibold text-slate-800">{column.name}</h2>
        <span className="text-xs text-slate-500">{cards.length}</span>
      </header>
      <ul className="flex flex-col gap-2">
        {cards.map((card, i) => (
          <li key={`${column.name}-${i}`}>
            <Card card={card} />
          </li>
        ))}
      </ul>
    </section>
  )
}
