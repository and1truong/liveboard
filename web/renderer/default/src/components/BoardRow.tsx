import type { BoardSummary } from '@shared/adapter.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { BoardIconPicker } from './BoardIconPicker.js'

export function BoardRow({ board }: { board: BoardSummary }): JSX.Element {
  const { active, setActive } = useActiveBoard()
  const isActive = active === board.id
  const hasSub = Boolean(board.updatedAgo) || Boolean(board.tags?.length)

  return (
    <li className={`lb-row${isActive ? ' lb-row--active' : ''}`}>
      <BoardIconPicker boardId={board.id} icon={board.icon} />
      <button
        type="button"
        onClick={() => setActive(board.id)}
        className="lb-row__body"
      >
        <span className="lb-row__label">{board.name}</span>
        {hasSub && (
          <span className="lb-row__sub">
            {board.updatedAgo && (
              <span className="lb-row__meta">{board.updatedAgo}</span>
            )}
            {board.tags?.length ? (
              <span className="lb-row__tags">
                {board.tags.map((t) => (
                  <span key={t} className="lb-row__tag">
                    {t}
                  </span>
                ))}
              </span>
            ) : null}
          </span>
        )}
      </button>
    </li>
  )
}
