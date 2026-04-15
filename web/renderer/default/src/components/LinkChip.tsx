import { useResolveLink } from '../queries/useResolveLink.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { useBoardFocus } from '../contexts/BoardFocusContext.js'

export function LinkChip({ target, onRemove }: { target: string; onRemove: () => void }): JSX.Element {
  const resolved = useResolveLink(target)
  const { setActive } = useActiveBoard()
  const { setFocused } = useBoardFocus()

  const navigate = (): void => {
    if (!resolved) return
    const idx = target.indexOf(':')
    const boardSlug = target.slice(0, idx)
    setActive(boardSlug)
    Promise.resolve().then(() => setFocused({ colIdx: resolved.colIdx, cardIdx: resolved.cardIdx }))
  }

  return (
    <li className="flex items-center gap-1 rounded bg-slate-100 dark:bg-slate-700 px-2 py-1 text-xs">
      <button type="button" onClick={navigate} className="flex-1 text-left">
        {resolved
          ? <span><span className="text-slate-500 dark:text-slate-400">{resolved.boardName} · </span>{resolved.cardTitle}</span>
          : <span className="italic text-slate-400">{target} (missing)</span>}
      </button>
      <button type="button" aria-label="remove link" onClick={onRemove}
        className="text-slate-400 hover:text-red-500">✕</button>
    </li>
  )
}
