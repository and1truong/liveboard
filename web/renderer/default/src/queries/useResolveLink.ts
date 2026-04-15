import { useQuery } from '@tanstack/react-query'
import { useClient } from '../queries.js'

export interface ResolvedLink {
  boardName: string
  cardTitle: string
  colIdx: number
  cardIdx: number
}

export function useResolveLink(target: string): ResolvedLink | null {
  const client = useClient()
  const q = useQuery({
    queryKey: ['resolve', target],
    queryFn: async (): Promise<ResolvedLink | null> => {
      const idx = target.indexOf(':')
      if (idx <= 0) return null
      const boardSlug = target.slice(0, idx)
      const cardId = target.slice(idx + 1)
      let board
      try {
        board = await client.getBoard(boardSlug)
      } catch {
        return null
      }
      const cols = board.columns ?? []
      for (let c = 0; c < cols.length; c++) {
        const cards = cols[c]?.cards ?? []
        for (let k = 0; k < cards.length; k++) {
          if (cards[k].id === cardId) {
            return {
              boardName: board.name ?? boardSlug,
              cardTitle: cards[k].title ?? '',
              colIdx: c,
              cardIdx: k,
            }
          }
        }
      }
      return null
    },
    enabled: target.length > 0,
  })
  return q.data ?? null
}
