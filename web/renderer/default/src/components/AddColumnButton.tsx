import { useState, useRef, useEffect } from 'react'
import { useBoardMutation } from '../mutations/useBoardMutation.js'

export function AddColumnButton({ boardId }: { boardId: string }): JSX.Element {
  const [open, setOpen] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useBoardMutation(boardId)
  const committedRef = useRef(false)

  useEffect(() => {
    if (open) {
      committedRef.current = false
      inputRef.current?.focus()
    }
  }, [open])

  const commit = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    const name = (inputRef.current?.value ?? '').trim()
    if (name) {
      mutation.mutate({ type: 'add_column', name })
    }
    Promise.resolve().then(() => setOpen(false))
  }

  const cancel = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    Promise.resolve().then(() => setOpen(false))
  }

  // Single persistent element — mirrors HTMX .add-column-bar which keeps ml-auto
  // and self-stretch in both states, only the width and content change.
  return (
    <div
      className={`ml-auto flex shrink-0 self-stretch items-center justify-center rounded-lg border border-[color:var(--color-border)] bg-[color:var(--color-column-bg)] ${open ? 'cursor-default' : 'cursor-pointer hover:bg-[color:var(--color-border)]'}`}
      style={{ width: open ? 220 : 40, minHeight: 120 }}
      onClick={!open ? () => setOpen(true) : undefined}
    >
      {!open && (
        <span
          className="select-none whitespace-nowrap text-sm font-medium text-[color:var(--color-text-muted)]"
          style={{ writingMode: 'vertical-rl', textOrientation: 'mixed', letterSpacing: '0.5px' }}
        >
          + Add list
        </span>
      )}
      {open && (
        <div className="flex w-full flex-col gap-2 p-3">
          <input
            ref={inputRef}
            aria-label="new column name"
            defaultValue=""
            onKeyDown={(e) => {
              if (e.key === 'Enter') { e.preventDefault(); commit() }
              else if (e.key === 'Escape') { e.preventDefault(); cancel() }
            }}
            placeholder="List name"
            className="w-full rounded bg-[color:var(--color-surface)] px-2 py-1.5 text-sm outline-none ring-1 ring-[color:var(--color-border)] focus:ring-[color:var(--accent-500)] dark:text-slate-100 dark:placeholder-slate-400"
          />
          <button
            type="button"
            onMouseDown={(e) => e.preventDefault()}
            onClick={commit}
            className="w-full rounded px-2 py-1.5 text-sm font-medium text-slate-600 hover:bg-[color:var(--color-hover)] dark:text-slate-300"
          >
            Add
          </button>
        </div>
      )}
    </div>
  )
}
