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

  if (open) {
    return (
      <div className="flex w-72 shrink-0 flex-col rounded-lg bg-slate-100 p-3">
        <input
          ref={inputRef}
          aria-label="new column name"
          defaultValue=""
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === 'Enter') { e.preventDefault(); commit() }
            else if (e.key === 'Escape') { e.preventDefault(); cancel() }
          }}
          placeholder="Column name…"
          className="w-full rounded bg-white px-2 py-1 text-sm outline-none ring-1 ring-slate-200 focus:ring-blue-400"
        />
      </div>
    )
  }

  return (
    <button
      type="button"
      onClick={() => setOpen(true)}
      className="flex w-72 shrink-0 items-center justify-center rounded-lg border-2 border-dashed border-slate-300 p-3 text-sm text-slate-500 hover:border-slate-400 hover:text-slate-700"
    >
      + Add column
    </button>
  )
}
