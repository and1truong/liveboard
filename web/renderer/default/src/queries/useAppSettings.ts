import { useMutation, useQuery, useQueryClient, type UseMutationResult } from '@tanstack/react-query'
import type { AppSettings } from '@shared/types.js'
import { useClient } from '../queries.js'
import { errorToast } from '../toast.js'

function code(err: unknown): string {
  if (err instanceof Error) return err.message
  return String(err)
}

export const APP_SETTINGS_DEFAULTS: AppSettings = {
  site_name: 'LiveBoard',
  theme: 'system',
  color_theme: 'aqua',
  font_family: 'system',
  column_width: 280,
  sidebar_position: 'left',
  default_columns: ['not now', 'maybe?', 'done'],
  show_checkbox: true,
  newline_trigger: 'shift-enter',
  card_position: 'append',
  card_display_mode: 'full',
  keyboard_shortcuts: false,
  week_start: 'sunday',
  pinned_boards: [],
  tags: [],
  tag_colors: {},
}

export function useAppSettings(): AppSettings {
  const client = useClient()
  const q = useQuery({
    queryKey: ['appSettings'],
    queryFn: () => client.getAppSettings(),
    staleTime: 60_000,
  })
  return q.data ?? APP_SETTINGS_DEFAULTS
}

export function useUpdateAppSettings(): UseMutationResult<void, Error, Partial<AppSettings>> {
  const client = useClient()
  const qc = useQueryClient()
  return useMutation<void, Error, Partial<AppSettings>>({
    mutationFn: (patch) => client.putAppSettings(patch),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['appSettings'] })
    },
    onError: (err) => errorToast(code(err)),
  })
}
