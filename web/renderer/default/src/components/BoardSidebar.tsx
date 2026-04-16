import { useState, useRef, useEffect } from 'react'
import { useBoardList, useWorkspaceInfo } from '../queries.js'
import { EmptyState } from './EmptyState.js'
import { BoardRow } from './BoardRow.js'
import { AddBoardButton } from './AddBoardButton.js'
import { ThemePicker } from './ThemePicker.js'

export function BoardSidebar(): JSX.Element {
  const boards = useBoardList()
  const ws = useWorkspaceInfo()
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!menuOpen) return
    const handler = (e: MouseEvent): void => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [menuOpen])

  return (
    <aside className="flex h-full w-60 shrink-0 flex-col border-r border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
      <header className="border-b border-slate-200 p-3 dark:border-slate-800">
        <p className="text-xs font-bold uppercase tracking-wide text-slate-500 dark:text-slate-400">Boards</p>
      </header>
      <div className="flex-1 overflow-y-auto p-2">
        {boards.isLoading ? (
          <EmptyState title="Loading…" />
        ) : boards.error ? (
          <EmptyState title="Failed to load" detail={String(boards.error)} />
        ) : !boards.data || boards.data.length === 0 ? (
          <EmptyState title="No boards yet" />
        ) : (
          <ul className="flex flex-col gap-1">
            {boards.data.map((b) => (
              <BoardRow key={b.id} board={b} />
            ))}
          </ul>
        )}
        <AddBoardButton />
      </div>
      <div ref={menuRef} className="relative shrink-0 border-t border-slate-200 dark:border-slate-800">
        {menuOpen && (
          <div className="absolute bottom-full left-2 right-2 mb-1 rounded-lg border border-slate-200 bg-white py-1 shadow-[0_-4px_16px_rgba(0,0,0,0.25)] dark:border-slate-700 dark:bg-slate-800">
            <a
              href="/reminders"
              className="flex items-center gap-2.5 px-3.5 py-2 text-sm text-slate-600 no-underline hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-700"
            >
              <span className="w-[18px] shrink-0 text-center">&#128276;</span>
              <span>Reminders</span>
            </a>
            <a
              href="/settings"
              className="flex items-center gap-2.5 px-3.5 py-2 text-sm text-slate-600 no-underline hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-700"
            >
              <span className="w-[18px] shrink-0 text-center">&#9881;</span>
              <span>Settings</span>
            </a>
            <div className="my-1 h-px bg-slate-200 dark:bg-slate-700" />
            <a
              href="https://and1truong.github.io/liveboard/"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2.5 px-3.5 py-2 text-sm text-slate-600 no-underline hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-700"
            >
              <span className="w-[18px] shrink-0 text-center">&#9432;</span>
              <span>About {ws.data?.name ?? 'LiveBoard'}</span>
            </a>
          </div>
        )}
        <div className="flex items-center gap-1 px-4 py-2">
          <a href="/" className="flex min-w-0 flex-1 items-center gap-2 no-underline">
            <svg className="shrink-0 text-[color:var(--accent,#3b82f6)]" width="24" height="24" viewBox="0 0 128 128" xmlns="http://www.w3.org/2000/svg">
              <rect x="10" y="16" width="108" height="96" rx="14" fill="none" stroke="currentColor" strokeWidth="3" />
              <line x1="46" y1="28" x2="46" y2="100" stroke="currentColor" strokeWidth="0.8" opacity="0.25" />
              <line x1="82" y1="28" x2="82" y2="100" stroke="currentColor" strokeWidth="0.8" opacity="0.25" />
              <rect x="18" y="32" width="22" height="8" rx="2" fill="currentColor" opacity="0.5" />
              <rect x="18" y="44" width="22" height="8" rx="2" fill="currentColor" opacity="0.5" />
              <rect x="18" y="56" width="22" height="8" rx="2" fill="currentColor" opacity="0.5" />
              <rect x="53" y="32" width="22" height="8" rx="2" fill="currentColor" opacity="0.7" />
              <rect x="53" y="44" width="22" height="8" rx="2" fill="currentColor" opacity="0.7" />
              <rect x="88" y="32" width="22" height="8" rx="2" fill="currentColor" />
            </svg>
            <span className="truncate text-base font-semibold text-slate-800 dark:text-slate-100">
              {ws.data?.name ?? '—'}
            </span>
          </a>
          <div className="flex shrink-0 items-center gap-0.5">
            <ThemePicker />
            <button
              type="button"
              onClick={() => setMenuOpen((p) => !p)}
              title="Menu"
              className="flex h-7 w-7 items-center justify-center rounded text-slate-500 transition-colors hover:bg-slate-200 dark:text-slate-400 dark:hover:bg-slate-700"
            >
              <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
                <path d="M3 7.5L6 10L9 7.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
                <path d="M3 4.5L6 2L9 4.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    </aside>
  )
}
