import { useRef, useState } from 'react'
import type { Attachment } from '@shared/types.js'
import { useClient } from '../queries.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'

type AttachmentListProps = {
  attachments: Attachment[]
  boardId: string
  colIdx: number
  cardIdx: number
  onInsertIntoBody?: (att: Attachment) => void
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function AttachmentList({
  attachments,
  boardId,
  colIdx,
  cardIdx,
  onInsertIntoBody,
}: AttachmentListProps): JSX.Element {
  const client = useClient()
  const mutation = useBoardMutation(boardId)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const dropAreaRef = useRef<HTMLDivElement>(null)

  const [uploading, setUploading] = useState(false)
  const [editingHash, setEditingHash] = useState<string | null>(null)
  const [editingName, setEditingName] = useState('')
  const [dragOver, setDragOver] = useState(false)
  const [draggingIdx, setDraggingIdx] = useState<number | null>(null)

  async function uploadFiles(files: FileList | File[]): Promise<void> {
    const fileArr = Array.from(files)
    if (!fileArr.length) return
    setUploading(true)
    try {
      const items: Attachment[] = []
      for (const f of fileArr) {
        items.push(await client.uploadAttachment(f))
      }
      mutation.mutate({ type: 'add_attachments', col_idx: colIdx, card_idx: cardIdx, items })
    } finally {
      setUploading(false)
    }
  }

  function handleDrop(e: React.DragEvent): void {
    e.preventDefault()
    setDragOver(false)
    if (e.dataTransfer.files?.length) {
      void uploadFiles(e.dataTransfer.files)
    }
  }

  function handleDragOver(e: React.DragEvent): void {
    if (e.dataTransfer.types.includes('Files')) {
      e.preventDefault()
      setDragOver(true)
    }
  }

  function handleDragLeave(e: React.DragEvent): void {
    if (dropAreaRef.current && !dropAreaRef.current.contains(e.relatedTarget as Node)) {
      setDragOver(false)
    }
  }

  function handlePaste(e: React.ClipboardEvent): void {
    const files: File[] = []
    for (const item of Array.from(e.clipboardData.items)) {
      if (item.type.startsWith('image/')) {
        const f = item.getAsFile()
        if (f) files.push(f)
      }
    }
    if (files.length) {
      void uploadFiles(files)
    }
  }

  function handleFileInput(e: React.ChangeEvent<HTMLInputElement>): void {
    if (e.target.files?.length) {
      void uploadFiles(e.target.files)
      e.target.value = ''
    }
  }

  function startRename(att: Attachment): void {
    setEditingHash(att.h)
    setEditingName(att.n)
  }

  function commitRename(att: Attachment): void {
    const trimmed = editingName.trim()
    if (trimmed && trimmed !== att.n) {
      mutation.mutate({
        type: 'rename_attachment',
        col_idx: colIdx,
        card_idx: cardIdx,
        hash: att.h,
        new_name: trimmed,
      })
    }
    setEditingHash(null)
  }

  function handleRemove(att: Attachment): void {
    if (!window.confirm(`Remove "${att.n}"?`)) return
    mutation.mutate({ type: 'remove_attachment', col_idx: colIdx, card_idx: cardIdx, hash: att.h })
  }

  function handleRowDragStart(e: React.DragEvent, idx: number): void {
    e.dataTransfer.setData('text/plain', String(idx))
    e.dataTransfer.effectAllowed = 'move'
    setDraggingIdx(idx)
  }

  function handleRowDragEnd(): void {
    setDraggingIdx(null)
  }

  function handleRowDrop(e: React.DragEvent, toIdx: number): void {
    e.preventDefault()
    e.stopPropagation()
    const fromStr = e.dataTransfer.getData('text/plain')
    const fromIdx = parseInt(fromStr, 10)
    if (isNaN(fromIdx) || fromIdx === toIdx) return
    const reordered = [...attachments]
    const [moved] = reordered.splice(fromIdx, 1)
    reordered.splice(toIdx, 0, moved)
    mutation.mutate({
      type: 'reorder_attachments',
      col_idx: colIdx,
      card_idx: cardIdx,
      hashes_in_order: reordered.map((a) => a.h),
    })
  }

  function handleRowDragOver(e: React.DragEvent): void {
    // Only allow internal row reorder (not file drops)
    if (!e.dataTransfer.types.includes('Files')) {
      e.preventDefault()
      e.dataTransfer.dropEffect = 'move'
    }
  }

  const isEmpty = attachments.length === 0

  return (
    <div
      ref={dropAreaRef}
      onDrop={handleDrop}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onPaste={handlePaste}
      className={`mt-1 rounded border ${dragOver ? 'border-[color:var(--accent-500)] bg-[color:var(--accent-500)]/5' : 'border-[color:var(--color-border)]'} transition-colors`}
    >
      {isEmpty ? (
        <div className="px-3 py-4 text-center text-xs text-slate-400 dark:text-slate-500">
          Drop files here or click + to upload
        </div>
      ) : (
        <ul className="divide-y divide-[color:var(--color-border)]">
          {attachments.map((att, idx) => (
            <li
              key={att.h}
              draggable
              onDragStart={(e) => handleRowDragStart(e, idx)}
              onDragEnd={handleRowDragEnd}
              onDrop={(e) => handleRowDrop(e, idx)}
              onDragOver={handleRowDragOver}
              className={`flex items-center gap-2 px-2 py-1.5 text-xs ${draggingIdx === idx ? 'opacity-40' : ''}`}
            >
              {/* drag handle */}
              <span
                className="cursor-grab select-none text-slate-400 dark:text-slate-500"
                aria-hidden
              >
                ⠿
              </span>

              {/* filename / rename input */}
              <span className="min-w-0 flex-1 truncate">
                {editingHash === att.h ? (
                  <input
                    autoFocus
                    value={editingName}
                    onChange={(e) => setEditingName(e.target.value)}
                    onBlur={() => commitRename(att)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') { e.preventDefault(); commitRename(att) }
                      else if (e.key === 'Escape') { e.preventDefault(); setEditingHash(null) }
                    }}
                    className="w-full rounded border border-[color:var(--accent-500)] bg-transparent px-1 outline-none text-xs"
                  />
                ) : (
                  <button
                    type="button"
                    title="Click to rename"
                    onClick={() => startRename(att)}
                    className="truncate text-left text-slate-700 dark:text-slate-200 hover:underline"
                  >
                    {att.n}
                  </button>
                )}
              </span>

              {/* size */}
              <span className="shrink-0 text-slate-400 dark:text-slate-500">
                {formatSize(att.s)}
              </span>

              {/* actions */}
              <a
                href={client.attachmentURL(att)}
                download={att.n}
                title="Download"
                className="shrink-0 text-slate-400 hover:text-[color:var(--accent-600)] dark:text-slate-500"
                onClick={(e) => e.stopPropagation()}
              >
                ↓
              </a>
              <button
                type="button"
                title="Rename"
                onClick={() => startRename(att)}
                className="shrink-0 text-slate-400 hover:text-[color:var(--accent-600)] dark:text-slate-500"
              >
                ✎
              </button>
              {onInsertIntoBody && (
                <button
                  type="button"
                  title="Insert into body"
                  onClick={() => onInsertIntoBody(att)}
                  className="shrink-0 text-slate-400 hover:text-[color:var(--accent-600)] dark:text-slate-500"
                >
                  ⤵
                </button>
              )}
              <button
                type="button"
                title="Remove"
                onClick={() => handleRemove(att)}
                className="shrink-0 text-slate-400 hover:text-red-500 dark:text-slate-500"
              >
                ✕
              </button>
            </li>
          ))}
        </ul>
      )}

      {/* footer: + button and upload status */}
      <div className={`flex items-center gap-2 border-t border-[color:var(--color-border)] px-2 py-1.5 ${isEmpty ? 'border-t-0' : ''}`}>
        <button
          type="button"
          disabled={uploading}
          onClick={() => fileInputRef.current?.click()}
          className="text-xs text-[color:var(--accent-600)] hover:underline disabled:cursor-not-allowed disabled:text-slate-400"
        >
          {uploading ? 'Uploading…' : '+ Attach files'}
        </button>
      </div>

      <input
        ref={fileInputRef}
        type="file"
        multiple
        hidden
        onChange={handleFileInput}
      />
    </div>
  )
}
