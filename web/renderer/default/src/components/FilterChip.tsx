import type { CSSProperties, ReactNode } from 'react'

export function FilterChip({
  label,
  onRemove,
  style,
  variant = 'neutral',
  removeAriaLabel,
}: {
  label: ReactNode
  onRemove: () => void
  style?: CSSProperties
  variant?: 'neutral' | 'tag' | 'accent'
  removeAriaLabel?: string
}): JSX.Element {
  const base =
    'group inline-flex h-6 items-center gap-1 rounded-full pl-2 pr-1 text-xs font-medium leading-none transition-colors'
  const variantClass =
    variant === 'tag'
      ? '' // colors come from inline style
      : variant === 'accent'
        ? 'bg-[color:var(--accent-500)]/12 text-[color:var(--accent-600)] dark:bg-[color:var(--accent-500)]/20 dark:text-[color:var(--accent-500)]'
        : 'bg-[color:var(--color-hover)] text-[color:var(--color-text-secondary)]'

  return (
    <span className={`${base} ${variantClass}`} style={style}>
      <span className="truncate max-w-[10rem]">{label}</span>
      <button
        type="button"
        onClick={onRemove}
        aria-label={removeAriaLabel ?? 'Remove filter'}
        className="flex h-4 w-4 shrink-0 items-center justify-center rounded-full text-current opacity-60 transition-opacity hover:bg-black/10 hover:opacity-100 dark:hover:bg-white/15"
      >
        <svg width="8" height="8" viewBox="0 0 8 8" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" aria-hidden>
          <path d="M1.5 1.5l5 5M6.5 1.5l-5 5" />
        </svg>
      </button>
    </span>
  )
}
