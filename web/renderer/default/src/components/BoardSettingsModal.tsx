import { useRef, useState, type FormEvent } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { useBoardSettings, useUpdateSettings } from '../queries/useBoardSettings.js'
import { useDeleteBoard } from '../mutations/useBoardCrud.js'
import { stageDelete } from '../mutations/undoable.js'

type ViewMode = 'board' | 'list' | 'calendar'
const VIEW_MODES: { value: ViewMode; label: string }[] = [
  { value: 'board', label: 'Board' },
  { value: 'list', label: 'List' },
  { value: 'calendar', label: 'Calendar' },
]

function normalizeViewMode(v: string | null | undefined): ViewMode {
  if (v === 'list' || v === 'calendar') return v
  return 'board'
}

export function BoardSettingsModal({
  boardId,
  boardName,
  open,
  onOpenChange,
}: {
  boardId: string
  boardName: string
  open: boolean
  onOpenChange: (next: boolean) => void
}): JSX.Element {
  const settings = useBoardSettings(boardId)
  const mutation = useUpdateSettings(boardId)
  const deleteMut = useDeleteBoard()
  const checkboxRef = useRef<HTMLInputElement>(null)
  const expandRef = useRef<HTMLInputElement>(null)
  const modeRef = useRef<HTMLSelectElement>(null)
  const [viewMode, setViewMode] = useState<ViewMode>(() => normalizeViewMode(settings.view_mode))
  const [confirmDelete, setConfirmDelete] = useState(false)

  const requestDelete = (): void => {
    if (!confirmDelete) {
      setConfirmDelete(true)
      return
    }
    setConfirmDelete(false)
    onOpenChange(false)
    stageDelete(() => deleteMut.mutate(boardId), boardName)
  }

  const submit = (e: FormEvent): void => {
    e.preventDefault()
    mutation.mutate(
      {
        show_checkbox: checkboxRef.current?.checked ?? true,
        expand_columns: expandRef.current?.checked ?? false,
        card_display_mode: (modeRef.current?.value as 'normal' | 'compact') ?? 'normal',
        view_mode: viewMode,
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
          aria-label="Board settings"
          aria-describedby={undefined}
          className="fixed left-1/2 top-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-lg bg-white p-6 shadow-xl dark:bg-slate-800"
        >
          <Dialog.Title className="text-lg font-semibold text-slate-800 dark:text-slate-100">
            Settings: {boardName}
          </Dialog.Title>
          <form onSubmit={submit} className="mt-4 space-y-4">
            <div className="text-sm text-slate-700 dark:text-slate-300">
              <span className="block text-xs font-medium text-slate-600 dark:text-slate-300">View</span>
              <div
                role="radiogroup"
                aria-label="view mode"
                className="mt-1 inline-flex rounded-md border border-slate-300 p-0.5 dark:border-slate-600"
              >
                {VIEW_MODES.map((m) => (
                  <button
                    key={m.value}
                    type="button"
                    role="radio"
                    aria-checked={viewMode === m.value}
                    aria-label={`view mode ${m.value}`}
                    onClick={() => setViewMode(m.value)}
                    className={`rounded px-3 py-1 text-sm transition-colors ${
                      viewMode === m.value
                        ? 'bg-[color:var(--accent-500)] text-white'
                        : 'text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-700'
                    }`}
                  >
                    {m.label}
                  </button>
                ))}
              </div>
            </div>
            <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
              <input
                ref={checkboxRef}
                aria-label="show complete checkbox"
                type="checkbox"
                defaultChecked={settings.show_checkbox}
                className="h-4 w-4"
              />
              Show complete checkbox on cards
            </label>
            <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
              <input
                ref={expandRef}
                aria-label="expand columns"
                type="checkbox"
                defaultChecked={settings.expand_columns}
                className="h-4 w-4"
              />
              <span>
                Expand columns
                <span className="block text-xs text-slate-500 dark:text-slate-400">Stretch to fill available space</span>
              </span>
            </label>
            <label className="block text-sm text-slate-700 dark:text-slate-300">
              <span className="block text-xs font-medium text-slate-600 dark:text-slate-300">Card display</span>
              <select
                ref={modeRef}
                aria-label="card display mode"
                defaultValue={settings.card_display_mode}
                className="mt-1 w-full rounded border border-slate-300 dark:border-slate-600 px-2 py-1 text-sm outline-none focus:border-[color:var(--accent-500)]"
              >
                <option value="normal">Normal</option>
                <option value="compact">Compact</option>
              </select>
            </label>
            <div className="mt-4 border-t border-slate-200 pt-4 dark:border-slate-700">
              <span className="block text-xs font-semibold uppercase tracking-wide text-rose-600 dark:text-rose-400">
                Danger zone
              </span>
              {!confirmDelete ? (
                <div className="mt-2 flex items-center justify-between gap-3">
                  <span className="text-sm text-slate-600 dark:text-slate-300">
                    Delete this board and its markdown file.
                  </span>
                  <button
                    type="button"
                    onClick={requestDelete}
                    className="rounded border border-rose-300 px-3 py-1 text-sm font-medium text-rose-600 hover:bg-rose-50 dark:border-rose-700/60 dark:text-rose-400 dark:hover:bg-rose-950/40"
                  >
                    Delete board
                  </button>
                </div>
              ) : (
                <div className="mt-2 rounded-md border border-rose-300 bg-rose-50 p-3 dark:border-rose-700/60 dark:bg-rose-950/30">
                  <p className="text-sm text-slate-700 dark:text-slate-200">
                    Delete <strong>{boardName}</strong>? You&rsquo;ll have 5 seconds to undo.
                  </p>
                  <div className="mt-2 flex justify-end gap-2">
                    <button
                      type="button"
                      onClick={() => setConfirmDelete(false)}
                      className="rounded px-3 py-1 text-sm text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-700"
                    >
                      Keep board
                    </button>
                    <button
                      type="button"
                      onClick={requestDelete}
                      className="rounded bg-rose-600 px-3 py-1 text-sm font-medium text-white hover:bg-rose-700"
                    >
                      Confirm delete
                    </button>
                  </div>
                </div>
              )}
            </div>
            <div className="mt-2 flex justify-end gap-2">
              <button
                type="button"
                onClick={() => onOpenChange(false)}
                className="rounded px-3 py-1 text-sm text-slate-600 hover:bg-slate-100"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={mutation.isPending}
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
