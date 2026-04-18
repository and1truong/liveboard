import type { Board } from '@shared/types.js'
import { useBoardFilter } from '../contexts/BoardFilterContext.js'
import { tagChipStyle } from '../utils/tagColor.js'
import { activeFilterCount, type Priority } from '../utils/cardFilter.js'
import { FilterChip } from './FilterChip.js'
import { BoardIcon } from './BoardIcon.js'

const PRIORITY_LABEL: Record<Priority, string> = {
  critical: 'Critical',
  high: 'High',
  medium: 'Medium',
  low: 'Low',
}

export function BoardHeader({
  data,
  onToggleSidebar,
  onOpenSettings,
  filterOpen,
  onOpenFilter,
  onOpenSearch,
}: {
  data: Board
  onToggleSidebar: () => void
  onOpenSettings: () => void
  filterOpen: boolean
  onOpenFilter: () => void
  onOpenSearch: () => void
}): JSX.Element {
  const { filter, setQuery, toggleTag, togglePriority, setHideCompleted } = useBoardFilter()
  const tagColors = data.tag_colors ?? {}
  const count = activeFilterCount(filter)
  const trimmedQuery = filter.query.trim()

  return (
    <header className="lb-board-header relative flex h-12 shrink-0 items-center gap-2 border-b border-[color:var(--header-border)] bg-[color:var(--header-bg)] px-3 backdrop-blur-md">
      <button
        type="button"
        onClick={onToggleSidebar}
        title="Toggle sidebar"
        className="hidden h-7 w-7 shrink-0 items-center justify-center rounded-md text-[color:var(--color-text-muted)] transition-colors hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-secondary)] md:flex"
      >
        <svg width="15" height="15" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
          <path d="M16.5 4A1.5 1.5 0 0 1 18 5.5v9a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 2 14.5v-9A1.5 1.5 0 0 1 3.5 4zM7 15h9.5a.5.5 0 0 0 .5-.5v-9a.5.5 0 0 0-.5-.5H7zM3.5 5a.5.5 0 0 0-.5.5v9a.5.5 0 0 0 .5.5H6V5z" />
        </svg>
      </button>

      <div className="flex min-w-0 items-center gap-2">
        {data.icon && <BoardIcon icon={data.icon} color={data.icon_color} size="sm" />}
        <h1
          className="truncate text-[15px] font-semibold tracking-[-0.01em] text-[color:var(--color-text-primary)]"
          style={{ fontFeatureSettings: '"ss01", "cv11"' }}
        >
          {data.name}
        </h1>
      </div>

      <button
        type="button"
        onClick={onOpenSettings}
        title="Board settings"
        className="ml-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-md text-[color:var(--color-text-muted)] transition-colors hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-secondary)]"
      >
        <svg width="13" height="13" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
          <path
            fillRule="evenodd"
            d="M11.49 3.17c-.38-1.56-2.6-1.56-2.98 0a1.532 1.532 0 01-2.286.948c-1.372-.836-2.942.734-2.106 2.106.54.886.061 2.042-.947 2.287-1.561.379-1.561 2.6 0 2.978a1.532 1.532 0 01.947 2.287c-.836 1.372.734 2.942 2.106 2.106a1.532 1.532 0 012.287.947c.379 1.561 2.6 1.561 2.978 0a1.533 1.533 0 012.287-.947c1.372.836 2.942-.734 2.106-2.106a1.533 1.533 0 01.947-2.287c1.561-.379 1.561-2.6 0-2.978a1.532 1.532 0 01-.947-2.287c.836-1.372-.734-2.942-2.106-2.106a1.532 1.532 0 01-2.287-.947zM10 13a3 3 0 100-6 3 3 0 000 6z"
            clipRule="evenodd"
          />
        </svg>
      </button>

      <div className="ml-auto flex min-w-0 items-center gap-1.5">
        {count > 0 && (
          <ul className="hidden min-w-0 items-center gap-1 overflow-x-auto pr-1 md:flex">
            {trimmedQuery && (
              <li>
                <FilterChip
                  variant="accent"
                  label={
                    <span>
                      Search: <span className="font-semibold">{trimmedQuery}</span>
                    </span>
                  }
                  onRemove={() => setQuery('')}
                  removeAriaLabel={`Clear search "${trimmedQuery}"`}
                />
              </li>
            )}
            {filter.tags.map((t) => (
              <li key={t}>
                <FilterChip
                  variant="tag"
                  label={t}
                  style={tagChipStyle(tagColors[t])}
                  onRemove={() => toggleTag(t)}
                  removeAriaLabel={`Remove tag ${t}`}
                />
              </li>
            ))}
            {filter.priorities.map((p) => (
              <li key={p}>
                <FilterChip
                  variant="neutral"
                  label={PRIORITY_LABEL[p]}
                  onRemove={() => togglePriority(p)}
                  removeAriaLabel={`Remove priority ${PRIORITY_LABEL[p]}`}
                />
              </li>
            ))}
            {filter.hideCompleted && (
              <li>
                <FilterChip
                  variant="neutral"
                  label="Completed hidden"
                  onRemove={() => setHideCompleted(false)}
                  removeAriaLabel="Show completed cards"
                />
              </li>
            )}
          </ul>
        )}

        {/* Desktop: inline search input */}
        <div className="relative hidden md:block">
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
            type="text"
            value={filter.query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Escape' && filter.query) {
                e.stopPropagation()
                setQuery('')
              }
            }}
            placeholder="Search cards…"
            aria-label="Search cards"
            className="h-8 w-[220px] rounded-lg border border-[color:var(--header-border)] bg-[color:var(--color-surface)]/60 py-1 pl-7 pr-2 text-xs text-[color:var(--color-text-primary)] placeholder-[color:var(--color-text-muted)] focus:border-[color:var(--accent-500)] focus:outline-none focus:ring-2 focus:ring-[color:var(--sb-focus-ring)]"
          />
        </div>

        {/* Mobile: search icon opens drawer with search focused */}
        <button
          type="button"
          aria-label="Search cards"
          onClick={onOpenSearch}
          className="flex h-8 w-8 items-center justify-center rounded-lg border border-[color:var(--header-border)] bg-[color:var(--color-surface)]/60 text-[color:var(--color-text-secondary)] transition-colors hover:bg-[color:var(--color-hover)] md:hidden"
        >
          <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden>
            <circle cx="6.5" cy="6.5" r="5" />
            <line x1="10" y1="10" x2="14.5" y2="14.5" strokeLinecap="round" />
          </svg>
        </button>

        <button
          type="button"
          aria-label={count > 0 ? `Filter (${count} active)` : 'Filter'}
          aria-pressed={filterOpen}
          onClick={onOpenFilter}
          className={`group flex h-8 items-center gap-1.5 rounded-lg border px-2.5 text-xs font-medium leading-none transition-colors ${
            filterOpen || count > 0
              ? 'border-[color:var(--accent-500)]/40 bg-[color:var(--accent-500)]/10 text-[color:var(--accent-600)] dark:text-[color:var(--accent-500)]'
              : 'border-[color:var(--header-border)] bg-[color:var(--color-surface)]/60 text-[color:var(--color-text-secondary)] hover:bg-[color:var(--color-hover)]'
          }`}
        >
          <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
            <path d="M2 3h12M4 8h8M6 13h4" />
          </svg>
          <span>Filter</span>
          {count > 0 && (
            <span className="ml-0.5 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-[color:var(--accent-500)] px-1 text-[10px] font-semibold text-white">
              {count}
            </span>
          )}
        </button>
      </div>
    </header>
  )
}
