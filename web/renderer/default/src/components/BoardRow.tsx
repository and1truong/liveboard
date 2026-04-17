import type { BoardSummary } from '@shared/adapter.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { BoardIconPicker } from './BoardIconPicker.js'
import { useTogglePin } from '../mutations/useBoardCrud.js'

export function BoardRow({ board }: { board: BoardSummary }): JSX.Element {
  const { active, setActive } = useActiveBoard()
  const isActive = active === board.id
  const hasSub = Boolean(board.updatedAgo) || Boolean(board.tags?.length)
  const togglePin = useTogglePin()

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
      <button
        type="button"
        className={`lb-row__pin-btn${board.pinned ? ' lb-row__pin-btn--active' : ''}`}
        title={board.pinned ? 'Unpin' : 'Pin to top'}
        aria-label={board.pinned ? 'Unpin board' : 'Pin board to top'}
        onClick={(e) => { e.stopPropagation(); togglePin.mutate(board.id) }}
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
          <path d="M12 17v5"/>
          <path d="M5 17h14v-1.76a2 2 0 0 0-1.11-1.79l-1.78-.9A2 2 0 0 1 15 10.76V6h1a2 2 0 0 0 0-4H8a2 2 0 0 0 0 4h1v4.76a2 2 0 0 1-1.11 1.79l-1.78.9A2 2 0 0 0 5 15.24z"/>
        </svg>
      </button>
    </li>
  )
}
