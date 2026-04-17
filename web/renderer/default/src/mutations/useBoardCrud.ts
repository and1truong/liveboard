import { useMutation, useQueryClient, type UseMutationResult } from '@tanstack/react-query'
import type { BoardSummary } from '@shared/adapter.js'
import { ProtocolError } from '@shared/protocol.js'
import { useClient } from '../queries.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { errorToast } from '../toast.js'

function code(err: unknown): string {
  return err instanceof ProtocolError ? err.code : 'INTERNAL'
}

export interface CreateBoardVars {
  name: string
  folder?: string
}

export function useCreateBoard(): UseMutationResult<BoardSummary, Error, CreateBoardVars | string> {
  const client = useClient()
  const qc = useQueryClient()
  const { setActive } = useActiveBoard()
  return useMutation<BoardSummary, Error, CreateBoardVars | string>({
    mutationFn: (vars) => {
      if (typeof vars === 'string') return client.createBoard(vars)
      return client.createBoard(vars.name, vars.folder)
    },
    onSuccess: (summary) => {
      void qc.invalidateQueries({ queryKey: ['boards'] })
      void qc.invalidateQueries({ queryKey: ['folders'] })
      setActive(summary.id)
    },
    onError: (err) => errorToast(code(err)),
  })
}

export interface RenameVars {
  boardId: string
  newName: string
  folder?: string
}

export function useRenameBoard(): UseMutationResult<BoardSummary, Error, RenameVars> {
  const client = useClient()
  const qc = useQueryClient()
  const { active, setActive } = useActiveBoard()
  return useMutation<BoardSummary, Error, RenameVars>({
    mutationFn: ({ boardId, newName, folder }) => client.renameBoard(boardId, newName, folder),
    onSuccess: (summary, { boardId }) => {
      void qc.invalidateQueries({ queryKey: ['boards'] })
      void qc.invalidateQueries({ queryKey: ['folders'] })
      if (active === boardId) setActive(summary.id)
    },
    onError: (err) => errorToast(code(err)),
  })
}

export interface MoveVars {
  boardId: string
  folder: string
}

// useMoveBoard is a thin wrapper that keeps the current display name but
// moves the board to a different folder ('' = root).
export function useMoveBoard(): UseMutationResult<BoardSummary, Error, MoveVars> {
  const client = useClient()
  const qc = useQueryClient()
  const { active, setActive } = useActiveBoard()
  return useMutation<BoardSummary, Error, MoveVars>({
    mutationFn: async ({ boardId, folder }) => {
      // Preserve the board's existing display name.
      const board = await client.getBoard(boardId)
      const name = board.name ?? boardId.split('/').pop() ?? boardId
      return client.renameBoard(boardId, name, folder)
    },
    onSuccess: (summary, { boardId }) => {
      void qc.invalidateQueries({ queryKey: ['boards'] })
      void qc.invalidateQueries({ queryKey: ['folders'] })
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
      void qc.invalidateQueries({ queryKey: ['folders'] })
      if (active === boardId) setActive(ctx?.fallbackActive ?? null)
    },
    onError: (err) => errorToast(code(err)),
  })
}
