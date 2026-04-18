import { useEffect, useRef } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import type { Board } from '@shared/types.js'
import { useBoardFilter } from '../contexts/BoardFilterContext.js'
import { activeFilterCount, PRIORITIES, type Priority } from '../utils/cardFilter.js'
import { tagChipStyle } from '../utils/tagColor.js'

const PRIORITY_LABEL: Record<Priority, string> = {
  critical: 'Critical',
  high: 'High',
  medium: 'Medium',
  low: 'Low',
}

const PRIORITY_DOT: Record<Priority, string> = {
  critical: 'bg-[color:var(--priority-critical,#e11d48)]',
  high: 'bg-[color:var(--priority-high,#f97316)]',
  medium: 'bg-[color:var(--priority-medium,#eab308)]',
  low: 'bg-[color:var(--priority-low,#64748b)]',
}

export function FilterPanelBody({
  board,
  availableTags,
  tagCounts,
  showSearch,
  onClose,
  initialFocus,
}: {
  board: Board
  availableTags: string[]
  tagCounts?: Record<string, number>
  showSearch: boolean
  onClose?: () => void
  initialFocus?: 'search' | null
}): JSX.Element {
  const {
    filter,
    setQuery,
    toggleTag,
    togglePriority,
    setHideCompleted,
    reset,
  } = useBoardFilter()
  const count = activeFilterCount(filter)
  const tagColors = board.tag_colors ?? {}
  const searchRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (showSearch && initialFocus === 'search') {
      requestAnimationFrame(() => searchRef.current?.focus())
    }
  }, [showSearch, initialFocus])

  return (
    <div className="flex h-full flex-col">
      <header className="flex h-12 shrink-0 items-center justify-between border-b border-[color:var(--header-border)] px-4">
        <h2 className="text-sm font-semibold text-[color:var(--color-text-primary)]">Filters</h2>
        {onClose && (
          <button
            type="button"
            aria-label="Close filters"
            onClick={onClose}
            className="flex h-7 w-7 items-center justify-center rounded-md text-[color:var(--color-text-muted)] transition-colors hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-secondary)]"
          >
            <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" aria-hidden>
              <path d="M3 3l10 10M13 3L3 13" />
            </svg>
          </button>
        )}
      </header>

      <div className="flex-1 overflow-y-auto px-4 py-4">
        {showSearch && (
          <div className="relative mb-4">
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
              className="h-9 w-full rounded-lg border border-[color:var(--header-border)] bg-[color:var(--color-surface)]/70 py-1 pl-8 pr-2 text-sm text-[color:var(--color-text-primary)] placeholder-[color:var(--color-text-muted)] focus:border-[color:var(--accent-500)] focus:outline-none focus:ring-2 focus:ring-[color:var(--sb-focus-ring)]"
            />
          </div>
        )}

        <section>
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
            <ul className="flex flex-wrap gap-1 pr-0.5">
              {availableTags.map((t) => {
                const selected = filter.tags.includes(t)
                const color = tagColors[t]
                const n = tagCounts?.[t] ?? 0
                return (
                  <li key={t}>
                    <button
                      type="button"
                      role="checkbox"
                      aria-checked={selected}
                      onClick={() => toggleTag(t)}
                      title={n ? `${t} (${n})` : t}
                      style={selected ? tagChipStyle(color) : undefined}
                      className={
                        selected
                          ? `inline-flex h-6 items-center gap-1 rounded-full px-2 text-xs font-medium transition-transform ${color ? '' : 'bg-[color:var(--accent-500)] text-white'}`
                          : 'inline-flex h-6 items-center gap-1 rounded-full border border-dashed border-[color:var(--color-border-dashed)] bg-transparent px-2 text-xs text-[color:var(--color-text-secondary)] transition-colors hover:border-solid hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)]'
                      }
                    >
                      <span>{t}</span>
                      {n > 0 && (
                        <span
                          className={
                            selected
                              ? 'text-[10px] opacity-70'
                              : 'text-[10px] text-[color:var(--color-text-muted)]'
                          }
                        >
                          - {n}
                        </span>
                      )}
                    </button>
                  </li>
                )
              })}
            </ul>
          )}
        </section>

        <section className="mt-4">
          <div className="mb-1.5 px-0.5">
            <span className="text-[11px] font-medium uppercase tracking-wider text-[color:var(--color-text-muted)]">
              Priority
            </span>
          </div>
          <ul className="flex flex-wrap gap-1 pr-0.5">
            {PRIORITIES.map((p) => {
              const selected = filter.priorities.includes(p)
              return (
                <li key={p}>
                  <button
                    type="button"
                    role="checkbox"
                    aria-checked={selected}
                    onClick={() => togglePriority(p)}
                    className={
                      selected
                        ? 'inline-flex h-6 items-center gap-1.5 rounded-full bg-[color:var(--accent-500)] px-2 text-xs font-medium text-white transition-colors'
                        : 'inline-flex h-6 items-center gap-1.5 rounded-full border border-dashed border-[color:var(--color-border-dashed)] bg-transparent px-2 text-xs text-[color:var(--color-text-secondary)] transition-colors hover:border-solid hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)]'
                    }
                  >
                    <span className={`h-1.5 w-1.5 rounded-full ${PRIORITY_DOT[p]}`} aria-hidden />
                    {PRIORITY_LABEL[p]}
                  </button>
                </li>
              )
            })}
          </ul>
        </section>

        <ToggleRow
          label="Hide completed"
          checked={filter.hideCompleted}
          onChange={setHideCompleted}
        />
      </div>

      <footer className="flex shrink-0 items-center justify-between border-t border-[color:var(--header-border)] px-4 py-2.5">
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
      </footer>
    </div>
  )
}

