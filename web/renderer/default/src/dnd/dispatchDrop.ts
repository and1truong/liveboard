import type { Board, MutationOp } from '@shared/types.js'

interface ActiveOver {
  id: string
  data: { current?: Record<string, unknown> }
}

export function dispatchDrop(
  active: ActiveOver,
  over: ActiveOver | null,
  board: Board,
): MutationOp | null {
  if (!over) return null
  const a = active.data.current
  const o = over.data.current
  if (!a || !o) return null

  if (a.type === 'card') {
    const fromCol = a.col_idx as number
    const fromIdx = a.card_idx as number

    if (o.type === 'card') {
      const toCol = o.col_idx as number
      const toIdx = o.card_idx as number
      if (fromCol === toCol && fromIdx === toIdx) return null
      const targetName = board.columns?.[toCol]?.name
      if (targetName == null) return null
      return {
        type: 'reorder_card',
        col_idx: fromCol,
        card_idx: fromIdx,
        before_idx: toIdx,
        target_column: targetName,
      }
    }

    if (o.type === 'column') {
      const targetName = o.name as string
      return {
        type: 'move_card',
        col_idx: fromCol,
        card_idx: fromIdx,
        target_column: targetName,
      }
    }
    return null
  }

  if (a.type === 'column') {
    if (o.type !== 'column') return null
    const fromIdx = a.col_idx as number
    const toIdx = o.col_idx as number
    if (fromIdx === toIdx) return null
    const name = a.name as string
    const cols = board.columns ?? []
    const after_col = fromIdx < toIdx ? (cols[toIdx]?.name ?? '') : ''
    return { type: 'move_column', name, after_col }
  }

  return null
}
