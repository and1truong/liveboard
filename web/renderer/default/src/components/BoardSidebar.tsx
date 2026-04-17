import { useState, useRef, useEffect, useMemo } from 'react'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import type { BoardSummary } from '@shared/adapter.js'
import { useBoardList, useFolderList, useWorkspaceInfo } from '../queries.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { EmptyState } from './EmptyState.js'
import { BoardRow } from './BoardRow.js'
import { BoardFolderGroup } from './BoardFolderGroup.js'
import { AddBoardButton } from './AddBoardButton.js'
import { ThemePicker } from './ThemePicker.js'
import { useGlobalSettingsContext } from '../contexts/GlobalSettingsContext.js'
import { useFolderCollapse } from '../hooks/useFolderCollapse.js'
import { useCreateFolder } from '../mutations/useFolderCrud.js'

interface Grouped {
  pinned: BoardSummary[]
  rootBoards: BoardSummary[]
  folderOrder: string[]
  byFolder: Map<string, BoardSummary[]>
}

// groupBoards partitions the board list into: pinned (cross-cutting), root
// boards, and per-folder groups. The server already sorts the list by
// pinned-first then folder then name, so we preserve relative order within
// each bucket.
function groupBoards(boards: BoardSummary[], knownFolders: string[]): Grouped {
  const pinned: BoardSummary[] = []
  const rootBoards: BoardSummary[] = []
  const byFolder = new Map<string, BoardSummary[]>()
  for (const b of boards) {
    if (b.pinned) {
      pinned.push(b)
      continue
    }
    const folder = b.folder ?? ''
    if (folder === '') {
      rootBoards.push(b)
      continue
    }
    let list = byFolder.get(folder)
    if (!list) {
      list = []
      byFolder.set(folder, list)
    }
    list.push(b)
  }
  // Ensure every known folder (including empty ones) shows up.
  for (const f of knownFolders) {
    if (!byFolder.has(f)) byFolder.set(f, [])
  }
  const folderOrder = [...byFolder.keys()].sort((a, b) => a.localeCompare(b))
  return { pinned, rootBoards, folderOrder, byFolder }
}

export function BoardSidebar({ collapsed = false }: { collapsed?: boolean }): JSX.Element {
  const boards = useBoardList()
  const folders = useFolderList()
  const ws = useWorkspaceInfo()
  const { active, setActive } = useActiveBoard()
  const activeBoard = boards.data?.find((b) => b.id === active)
  const [menuOpen, setMenuOpen] = useState(false)
  const { openSettings: openGlobalSettings } = useGlobalSettingsContext()
  const menuRef = useRef<HTMLDivElement>(null)
  const { isCollapsed, toggle } = useFolderCollapse()
  const createFolder = useCreateFolder()
  const [newFolderOpen, setNewFolderOpen] = useState(false)
  const newFolderInputRef = useRef<HTMLInputElement>(null)

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

  useEffect(() => {
    if (newFolderOpen) newFolderInputRef.current?.focus()
  }, [newFolderOpen])

  const allFolders = folders.data ?? []
  const groups = useMemo(
    () => groupBoards(boards.data ?? [], allFolders),
    [boards.data, allFolders],
  )
  const count = boards.data?.length ?? 0

  const commitNewFolder = (): void => {
    const name = (newFolderInputRef.current?.value ?? '').trim()
    setNewFolderOpen(false)
    if (name) createFolder.mutate(name)
  }

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
            {groups.pinned.map((b) => (
              <BoardRow key={b.id} board={b} folders={allFolders} />
            ))}
            {groups.rootBoards.map((b) => (
              <BoardRow key={b.id} board={b} folders={allFolders} />
            ))}
            {groups.folderOrder.map((f) => (
              <BoardFolderGroup
                key={f}
                folder={f}
                boards={groups.byFolder.get(f) ?? []}
                collapsed={isCollapsed(f)}
                allFolders={allFolders}
                onToggle={() => toggle(f)}
              />
            ))}
          </ul>
        )}
        <hr className="lb-sidebar__sep" />
        <AddBoardButton folders={allFolders} />
        {newFolderOpen ? (
          <div className="lb-row lb-row--add-input">
            <input
              ref={newFolderInputRef}
              aria-label="new folder name"
              placeholder="Folder name…"
              className="lb-row__input"
              onBlur={commitNewFolder}
              onKeyDown={(e) => {
                if (e.key === 'Enter') { e.preventDefault(); commitNewFolder() }
                else if (e.key === 'Escape') { e.preventDefault(); setNewFolderOpen(false) }
              }}
            />
          </div>
        ) : (
          <button
            type="button"
            onClick={() => setNewFolderOpen(true)}
            className="lb-row lb-row--add"
          >
            <span className="lb-row__plus" aria-hidden>+</span>
            <span className="lb-row__label">+ New folder</span>
          </button>
        )}
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
