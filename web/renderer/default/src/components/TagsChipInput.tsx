import { useEffect, useMemo, useRef, useState } from 'react'
import { tagChipStyle } from '../utils/tagColor.js'

export function TagsChipInput({
  value,
  onChange,
  available,
  tagColors,
  ariaLabel,
}: {
  value: string[]
  onChange: (next: string[]) => void
  available: string[]
  tagColors: Record<string, string>
  ariaLabel?: string
}): JSX.Element {
  const [draft, setDraft] = useState('')
  const [focused, setFocused] = useState(false)
  const [highlight, setHighlight] = useState(0)
  const [inputKey, setInputKey] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)

  const suggestions = useMemo(() => {
    const q = draft.trim().toLowerCase()
    const picked = new Set(value)
    return available
      .filter((t) => !picked.has(t))
      .filter((t) => !q || t.toLowerCase().includes(q))
      .slice(0, 8)
  }, [draft, value, available])

  const clearInput = (): void => {
    setDraft('')
    setHighlight(0)
    setInputKey((k) => k + 1)
    if (inputRef.current) inputRef.current.value = ''
  }

  const commit = (tag: string): void => {
    const t = tag.trim()
    if (!t) return
    if (!value.includes(t)) onChange([...value, t])
    clearInput()
  }

  const removeAt = (idx: number): void => {
    const next = value.slice()
    next.splice(idx, 1)
    onChange(next)
  }

  useEffect(() => {
    const input = inputRef.current
    if (!input) return
    const handler = (e: KeyboardEvent): void => {
      const current = input.value
      if (e.key === 'Enter' || e.key === ',') {
        const picked = suggestions[highlight]
        const choice = current.trim().length > 0 ? current : picked ?? ''
        if (choice.trim()) {
          e.preventDefault()
          commit(choice)
        } else if (e.key === ',') {
          e.preventDefault()
        }
      } else if (e.key === 'Tab' && current.trim() && suggestions[highlight]) {
        e.preventDefault()
        commit(suggestions[highlight])
      } else if (e.key === 'Backspace' && current === '' && value.length > 0) {
        e.preventDefault()
        removeAt(value.length - 1)
      } else if (e.key === 'ArrowDown' && suggestions.length > 0) {
        e.preventDefault()
        setHighlight((h) => (h + 1) % suggestions.length)
      } else if (e.key === 'ArrowUp' && suggestions.length > 0) {
        e.preventDefault()
        setHighlight((h) => (h - 1 + suggestions.length) % suggestions.length)
      } else if (e.key === 'Escape' && focused) {
        e.preventDefault()
        input.blur()
      }
    }
    input.addEventListener('keydown', handler)
    return () => input.removeEventListener('keydown', handler)
  })

  const open = focused && suggestions.length > 0

  return (
    <div className="relative">
      <div
        className="mt-1 flex flex-wrap items-center gap-1 rounded border border-[color:var(--color-border)] px-2 py-1 text-sm focus-within:border-[color:var(--accent-500)]"
        onClick={() => inputRef.current?.focus()}
      >
        {value.map((t, i) => {
          const style = tagChipStyle(tagColors[t])
          return (
            <span
              key={t}
              style={style}
              className={
                (style
                  ? 'rounded px-1.5 py-0.5 text-xs'
                  : 'rounded bg-[color:var(--color-column-bg)] px-1.5 py-0.5 text-xs text-slate-700 dark:text-slate-200') +
                ' flex items-center gap-1'
              }
            >
              {t}
              <button
                type="button"
                aria-label={`remove tag ${t}`}
                onClick={(e) => {
                  e.stopPropagation()
                  removeAt(i)
                }}
                className="opacity-70 hover:opacity-100"
              >
                ✕
              </button>
            </span>
          )
        })}
        <input
          key={inputKey}
          ref={inputRef}
          aria-label={ariaLabel ?? 'tags'}
          defaultValue=""
          onInput={(e) => {
            setDraft(e.currentTarget.value)
            setHighlight(0)
          }}
          onFocus={() => setFocused(true)}
          onBlur={() => window.setTimeout(() => setFocused(false), 120)}
          className="min-w-[80px] flex-1 bg-transparent py-0.5 outline-none"
          placeholder={value.length === 0 ? 'Add tags…' : ''}
        />
      </div>
      {open && (
        <ul
          role="listbox"
          className="absolute left-0 right-0 z-10 mt-1 max-h-48 overflow-auto rounded border border-[color:var(--color-border)] bg-[color:var(--color-surface)] py-1 shadow-[var(--shadow-raised)]"
        >
          {suggestions.map((t, i) => {
            const style = tagChipStyle(tagColors[t])
            return (
              <li
                key={t}
                role="option"
                aria-selected={i === highlight}
                onMouseDown={(e) => {
                  e.preventDefault()
                  commit(t)
                }}
                onMouseEnter={() => setHighlight(i)}
                className={
                  'flex cursor-pointer items-center gap-2 px-2 py-1 text-sm ' +
                  (i === highlight ? 'bg-[color:var(--color-hover)]' : '')
                }
              >
                <span
                  style={style}
                  className={
                    style
                      ? 'rounded px-1.5 py-0.5 text-xs'
                      : 'rounded bg-[color:var(--color-column-bg)] px-1.5 py-0.5 text-xs text-slate-700 dark:text-slate-200'
                  }
                >
                  {t}
                </span>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
