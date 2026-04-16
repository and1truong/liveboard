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
      <div className="lb-row lb-row--add-input">
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
          className="lb-row__input"
        />
      </div>
    )
  }

  return (
    <button
      type="button"
      onClick={() => setOpen(true)}
      className="lb-row lb-row--add"
    >
      <span className="lb-row__plus" aria-hidden>+</span>
      <span className="lb-row__label">+ New board</span>
    </button>
  )
}
