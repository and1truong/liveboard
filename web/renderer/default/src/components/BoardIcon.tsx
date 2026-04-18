import type { JSX } from 'react'
import { getBoardIcon, isEmojiIcon } from '../icons/boardIcons.js'
import { resolveIconColor } from '../icons/iconColors.js'

type Size = 'sm' | 'md' | 'lg'
type Shape = 'rounded' | 'circle'

const GLYPH_SIZE: Record<Size, number> = { sm: 14, md: 16, lg: 22 }
const PLACEHOLDER = '\u2637' // ☷ — same placeholder HTMX uses for empty boards

export function BoardIcon({
  icon,
  color,
  size = 'md',
  shape = 'rounded',
  className,
  title,
}: {
  icon?: string
  color?: string
  size?: Size
  shape?: Shape
  className?: string
  title?: string
}): JSX.Element {
  const colorKey = resolveIconColor(color)
  const classes = [
    'lb-bicon',
    `lb-bicon--${size}`,
    `lb-bicon--${shape}`,
  ]

  if (!icon) {
    classes.push('lb-bicon--placeholder', 'lb-bicon--emoji')
    return (
      <span className={joinClasses(classes, className)} aria-hidden title={title}>
        {PLACEHOLDER}
      </span>
    )
  }

  if (isEmojiIcon(icon)) {
    classes.push('lb-bicon--emoji')
    return (
      <span className={joinClasses(classes, className)} aria-hidden title={title}>
        {icon}
      </span>
    )
  }

  const Lucide = getBoardIcon(icon)
  if (!Lucide) {
    classes.push('lb-bicon--placeholder', 'lb-bicon--emoji')
    return (
      <span className={joinClasses(classes, className)} aria-hidden title={title}>
        {PLACEHOLDER}
      </span>
    )
  }

  classes.push('lb-bicon--svg', `lb-bicon--c-${colorKey}`)
  return (
    <span className={joinClasses(classes, className)} aria-hidden title={title}>
      <Lucide size={GLYPH_SIZE[size]} strokeWidth={1.75} />
    </span>
  )
}

function joinClasses(base: string[], extra?: string): string {
  return extra ? `${base.join(' ')} ${extra}` : base.join(' ')
}
