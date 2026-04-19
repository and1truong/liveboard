import { useRef, useState, useEffect, type DragEvent, type ReactNode } from 'react'
import * as ContextMenu from '@radix-ui/react-context-menu'
import type { BoardSummary } from '@shared/adapter.js'
import { BoardRow, BOARD_DRAG_MIME } from './BoardRow.js'
import { useRenameFolder, useDeleteFolder } from '../mutations/useFolderCrud.js'
import { errorToast } from '../toast.js'

const contentCls = 'z-50 min-w-44 rounded-md bg-white p-1 shadow-lg ring-1 ring-slate-200 dark:bg-slate-800 dark:ring-slate-700 dark:text-slate-100'
const itemCls = 'cursor-pointer rounded px-2 py-1 text-sm outline-none hover:bg-slate-100 dark:hover:bg-slate-700 data-[disabled]:cursor-not-allowed data-[disabled]:text-slate-300 dark:data-[disabled]:text-slate-600'

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
  const [renaming, setRenaming] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const renameFolder = useRenameFolder()
  const deleteFolder = useDeleteFolder()

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
      </div>
    )
  }

  return (
    <ContextMenu.Root>
      <ContextMenu.Trigger asChild>
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
      </ContextMenu.Trigger>
      <ContextMenu.Portal>
        <ContextMenu.Content className={contentCls}>
          <ContextMenu.Item className={itemCls} onSelect={() => setRenaming(true)}>
            Rename folder
          </ContextMenu.Item>
          <ContextMenu.Item className={itemCls} onSelect={onDelete} disabled={boards.length > 0}>
            Delete folder
          </ContextMenu.Item>
        </ContextMenu.Content>
      </ContextMenu.Portal>
    </ContextMenu.Root>
  )
}
