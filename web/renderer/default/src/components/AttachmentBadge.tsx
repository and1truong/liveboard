import type { Attachment } from '@shared/types.js'

export function AttachmentBadge({ attachments }: { attachments?: Attachment[] }): JSX.Element | null {
  if (!attachments || attachments.length === 0) return null
  return (
    <span
      title={`${attachments.length} attachment${attachments.length === 1 ? '' : 's'}`}
      className="inline-flex items-center gap-0.5 rounded bg-[color:var(--color-column-bg)] px-1 py-0.5 text-[10px] text-slate-600 dark:text-slate-300"
      aria-label={`${attachments.length} attachments`}
    >
      📎 {attachments.length}
    </span>
  )
}
