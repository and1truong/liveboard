import type { Board, MutationOp } from './types.js'
import { OpError } from './types.js'

// applyOp returns a new board with op applied. Input is not mutated.
// Mirrors internal/api/v1.Apply semantics. Shared parity vectors guard drift.
export function applyOp(board: Board, op: MutationOp): Board {
  const b: Board = structuredClone(board)

  switch (op.type) {
    default:
      throw new OpError('INTERNAL', `unimplemented op: ${(op as MutationOp).type}`)
  }
}
