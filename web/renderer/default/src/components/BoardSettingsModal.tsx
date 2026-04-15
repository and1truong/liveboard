import { useRef, type FormEvent } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { useBoardSettings, useUpdateSettings } from '../queries/useBoardSettings.js'

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
  const checkboxRef = useRef<HTMLInputElement>(null)
  const modeRef = useRef<HTMLSelectElement>(null)

  const submit = (e: FormEvent): void => {
    e.preventDefault()
    mutation.mutate(
      {
        show_checkbox: checkboxRef.current?.checked ?? true,
        card_display_mode: (modeRef.current?.value as 'normal' | 'compact') ?? 'normal',
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
          className="fixed left-1/2 top-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-lg bg-white p-6 shadow-xl"
        >
          <Dialog.Title className="text-lg font-semibold text-slate-800">
            Settings: {boardName}
          </Dialog.Title>
          <form onSubmit={submit} className="mt-4 space-y-4">
            <label className="flex items-center gap-2 text-sm text-slate-700">
              <input
                ref={checkboxRef}
                aria-label="show complete checkbox"
                type="checkbox"
                defaultChecked={settings.show_checkbox}
                className="h-4 w-4"
              />
              Show complete checkbox on cards
            </label>
            <label className="block text-sm text-slate-700">
              <span className="block text-xs font-medium text-slate-600">Card display</span>
              <select
                ref={modeRef}
                aria-label="card display mode"
                defaultValue={settings.card_display_mode}
                className="mt-1 w-full rounded border border-slate-300 px-2 py-1 text-sm outline-none focus:border-blue-400"
              >
                <option value="normal">Normal</option>
                <option value="compact">Compact</option>
              </select>
            </label>
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
                className="rounded bg-blue-600 px-3 py-1 text-sm font-medium text-white disabled:cursor-not-allowed disabled:bg-slate-300"
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
