export function moveColumnTarget(
  columnNames: string[],
  colIdx: number,
  dir: 'left' | 'right',
): string | null {
  if (dir === 'left') {
    if (colIdx <= 0) return null
    return columnNames[colIdx - 2] ?? ''
  }
  if (colIdx >= columnNames.length - 1) return null
  return columnNames[colIdx + 1]!
}
