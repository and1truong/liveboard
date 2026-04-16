import type { Card } from '@shared/types.js'

export interface BoardFilter {
  query: string
  tags: string[]
  hideCompleted: boolean
}

export const EMPTY_FILTER: BoardFilter = { query: '', tags: [], hideCompleted: false }

export function filterCard(card: Card, filter: BoardFilter): boolean {
  if (filter.hideCompleted && card.completed) return false

  if (filter.tags.length > 0) {
    const cardTags = card.tags ?? []
    for (const t of filter.tags) {
      if (!cardTags.includes(t)) return false
    }
  }

  const q = filter.query.trim().toLowerCase()
  if (q) {
    const hay = [card.title, card.body ?? '', ...(card.tags ?? []), card.assignee ?? '']
      .join(' ')
      .toLowerCase()
    if (!hay.includes(q)) return false
  }

  return true
}

export function activeFilterCount(filter: BoardFilter): number {
  let n = 0
  if (filter.query.trim()) n++
  n += filter.tags.length
  if (filter.hideCompleted) n++
  return n
}
