import { useState, useRef, useEffect } from 'react'
import { useBoardMutation } from '../mutations/useBoardMutation.js'

const HIG_EASE = 'cubic-bezier(0.32, 0.72, 0, 1)'
const HIG_FONT =
  "-apple-system, BlinkMacSystemFont, 'SF Pro Text', 'Helvetica Neue', system-ui, sans-serif"

export function AddColumnButton({ boardId }: { boardId: string }): JSX.Element {
  const [open, setOpen] = useState(false)
  const [hasValue, setHasValue] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useBoardMutation(boardId)
  const committedRef = useRef(false)

  useEffect(() => {
    if (open) {
      committedRef.current = false
      setHasValue(false)
      const t = setTimeout(() => inputRef.current?.focus(), 140)
      return () => clearTimeout(t)
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

  if (!open) {
    return (
      <button
        type="button"
        aria-label="add list"
        onClick={() => setOpen(true)}
        className="group ml-auto flex h-12 w-12 shrink-0 cursor-pointer items-center justify-center self-start rounded-[10px] border border-dashed border-[color:var(--color-border)] bg-transparent text-[color:var(--sb-text-tertiary)] hover:border-solid hover:bg-[color:var(--color-hover)] hover:text-[color:var(--sb-text-secondary)]"
        style={{
          fontFamily: HIG_FONT,
          transition: `background-color 180ms ${HIG_EASE}, border-color 180ms ${HIG_EASE}, color 180ms ${HIG_EASE}`,
        }}
      >
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden>
          <path
            d="M8 3v10M3 8h10"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
          />
        </svg>
      </button>
    )
  }

  return (
    <div
      className="ml-auto shrink-0 self-stretch rounded-[10px] bg-[color:var(--color-surface)]"
      style={{
        width: 288,
        minHeight: 120,
        fontFamily: HIG_FONT,
        boxShadow:
          '0 0 0 0.5px var(--color-border), ' +
          'inset 0 1px 0 rgba(255,255,255,0.5), ' +
          'var(--shadow-raised)',
      }}
    >
      <div className="flex flex-col gap-3 p-4">
        <div
          className="text-[10px] font-semibold uppercase text-[color:var(--sb-text-tertiary)]"
          style={{ letterSpacing: '0.08em' }}
        >
          New list
        </div>

        <input
          ref={inputRef}
          aria-label="new column name"
          defaultValue=""
          autoComplete="off"
          spellCheck={false}
          placeholder="Title"
          onChange={(e) => setHasValue(e.currentTarget.value.trim().length > 0)}
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === 'Enter') { e.preventDefault(); commit() }
            else if (e.key === 'Escape') { e.preventDefault(); cancel() }
          }}
          className="w-full border-0 bg-transparent p-0 text-[15px] leading-tight text-[color:var(--sb-text-primary)] placeholder-[color:var(--sb-text-tertiary)] outline-none"
          style={{ letterSpacing: '-0.01em' }}
        />

        <div
          aria-hidden
          className="h-px w-full"
          style={{ backgroundColor: 'var(--sb-divider)' }}
        />

        <div className="flex items-center justify-between gap-2">
          <div
            className="flex items-center gap-1.5 text-[11px] text-[color:var(--sb-text-tertiary)]"
          >
            <Keycap>↵</Keycap>
            <span>Add</span>
            <span className="mx-1 opacity-40">·</span>
            <Keycap wide>esc</Keycap>
            <span>Cancel</span>
          </div>

          <div className="flex items-center gap-1">
            <button
              type="button"
              onMouseDown={(e) => e.preventDefault()}
              onClick={cancel}
              className="rounded-md px-2.5 py-[5px] text-[12px] font-medium text-[color:var(--sb-text-secondary)] hover:bg-[color:var(--color-hover)]"
              style={{ transition: `background-color 120ms ${HIG_EASE}` }}
            >
              Cancel
            </button>
            <button
              type="button"
              disabled={!hasValue}
              onMouseDown={(e) => e.preventDefault()}
              onClick={commit}
              className="rounded-md px-3 py-[5px] text-[12px] font-semibold text-white active:scale-[0.97]"
              style={{
                background: hasValue
                  ? 'linear-gradient(180deg, color-mix(in srgb, var(--accent-500) 92%, #fff 8%), var(--accent-500))'
                  : 'color-mix(in srgb, var(--accent-500) 45%, transparent)',
                boxShadow: hasValue
                  ? 'inset 0 0.5px 0 rgba(255,255,255,0.4), 0 1px 2px color-mix(in srgb, var(--accent-600) 50%, transparent)'
                  : 'none',
                opacity: hasValue ? 1 : 0.55,
                cursor: hasValue ? 'pointer' : 'default',
                transition: `transform 120ms ${HIG_EASE}, opacity 160ms ${HIG_EASE}, background 160ms ${HIG_EASE}, box-shadow 160ms ${HIG_EASE}`,
              }}
            >
              Add list
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

function Keycap({ children, wide = false }: { children: React.ReactNode; wide?: boolean }): JSX.Element {
  return (
    <kbd
      className="inline-flex h-[17px] items-center justify-center rounded-[4px] font-mono text-[10px] font-medium text-[color:var(--sb-text-secondary)]"
      style={{
        minWidth: wide ? 22 : 17,
        padding: '0 4px',
        background: 'color-mix(in srgb, var(--sb-text-primary) 7%, transparent)',
        boxShadow:
          'inset 0 -1px 0 color-mix(in srgb, var(--sb-text-primary) 10%, transparent), ' +
          '0 0 0 0.5px color-mix(in srgb, var(--sb-text-primary) 12%, transparent)',
      }}
    >
      {children}
    </kbd>
  )
}
