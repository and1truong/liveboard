export interface IconColor {
  key: string
  label: string
}

export const ICON_COLORS: readonly IconColor[] = [
  { key: 'slate', label: 'Slate' },
  { key: 'red', label: 'Red' },
  { key: 'orange', label: 'Orange' },
  { key: 'amber', label: 'Amber' },
  { key: 'yellow', label: 'Yellow' },
  { key: 'lime', label: 'Lime' },
  { key: 'green', label: 'Green' },
  { key: 'teal', label: 'Teal' },
  { key: 'cyan', label: 'Cyan' },
  { key: 'blue', label: 'Blue' },
  { key: 'indigo', label: 'Indigo' },
  { key: 'violet', label: 'Violet' },
  { key: 'fuchsia', label: 'Fuchsia' },
  { key: 'pink', label: 'Pink' },
]

export const DEFAULT_ICON_COLOR = 'slate'

const VALID = new Set(ICON_COLORS.map((c) => c.key))

export function resolveIconColor(key: string | undefined): string {
  return key && VALID.has(key) ? key : DEFAULT_ICON_COLOR
}
