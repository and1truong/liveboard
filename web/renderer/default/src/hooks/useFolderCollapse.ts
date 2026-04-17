import { useCallback, useEffect, useState } from 'react'

const STORAGE_KEY = 'lb:folder-collapse'

function readState(): Record<string, boolean> {
  if (typeof localStorage === 'undefined') return {}
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as unknown
    if (parsed && typeof parsed === 'object') return parsed as Record<string, boolean>
  } catch {
    // ignore
  }
  return {}
}

function writeState(state: Record<string, boolean>): void {
  if (typeof localStorage === 'undefined') return
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
  } catch {
    // ignore — private mode / quota
  }
}

export interface FolderCollapseAPI {
  isCollapsed(folder: string): boolean
  toggle(folder: string): void
}

// useFolderCollapse persists per-folder expand/collapse state in localStorage.
// Default (missing key) is expanded, matching Apple Reminders behavior.
export function useFolderCollapse(): FolderCollapseAPI {
  const [state, setState] = useState<Record<string, boolean>>(() => readState())

  useEffect(() => {
    writeState(state)
  }, [state])

  const isCollapsed = useCallback((folder: string) => state[folder] === true, [state])
  const toggle = useCallback((folder: string) => {
    setState((prev) => ({ ...prev, [folder]: !prev[folder] }))
  }, [])

  return { isCollapsed, toggle }
}
