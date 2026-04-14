import type { Board, Column, Card, MutationOp } from './types.js'
import { OpError } from './types.js'

// applyOp returns a new board with op applied. Input is not mutated.
// Mirrors internal/api/v1.Apply semantics. Shared parity vectors guard drift.
export function applyOp(board: Board, op: MutationOp): Board {
  const b: Board = structuredClone(board)

  switch (op.type) {
    case 'add_card': {
      const col = colByName(b, op.column)
      if (!col) throw new OpError('NOT_FOUND', `column ${op.column}`)
      const card: Card = { title: op.title }
      col.cards = op.prepend ? [card, ...(col.cards ?? [])] : [...(col.cards ?? []), card]
      return b
    }
    case 'move_card': {
      const src = colAt(b, op.col_idx)
      cardAt(src, op.card_idx, op.col_idx)
      const card = src.cards[op.card_idx]!
      src.cards = src.cards.filter((_, i) => i !== op.card_idx)
      const dst = colByName(b, op.target_column)
      if (!dst) throw new OpError('NOT_FOUND', `target column ${op.target_column}`)
      dst.cards = [...(dst.cards ?? []), card]
      return b
    }
    case 'reorder_card': {
      const src = colAt(b, op.col_idx)
      cardAt(src, op.card_idx, op.col_idx)
      const card = src.cards[op.card_idx]!
      src.cards = src.cards.filter((_, i) => i !== op.card_idx)
      const dst = colByName(b, op.target_column)
      if (!dst) throw new OpError('NOT_FOUND', `target column ${op.target_column}`)
      const cards = dst.cards ?? []
      if (op.before_idx < 0 || op.before_idx >= cards.length) {
        dst.cards = [...cards, card]
      } else {
        dst.cards = [...cards.slice(0, op.before_idx), card, ...cards.slice(op.before_idx)]
      }
      return b
    }
    case 'edit_card': {
      const col = colAt(b, op.col_idx)
      cardAt(col, op.card_idx, op.col_idx)
      const card = col.cards[op.card_idx]!
      if (op.title !== '') card.title = op.title
      card.body = op.body
      card.tags = op.tags
      card.priority = op.priority
      card.due = op.due
      card.assignee = op.assignee
      return b
    }
    case 'delete_card': {
      const col = colAt(b, op.col_idx)
      cardAt(col, op.card_idx, op.col_idx)
      col.cards = col.cards.filter((_, i) => i !== op.card_idx)
      return b
    }
    case 'complete_card': {
      const col = colAt(b, op.col_idx)
      cardAt(col, op.card_idx, op.col_idx)
      const card = col.cards[op.card_idx]!
      card.completed = !card.completed
      return b
    }
    case 'tag_card': {
      const col = colAt(b, op.col_idx)
      cardAt(col, op.card_idx, op.col_idx)
      const card = col.cards[op.card_idx]!
      const existing = new Set(card.tags ?? [])
      const merged = [...(card.tags ?? [])]
      for (const t of op.tags) {
        if (!existing.has(t)) {
          merged.push(t)
          existing.add(t)
        }
      }
      card.tags = merged
      return b
    }
    default:
      throw new OpError('INTERNAL', `unimplemented op: ${(op as MutationOp).type}`)
  }
}

function colByName(b: Board, name: string): Column | undefined {
  return (b.columns ?? []).find((c) => c.name === name)
}

function colAt(b: Board, idx: number): Column {
  const cols = b.columns ?? []
  if (idx < 0 || idx >= cols.length) {
    throw new OpError('OUT_OF_RANGE', `column index ${idx}`)
  }
  return cols[idx]!
}

function cardAt(col: Column, idx: number, colIdx: number): void {
  const cards = col.cards ?? []
  if (idx < 0 || idx >= cards.length) {
    throw new OpError('OUT_OF_RANGE', `card index ${idx} in column ${colIdx}`)
  }
}
