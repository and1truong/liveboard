import { useState } from 'react'
import type { Board } from '@shared/types.js'
import { useBoardFilter } from '../contexts/BoardFilterContext.js'
import { tagChipStyle } from '../utils/tagColor.js'
import { activeFilterCount } from '../utils/cardFilter.js'
import { FilterChip } from './FilterChip.js'
import { FilterPopover } from './FilterPopover.js'

export function BoardHeader({
  data,
  availableTags,
  onToggleSidebar,
  onOpenSettings,
}: {
  data: Board
  availableTags: string[]
  onToggleSidebar: () => void
  onOpenSettings: () => void
}): JSX.Element {
  const { filter, setQuery, toggleTag, setHideCompleted } = useBoardFilter()
  const [open, setOpen] = useState(false)
  const [initialFocus, setInitialFocus] = useState<'search' | null>(null)
  const tagColors = data.tag_colors ?? {}
  const count = activeFilterCount(filter)
  const trimmedQuery = filter.query.trim()

  const openWith = (focus: 'search' | null): void => {
    setInitialFocus(focus)
    setOpen(true)
  }

  return (
    <header className="relative flex h-12 shrink-0 items-center gap-2 border-b border-[color:var(--header-border)] bg-[color:var(--header-bg)] px-3 backdrop-blur-md">
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
        {data.icon && (
          <span aria-hidden className="select-none text-[15px] leading-none">
            {data.icon}
          </span>
        )}
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
          <ul className="flex min-w-0 items-center gap-1 overflow-x-auto pr-1">
            {trimmedQuery && (
              <li>
                <FilterChip
                  variant="accent"
                  label={
                    <button
                      type="button"
                      onClick={() => openWith('search')}
                      className="cursor-pointer bg-transparent p-0 text-current"
                    >
                      Search: <span className="font-semibold">{trimmedQuery}</span>
                    </button>
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

        <FilterPopover
          board={data}
          availableTags={availableTags}
          open={open}
          onOpenChange={(next) => {
            setOpen(next)
            if (!next) setInitialFocus(null)
          }}
          initialFocus={initialFocus}
        />
      </div>
    </header>
  )
}
