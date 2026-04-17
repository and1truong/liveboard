import { useMutation, useQueryClient, type UseMutationResult } from '@tanstack/react-query'
import { ProtocolError } from '@shared/protocol.js'
import { useClient } from '../queries.js'
import { errorToast } from '../toast.js'

function code(err: unknown): string {
  return err instanceof ProtocolError ? err.code : 'INTERNAL'
}

function invalidateAll(qc: ReturnType<typeof useQueryClient>): void {
  void qc.invalidateQueries({ queryKey: ['folders'] })
  void qc.invalidateQueries({ queryKey: ['boards'] })
}

export function useCreateFolder(): UseMutationResult<void, Error, string> {
  const client = useClient()
  const qc = useQueryClient()
  return useMutation<void, Error, string>({
    mutationFn: (name) => client.createFolder(name),
    onSuccess: () => invalidateAll(qc),
    onError: (err) => errorToast(code(err)),
  })
}

export interface RenameFolderVars {
  oldName: string
  newName: string
}

export function useRenameFolder(): UseMutationResult<void, Error, RenameFolderVars> {
  const client = useClient()
  const qc = useQueryClient()
  return useMutation<void, Error, RenameFolderVars>({
    mutationFn: ({ oldName, newName }) => client.renameFolder(oldName, newName),
    onSuccess: () => invalidateAll(qc),
    onError: (err) => errorToast(code(err)),
  })
}

export function useDeleteFolder(): UseMutationResult<void, Error, string> {
  const client = useClient()
  const qc = useQueryClient()
  return useMutation<void, Error, string>({
    mutationFn: (name) => client.deleteFolder(name),
    onSuccess: () => invalidateAll(qc),
    onError: (err) => errorToast(code(err)),
  })
}
