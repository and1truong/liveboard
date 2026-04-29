export interface IconColor {
  key: string
  label: string
  /** Saturated hex used for the swatch dot in the picker. */
  swatch: string
}

// Order matches the picker layout: 6 columns × 2 rows.
export const ICON_COLORS: readonly IconColor[] = [
  { key: 'red',     label: 'Red',      swatch: '#ff6b6b' },
  { key: 'orange',  label: 'Orange',   swatch: '#ffa94d' },
  { key: 'yellow',  label: 'Yellow',   swatch: '#ffd43b' },
  { key: 'green',   label: 'Green',    swatch: '#51cf66' },
  { key: 'cyan',    label: 'Sky',      swatch: '#74c0fc' },
  { key: 'blue',    label: 'Blue',     swatch: '#4dabf7' },
  { key: 'indigo',  label: 'Purple',   swatch: '#748ffc' },
  { key: 'fuchsia', label: 'Pink',     swatch: '#f06595' },
  { key: 'violet',  label: 'Lavender', swatch: '#b197fc' },
  { key: 'amber',   label: 'Tan',      swatch: '#d4a574' },
  { key: 'slate',   label: 'Grey',     swatch: '#adb5bd' },
  { key: 'pink',    label: 'Peach',    swatch: '#ffa8a8' },
]

export const DEFAULT_ICON_COLOR = 'slate'

const VALID = new Set(ICON_COLORS.map((c) => c.key))

// Removed keys map to their closest surviving palette entry so existing
// boards keep rendering after the trim.
const ALIASES: Record<string, string> = {
  lime: 'green',
  teal: 'cyan',
}

export function resolveIconColor(key: string | undefined): string {
  if (!key) return DEFAULT_ICON_COLOR
  if (VALID.has(key)) return key
  return ALIASES[key] ?? DEFAULT_ICON_COLOR
}
