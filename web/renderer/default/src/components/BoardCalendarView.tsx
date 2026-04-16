import { Suspense, lazy, useCallback, useMemo, useState, type DragEvent } from 'react'
import type { Board, Card as CardModel, Column } from '@shared/types.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { useBoardSettings } from '../queries/useBoardSettings.js'

const CardDetailModal = lazy(() =>
  import('./CardDetailModal.js').then((m) => ({ default: m.CardDetailModal })),
)

interface FlatCard {
  card: CardModel
  colIdx: number
  cardIdx: number
  columnName: string
}

interface DayCell {
  date: number
  dateStr: string
  isToday: boolean
  isOtherMonth: boolean
  monthLabel?: string
}

const MONTH_NAMES = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
]
const MONTH_NAMES_SHORT = [
  'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
  'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec',
]
const DAY_NAMES = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
const WEEKDAY_SHORT = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa']

const PRIORITY_DOT: Record<string, string> = {
  critical: 'bg-red-500',
  high: 'bg-orange-500',
  medium: 'bg-yellow-500',
  low: 'bg-slate-400',
}

const pad = (n: number): string => (n < 10 ? '0' + n : '' + n)
const fmt = (y: number, m: number, d: number): string => `${y}-${pad(m + 1)}-${pad(d)}`
const todayStr = (): string => {
  const t = new Date()
  return fmt(t.getFullYear(), t.getMonth(), t.getDate())
}

type SubView = 'month' | 'week' | 'day'

