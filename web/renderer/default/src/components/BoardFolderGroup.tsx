import { useRef, useState, useEffect, type DragEvent, type ReactNode } from 'react'
import type { BoardSummary } from '@shared/adapter.js'
import { BoardRow, BOARD_DRAG_MIME } from './BoardRow.js'
import { useRenameFolder, useDeleteFolder } from '../mutations/useFolderCrud.js'
import { errorToast } from '../toast.js'

export interface BoardFolderGroupProps {
  folder: string
  boards: BoardSummary[]
  collapsed: boolean
  onToggle(): void
  onBoardDrop(boardId: string): void
}

// BoardFolderGroup renders a collapsible section of boards sharing the same
// parent folder.
export function BoardFolderGroup({
  folder,
  boards,
  collapsed,
  onToggle,
  onBoardDrop,
}: BoardFolderGroupProps): JSX.Element {
  const [menuOpen, setMenuOpen] = useState(false)
  const [renaming, setRenaming] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const renameFolder = useRenameFolder()
  const deleteFolder = useDeleteFolder()

  useEffect(() => {
    if (!menuOpen) return
    const handler = (e: MouseEvent): void => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setMenuOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [menuOpen])

  useEffect(() => {
    if (renaming) inputRef.current?.focus()
  }, [renaming])

  const commitRename = (): void => {
    const next = (inputRef.current?.value ?? '').trim()
    setRenaming(false)
    if (!next || next === folder) return
    renameFolder.mutate({ oldName: folder, newName: next })
  }

  const onDelete = (): void => {
    setMenuOpen(false)
    if (boards.length > 0) {
      errorToast('INVALID')
      return
    }
    deleteFolder.mutate(folder)
  }

  const hasBoardPayload = (e: DragEvent): boolean =>
    e.dataTransfer.types.includes(BOARD_DRAG_MIME)

  const onDragOver = (e: DragEvent<HTMLLIElement>): void => {
    if (!hasBoardPayload(e)) return
    e.preventDefault()
    e.stopPropagation()
    e.dataTransfer.dropEffect = 'move'
    if (!dragOver) setDragOver(true)
  }

  const onDragLeave = (e: DragEvent<HTMLLIElement>): void => {
    if (e.currentTarget.contains(e.relatedTarget as Node | null)) return
    setDragOver(false)
  }

  const onDrop = (e: DragEvent<HTMLLIElement>): void => {
    const boardId = e.dataTransfer.getData(BOARD_DRAG_MIME)
    setDragOver(false)
    if (!boardId) return
    e.preventDefault()
    e.stopPropagation()
    onBoardDrop(boardId)
  }

  let header: ReactNode
  if (renaming) {
    header = (
      <div className="lb-folder__header">
        <input
          ref={inputRef}
          aria-label="rename folder"
          defaultValue={folder}
          onBlur={commitRename}
          onKeyDown={(e) => {
            if (e.key === 'Enter') { e.preventDefault(); commitRename() }
            else if (e.key === 'Escape') { e.preventDefault(); setRenaming(false) }
          }}
          className="lb-folder__input"
        />
      </div>
    )
  } else {
    header = (
      <div className="lb-folder__header">
        <button
          type="button"
          className="lb-folder__toggle"
          aria-expanded={!collapsed}
          aria-controls={`lb-folder-${folder}`}
          onClick={onToggle}
        >
          <span className={`lb-folder__caret${collapsed ? ' lb-folder__caret--collapsed' : ''}`} aria-hidden>
            <svg width="10" height="10" viewBox="0 0 10 10">
              <path d="M3 2 L7 5 L3 8" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          </span>
          <span className="lb-folder__name">{folder}</span>
          <span className="lb-folder__count">{boards.length}</span>
        </button>
        <div ref={menuRef} className="lb-folder__menu-wrap">
          <button
            type="button"
            className="lb-folder__menu-btn"
            title="Folder actions"
            aria-label="Folder actions"
            aria-expanded={menuOpen}
            onClick={(e) => { e.stopPropagation(); setMenuOpen((p) => !p) }}
          >
            <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden>
              <circle cx="2" cy="6" r="1.25" fill="currentColor" />
              <circle cx="6" cy="6" r="1.25" fill="currentColor" />
              <circle cx="10" cy="6" r="1.25" fill="currentColor" />
            </svg>
          </button>
          {menuOpen && (
            <div className="lb-popover" role="menu">
              <button
                type="button"
                className="lb-popover__item"
                role="menuitem"
                onClick={() => { setMenuOpen(false); setRenaming(true) }}
              >
                <span>Rename folder</span>
              </button>
              <button
                type="button"
                className="lb-popover__item"
                role="menuitem"
                onClick={onDelete}
                disabled={boards.length > 0}
                title={boards.length > 0 ? 'Folder must be empty' : 'Delete folder'}
              >
                <span>Delete folder</span>
              </button>
            </div>
          )}
        </div>
      </div>
    )
  }

  return (
    <li
      className={`lb-folder${dragOver ? ' lb-folder--drop-target' : ''}`}
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
    >
      {header}
      {!collapsed && !renaming && (
        <ul
          id={`lb-folder-${folder}`}
          className="lb-folder__children"
        >
          {boards.map((b) => (
            <BoardRow key={b.id} board={b} />
          ))}
        </ul>
      )}
    </li>
  )
}
