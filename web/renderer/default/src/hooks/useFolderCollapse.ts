import { useCallback, useEffect, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import type { AppSettings } from '@shared/types.js'
import { useClient } from '../queries.js'

export interface FolderCollapseAPI {
  isCollapsed(folder: string): boolean
  toggle(folder: string): void
}

// useFolderCollapse persists per-folder expand/collapse state to the backend
// settings. Default (missing key) is expanded.
export function useFolderCollapse(): FolderCollapseAPI {
  const client = useClient()
  const { data: appSettings } = useQuery<AppSettings>({
    queryKey: ['appSettings'],
    queryFn: () => client.getAppSettings(),
    staleTime: 60_000,
  })

  const [state, setState] = useState<Record<string, boolean>>({})
  const initialized = useRef(false)

  useEffect(() => {
    if (initialized.current || !appSettings) return
    initialized.current = true
    setState(appSettings.folder_collapse ?? {})
  }, [appSettings])

  const toggle = useCallback(
    (folder: string) => {
      setState((prev) => {
        const next = { ...prev, [folder]: !prev[folder] }
        void client.putAppSettings({ folder_collapse: next })
        return next
      })
    },
    [client],
  )

  const isCollapsed = useCallback((folder: string) => state[folder] === true, [state])
  return { isCollapsed, toggle }
}
