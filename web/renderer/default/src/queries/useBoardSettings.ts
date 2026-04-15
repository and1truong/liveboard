import { useMutation, useQuery, useQueryClient, type UseMutationResult } from '@tanstack/react-query'
import type { ResolvedSettings } from '@shared/adapter.js'
import type { BoardSettings } from '@shared/types.js'
import { ProtocolError } from '@shared/protocol.js'
import { useClient } from '../queries.js'
import { errorToast } from '../toast.js'

export const SETTINGS_DEFAULTS: ResolvedSettings = {
  show_checkbox: true,
  card_position: 'bottom',
  expand_columns: false,
  view_mode: 'board',
  card_display_mode: 'normal',
  week_start: 'monday',
}

export function useBoardSettings(boardId: string | null): ResolvedSettings {
  const client = useClient()
  const q = useQuery({
    queryKey: ['settings', boardId],
    queryFn: () => client.getSettings(boardId!),
    enabled: !!boardId,
  })
  return q.data ?? SETTINGS_DEFAULTS
}

function code(err: unknown): string {
  return err instanceof ProtocolError ? err.code : 'INTERNAL'
}

export function useUpdateSettings(
  boardId: string,
): UseMutationResult<void, Error, Partial<BoardSettings>> {
  const client = useClient()
  const qc = useQueryClient()
  return useMutation<void, Error, Partial<BoardSettings>>({
    mutationFn: (patch) => client.putBoardSettings(boardId, patch),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['settings', boardId] })
    },
    onError: (err) => errorToast(code(err)),
  })
}
