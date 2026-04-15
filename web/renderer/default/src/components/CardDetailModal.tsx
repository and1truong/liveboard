import { useRef, useState, useEffect, type FormEvent } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import type { Card as CardModel } from '@shared/types.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { renderMarkdown } from './markdownPreview.js'

export function CardDetailModal({
  card,
  colIdx,
  cardIdx,
  boardId,
  open,
  onOpenChange,
}: {
  card: CardModel
  colIdx: number
  cardIdx: number
  boardId: string
  open: boolean
  onOpenChange: (next: boolean) => void
}): JSX.Element {
  const titleRef = useRef<HTMLInputElement>(null)
  const bodyRef = useRef<HTMLTextAreaElement>(null)
  const tagsRef = useRef<HTMLInputElement>(null)
  const priorityRef = useRef<HTMLSelectElement>(null)
  const dueRef = useRef<HTMLInputElement>(null)
  const assigneeRef = useRef<HTMLInputElement>(null)
  const [tab, setTab] = useState<'edit' | 'preview'>('edit')
  const [previewHtml, setPreviewHtml] = useState<string | null>(null)
  const renderGenRef = useRef(0)

  const [titleValid, setTitleValid] = useState((card.title ?? '').trim().length > 0)
  const mutation = useBoardMutation(boardId)

  const onPickPreview = (): void => {
    setTab('preview')
    setPreviewHtml(null)
    const gen = ++renderGenRef.current
    void renderMarkdown(bodyRef.current?.value ?? '').then((html) => {
      if (renderGenRef.current === gen) setPreviewHtml(html)
    })
  }

  const onPickEdit = (): void => {
    setTab('edit')
    setPreviewHtml(null)
  }

  useEffect(() => {
    if (open) setTitleValid((card.title ?? '').trim().length > 0)
  }, [open, card.title])

  const submit = (e: FormEvent): void => {
    e.preventDefault()
    const title = (titleRef.current?.value ?? '').trim()
    if (!title) return
    const tags = (tagsRef.current?.value ?? '')
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean)
    mutation.mutate(
      {
        type: 'edit_card',
        col_idx: colIdx,
        card_idx: cardIdx,
        title,
        body: bodyRef.current?.value ?? '',
        tags,
        priority: priorityRef.current?.value ?? '',
        due: dueRef.current?.value ?? '',
        assignee: assigneeRef.current?.value ?? '',
      },
      {
        onSuccess: () => onOpenChange(false),
      },
    )
  }

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/40" />
        <Dialog.Content
          key={String(open)}
          aria-describedby={undefined}
          className="fixed left-1/2 top-1/2 z-50 w-full max-w-lg -translate-x-1/2 -translate-y-1/2 rounded-lg bg-white p-6 shadow-xl dark:bg-slate-800"
        >
          <Dialog.Title className="text-lg font-semibold text-slate-800 dark:text-slate-100">Edit card</Dialog.Title>
          <form onSubmit={submit} className="mt-4 space-y-3">
            <label className="block">
              <span className="block text-xs font-medium text-slate-600 dark:text-slate-300">Title</span>
              <input
                ref={titleRef}
                aria-label="card title"
                defaultValue={card.title ?? ''}
                onInput={(e) => setTitleValid((e.currentTarget.value ?? '').trim().length > 0)}
                className="mt-1 w-full rounded border border-slate-300 dark:border-slate-600 px-2 py-1 text-sm outline-none focus:border-[color:var(--accent-500)]"
              />
            </label>
            <div>
              <div className="flex items-center justify-between">
                <span className="block text-xs font-medium text-slate-600 dark:text-slate-300">Body</span>
                <div role="tablist" className="flex gap-1 text-xs">
                  <button
                    type="button"
                    role="tab"
                    aria-selected={tab === 'edit'}
                    onClick={onPickEdit}
                    className={
                      'px-2 py-1 rounded ' +
                      (tab === 'edit'
                        ? 'border-b-2 border-[color:var(--accent-500)] font-semibold text-slate-800 dark:text-slate-100'
                        : 'text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700')
                    }
                  >
                    Edit
                  </button>
                  <button
                    type="button"
                    role="tab"
                    aria-selected={tab === 'preview'}
                    onClick={onPickPreview}
                    className={
                      'px-2 py-1 rounded ' +
                      (tab === 'preview'
                        ? 'border-b-2 border-[color:var(--accent-500)] font-semibold text-slate-800 dark:text-slate-100'
                        : 'text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700')
                    }
                  >
                    Preview
                  </button>
                </div>
              </div>
              <textarea
                ref={bodyRef}
                aria-label="card body"
                rows={6}
                defaultValue={card.body ?? ''}
                hidden={tab === 'preview'}
                className="mt-1 w-full rounded border border-slate-300 dark:border-slate-600 px-2 py-1 text-sm outline-none focus:border-[color:var(--accent-500)]"
              />
              {tab === 'preview' && (
                previewHtml === null ? (
                  <div className="mt-1 min-h-32 rounded border border-slate-200 dark:border-slate-700 px-2 py-1 text-xs italic text-slate-400">
                    Rendering…
                  </div>
                ) : (
                  <div
                    aria-label="card body preview"
                    className="mt-1 min-h-32 rounded border border-slate-200 dark:border-slate-700 px-2 py-1 text-sm prose prose-sm dark:prose-invert max-w-none"
                    dangerouslySetInnerHTML={{ __html: previewHtml }}
                  />
                )
              )}
            </div>
            <label className="block">
              <span className="block text-xs font-medium text-slate-600 dark:text-slate-300">Tags (comma separated)</span>
              <input
                ref={tagsRef}
                aria-label="card tags"
                defaultValue={(card.tags ?? []).join(', ')}
                className="mt-1 w-full rounded border border-slate-300 dark:border-slate-600 px-2 py-1 text-sm outline-none focus:border-[color:var(--accent-500)]"
              />
            </label>
            <div className="grid grid-cols-3 gap-3">
              <label className="block">
                <span className="block text-xs font-medium text-slate-600 dark:text-slate-300">Priority</span>
                <select
                  ref={priorityRef}
                  aria-label="card priority"
                  defaultValue={card.priority ?? ''}
                  className="mt-1 w-full rounded border border-slate-300 dark:border-slate-600 px-2 py-1 text-sm outline-none focus:border-[color:var(--accent-500)]"
                >
                  <option value="">—</option>
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                  <option value="critical">Critical</option>
                </select>
              </label>
              <label className="block">
                <span className="block text-xs font-medium text-slate-600 dark:text-slate-300">Due</span>
                <input
                  ref={dueRef}
                  aria-label="card due"
                  type="date"
                  defaultValue={card.due ?? ''}
                  className="mt-1 w-full rounded border border-slate-300 dark:border-slate-600 px-2 py-1 text-sm outline-none focus:border-[color:var(--accent-500)] dark:[color-scheme:dark] dark:text-slate-100"
                />
              </label>
              <label className="block">
                <span className="block text-xs font-medium text-slate-600 dark:text-slate-300">Assignee</span>
                <input
                  ref={assigneeRef}
                  aria-label="card assignee"
                  defaultValue={card.assignee ?? ''}
                  className="mt-1 w-full rounded border border-slate-300 dark:border-slate-600 px-2 py-1 text-sm outline-none focus:border-[color:var(--accent-500)]"
                />
              </label>
            </div>
            <div className="mt-2 flex justify-end gap-2">
              <button
                type="button"
                onClick={() => onOpenChange(false)}
                className="rounded px-3 py-1 text-sm text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-700"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={!titleValid || mutation.isPending}
                className="rounded bg-[color:var(--accent-600)] px-3 py-1 text-sm font-medium text-white hover:bg-[color:var(--accent-500)] disabled:cursor-not-allowed disabled:bg-slate-300"
              >
                {mutation.isPending ? 'Saving…' : 'Save'}
              </button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
