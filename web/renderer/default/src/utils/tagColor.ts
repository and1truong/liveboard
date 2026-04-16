// Keep the palette in sync with web/js/liveboard.board-settings.js so colors
// look identical across the htmx UI and the shell/renderer.
export const TAG_PALETTE = [
  '#e05252',
  '#d4722c',
  '#c9a227',
  '#4caf76',
  '#45aab5',
  '#4080c4',
  '#8060c4',
  '#c060a0',
  '#607080',
  '#a07040',
] as const

// WCAG relative luminance, matching LB.colorLuminance in web/js/liveboard.core.js.
export function colorLuminance(hex: string): number {
  const r = parseInt(hex.slice(1, 3), 16) / 255
  const g = parseInt(hex.slice(3, 5), 16) / 255
  const b = parseInt(hex.slice(5, 7), 16) / 255
  const toLinear = (c: number): number =>
    c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4)
  return 0.2126 * toLinear(r) + 0.7152 * toLinear(g) + 0.0722 * toLinear(b)
}

export function tagChipStyle(color: string | undefined): React.CSSProperties | undefined {
  if (!color) return undefined
  return {
    backgroundColor: color,
    color: colorLuminance(color) > 0.35 ? '#111' : '#fff',
  }
}
