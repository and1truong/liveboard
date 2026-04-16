import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import type { BoardFilter } from '../utils/cardFilter.js'

interface PersistedFilter {
  tags: string[]
  hideCompleted: boolean
}

export interface BoardFilterCtx {
  filter: BoardFilter
  setQuery: (q: string) => void
  toggleTag: (tag: string) => void
  setTags: (tags: string[]) => void
  setHideCompleted: (next: boolean) => void
  reset: () => void
}

const Ctx = createContext<BoardFilterCtx | null>(null)

const LEGACY_HIDE_COMPLETED = 'lb_hideCompleted'
const keyFor = (boardId: string): string => `lb_filters_${boardId}`

function loadPersisted(boardId: string): PersistedFilter {
  try {
    const raw = localStorage.getItem(keyFor(boardId))
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<PersistedFilter>
      return {
        tags: Array.isArray(parsed.tags) ? parsed.tags.filter((t) => typeof t === 'string') : [],
        hideCompleted: !!parsed.hideCompleted,
      }
    }
  } catch {
    // fall through
  }
  // Lazy migration from legacy global key.
  const legacy = localStorage.getItem(LEGACY_HIDE_COMPLETED)
  return { tags: [], hideCompleted: legacy === 'true' }
}

function persist(boardId: string, value: PersistedFilter): void {
  try {
    localStorage.setItem(keyFor(boardId), JSON.stringify(value))
  } catch {
    // ignore quota / serialization errors
  }
}

export function BoardFilterProvider({
  boardId,
  availableTags,
  children,
}: {
  boardId: string
  availableTags: string[]
  children: ReactNode
}): JSX.Element {
  const [query, _setQuery] = useState('')
  const initial = useRef<PersistedFilter | null>(null)
  if (initial.current === null) initial.current = loadPersisted(boardId)
  const [tags, _setTags] = useState<string[]>(initial.current.tags)
  const [hideCompleted, _setHideCompleted] = useState(initial.current.hideCompleted)

  // When the active board changes, reload persisted state for it and clear search.
  const lastBoardRef = useRef(boardId)
  useEffect(() => {
    if (lastBoardRef.current === boardId) return
    lastBoardRef.current = boardId
    const next = loadPersisted(boardId)
    _setTags(next.tags)
    _setHideCompleted(next.hideCompleted)
    _setQuery('')
  }, [boardId])

  // Prune any selected tags that no longer exist on the current board.
  useEffect(() => {
    if (tags.length === 0) return
    const allowed = new Set(availableTags)
    const pruned = tags.filter((t) => allowed.has(t))
    if (pruned.length !== tags.length) _setTags(pruned)
  }, [availableTags, tags])

  // Persist tags + hideCompleted whenever they change (search is intentionally not persisted).
  useEffect(() => {
    persist(boardId, { tags, hideCompleted })
  }, [boardId, tags, hideCompleted])

  const setQuery = useCallback((q: string) => _setQuery(q), [])
  const setTags = useCallback((next: string[]) => _setTags(next), [])
  const toggleTag = useCallback((tag: string) => {
    _setTags((prev) => (prev.includes(tag) ? prev.filter((t) => t !== tag) : [...prev, tag]))
  }, [])
  const setHideCompleted = useCallback((next: boolean) => _setHideCompleted(next), [])
  const reset = useCallback(() => {
    _setQuery('')
    _setTags([])
    _setHideCompleted(false)
  }, [])

  const filter = useMemo<BoardFilter>(
    () => ({ query, tags, hideCompleted }),
    [query, tags, hideCompleted],
  )

  const value = useMemo<BoardFilterCtx>(
    () => ({ filter, setQuery, toggleTag, setTags, setHideCompleted, reset }),
    [filter, setQuery, toggleTag, setTags, setHideCompleted, reset],
  )

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useBoardFilter(): BoardFilterCtx {
  const v = useContext(Ctx)
  if (!v) throw new Error('useBoardFilter must be used within BoardFilterProvider')
  return v
}
