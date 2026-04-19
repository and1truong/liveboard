import { useMemo } from 'react'
import type { Board } from '@shared/types.js'
import { useAppSettings } from './useAppSettings.js'

export function useAvailableTags(board: Board | null | undefined): string[] {
  const workspaceTags = useAppSettings().tags ?? []
  return useMemo(() => collectTags(board, workspaceTags), [board, workspaceTags])
}

export function collectTags(
  board: Board | null | undefined,
  workspaceTags: string[] = [],
): string[] {
  const set = new Set<string>(workspaceTags)
  for (const col of board?.columns ?? []) {
    for (const card of col.cards ?? []) {
      for (const t of card.tags ?? []) set.add(t)
    }
  }
  return Array.from(set).sort((a, b) => a.localeCompare(b))
}

export function useTagCounts(
  board: Board | null | undefined,
  hideCompleted = false,
): Record<string, number> {
  return useMemo(() => collectTagCounts(board, hideCompleted), [board, hideCompleted])
}

export function collectTagCounts(
  board: Board | null | undefined,
  hideCompleted = false,
): Record<string, number> {
  const counts: Record<string, number> = {}
  for (const col of board?.columns ?? []) {
    for (const card of col.cards ?? []) {
      if (hideCompleted && card.completed) continue
      for (const t of card.tags ?? []) {
        counts[t] = (counts[t] ?? 0) + 1
      }
    }
  }
  return counts
}
