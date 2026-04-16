import { useState, useRef, useEffect } from 'react'
import { useBoardMutation } from '../mutations/useBoardMutation.js'

export function AddCardButton({
  columnName,
  boardId,
}: {
  columnName: string
  boardId: string
}): JSX.Element {
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
    const title = (inputRef.current?.value ?? '').trim()
    if (title) {
      mutation.mutate({ type: 'add_card', column: columnName, title })
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
      <input
        ref={inputRef}
        aria-label={`new card in ${columnName}`}
        defaultValue=""
        onBlur={commit}
        onKeyDown={(e) => {
          if (e.key === 'Enter') { e.preventDefault(); commit() }
          else if (e.key === 'Escape') { e.preventDefault(); cancel() }
        }}
        placeholder="Card title…"
        className="mt-2 w-full rounded-md bg-white p-2 text-sm text-slate-900 shadow-sm ring-1 ring-slate-200 outline-none focus:ring-[color:var(--accent-500)] dark:bg-slate-700 dark:text-slate-100 dark:ring-slate-600"
      />
    )
  }

  return (
    <button
      type="button"
      onClick={() => setOpen(true)}
      className="mt-2 w-full rounded-md px-2 py-1 text-left text-xs text-slate-500 hover:bg-slate-200 dark:text-slate-400 dark:hover:bg-slate-800"
    >
      + Add card
    </button>
  )
}
