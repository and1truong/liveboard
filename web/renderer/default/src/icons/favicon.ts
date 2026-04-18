import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { getBoardIcon, isEmojiIcon } from './boardIcons.js'
import { resolveIconColor } from './iconColors.js'

// Mirrors boardIconPalette in internal/templates/board_icons.go
const PALETTE: Record<string, readonly [string, string]> = {
  slate:   ['#e2e8f0', '#334155'],
  red:     ['#fee2e2', '#b91c1c'],
  orange:  ['#ffedd5', '#c2410c'],
  amber:   ['#fef3c7', '#b45309'],
  yellow:  ['#fef9c3', '#a16207'],
  lime:    ['#ecfccb', '#4d7c0f'],
  green:   ['#dcfce7', '#15803d'],
  teal:    ['#ccfbf1', '#0f766e'],
  cyan:    ['#cffafe', '#0e7490'],
  blue:    ['#dbeafe', '#1d4ed8'],
  indigo:  ['#e0e7ff', '#4338ca'],
  violet:  ['#ede9fe', '#6d28d9'],
  fuchsia: ['#fae8ff', '#a21caf'],
  pink:    ['#fce7f3', '#be185d'],
}

export function buildFaviconHref(
  icon: string | undefined | null,
  colorKey: string | undefined | null,
): string | null {
  if (!icon) return null

  if (isEmojiIcon(icon)) {
    return `data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><text y=".9em" font-size="90">${encodeURIComponent(icon)}</text></svg>`
  }

  const Lucide = getBoardIcon(icon)
  if (!Lucide) return null

  const key = resolveIconColor(colorKey ?? undefined)
  const [bg, fg] = PALETTE[key] ?? PALETTE['slate']

  const svgStr = renderToStaticMarkup(
    createElement(Lucide, { size: 16, strokeWidth: 1.75, color: fg }),
  )
  const withBg = svgStr.replace('<svg ', `<svg style="border-radius:3px;background:${bg};" `)
  return `data:image/svg+xml,${encodeURIComponent(withBg)}`
}
