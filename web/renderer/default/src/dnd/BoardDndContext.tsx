import { useState, type ReactNode } from 'react'
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  TouchSensor,
  KeyboardSensor,
  closestCorners,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import { sortableKeyboardCoordinates } from '@dnd-kit/sortable'
import { useQueryClient } from '@tanstack/react-query'
import type { Board } from '@shared/types.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { Card } from '../components/Card.js'
import { dispatchDrop } from './dispatchDrop.js'
import { decodeCardId, decodeColumnId } from './cardId.js'

export function BoardDndContext({
  boardId,
  children,
}: {
  boardId: string
  children: ReactNode
}): JSX.Element {
  const qc = useQueryClient()
  const mutation = useBoardMutation(boardId)
  const [activeId, setActiveId] = useState<string | null>(null)

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 200, tolerance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

  const onDragStart = (e: DragStartEvent): void => {
    setActiveId(String(e.active.id))
  }

  const onDragEnd = (e: DragEndEvent): void => {
    setActiveId(null)
    const board = qc.getQueryData<Board>(['board', boardId])
    if (!board) return
    const op = dispatchDrop(
      { id: String(e.active.id), data: { current: e.active.data.current } },
      e.over ? { id: String(e.over.id), data: { current: e.over.data.current } } : null,
      board,
    )
    if (op) mutation.mutate(op)
  }

  const overlay = renderOverlay(activeId, qc.getQueryData<Board>(['board', boardId]))

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCorners}
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
      onDragCancel={() => setActiveId(null)}
    >
      {children}
      <DragOverlay>{overlay}</DragOverlay>
    </DndContext>
  )
}

function renderOverlay(activeId: string | null, board: Board | undefined): JSX.Element | null {
  if (!activeId || !board) return null
  const cardId = decodeCardId(activeId)
  if (cardId) {
    const card = board.columns?.[cardId.colIdx]?.cards?.[cardId.cardIdx]
    if (!card) return null
    return (
      <div className="w-72">
        <Card card={card} />
      </div>
    )
  }
  const columnName = decodeColumnId(activeId)
  if (columnName) {
    const col = board.columns?.find((c) => c.name === columnName)
    if (!col) return null
    return (
      <div className="w-72 rounded-lg bg-slate-100 p-3 shadow-lg">
        <div className="text-sm font-semibold text-slate-800">{col.name}</div>
        <div className="text-xs text-slate-500">{col.cards?.length ?? 0} cards</div>
      </div>
    )
  }
  return null
}
