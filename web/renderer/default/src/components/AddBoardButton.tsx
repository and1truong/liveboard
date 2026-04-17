import { useState, useRef, useEffect } from 'react'
import { useCreateBoard } from '../mutations/useBoardCrud.js'

export interface AddBoardButtonProps {
  // Available folders, for the target selector. Undefined = hide the selector
  // and default to root.
  folders?: string[]
}

export function AddBoardButton({ folders }: AddBoardButtonProps): JSX.Element {
  const [open, setOpen] = useState(false)
  const [folder, setFolder] = useState<string>('')
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
    if (name) mutation.mutate({ name, folder: folder || undefined })
    Promise.resolve().then(() => { setOpen(false); setFolder('') })
  }

  const cancel = (): void => {
    if (committedRef.current) return
    committedRef.current = true
    Promise.resolve().then(() => { setOpen(false); setFolder('') })
  }

  if (open) {
    const showFolderSelect = folders !== undefined && folders.length > 0
    return (
      <div className="lb-row lb-row--add-input">
        <input
          ref={inputRef}
          aria-label="new board name"
          defaultValue=""
          // Do not commit on blur when focus moves to the folder select.
          onBlur={(e) => {
            if (e.relatedTarget && (e.relatedTarget as HTMLElement).classList.contains('lb-row__folder-select')) {
              return
            }
            commit()
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter') { e.preventDefault(); commit() }
            else if (e.key === 'Escape') { e.preventDefault(); cancel() }
          }}
          placeholder="Board name…"
          className="lb-row__input"
        />
        {showFolderSelect && (
          <select
            className="lb-row__folder-select"
            aria-label="new board folder"
            value={folder}
            onChange={(e) => setFolder(e.target.value)}
            onBlur={(e) => {
              if (e.relatedTarget && e.relatedTarget === inputRef.current) return
              commit()
            }}
          >
            <option value="">(Root)</option>
            {folders!.map((f) => (
              <option key={f} value={f}>{f}</option>
            ))}
          </select>
        )}
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