export function BoardCalendarView({
  data,
  active,
  columns,
  filterQuery,
  hideCompleted,
}: {
  data: Board
  active: string
  columns: Column[]
  filterQuery: string
  hideCompleted: boolean
}): JSX.Element {
  void data
  const settings = useBoardSettings(active)
  const weekStart: 0 | 1 = settings.week_start === 'sunday' ? 0 : 1

  const now = new Date()
  const [subView, setSubView] = useState<SubView>('month')
  const [viewYear, setViewYear] = useState(now.getFullYear())
  const [viewMonth, setViewMonth] = useState(now.getMonth())
  const [viewDay, setViewDay] = useState(now.getDate())
  const [unschedOpen, setUnschedOpen] = useState(true)

  const { cardsByDate, unscheduled } = useMemo(() => {
    const byDate: Record<string, FlatCard[]> = {}
    const unsched: FlatCard[] = []
    columns.forEach((col, colIdx) => {
      col.cards.forEach((card, cardIdx) => {
        const entry: FlatCard = { card, colIdx, cardIdx, columnName: col.name }
        if (card.due) {
          if (!byDate[card.due]) byDate[card.due] = []
          byDate[card.due].push(entry)
        } else {
          unsched.push(entry)
        }
      })
    })
    return { cardsByDate: byDate, unscheduled: unsched }
  }, [columns])

  const q = filterQuery.trim().toLowerCase()
  const matchesFilter = useCallback(
    (fc: FlatCard): boolean => {
      if (hideCompleted && fc.card.completed) return false
      if (!q) return true
      const hay = [
        fc.card.title,
        fc.card.body ?? '',
        (fc.card.tags ?? []).join(' '),
        fc.card.assignee ?? '',
        fc.columnName,
      ]
        .join(' ')
        .toLowerCase()
      return hay.includes(q)
    },
    [hideCompleted, q],
  )

  const cardsForDate = (dateStr: string): FlatCard[] =>
    (cardsByDate[dateStr] ?? []).filter(matchesFilter)

  const filteredUnscheduled = unscheduled.filter(matchesFilter)

  const weekStartDate = (): Date => {
    const d = new Date(viewYear, viewMonth, viewDay)
    const diff = (d.getDay() - weekStart + 7) % 7
    d.setDate(d.getDate() - diff)
    return d
  }

  const weekdayLabels: string[] = []
  for (let i = 0; i < 7; i++) weekdayLabels.push(WEEKDAY_SHORT[(weekStart + i) % 7])

  const title = ((): string => {
    if (subView === 'month') return `${MONTH_NAMES[viewMonth]} ${viewYear}`
    if (subView === 'day') {
      const d = new Date(viewYear, viewMonth, viewDay)
      return `${DAY_NAMES[d.getDay()]}, ${MONTH_NAMES[viewMonth]} ${viewDay}, ${viewYear}`
    }
    const start = weekStartDate()
    const end = new Date(start.getTime())
    end.setDate(end.getDate() + 6)
    const head = `${MONTH_NAMES_SHORT[start.getMonth()]} ${start.getDate()}`
    const tail =
      start.getMonth() !== end.getMonth() || start.getFullYear() !== end.getFullYear()
        ? `${MONTH_NAMES_SHORT[end.getMonth()]} ${end.getDate()}, ${end.getFullYear()}`
        : `${end.getDate()}, ${end.getFullYear()}`
    return `${head} – ${tail}`
  })()

  const monthDays = ((): DayCell[] => {
    const firstDow = new Date(viewYear, viewMonth, 1).getDay()
    const leading = (firstDow - weekStart + 7) % 7
    const daysInMonth = new Date(viewYear, viewMonth + 1, 0).getDate()
    const prevMonthDays = new Date(viewYear, viewMonth, 0).getDate()
    const today = todayStr()
    const out: DayCell[] = []

    for (let i = leading - 1; i >= 0; i--) {
      const pd = prevMonthDays - i
      let pm = viewMonth - 1
      let py = viewYear
      if (pm < 0) { pm = 11; py-- }
      out.push({ date: pd, dateStr: fmt(py, pm, pd), isToday: false, isOtherMonth: true })
    }
    for (let d = 1; d <= daysInMonth; d++) {
      const ds = fmt(viewYear, viewMonth, d)
      out.push({ date: d, dateStr: ds, isToday: ds === today, isOtherMonth: false })
    }
    // Always emit 6 weeks (42 cells) to keep the grid stable across months.
    const needed = 42 - out.length
    let nm = viewMonth + 1
    let ny = viewYear
    if (nm > 11) { nm = 0; ny++ }
    for (let d = 1; d <= needed; d++) {
      out.push({ date: d, dateStr: fmt(ny, nm, d), isToday: false, isOtherMonth: true })
    }
    return out
  })()

  const weekDays = ((): DayCell[] => {
    const start = weekStartDate()
    const today = todayStr()
    const out: DayCell[] = []
    for (let i = 0; i < 7; i++) {
      const d = new Date(start.getTime())
      d.setDate(d.getDate() + i)
      const ds = fmt(d.getFullYear(), d.getMonth(), d.getDate())
      out.push({
        date: d.getDate(),
        dateStr: ds,
        isToday: ds === today,
        isOtherMonth: d.getMonth() !== viewMonth,
        monthLabel: MONTH_NAMES_SHORT[d.getMonth()],
      })
    }
    return out
  })()

  const dayCell: DayCell = {
    date: viewDay,
    dateStr: fmt(viewYear, viewMonth, viewDay),
    isToday: fmt(viewYear, viewMonth, viewDay) === todayStr(),
    isOtherMonth: false,
  }

  const prev = (): void => {
    if (subView === 'month') {
      const m = viewMonth - 1
      if (m < 0) { setViewMonth(11); setViewYear(viewYear - 1) } else setViewMonth(m)
    } else {
      const delta = subView === 'week' ? -7 : -1
      const d = new Date(viewYear, viewMonth, viewDay + delta)
      setViewYear(d.getFullYear()); setViewMonth(d.getMonth()); setViewDay(d.getDate())
    }
  }
  const next = (): void => {
    if (subView === 'month') {
      const m = viewMonth + 1
      if (m > 11) { setViewMonth(0); setViewYear(viewYear + 1) } else setViewMonth(m)
    } else {
      const delta = subView === 'week' ? 7 : 1
      const d = new Date(viewYear, viewMonth, viewDay + delta)
      setViewYear(d.getFullYear()); setViewMonth(d.getMonth()); setViewDay(d.getDate())
    }
  }
  const goToday = (): void => {
    const t = new Date()
    setViewYear(t.getFullYear()); setViewMonth(t.getMonth()); setViewDay(t.getDate())
  }
  const selectDay = (dateStr: string): void => {
    const [y, m, d] = dateStr.split('-').map((s) => parseInt(s, 10))
    setViewYear(y); setViewMonth(m - 1); setViewDay(d); setSubView('day')
  }

  const mutation = useBoardMutation(active)

  const onDropOnDay = (dateStr: string) => (ev: DragEvent<HTMLElement>): void => {
    ev.preventDefault()
    const raw = ev.dataTransfer.getData('application/x-liveboard-card')
    if (!raw) return
    let payload: { colIdx: number; cardIdx: number }
    try { payload = JSON.parse(raw) } catch { return }
    const col = columns[payload.colIdx]
    const card = col?.cards[payload.cardIdx]
    if (!card) return
    if (card.due === dateStr) return
    mutation.mutate({
      type: 'edit_card',
      col_idx: payload.colIdx,
      card_idx: payload.cardIdx,
      title: card.title,
      body: card.body ?? '',
      tags: card.tags ?? [],
      links: card.links ?? [],
      priority: card.priority ?? '',
      due: dateStr,
      assignee: card.assignee ?? '',
    })
  }

  return (
    <div className="flex flex-1 flex-col overflow-y-auto">
      <Toolbar
        title={title}
        subView={subView}
        onSubViewChange={setSubView}
        onPrev={prev}
        onNext={next}
        onToday={goToday}
      />

      {subView === 'month' && (
        <MonthGrid
          days={monthDays}
          weekdayLabels={weekdayLabels}
          cardsForDate={cardsForDate}
          onSelectDay={selectDay}
          onDropOnDay={onDropOnDay}
          boardId={active}
        />
      )}
      {subView === 'week' && (
        <WeekGrid
          days={weekDays}
          weekdayLabels={weekdayLabels}
          cardsForDate={cardsForDate}
          onSelectDay={selectDay}
          onDropOnDay={onDropOnDay}
          boardId={active}
        />
      )}
      {subView === 'day' && (
        <DayView day={dayCell} cardsForDate={cardsForDate} boardId={active} />
      )}

      <UnscheduledSection
        cards={filteredUnscheduled}
        open={unschedOpen}
        onToggle={() => setUnschedOpen((v) => !v)}
        boardId={active}
      />
    </div>
  )
}

