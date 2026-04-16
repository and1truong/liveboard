import { useEffect, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useBoardMutation } from '../mutations/useBoardMutation.js'

const EMOJIS = [
  '📋','📌','📝','📊','📈','🎯','🚀','💡','🔥','⭐',
  '❤️','💼','🏠','🎨','🎵','📚','🔧','⚡','🌟','🎮',
  '🧪','📦','🔔','💬','🌈','🍀','🦊','🐱','🐶','🌻',
  '🌙','☀️','🏔️','🌊','🎪','🏆','💎','🔑','🎁','🧩',
] as const

const PLACEHOLDER = '\u2637' // ☷ — matches HTMX sidebar placeholder

export function BoardIconPicker({
  boardId,
  icon,
}: {
  boardId: string
  icon?: string
}): JSX.Element {
  const [open, setOpen] = useState(false)
  const wrapperRef = useRef<HTMLSpanElement>(null)
  const mutation = useBoardMutation(boardId)
  const qc = useQueryClient()

  useEffect(() => {
    if (!open) return
    const onMouseDown = (e: MouseEvent): void => {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    const onKey = (e: KeyboardEvent): void => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onMouseDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onMouseDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const pick = (next: string): void => {
    setOpen(false)
    mutation.mutate(
      { type: 'update_board_icon', icon: next },
      {
        onSuccess: () => {
          void qc.invalidateQueries({ queryKey: ['boards'] })
        },
      },
    )
  }

  return (
    <span ref={wrapperRef} style={{ position: 'relative', flexShrink: 0 }}>
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation()
          setOpen((p) => !p)
        }}
        title="Change icon"
        aria-label="Change board icon"
        className={`lb-row__icon${icon ? '' : ' lb-row__icon--placeholder'}`}
      >
        <span aria-hidden>{icon || PLACEHOLDER}</span>
      </button>
      {open && (
        <div
          role="dialog"
          aria-label="Choose board icon"
          onClick={(e) => e.stopPropagation()}
          className="lb-popover lb-popover--grid"
          style={{ position: 'absolute', left: '100%', top: 0, marginLeft: 8 }}
        >
          <button
            type="button"
            onClick={() => pick('')}
            title="Remove icon"
            aria-label="Remove icon"
            className="lb-popover__cell lb-popover__cell--clear"
          >
            ✕
          </button>
          {EMOJIS.map((e) => (
            <button
              key={e}
              type="button"
              onClick={() => pick(e)}
              className="lb-popover__cell"
            >
              {e}
            </button>
          ))}
        </div>
      )}
    </span>
  )
}
