import type { Attachment } from '@shared/types.js'
import { useClient } from '../queries.js'

const IMAGE_RE = /^image\//
const MAX_VISIBLE = 3

export function AttachmentThumbStrip({ attachments }: { attachments?: Attachment[] }): JSX.Element | null {
  const client = useClient()
  if (!attachments || attachments.length === 0) return null
  const visible = attachments.slice(0, MAX_VISIBLE)
  const overflow = attachments.length - visible.length
  return (
    <div className="mt-2 flex items-center gap-1">
      {visible.map((a) => (
        <div key={a.h} className="h-10 w-10 overflow-hidden rounded border border-[color:var(--color-border)] bg-[color:var(--color-column-bg)]">
          {IMAGE_RE.test(a.m) ? (
            <img
              src={client.attachmentThumbURL(a)}
              alt={a.n}
              loading="lazy"
              className="h-full w-full object-cover"
              onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none' }}
            />
          ) : (
            <span className="flex h-full w-full items-center justify-center text-base" title={a.n}>📄</span>
          )}
        </div>
      ))}
      {overflow > 0 && (
        <span className="rounded bg-[color:var(--color-column-bg)] px-1.5 py-0.5 text-[10px] text-slate-600 dark:text-slate-300">
          +{overflow}
        </span>
      )}
    </div>
  )
}
