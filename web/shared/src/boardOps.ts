import type { Board, MutationOp } from './types.js'
import { OpError } from './types.js'

// applyOp returns a new board with op applied. Input is not mutated.
// Mirrors internal/api/v1.Apply semantics. Shared parity vectors guard drift.
export function applyOp(board: Board, op: MutationOp): Board {
  const b: Board = structuredClone(board)

  switch (op.type) {
    case 'add_card': {
      const col = (b.columns ?? []).find((c) => c.name === op.column)
      if (!col) throw new OpError('NOT_FOUND', `column ${op.column}`)
      const card = { title: op.title }
      if (op.prepend) {
        col.cards = [card, ...(col.cards ?? [])]
      } else {
        col.cards = [...(col.cards ?? []), card]
      }
      return b
    }
    default:
      throw new OpError('INTERNAL', `unimplemented op: ${(op as MutationOp).type}`)
  }
}
