import { useMutation, useQueryClient, type UseMutationResult } from '@tanstack/react-query'
import type { BoardSummary } from '@shared/adapter.js'
import { ProtocolError } from '@shared/protocol.js'
import { useClient } from '../queries.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { errorToast } from '../toast.js'

function code(err: unknown): string {
  return err instanceof ProtocolError ? err.code : 'INTERNAL'
}

export function useCreateBoard(): UseMutationResult<BoardSummary, Error, string> {
  const client = useClient()
  const qc = useQueryClient()
  const { setActive } = useActiveBoard()
  return useMutation<BoardSummary, Error, string>({
    mutationFn: (name) => client.createBoard(name),
    onSuccess: (summary) => {
      void qc.invalidateQueries({ queryKey: ['boards'] })
      setActive(summary.id)
    },
    onError: (err) => errorToast(code(err)),
  })
}

interface RenameVars {
  boardId: string
  newName: string
}

export function useRenameBoard(): UseMutationResult<BoardSummary, Error, RenameVars> {
  const client = useClient()
  const qc = useQueryClient()
  const { active, setActive } = useActiveBoard()
  return useMutation<BoardSummary, Error, RenameVars>({
    mutationFn: ({ boardId, newName }) => client.renameBoard(boardId, newName),
    onSuccess: (summary, { boardId }) => {
      void qc.invalidateQueries({ queryKey: ['boards'] })
      if (active === boardId) setActive(summary.id)
    },
    onError: (err) => errorToast(code(err)),
  })
}

interface DeleteCtx {
  fallbackActive: string | null
}

export function useTogglePin(): UseMutationResult<void, Error, string> {
  const client = useClient()
  const qc = useQueryClient()
  return useMutation<void, Error, string>({
    mutationFn: (boardId) => client.togglePin(boardId),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['boards'] }),
    onError: (err) => errorToast(code(err)),
  })
}

export function useDeleteBoard(): UseMutationResult<void, Error, string, DeleteCtx> {
  const client = useClient()
  const qc = useQueryClient()
  const { active, setActive } = useActiveBoard()
  return useMutation<void, Error, string, DeleteCtx>({
    mutationFn: (boardId) => client.deleteBoard(boardId),
    onMutate: (boardId) => {
      const list = qc.getQueryData<BoardSummary[]>(['boards']) ?? []
      const fallbackActive = list.find((b) => b.id !== boardId)?.id ?? null
      return { fallbackActive }
    },
    onSuccess: (_void, boardId, ctx) => {
      void qc.invalidateQueries({ queryKey: ['boards'] })
      if (active === boardId) setActive(ctx?.fallbackActive ?? null)
    },
    onError: (err) => errorToast(code(err)),
  })
}
