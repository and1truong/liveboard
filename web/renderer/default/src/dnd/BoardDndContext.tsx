import { createContext, useContext, useState, type ReactNode } from 'react'
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
  type DragOverEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import { sortableKeyboardCoordinates } from '@dnd-kit/sortable'
import { useQueryClient } from '@tanstack/react-query'
import type { Board } from '@shared/types.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { Card } from '../components/Card.js'
import { dispatchDrop } from './dispatchDrop.js'
import { decodeCardId, decodeColumnId, decodeColumnEndId } from './cardId.js'

export type DropTarget =
  | { type: 'card'; colIdx: number; cardIdx: number }
  | { type: 'column'; colIdx: number }
  | null

interface DragState { drop: DropTarget; isDragActive: boolean }
const DragStateContext = createContext<DragState>({ drop: null, isDragActive: false })
export function useDragState(): DragState { return useContext(DragStateContext) }

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
  const [drop, setDrop] = useState<DropTarget>(null)

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 200, tolerance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

  const onDragStart = (e: DragStartEvent): void => {
    setActiveId(String(e.active.id))
  }

  const onDragOver = (e: DragOverEvent): void => {
    // Only show drop indicator when dragging a card; column reorder uses sortable animation only.
    const activeType = e.active.data.current?.type
    if (activeType !== 'card' || !e.over) { setDrop(null); return }
    const overId = String(e.over.id)
    const card = decodeCardId(overId)
    if (card) { setDrop({ type: 'card', colIdx: card.colIdx, cardIdx: card.cardIdx }); return }
    const endName = decodeColumnEndId(overId)
    if (endName) {
      const board = qc.getQueryData<Board>(['board', boardId])
      const idx = board?.columns?.findIndex((c) => c.name === endName) ?? -1
      setDrop(idx >= 0 ? { type: 'column', colIdx: idx } : null)
      return
    }
    const colName = decodeColumnId(overId)
    if (colName) {
      const board = qc.getQueryData<Board>(['board', boardId])
      const idx = board?.columns?.findIndex((c) => c.name === colName) ?? -1
      setDrop(idx >= 0 ? { type: 'column', colIdx: idx } : null)
      return
    }
    setDrop(null)
  }

  const onDragEnd = (e: DragEndEvent): void => {
    setActiveId(null)
    setDrop(null)
    const board = qc.getQueryData<Board>(['board', boardId])
    if (!board) return
    const op = dispatchDrop(
      { id: String(e.active.id), data: { current: e.active.data.current } },
      e.over ? { id: String(e.over.id), data: { current: e.over.data.current } } : null,
      board,
    )
    if (op) mutation.mutate(op)
  }

  const board = qc.getQueryData<Board>(['board', boardId])
  const overlay = renderOverlay(activeId, board)

  return (
    <DragStateContext.Provider value={{ drop, isDragActive: activeId != null }}>
      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        onDragStart={onDragStart}
        onDragOver={onDragOver}
        onDragEnd={onDragEnd}
        onDragCancel={() => { setActiveId(null); setDrop(null) }}
      >
        {children}
        <DragOverlay>{overlay}</DragOverlay>
      </DndContext>
    </DragStateContext.Provider>
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
        <Card card={card} tagColors={board.tag_colors} />
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
