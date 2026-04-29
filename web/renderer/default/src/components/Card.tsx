import { useState } from 'react'
import type { Card as CardModel, Attachment } from '@shared/types.js'
import { useClient } from '../queries.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { AttachmentBadge } from './AttachmentBadge.js'
import { AttachmentThumbStrip } from './AttachmentThumbStrip.js'

const PRIORITY_DOT: Record<string, string> = {
  critical: 'bg-red-600',
  high: 'bg-orange-500',
  medium: 'bg-yellow-400',
  low: 'bg-slate-300',
}

export function Card({
  card,
  tagColors,
  boardId = '',
  colIdx,
  cardIdx,
  displayMode,
}: {
  card: CardModel
  tagColors?: Record<string, string>
  boardId?: string
  colIdx?: number
  cardIdx?: number
  displayMode?: 'compact' | 'normal' | 'full'
}): JSX.Element {
  const client = useClient()
  const mutation = useBoardMutation(boardId)
  const [uploading, setUploading] = useState(false)

  const compact = displayMode === 'compact'
  const canDrop = boardId !== '' && colIdx != null && cardIdx != null

  async function handleDrop(e: React.DragEvent): Promise<void> {
    if (!e.dataTransfer.files?.length || !canDrop) return
    e.preventDefault()
    setUploading(true)
    try {
      const items: Attachment[] = []
      for (const f of Array.from(e.dataTransfer.files)) {
        items.push(await client.uploadAttachment(f))
      }
      mutation.mutate({ type: 'add_attachments', col_idx: colIdx!, card_idx: cardIdx!, items })
    } finally {
      setUploading(false)
    }
  }

  return (
    <article
      onDrop={canDrop ? (e) => void handleDrop(e) : undefined}
      onDragOver={canDrop ? (e) => { if (e.dataTransfer.types.includes('Files')) e.preventDefault() } : undefined}
      className={`rounded-md bg-[color:var(--color-surface)] p-3 border ${uploading ? 'border-[color:var(--accent-500)] ring-2 ring-[color:var(--accent-500)]/30' : 'border-[color:var(--color-border)]'} shadow-[var(--shadow-card)]`}
    >
      <div className="flex items-start gap-2">
        {card.priority && (
          <span
            aria-label={`priority ${card.priority}`}
            className={`mt-1.5 h-2 w-2 shrink-0 rounded-full ${PRIORITY_DOT[card.priority] ?? 'bg-slate-300'}`}
          />
        )}
        <h3 className={`text-sm font-semibold dark:text-slate-100 ${card.completed ? 'text-slate-400 dark:text-slate-500' : ''}`}>
          {card.title}
        </h3>
        {compact && <AttachmentBadge attachments={card.attachments} />}
      </div>
      {card.tags && card.tags.length > 0 && (
        <ul className="mt-2 flex flex-wrap gap-x-2 gap-y-0.5">
          {card.tags.map((t) => {
            const color = tagColors?.[t]
            return (
              <li
                key={t}
                style={color ? { color } : undefined}
                className={
                  color
                    ? 'text-[11px] font-medium uppercase tracking-wide'
                    : 'text-[11px] font-medium uppercase tracking-wide text-slate-600 dark:text-slate-300'
                }
              >
                {t}
              </li>
            )
          })}
        </ul>
      )}
      {!compact && <AttachmentThumbStrip attachments={card.attachments} />}
    </article>
  )
}
