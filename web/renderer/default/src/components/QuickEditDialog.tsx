import { useRef, type FormEvent } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import type { Card as CardModel } from '@shared/types.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'

export function QuickEditDialog({
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
  const mutation = useBoardMutation(boardId)

  const submit = (e: FormEvent): void => {
    e.preventDefault()
    const title = (titleRef.current?.value ?? card.title).trim() || card.title
    const body = bodyRef.current?.value ?? card.body ?? ''
    const tags = (tagsRef.current?.value ?? '')
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean)
    const priority = priorityRef.current?.value ?? card.priority ?? ''
    const due = dueRef.current?.value ?? card.due ?? ''
    const assignee = assigneeRef.current?.value ?? card.assignee ?? ''
    mutation.mutate({
      type: 'edit_card',
      col_idx: colIdx,
      card_idx: cardIdx,
      title,
      body,
      tags,
      links: card.links ?? [],
      priority,
      due,
      assignee,
    })
    onOpenChange(false)
  }

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/40" />
        <Dialog.Content
          aria-label="quick edit card"
          className="fixed left-1/2 top-1/2 z-50 w-[min(480px,90vw)] -translate-x-1/2 -translate-y-1/2 rounded-lg bg-white p-4 shadow-xl dark:bg-slate-800 dark:text-slate-100"
        >
          <Dialog.Title className="mb-3 text-sm font-semibold">Quick edit</Dialog.Title>
          <form onSubmit={submit} className="flex flex-col gap-2">
            <input
              ref={titleRef}
              aria-label="title"
              defaultValue={card.title}
              className="rounded border px-2 py-1 text-sm dark:border-slate-600 dark:bg-slate-700"
              autoFocus
            />
            <textarea
              ref={bodyRef}
              aria-label="body"
              defaultValue={card.body ?? ''}
              rows={3}
              className="rounded border px-2 py-1 text-sm dark:border-slate-600 dark:bg-slate-700"
            />
            <input
              ref={tagsRef}
              aria-label="tags"
              placeholder="tags (comma separated)"
              defaultValue={(card.tags ?? []).join(', ')}
              className="rounded border px-2 py-1 text-sm dark:border-slate-600 dark:bg-slate-700"
            />
            <div className="flex gap-2">
              <select
                ref={priorityRef}
                aria-label="priority"
                defaultValue={card.priority ?? ''}
                className="flex-1 rounded border px-2 py-1 text-sm dark:border-slate-600 dark:bg-slate-700"
              >
                <option value="">no priority</option>
                <option value="critical">critical</option>
                <option value="high">high</option>
                <option value="medium">medium</option>
                <option value="low">low</option>
              </select>
              <input
                ref={dueRef}
                aria-label="due"
                type="date"
                defaultValue={card.due ?? ''}
                className="flex-1 rounded border px-2 py-1 text-sm dark:border-slate-600 dark:bg-slate-700"
              />
            </div>
            <input
              ref={assigneeRef}
              aria-label="assignee"
              placeholder="assignee"
              defaultValue={card.assignee ?? ''}
              className="rounded border px-2 py-1 text-sm dark:border-slate-600 dark:bg-slate-700"
            />
            <div className="mt-2 flex justify-end gap-2">
              <Dialog.Close className="rounded px-3 py-1 text-sm text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-700">
                Cancel
              </Dialog.Close>
              <button
                type="submit"
                className="rounded bg-[color:var(--accent-500)] px-3 py-1 text-sm text-white hover:opacity-90"
              >
                Save
              </button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