function Toolbar({
  title, subView, onSubViewChange, onPrev, onNext, onToday,
}: {
  title: string
  subView: SubView
  onSubViewChange: (v: SubView) => void
  onPrev: () => void
  onNext: () => void
  onToday: () => void
}): JSX.Element {
  return (
    <div className="flex items-center gap-2 border-b border-[color:var(--color-border)] px-4 py-2">
      <button
        type="button"
        aria-label="previous"
        onClick={onPrev}
        className="flex h-7 w-7 items-center justify-center rounded border border-[color:var(--color-border)] bg-[color:var(--color-surface)] text-sm text-[color:var(--color-text-secondary)] hover:bg-[color:var(--color-column-bg)]"
      >
        ‹
      </button>
      <button
        type="button"
        onClick={onToday}
        className="rounded border border-[color:var(--color-border)] bg-[color:var(--color-surface)] px-2 py-1 text-xs text-[color:var(--color-text-secondary)] hover:bg-[color:var(--color-column-bg)]"
      >
        Today
      </button>
      <button
        type="button"
        aria-label="next"
        onClick={onNext}
        className="flex h-7 w-7 items-center justify-center rounded border border-[color:var(--color-border)] bg-[color:var(--color-surface)] text-sm text-[color:var(--color-text-secondary)] hover:bg-[color:var(--color-column-bg)]"
      >
        ›
      </button>
      <div className="ml-2 text-sm font-semibold text-slate-800 dark:text-slate-100">{title}</div>
      <div
        role="radiogroup"
        aria-label="calendar sub-view"
        className="ml-auto inline-flex rounded border border-[color:var(--color-border)] p-0.5"
      >
        {(['month', 'week', 'day'] as const).map((v) => (
          <button
            key={v}
            type="button"
            role="radio"
            aria-checked={subView === v}
            aria-label={`${v} view`}
            onClick={() => onSubViewChange(v)}
            className={`rounded px-2 py-0.5 text-xs capitalize transition-colors ${
              subView === v
                ? 'bg-[color:var(--accent-500)] text-white'
                : 'text-[color:var(--color-text-secondary)] hover:bg-[color:var(--color-column-bg)]'
            }`}
          >
            {v}
          </button>
        ))}
      </div>
    </div>
  )
}

function MonthGrid({
  days, weekdayLabels, cardsForDate, onSelectDay, onDropOnDay, boardId,
}: {
  days: DayCell[]
  weekdayLabels: string[]
  cardsForDate: (d: string) => FlatCard[]
  onSelectDay: (d: string) => void
  onDropOnDay: (d: string) => (ev: DragEvent<HTMLElement>) => void
  boardId: string
}): JSX.Element {
  return (
    <div className="flex flex-1 flex-col p-2">
      <div className="grid grid-cols-7 border-b border-[color:var(--color-border)] pb-1 text-center text-xs font-medium text-[color:var(--color-text-muted)]">
        {weekdayLabels.map((l, i) => <div key={i}>{l}</div>)}
      </div>
      <div
        role="grid"
        aria-label="calendar month grid"
        className="grid flex-1 grid-cols-7 grid-rows-6"
      >
        {days.map((day) => (
          <DayCellView
            key={day.dateStr}
            day={day}
            cards={cardsForDate(day.dateStr)}
            onSelectDay={onSelectDay}
            onDropOnDay={onDropOnDay}
            boardId={boardId}
            minHeight="min-h-[96px]"
          />
        ))}
      </div>
    </div>
  )
}

