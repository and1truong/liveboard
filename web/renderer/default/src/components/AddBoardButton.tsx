import { useState, useRef, useEffect } from 'react'
import { useCreateBoard } from '../mutations/useBoardCrud.js'

export function AddBoardButton(): JSX.Element {
  const [open, setOpen] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useCreateBoard()
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
    if (name) mutation.mutate(name)
    Promise.resolve().then(() => setOpen(false))
  }

  const cancel = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    Promise.resolve().then(() => setOpen(false))
  }

  if (open) {
    return (
      <div className="border-t border-slate-200 p-2 dark:border-slate-800">
        <input
          ref={inputRef}
          aria-label="new board name"
          defaultValue=""
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === 'Enter') { e.preventDefault(); commit() }
            else if (e.key === 'Escape') { e.preventDefault(); cancel() }
          }}
          placeholder="Board name…"
          className="w-full rounded bg-white px-2 py-1 text-sm outline-none ring-1 ring-slate-200 focus:ring-[color:var(--accent-500)] dark:bg-slate-800 dark:text-slate-100 dark:ring-slate-700 dark:placeholder:text-slate-500"
        />
      </div>
    )
  }

  return (
    <button
      type="button"
      onClick={() => setOpen(true)}
      className="w-full border-t border-slate-200 px-3 py-2 text-left text-sm text-slate-500 hover:bg-slate-50 dark:border-slate-800 dark:text-slate-400 dark:hover:bg-slate-800"
    >
      + New board
    </button>
  )
}
