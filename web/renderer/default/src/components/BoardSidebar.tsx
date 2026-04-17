import { useState, useRef, useEffect } from 'react'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { useBoardList, useWorkspaceInfo } from '../queries.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { EmptyState } from './EmptyState.js'
import { BoardRow } from './BoardRow.js'
import { AddBoardButton } from './AddBoardButton.js'
import { ThemePicker } from './ThemePicker.js'
import { useGlobalSettingsContext } from '../contexts/GlobalSettingsContext.js'

export function BoardSidebar({ collapsed = false }: { collapsed?: boolean }): JSX.Element {
  const boards = useBoardList()
  const ws = useWorkspaceInfo()
  const { active, setActive } = useActiveBoard()
  const activeBoard = boards.data?.find((b) => b.id === active)
  const [menuOpen, setMenuOpen] = useState(false)
  const { openSettings: openGlobalSettings } = useGlobalSettingsContext()
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

  const count = boards.data?.length ?? 0

  return (
    <aside
      className={`lb-sidebar${collapsed ? ' lb-sidebar--collapsed' : ''}`}
      role="complementary"
      aria-label="Boards sidebar"
      aria-hidden={collapsed || undefined}
    >
      <header className="lb-sidebar__header">
        <span className="lb-sidebar__title">Boards</span>
        {!boards.isLoading && !boards.error && count > 0 && (
          <span className="lb-sidebar__count" aria-label={`${count} boards`}>
            {count}
          </span>
        )}
      </header>
      <div className="lb-sidebar__body">
        {boards.isLoading ? (
          <EmptyState title="Loading…" />
        ) : boards.error ? (
          <EmptyState title="Failed to load" detail={String(boards.error)} />
        ) : !boards.data || boards.data.length === 0 ? (
          <EmptyState title="No boards yet" />
        ) : (
          <ul className="lb-sidebar__list">
            {boards.data.map((b) => (
              <BoardRow key={b.id} board={b} />
            ))}
          </ul>
        )}
        <hr className="lb-sidebar__sep" />
        <AddBoardButton />
      </div>
      <div className="lb-sidebar__mobile-dropdown">
        <DropdownMenu.Root>
          <DropdownMenu.Trigger className="lb-mobile-trigger" aria-label="Switch board">
            <span>{activeBoard?.name ?? 'Boards'}</span>
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden>
              <path d="M3 4.5L6 7.5L9 4.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          </DropdownMenu.Trigger>
          <DropdownMenu.Portal>
            <DropdownMenu.Content className="lb-popover" sideOffset={4} align="end">
              {boards.data?.map((b) => (
                <DropdownMenu.Item
                  key={b.id}
                  onSelect={() => setActive(b.id)}
                  className={`lb-popover__item${b.id === active ? ' lb-popover__item--active' : ''}`}
                >
                  <span className="lb-popover__icon" aria-hidden>{b.icon || '\u25A6'}</span>
                  <span>{b.name}</span>
                </DropdownMenu.Item>
              ))}
            </DropdownMenu.Content>
          </DropdownMenu.Portal>
        </DropdownMenu.Root>
      </div>
      <div ref={menuRef} className="lb-sidebar__footer">
        {menuOpen && (
          <div className="lb-popover lb-popover--bottom-up" role="menu">
            <button
              type="button"
              onClick={() => { setMenuOpen(false); openGlobalSettings() }}
              className="lb-popover__item"
              role="menuitem"
            >
              <span className="lb-popover__icon" aria-hidden>&#9881;</span>
              <span>Settings</span>
            </button>
            <hr className="lb-popover__sep" />
            <a
              href="https://and1truong.github.io/liveboard/"
              target="_blank"
              rel="noopener noreferrer"
              className="lb-popover__item"
              role="menuitem"
            >
              <span className="lb-popover__icon" aria-hidden>&#9432;</span>
              <span>About {ws.data?.name ?? 'LiveBoard'}</span>
            </a>
          </div>
        )}
        <a href="/" className="lb-sidebar__brand">
          <svg width="20" height="20" viewBox="0 0 128 128" xmlns="http://www.w3.org/2000/svg" aria-hidden>
            <rect x="10" y="16" width="108" height="96" rx="14" fill="none" stroke="currentColor" strokeWidth="3" />
            <line x1="46" y1="28" x2="46" y2="100" stroke="currentColor" strokeWidth="0.8" opacity="0.25" />
            <line x1="82" y1="28" x2="82" y2="100" stroke="currentColor" strokeWidth="0.8" opacity="0.25" />
            <rect x="18" y="32" width="22" height="8" rx="2" fill="currentColor" opacity="0.55" />
            <rect x="18" y="44" width="22" height="8" rx="2" fill="currentColor" opacity="0.55" />
            <rect x="18" y="56" width="22" height="8" rx="2" fill="currentColor" opacity="0.55" />
            <rect x="53" y="32" width="22" height="8" rx="2" fill="currentColor" opacity="0.75" />
            <rect x="53" y="44" width="22" height="8" rx="2" fill="currentColor" opacity="0.75" />
            <rect x="88" y="32" width="22" height="8" rx="2" fill="currentColor" />
          </svg>
          <span className="lb-sidebar__brand-name">{ws.data?.name ?? '—'}</span>
        </a>
        <ThemePicker />
        <button
          type="button"
          onClick={() => setMenuOpen((p) => !p)}
          title="Menu"
          aria-label="Sidebar menu"
          aria-expanded={menuOpen}
          data-state={menuOpen ? 'open' : 'closed'}
          className="lb-iconbtn"
        >
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden>
            <path d="M3 7.5L6 10L9 7.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            <path d="M3 4.5L6 2L9 4.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </button>
      </div>
    </aside>
  )
}