function WeekGrid({
  days, weekdayLabels, cardsForDate, onSelectDay, onDropOnDay, boardId,
}: {
  days: DayCell[]
  weekdayLabels: string[]
  cardsForDate: (d: string) => FlatCard[]
  onSelectDay: (d: string) => void
  onDropOnDay: (d: string) => (ev: DragEvent<HTMLElement>) => void
  boardId: string
}): JSX.Element {
  return (
    <div className="flex flex-1 flex-col p-2">
      <div className="grid grid-cols-7 border-b border-[color:var(--color-border)] pb-1 text-center text-xs font-medium text-[color:var(--color-text-muted)]">
        {weekdayLabels.map((l, i) => <div key={i}>{l}</div>)}
      </div>
      <div role="grid" aria-label="calendar week grid" className="grid flex-1 grid-cols-7">
        {days.map((day) => (
          <DayCellView
            key={day.dateStr}
            day={day}
            cards={cardsForDate(day.dateStr)}
            onSelectDay={onSelectDay}
            onDropOnDay={onDropOnDay}
            boardId={boardId}
            minHeight="min-h-[200px]"
            showMonthLabel
          />
        ))}
      </div>
    </div>
  )
}

function DayCellView({
  day, cards, onSelectDay, onDropOnDay, boardId, minHeight, showMonthLabel,
}: {
  day: DayCell
  cards: FlatCard[]
  onSelectDay: (d: string) => void
  onDropOnDay: (d: string) => (ev: DragEvent<HTMLElement>) => void
  boardId: string
  minHeight: string
  showMonthLabel?: boolean
}): JSX.Element {
  const [over, setOver] = useState(false)
  return (
    <div
      role="gridcell"
      aria-label={`day ${day.dateStr}`}
      onDragOver={(e) => { e.preventDefault(); setOver(true) }}
      onDragLeave={() => setOver(false)}
      onDrop={(e) => { setOver(false); onDropOnDay(day.dateStr)(e) }}
      className={`flex flex-col border-b border-r border-[color:var(--color-border)] p-1 ${minHeight} ${
        day.isOtherMonth ? 'opacity-40' : ''
      } ${over ? 'bg-[color:var(--color-column-bg)] outline outline-2 outline-dashed outline-[color:var(--accent-500)]' : ''}`}
    >
      <button
        type="button"
        onClick={() => onSelectDay(day.dateStr)}
        className="flex items-center gap-1 self-start text-xs text-[color:var(--color-text-secondary)] hover:text-[color:var(--color-text-primary)]"
      >
        {showMonthLabel && day.monthLabel && (
          <span className="text-[10px] text-[color:var(--color-text-muted)]">{day.monthLabel}</span>
        )}
        <span
          className={
            day.isToday
              ? 'flex h-5 w-5 items-center justify-center rounded-full bg-[color:var(--accent-500)] text-[11px] font-semibold text-white'
              : 'px-1 font-medium'
          }
        >
          {day.date}
        </span>
      </button>
      <div className="mt-1 flex min-h-0 flex-col gap-0.5 overflow-hidden">
        {cards.map((fc) => (
          <CardChip key={`${fc.colIdx}-${fc.cardIdx}`} fc={fc} boardId={boardId} />
        ))}
      </div>
    </div>
  )
}

function DayView({
  day, cardsForDate, boardId,
}: {
  day: DayCell
  cardsForDate: (d: string) => FlatCard[]
  boardId: string
}): JSX.Element {
  const cards = cardsForDate(day.dateStr)
  return (
    <div className="flex flex-1 flex-col gap-2 p-4">
      {cards.length === 0 ? (
        <div className="text-sm text-[color:var(--color-text-muted)]">No cards scheduled for this day.</div>
      ) : (
        cards.map((fc) => (
          <DayCard key={`${fc.colIdx}-${fc.cardIdx}`} fc={fc} boardId={boardId} />
        ))
      )}
    </div>
  )
}

