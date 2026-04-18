const WEEKDAY_FMT = new Intl.DateTimeFormat(undefined, { weekday: 'short' })
const SAME_YEAR_FMT = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' })
const FULL_FMT = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: 'numeric' })

export interface DueBadge {
  label: string
  overdue: boolean
  soon: boolean
}

function startOfDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate())
}

function parseDue(iso: string): Date | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(iso)
  if (!m) {
    const d = new Date(iso)
    return Number.isNaN(d.getTime()) ? null : startOfDay(d)
  }
  return new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]))
}

export function formatDueBadge(iso: string, now: Date = new Date()): DueBadge | null {
  const due = parseDue(iso)
  if (!due) return null
  const today = startOfDay(now)
  const dayMs = 86_400_000
  const diffDays = Math.round((due.getTime() - today.getTime()) / dayMs)
  const overdue = diffDays < 0
  const soon = diffDays >= 0 && diffDays <= 1

  let label: string
  if (diffDays === 0) label = 'Today'
  else if (diffDays === 1) label = 'Tomorrow'
  else if (diffDays === -1) label = 'Yesterday'
  else if (diffDays > 1 && diffDays < 7) label = WEEKDAY_FMT.format(due)
  else if (due.getFullYear() === today.getFullYear()) label = SAME_YEAR_FMT.format(due)
  else label = FULL_FMT.format(due)

  return { label, overdue, soon }
}
