import { useState, useRef, useEffect, useMemo, type DragEvent } from 'react'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import type { BoardSummary } from '@shared/adapter.js'
import { useBoardList, useFolderList, useWorkspaceInfo } from '../queries.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { EmptyState } from './EmptyState.js'
import { BoardRow, BOARD_DRAG_MIME } from './BoardRow.js'
import { BoardFolderGroup } from './BoardFolderGroup.js'
import { AddBoardButton } from './AddBoardButton.js'
import { ThemePicker } from './ThemePicker.js'
import { useGlobalSettingsContext } from '../contexts/GlobalSettingsContext.js'
import { useFolderCollapse } from '../hooks/useFolderCollapse.js'
import { useCreateFolder } from '../mutations/useFolderCrud.js'
import { useMoveBoard } from '../mutations/useBoardCrud.js'
import { useHasCapability } from '../queries/useCapabilities.js'

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
function groupBoards(
  boards: BoardSummary[],
  knownFolders: string[],
  foldersEnabled: boolean,
): Grouped {
  const pinned: BoardSummary[] = []
  const rootBoards: BoardSummary[] = []
  const byFolder = new Map<string, BoardSummary[]>()
  for (const b of boards) {
    if (b.pinned) {
      pinned.push(b)
      continue
    }
    const folder = foldersEnabled ? (b.folder ?? '') : ''
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
  if (foldersEnabled) {
    // Ensure every known folder (including empty ones) shows up.
    for (const f of knownFolders) {
      if (!byFolder.has(f)) byFolder.set(f, [])
    }
  }
  const folderOrder = [...byFolder.keys()].sort((a, b) => a.localeCompare(b))
  return { pinned, rootBoards, folderOrder, byFolder }
}

export function BoardSidebar({ collapsed = false }: { collapsed?: boolean }): JSX.Element {
  const foldersEnabled = useHasCapability('folders')
  const boards = useBoardList()
  const folders = useFolderList({ enabled: foldersEnabled })
  const ws = useWorkspaceInfo()
  const { active, setActive } = useActiveBoard()
  const activeBoard = boards.data?.find((b) => b.id === active)
  const [menuOpen, setMenuOpen] = useState(false)
  const { openSettings: openGlobalSettings } = useGlobalSettingsContext()
  const menuRef = useRef<HTMLDivElement>(null)
  const { isCollapsed, toggle } = useFolderCollapse()
  const createFolder = useCreateFolder()
  const moveBoard = useMoveBoard()
  const [newBoardOpen, setNewBoardOpen] = useState(false)
  const [newFolderOpen, setNewFolderOpen] = useState(false)
  const [rootDragOver, setRootDragOver] = useState(false)
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

  useEffect(() => {
    const handler = (e: KeyboardEvent): void => {
      const mod = e.metaKey || e.ctrlKey
      if (!mod) return
      if (e.key !== 'n' && e.key !== 'N') return
      const target = e.target as HTMLElement | null
      // Don't steal focus when the user is typing.
      if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)) return
      if (e.shiftKey) {
        if (!foldersEnabled) return
        e.preventDefault()
        setNewBoardOpen(false)
        setNewFolderOpen(true)
      } else {
        e.preventDefault()
        setNewFolderOpen(false)
        setNewBoardOpen(true)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [foldersEnabled])

  const allFolders = foldersEnabled ? (folders.data ?? []) : []
  const groups = useMemo(
    () => groupBoards(boards.data ?? [], allFolders, foldersEnabled),
    [boards.data, allFolders, foldersEnabled],
  )
  const count = boards.data?.length ?? 0

  const commitNewFolder = (): void => {
    const name = (newFolderInputRef.current?.value ?? '').trim()
    setNewFolderOpen(false)
    if (name) createFolder.mutate(name)
  }

  const moveToFolder = (boardId: string, folder: string): void => {
    const current = boards.data?.find((b) => b.id === boardId)
    if (current && (current.folder ?? '') === folder) return
    moveBoard.mutate({ boardId, folder })
  }

  const hasBoardPayload = (e: DragEvent): boolean =>
    e.dataTransfer.types.includes(BOARD_DRAG_MIME)

  const onRootDragOver = (e: DragEvent<HTMLUListElement>): void => {
    if (!foldersEnabled || !hasBoardPayload(e)) return
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    if (!rootDragOver) setRootDragOver(true)
  }

  const onRootDragLeave = (e: DragEvent<HTMLUListElement>): void => {
    if (e.currentTarget.contains(e.relatedTarget as Node | null)) return
    setRootDragOver(false)
  }

  const onRootDrop = (e: DragEvent<HTMLUListElement>): void => {
    const boardId = e.dataTransfer.getData(BOARD_DRAG_MIME)
    setRootDragOver(false)
    if (!boardId) return
    e.preventDefault()
    moveToFolder(boardId, '')
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
      <div className="lb-sidebar__toolbar">
        <button
          type="button"
          className="lb-sidebar__pill"
          onClick={() => { setNewFolderOpen(false); setNewBoardOpen(true) }}
        >
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden>
            <rect x="2" y="3" width="12" height="10" rx="1.5" stroke="currentColor" strokeWidth="1.3" />
            <line x1="6.2" y1="3.5" x2="6.2" y2="12.5" stroke="currentColor" strokeWidth="1.1" />
            <line x1="9.8" y1="3.5" x2="9.8" y2="12.5" stroke="currentColor" strokeWidth="1.1" />
          </svg>
          <span className="lb-sidebar__pill-label">Board</span>
          <kbd className="lb-sidebar__kbd" aria-hidden>&#8984;N</kbd>
        </button>
        {foldersEnabled && (
          <button
            type="button"
            className="lb-sidebar__pill"
            onClick={() => { setNewBoardOpen(false); setNewFolderOpen(true) }}
          >
            <svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden>
              <path
                d="M2 5.2c0-.66.54-1.2 1.2-1.2h3.1l1.3 1.4h5.2c.66 0 1.2.54 1.2 1.2v5.4c0 .66-.54 1.2-1.2 1.2H3.2c-.66 0-1.2-.54-1.2-1.2V5.2Z"
                stroke="currentColor"
                strokeWidth="1.3"
                strokeLinejoin="round"
              />
            </svg>
            <span className="lb-sidebar__pill-label">Folder</span>
            <kbd className="lb-sidebar__kbd" aria-hidden>&#8984;&#8679;N</kbd>
          </button>
        )}
      </div>
      <div className="lb-sidebar__body">
        {boards.isLoading ? (
          <EmptyState title="Loading…" />
        ) : boards.error ? (
          <EmptyState title="Failed to load" detail={String(boards.error)} />
        ) : !boards.data || boards.data.length === 0 ? (
          <EmptyState title="No boards yet" />
        ) : (
          <ul
            className={`lb-sidebar__list${rootDragOver ? ' lb-sidebar__list--drop-target' : ''}`}
            onDragOver={onRootDragOver}
            onDragLeave={onRootDragLeave}
            onDrop={onRootDrop}
          >
            {groups.pinned.map((b) => (
              <BoardRow key={b.id} board={b} />
            ))}
            {groups.rootBoards.map((b) => (
              <BoardRow key={b.id} board={b} />
            ))}
            {foldersEnabled && groups.folderOrder.map((f) => (
              <BoardFolderGroup
                key={f}
                folder={f}
                boards={groups.byFolder.get(f) ?? []}
                collapsed={isCollapsed(f)}
                onToggle={() => toggle(f)}
                onBoardDrop={(boardId) => moveToFolder(boardId, f)}
              />
            ))}
          </ul>
        )}
        {newBoardOpen && (
          <AddBoardButton
            folders={foldersEnabled ? allFolders : undefined}
            onClose={() => setNewBoardOpen(false)}
          />
        )}
        {foldersEnabled && newFolderOpen && (
          <div className="lb-row lb-row--add-input">
            <svg
              className="lb-row__input-icon"
              width="16"
              height="16"
              viewBox="0 0 16 16"
              fill="none"
              aria-hidden
            >
              <path
                d="M2 5.2c0-.66.54-1.2 1.2-1.2h3.1l1.3 1.4h5.2c.66 0 1.2.54 1.2 1.2v5.4c0 .66-.54 1.2-1.2 1.2H3.2c-.66 0-1.2-.54-1.2-1.2V5.2Z"
                stroke="currentColor"
                strokeWidth="1.3"
                strokeLinejoin="round"
              />
            </svg>
            <input
              ref={newFolderInputRef}
              aria-label="new folder name"
              placeholder="Folder name"
              className="lb-row__input"
              onBlur={commitNewFolder}
              onKeyDown={(e) => {
                if (e.key === 'Enter') { e.preventDefault(); commitNewFolder() }
                else if (e.key === 'Escape') { e.preventDefault(); setNewFolderOpen(false) }
              }}
            />
          </div>
        )}
      </div>
      <div className="lb-sidebar__archived" role="button" tabIndex={0} aria-label="Archived boards">
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden>
          <rect x="2" y="3.5" width="12" height="2.5" rx="0.6" stroke="currentColor" strokeWidth="1.2" />
          <path d="M3 6.5v5.2c0 .55.45 1 1 1h8c.55 0 1-.45 1-1V6.5" stroke="currentColor" strokeWidth="1.2" strokeLinejoin="round" />
          <line x1="6.5" y1="9" x2="9.5" y2="9" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
        </svg>
        <span className="lb-sidebar__archived-label">Archived</span>
        <span className="lb-sidebar__archived-count" aria-label="3 archived boards">3</span>
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
