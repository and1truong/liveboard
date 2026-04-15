import { useMutation, useQueryClient, type UseMutationResult } from '@tanstack/react-query'
import type { Board, MutationOp } from '@shared/types.js'
import { applyOp } from '@shared/boardOps.js'
import { ProtocolError } from '@shared/protocol.js'
import { useClient } from '../queries.js'
import { errorToast } from '../toast.js'

interface Ctx {
  prev?: Board
}

export function useBoardMutation(
  boardId: string,
): UseMutationResult<Board, Error, MutationOp, Ctx> {
  const client = useClient()
  const qc = useQueryClient()

  return useMutation<Board, Error, MutationOp, Ctx>({
    mutationFn: (op) => {
      const cached = qc.getQueryData<Board>(['board', boardId])
      return client.mutateBoard(boardId, cached?.version ?? -1, op)
    },
    onMutate: async (op) => {
      await qc.cancelQueries({ queryKey: ['board', boardId] })
      const prev = qc.getQueryData<Board>(['board', boardId])
      if (prev) {
        try {
          qc.setQueryData(['board', boardId], applyOp(prev, op))
        } catch {
          // Optimistic apply failed (e.g. stale indices). Keep prev; let server reject.
        }
      }
      return { prev }
    },
    onSuccess: (board) => {
      qc.setQueryData(['board', boardId], board)
    },
    onError: (err, _op, ctx) => {
      if (ctx?.prev) qc.setQueryData(['board', boardId], ctx.prev)
      const code = err instanceof ProtocolError ? err.code : 'INTERNAL'
      if (code === 'VERSION_CONFLICT') {
        void qc.invalidateQueries({ queryKey: ['board', boardId] })
      }
      errorToast(code)
    },
  })
}
