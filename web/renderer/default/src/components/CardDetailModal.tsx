import { useRef, useState, useEffect, type FormEvent } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import type { Card as CardModel } from '@shared/types.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { renderMarkdown } from './markdownPreview.js'
import { useBacklinks } from '../queries/useBacklinks.js'
import { useOptionalActiveBoard } from '../contexts/ActiveBoardContext.js'
import { useOptionalBoardFocus } from '../contexts/BoardFocusContext.js'
import { LinkChip } from './LinkChip.js'
import { LinkPicker } from './LinkPicker.js'

export interface CreateCardParams {
  title: string
  body: string
  tags: string[]
  links: string[]
  priority: string
  due: string
  assignee: string
}

export function CardDetailModal({
  card,
  colIdx,
  cardIdx,
  boardId,
  open,
  onOpenChange,
  initialDue,
  isNew,
  onCreateCard,
}: {
  card: CardModel
  colIdx: number
  cardIdx: number
  boardId: string
  open: boolean
  onOpenChange: (next: boolean) => void
  initialDue?: string
  isNew?: boolean
  onCreateCard?: (params: CreateCardParams) => void
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
  const [links, setLinks] = useState<string[]>(card.links ?? [])
  const [pickerOpen, setPickerOpen] = useState(false)
  const mutation = useBoardMutation(boardId)

  const backlinks = useBacklinks(card.id)
  const activeBoardCtx = useOptionalActiveBoard()
  const focusCtx = useOptionalBoardFocus()

  useEffect(() => {
    if (open) setLinks(card.links ?? [])
    else setPickerOpen(false)
  }, [open, card.links])

  const addLink = (t: string): void => {
    if (!links.includes(t)) setLinks([...links, t])
    setPickerOpen(false)
  }
  const removeLink = (t: string): void => setLinks(links.filter((l) => l !== t))
  const navigateToBacklink = (b: typeof backlinks[number]): void => {
    activeBoardCtx?.setActive(b.boardId)
    Promise.resolve().then(() => focusCtx?.setFocused({ colIdx: b.colIdx, cardIdx: b.cardIdx }))
    onOpenChange(false)
  }

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
    if (onCreateCard) {
      onCreateCard({
        title,
        body: bodyRef.current?.value ?? '',
        tags,
        links,
        priority: priorityRef.current?.value ?? '',
        due: dueRef.current?.value ?? '',
        assignee: assigneeRef.current?.value ?? '',
      })
      onOpenChange(false)
    } else {
      mutation.mutate(
        {
          type: 'edit_card',
          col_idx: colIdx,
          card_idx: cardIdx,
          title,
          body: bodyRef.current?.value ?? '',
          tags,
          links,
          priority: priorityRef.current?.value ?? '',
          due: dueRef.current?.value ?? '',
          assignee: assigneeRef.current?.value ?? '',
        },
        {
          onSuccess: () => onOpenChange(false),
        },
      )
    }
  }

  return (
    <>
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/40 backdrop-blur-sm" />
        <Dialog.Content
          key={String(open)}
          aria-describedby={undefined}
          className="fixed left-1/2 top-1/2 z-50 w-full max-w-lg -translate-x-1/2 -translate-y-1/2 rounded-lg border border-[color:var(--color-border)] bg-[color:var(--color-surface)] p-6 shadow-[var(--shadow-raised)]"
        >
          <Dialog.Title className="text-lg font-semibold text-slate-800 dark:text-slate-100">{isNew ? 'New card' : 'Edit card'}</Dialog.Title>
          <form onSubmit={submit} className="mt-4 space-y-3">
            <label className="block">
              <span className="block text-xs font-medium text-slate-600 dark:text-slate-300">Title</span>
              <input
                ref={titleRef}
                aria-label="card title"
                autoFocus={isNew}
                placeholder={isNew ? 'Card title' : undefined}
                defaultValue={card.title ?? ''}
                onInput={(e) => setTitleValid((e.currentTarget.value ?? '').trim().length > 0)}
                className="mt-1 w-full rounded border border-[color:var(--color-border)] px-2 py-1 text-sm outline-none focus:border-[color:var(--accent-500)]"
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
                        ? 'border-b-2 border-[color:var(--accent-500)] font-semibold text-[color:var(--color-text-primary)]'
                        : 'text-[color:var(--color-text-secondary)] hover:bg-[color:var(--color-hover)]')
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
                        ? 'border-b-2 border-[color:var(--accent-500)] font-semibold text-[color:var(--color-text-primary)]'
                        : 'text-[color:var(--color-text-secondary)] hover:bg-[color:var(--color-hover)]')
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
                className="mt-1 w-full rounded border border-[color:var(--color-border)] px-2 py-1 text-sm outline-none focus:border-[color:var(--accent-500)]"
              />
              {tab === 'preview' && (
                previewHtml === null ? (
                  <div className="mt-1 min-h-32 rounded border border-[color:var(--color-border-dashed)] px-2 py-1 text-xs italic text-slate-400">
                    Rendering…
                  </div>
                ) : (
                  <div
                    aria-label="card body preview"
                    className="mt-1 min-h-32 rounded border border-[color:var(--color-border-dashed)] px-2 py-1 text-sm prose prose-sm dark:prose-invert max-w-none"
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
                className="mt-1 w-full rounded border border-[color:var(--color-border)] px-2 py-1 text-sm outline-none focus:border-[color:var(--accent-500)]"
              />
            </label>
            <div className="grid grid-cols-3 gap-3">
              <label className="block">
                <span className="block text-xs font-medium text-slate-600 dark:text-slate-300">Priority</span>
                <select
                  ref={priorityRef}
                  aria-label="card priority"
                  defaultValue={card.priority ?? ''}
                  className="mt-1 w-full rounded border border-[color:var(--color-border)] px-2 py-1 text-sm outline-none focus:border-[color:var(--accent-500)]"
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
                  defaultValue={card.due ?? initialDue ?? ''}
                  className="mt-1 w-full rounded border border-[color:var(--color-border)] px-2 py-1 text-sm outline-none focus:border-[color:var(--accent-500)] dark:[color-scheme:dark] dark:text-slate-100"
                />
              </label>
              <label className="block">
                <span className="block text-xs font-medium text-slate-600 dark:text-slate-300">Assignee</span>
                <input
                  ref={assigneeRef}
                  aria-label="card assignee"
                  defaultValue={card.assignee ?? ''}
                  className="mt-1 w-full rounded border border-[color:var(--color-border)] px-2 py-1 text-sm outline-none focus:border-[color:var(--accent-500)]"
                />
              </label>
            </div>
            <div>
              <span className="block text-xs font-medium text-slate-600 dark:text-slate-300">Links</span>
              <ul className="mt-1 flex flex-wrap gap-1">
                {links.map((l) => (
                  <LinkChip key={l} target={l} onRemove={() => removeLink(l)} />
                ))}
              </ul>
              <button
                type="button"
                onClick={() => setPickerOpen(true)}
                className="mt-1 text-xs text-[color:var(--accent-600)] hover:underline"
              >
                + Add link
              </button>
            </div>
            {backlinks.length > 0 && (
              <div>
                <span className="block text-xs font-medium text-slate-600 dark:text-slate-300">Linked from</span>
                <ul className="mt-1 flex flex-wrap gap-1">
                  {backlinks.map((b) => (
                    <li
                      key={`${b.boardId}:${b.colIdx}:${b.cardIdx}`}
                      className="flex items-center gap-1 rounded bg-[color:var(--color-column-bg)] px-2 py-1 text-xs"
                    >
                      <button type="button" onClick={() => navigateToBacklink(b)} className="text-left">
                        <span className="text-slate-500 dark:text-slate-400">{b.boardName} · </span>
                        {b.cardTitle}
                      </button>
                    </li>
                  ))}
                </ul>
              </div>
            )}
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
    <LinkPicker
      open={pickerOpen}
      onOpenChange={setPickerOpen}
      onPick={addLink}
      excludeBoardId={boardId}
      excludeCardId={card.id}
    />
    </>
  )
}