function CardChip({ fc, boardId }: { fc: FlatCard; boardId: string }): JSX.Element {
  const { activeCard, setActiveCard } = useActiveBoard()
  const open = activeCard?.colIdx === fc.colIdx && activeCard?.cardIdx === fc.cardIdx
  const onOpenChange = useCallback(
    (next: boolean): void => setActiveCard(next ? { colIdx: fc.colIdx, cardIdx: fc.cardIdx } : null),
    [setActiveCard, fc.colIdx, fc.cardIdx],
  )
  const onDragStart = (ev: DragEvent<HTMLButtonElement>): void => {
    ev.dataTransfer.setData(
      'application/x-liveboard-card',
      JSON.stringify({ colIdx: fc.colIdx, cardIdx: fc.cardIdx }),
    )
    ev.dataTransfer.effectAllowed = 'move'
  }
  return (
    <>
      <button
        type="button"
        draggable
        onDragStart={onDragStart}
        onClick={() => onOpenChange(true)}
        aria-label={`card ${fc.card.title}`}
        className={`flex items-center gap-1 rounded bg-[color:var(--color-surface)] px-1.5 py-0.5 text-left text-[11px] text-slate-700 hover:bg-[color:var(--color-column-bg)] dark:text-slate-200 ${
          fc.card.completed ? 'opacity-50 line-through' : ''
        }`}
      >
        {fc.card.priority && (
          <span
            className={`h-1.5 w-1.5 shrink-0 rounded-full ${PRIORITY_DOT[fc.card.priority] ?? PRIORITY_DOT.low}`}
            aria-hidden
          />
        )}
        <span className="min-w-0 flex-1 truncate">{fc.card.title}</span>
        {fc.card.assignee && (
          <span className="shrink-0 text-[10px] text-[color:var(--color-text-muted)]">
            {fc.card.assignee}
          </span>
        )}
      </button>
      <Suspense fallback={null}>
        <CardDetailModal
          card={fc.card}
          colIdx={fc.colIdx}
          cardIdx={fc.cardIdx}
          boardId={boardId}
          open={open}
          onOpenChange={onOpenChange}
        />
      </Suspense>
    </>
  )
}

function DayCard({ fc, boardId }: { fc: FlatCard; boardId: string }): JSX.Element {
  const { activeCard, setActiveCard } = useActiveBoard()
  const open = activeCard?.colIdx === fc.colIdx && activeCard?.cardIdx === fc.cardIdx
  const onOpenChange = useCallback(
    (next: boolean): void => setActiveCard(next ? { colIdx: fc.colIdx, cardIdx: fc.cardIdx } : null),
    [setActiveCard, fc.colIdx, fc.cardIdx],
  )
  return (
    <>
      <button
        type="button"
        onClick={() => onOpenChange(true)}
        aria-label={`card ${fc.card.title}`}
        className={`flex flex-col gap-1 rounded border border-[color:var(--color-border)] bg-[color:var(--color-surface)] p-3 text-left hover:bg-[color:var(--color-column-bg)] ${
          fc.card.completed ? 'opacity-50' : ''
        }`}
      >
        <div className="flex items-center gap-2">
          {fc.card.priority && (
            <span
              className={`h-2 w-2 shrink-0 rounded-full ${PRIORITY_DOT[fc.card.priority] ?? PRIORITY_DOT.low}`}
              aria-hidden
            />
          )}
          <span className="text-sm font-medium text-slate-800 dark:text-slate-100">
            {fc.card.title}
          </span>
        </div>
        <div className="flex flex-wrap gap-2 text-[11px] text-[color:var(--color-text-muted)]">
          <span>in {fc.columnName}</span>
          {fc.card.assignee && <span>· 👤 {fc.card.assignee}</span>}
          {fc.card.priority && <span>· {fc.card.priority}</span>}
        </div>
        {fc.card.body && (
          <div className="line-clamp-3 text-xs text-slate-600 dark:text-slate-300">
            {fc.card.body}
          </div>
        )}
      </button>
      <Suspense fallback={null}>
        <CardDetailModal
          card={fc.card}
          colIdx={fc.colIdx}
          cardIdx={fc.cardIdx}
          boardId={boardId}
          open={open}
          onOpenChange={onOpenChange}
        />
      </Suspense>
    </>
  )
}

function UnscheduledSection({
  cards, open, onToggle, boardId,
}: {
  cards: FlatCard[]
  open: boolean
  onToggle: () => void
  boardId: string
}): JSX.Element {
  return (
    <div className="border-t border-[color:var(--color-border)] px-4 py-2">
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={open}
        aria-label="toggle unscheduled"
        className="flex items-center gap-2 text-xs font-medium text-[color:var(--color-text-secondary)] hover:text-[color:var(--color-text-primary)]"
      >
        <span>{open ? '▾' : '▸'}</span>
        <span>Unscheduled</span>
        <span className="text-[color:var(--color-text-muted)]">({cards.length})</span>
      </button>
      {open && cards.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1">
          {cards.map((fc) => (
            <CardChip key={`${fc.colIdx}-${fc.cardIdx}`} fc={fc} boardId={boardId} />
          ))}
        </div>
      )}
    </div>
  )
}
