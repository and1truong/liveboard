import { useFocusedColumn } from '../contexts/FocusedColumnContext.js'

export function FocusExitBar(): JSX.Element | null {
  const { focused, setFocused } = useFocusedColumn()
  if (focused === null) return null
  return (
    <div className="mb-3 flex shrink-0 items-center justify-between rounded-md border border-slate-200 bg-white px-4 py-2 dark:border-slate-700 dark:bg-slate-900">
      <span className="text-sm font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
        Focusing: <span className="text-slate-800 dark:text-slate-100">{focused}</span>
      </span>
      <button
        type="button"
        aria-label="Exit focus mode"
        onClick={() => setFocused(null)}
        className="inline-flex items-center gap-1.5 rounded border border-slate-200 bg-slate-50 px-3 py-1 text-xs font-semibold text-slate-600 transition-colors hover:border-[color:var(--accent-500)] hover:bg-[color:var(--accent-500)] hover:text-white dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300"
      >
        Exit Focus
        <kbd className="rounded border border-current px-1 py-0.5 font-sans text-[10px] opacity-60">Esc</kbd>
      </button>
    </div>
  )
}
