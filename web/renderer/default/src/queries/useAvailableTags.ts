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

export function useTagCounts(board: Board | null | undefined): Record<string, number> {
  return useMemo(() => collectTagCounts(board), [board])
}

export function collectTagCounts(board: Board | null | undefined): Record<string, number> {
  const counts: Record<string, number> = {}
  for (const col of board?.columns ?? []) {
    for (const card of col.cards ?? []) {
      for (const t of card.tags ?? []) {
        counts[t] = (counts[t] ?? 0) + 1
      }
    }
  }
  return counts
}
