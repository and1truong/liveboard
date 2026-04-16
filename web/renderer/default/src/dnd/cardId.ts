export function encodeCardId(colIdx: number, cardIdx: number): string {
  return `card:${colIdx}:${cardIdx}`
}

export function decodeCardId(id: string): { colIdx: number; cardIdx: number } | null {
  const m = /^card:(\d+):(\d+)$/.exec(id)
  if (!m) return null
  return { colIdx: Number(m[1]), cardIdx: Number(m[2]) }
}

export function encodeColumnId(name: string): string {
  return `column:${name}`
}

export function decodeColumnId(id: string): string | null {
  if (!id.startsWith('column:')) return null
  return id.slice('column:'.length)
}

export function encodeColumnEndId(name: string): string {
  return `colend:${name}`
}

export function decodeColumnEndId(id: string): string | null {
  if (!id.startsWith('colend:')) return null
  return id.slice('colend:'.length)
}
