import { useMemo } from 'react'
import type { Board } from '@shared/types.js'

export function useAvailableTags(board: Board | null | undefined): string[] {
  return useMemo(() => collectTags(board), [board])
}

export function collectTags(board: Board | null | undefined): string[] {
  const set = new Set<string>(board?.tags ?? [])
  for (const col of board?.columns ?? []) {
    for (const card of col.cards ?? []) {
      for (const t of card.tags ?? []) set.add(t)
    }
  }
  return Array.from(set).sort((a, b) => a.localeCompare(b))
}
