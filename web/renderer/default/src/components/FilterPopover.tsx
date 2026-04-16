import { useEffect, useRef, useState } from 'react'
import * as Popover from '@radix-ui/react-popover'
import type { Board } from '@shared/types.js'
import { useBoardFilter } from '../contexts/BoardFilterContext.js'
import { activeFilterCount } from '../utils/cardFilter.js'
import { tagChipStyle } from '../utils/tagColor.js'

export function FilterPopover({
  board,
  availableTags,
  open,
  onOpenChange,
  initialFocus,
}: {
  board: Board
  availableTags: string[]
  open: boolean
  onOpenChange: (next: boolean) => void
  initialFocus?: 'search' | null
}): JSX.Element {
  const { filter, setQuery, toggleTag, setHideCompleted, reset } = useBoardFilter()
  const count = activeFilterCount(filter)
  const tagColors = board.tag_colors ?? {}
  const searchRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (!open) return
    if (initialFocus === 'search') {
      // Defer to next tick so Radix has mounted the input.
      requestAnimationFrame(() => searchRef.current?.focus())
    }
  }, [open, initialFocus])

  return (
    <Popover.Root open={open} onOpenChange={onOpenChange}>
      <Popover.Trigger asChild>
        <button
          type="button"
          aria-label={count > 0 ? `Filter (${count} active)` : 'Filter'}
          className={`group flex h-7 items-center gap-1.5 rounded-full border pl-2 pr-2.5 text-xs font-medium leading-none transition-colors ${
            count > 0
              ? 'border-[color:var(--accent-500)]/40 bg-[color:var(--accent-500)]/10 text-[color:var(--accent-600)] dark:text-[color:var(--accent-500)]'
              : 'border-[color:var(--header-border)] bg-[color:var(--color-surface)]/60 text-[color:var(--color-text-secondary)] hover:bg-[color:var(--color-hover)]'
          }`}
        >
          <svg width="11" height="11" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
            <path d="M1.5 2h9l-3.4 4v3.5l-2.2 1V6L1.5 2z" />
          </svg>
          <span>Filter</span>
          {count > 0 && (
            <span className="ml-0.5 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-[color:var(--accent-500)] px-1 text-[10px] font-semibold text-white">
              {count}
            </span>
          )}
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          align="end"
          sideOffset={6}
          className="lb-filter-popover z-50 w-[300px] rounded-xl border border-[color:var(--sb-popover-ring)] bg-[color:var(--sb-popover-bg)] p-3 shadow-[var(--sb-popover-shadow)] backdrop-blur"
          onOpenAutoFocus={(e) => {
            // Don't auto-focus the first interactive element when opening from the trigger;
            // we focus the search input ourselves only when initialFocus === 'search'.
            if (initialFocus !== 'search') e.preventDefault()
          }}
        >
          <div className="relative">
            <svg
              className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-[color:var(--color-text-muted)]"
              width="12"
              height="12"
              viewBox="0 0 16 16"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
              aria-hidden
            >
              <circle cx="6.5" cy="6.5" r="5" />
              <line x1="10" y1="10" x2="14.5" y2="14.5" />
            </svg>
            <input
              ref={searchRef}
              type="text"
              value={filter.query}
              placeholder="Search cards…"
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Escape' && filter.query) {
                  e.stopPropagation()
                  setQuery('')
                }
              }}
              className="h-8 w-full rounded-lg border border-[color:var(--header-border)] bg-[color:var(--color-surface)]/70 py-1 pl-8 pr-2 text-sm text-[color:var(--color-text-primary)] placeholder-[color:var(--color-text-muted)] focus:border-[color:var(--accent-500)] focus:outline-none focus:ring-2 focus:ring-[color:var(--sb-focus-ring)]"
            />
          </div>

          <div className="mt-3">
            <div className="mb-1.5 flex items-baseline justify-between px-0.5">
              <span className="text-[11px] font-medium uppercase tracking-wider text-[color:var(--color-text-muted)]">
                Tags
              </span>
              {filter.tags.length > 1 && (
                <span className="text-[10px] text-[color:var(--color-text-muted)]">match all</span>
              )}
            </div>
            {availableTags.length === 0 ? (
              <p className="px-0.5 py-2 text-xs text-[color:var(--color-text-muted)]">
                No tags on this board yet.
              </p>
            ) : (
              <ul className="flex max-h-44 flex-wrap gap-1 overflow-y-auto pr-0.5">
                {availableTags.map((t) => {
                  const selected = filter.tags.includes(t)
                  const color = tagColors[t]
                  return (
                    <li key={t}>
                      <button
                        type="button"
                        role="checkbox"
                        aria-checked={selected}
                        onClick={() => toggleTag(t)}
                        title={t}
                        style={selected ? tagChipStyle(color) : undefined}
                        className={
                          selected
                            ? `inline-flex h-6 items-center rounded-full px-2 text-xs font-medium transition-transform ${color ? '' : 'bg-[color:var(--accent-500)] text-white'}`
                            : 'inline-flex h-6 items-center rounded-full border border-dashed border-[color:var(--color-border-dashed)] bg-transparent px-2 text-xs text-[color:var(--color-text-secondary)] transition-colors hover:border-solid hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)]'
                        }
                      >
                        {t}
                      </button>
                    </li>
                  )
                })}
              </ul>
            )}
          </div>

          <ToggleRow
            label="Hide completed"
            checked={filter.hideCompleted}
            onChange={setHideCompleted}
          />

          <div className="mt-3 flex items-center justify-between border-t border-[color:var(--header-border)] pt-2.5">
            <span className="text-[11px] text-[color:var(--color-text-muted)]">
              {count === 0 ? 'No filters active' : `${count} active`}
            </span>
            <button
              type="button"
              onClick={reset}
              disabled={count === 0}
              className="text-xs font-medium text-[color:var(--accent-600)] transition-opacity hover:opacity-70 disabled:cursor-default disabled:opacity-30 dark:text-[color:var(--accent-500)]"
            >
              Reset
            </button>
          </div>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}

function ToggleRow({
  label,
  checked,
  onChange,
}: {
  label: string
  checked: boolean
  onChange: (next: boolean) => void
}): JSX.Element {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={() => onChange(!checked)}
      className="mt-3 flex w-full items-center justify-between rounded-lg px-1 py-1.5 text-sm text-[color:var(--color-text-primary)] transition-colors hover:bg-[color:var(--color-hover)]"
    >
      <span>{label}</span>
      <span
        className={`relative inline-flex h-[18px] w-[30px] shrink-0 items-center rounded-full transition-colors ${
          checked
            ? 'bg-[color:var(--accent-500)]'
            : 'bg-[color:var(--color-border)]'
        }`}
      >
        <span
          className={`absolute h-[14px] w-[14px] rounded-full bg-white shadow-sm transition-transform ${
            checked ? 'translate-x-[14px]' : 'translate-x-[2px]'
          }`}
        />
      </span>
    </button>
  )
}

// Re-export a controlled-trigger variant useful for the BoardHeader to open the
// popover programmatically (e.g. when clicking an active-filter chip).
export function useFilterPopoverState(): {
  open: boolean
  setOpen: (next: boolean) => void
  initialFocus: 'search' | null
  openWith: (focus: 'search' | null) => void
} {
  const [open, setOpen] = useState(false)
  const [initialFocus, setInitialFocus] = useState<'search' | null>(null)
  const openWith = (focus: 'search' | null): void => {
    setInitialFocus(focus)
    setOpen(true)
  }
  return { open, setOpen, initialFocus, openWith }
}
