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
    <span ref={wrapperRef} className="relative shrink-0">
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation()
          setOpen((p) => !p)
        }}
        title="Change icon"
        aria-label="Change board icon"
        className="flex h-7 w-7 items-center justify-center rounded text-base leading-none transition-colors hover:bg-slate-200 dark:hover:bg-slate-700"
      >
        {icon ? (
          <span aria-hidden>{icon}</span>
        ) : (
          <span aria-hidden className="text-slate-400 dark:text-slate-500">{PLACEHOLDER}</span>
        )}
      </button>
      {open && (
        <div
          role="dialog"
          aria-label="Choose board icon"
          onClick={(e) => e.stopPropagation()}
          className="absolute left-full top-0 z-50 ml-2 grid w-[260px] grid-cols-8 gap-1 rounded-lg border border-slate-200 bg-white p-2 shadow-xl dark:border-slate-700 dark:bg-slate-800"
        >
          <button
            type="button"
            onClick={() => pick('')}
            title="Remove icon"
            aria-label="Remove icon"
            className="flex h-8 w-8 items-center justify-center rounded text-sm text-slate-400 hover:bg-slate-100 dark:text-slate-500 dark:hover:bg-slate-700"
          >
            ✕
          </button>
          {EMOJIS.map((e) => (
            <button
              key={e}
              type="button"
              onClick={() => pick(e)}
              className="flex h-8 w-8 items-center justify-center rounded text-lg leading-none hover:bg-slate-100 dark:hover:bg-slate-700"
            >
              {e}
            </button>
          ))}
        </div>
      )}
    </span>
  )
}
