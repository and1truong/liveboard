import type { Card as CardModel } from '@shared/types.js'

const PRIORITY_DOT: Record<string, string> = {
  critical: 'bg-red-600',
  high: 'bg-orange-500',
  medium: 'bg-yellow-400',
  low: 'bg-slate-300',
}

export function Card({ card }: { card: CardModel }): JSX.Element {
  return (
    <article className="rounded-md bg-white p-3 shadow-sm ring-1 ring-slate-200 dark:bg-slate-800 dark:ring-slate-700">
      <div className="flex items-start gap-2">
        {card.priority && (
          <span
            aria-label={`priority ${card.priority}`}
            className={`mt-1.5 h-2 w-2 shrink-0 rounded-full ${PRIORITY_DOT[card.priority] ?? 'bg-slate-300'}`}
          />
        )}
        <h3 className={`text-sm font-semibold dark:text-slate-100 ${card.completed ? 'line-through text-slate-400 dark:text-slate-500' : ''}`}>
          {card.title}
        </h3>
      </div>
      {card.tags && card.tags.length > 0 && (
        <ul className="mt-2 flex flex-wrap gap-1">
          {card.tags.map((t) => (
            <li key={t} className="rounded bg-slate-100 px-1.5 py-0.5 text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
              {t}
            </li>
          ))}
        </ul>
      )}
    </article>
  )
}