export function FilterSidePanel({
  board,
  availableTags,
  tagCounts,
  open,
  onOpenChange,
}: {
  board: Board
  availableTags: string[]
  tagCounts?: Record<string, number>
  open: boolean
  onOpenChange: (next: boolean) => void
}): JSX.Element | null {
  if (!open) return null
  return (
    <aside className="lb-filter-panel flex h-full w-[320px] shrink-0 flex-col border-l border-[color:var(--sb-popover-ring)] bg-[color:var(--sb-popover-bg)]">
      <FilterPanelBody
        board={board}
        availableTags={availableTags}
        tagCounts={tagCounts}
        showSearch={false}
        onClose={() => onOpenChange(false)}
      />
    </aside>
  )
}

export function FilterDrawer({
  board,
  availableTags,
  tagCounts,
  open,
  onOpenChange,
  initialFocus,
}: {
  board: Board
  availableTags: string[]
  tagCounts?: Record<string, number>
  open: boolean
  onOpenChange: (next: boolean) => void
  initialFocus?: 'search' | null
}): JSX.Element {
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/30 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=open]:fade-in" />
        <Dialog.Content
          aria-label="Filter cards"
          aria-describedby={undefined}
          onOpenAutoFocus={(e) => {
            if (initialFocus !== 'search') e.preventDefault()
          }}
          className="lb-filter-drawer fixed right-0 top-0 z-50 h-full w-[min(360px,92vw)] border-l border-[color:var(--sb-popover-ring)] bg-[color:var(--sb-popover-bg)] shadow-[var(--sb-popover-shadow)] backdrop-blur data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:slide-out-to-right data-[state=open]:slide-in-from-right"
        >
          <Dialog.Title className="sr-only">Filter cards</Dialog.Title>
          <FilterPanelBody
            board={board}
            availableTags={availableTags}
            tagCounts={tagCounts}
            showSearch
            initialFocus={initialFocus}
            onClose={() => onOpenChange(false)}
          />
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

// Back-compat shim — kept so existing tests continue to exercise the panel contents.
// New code should use FilterSidePanel (desktop) or FilterDrawer (mobile).
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
  return (
    <FilterDrawer
      board={board}
      availableTags={availableTags}
      open={open}
      onOpenChange={onOpenChange}
      initialFocus={initialFocus}
    />
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
      className="mt-4 flex w-full items-center justify-between rounded-lg px-1 py-1.5 text-sm text-[color:var(--color-text-primary)] transition-colors hover:bg-[color:var(--color-hover)]"
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
