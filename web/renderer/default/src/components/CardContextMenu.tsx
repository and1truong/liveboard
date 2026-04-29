import { type ReactNode } from 'react'
import * as ContextMenu from '@radix-ui/react-context-menu'
import {
  Pencil,
  Maximize2,
  CheckSquare,
  Square,
  ArrowRightLeft,
  FolderOutput,
  Trash2,
} from 'lucide-react'
import type { Card as CardModel } from '@shared/types.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { stageDelete } from '../mutations/undoable.js'
import { MoveToBoardSubmenu } from './MoveToBoardSubmenu.js'

const itemCls =
  'cursor-pointer rounded px-2 py-1 text-sm outline-none flex items-center gap-2 hover:bg-slate-100 dark:hover:bg-slate-700 data-[disabled]:cursor-not-allowed data-[disabled]:text-slate-300 dark:data-[disabled]:text-slate-600'

const iconCls = 'shrink-0 text-[color:var(--color-text-muted)]'

export function CardContextMenu({
  card,
  colIdx,
  cardIdx,
  boardId,
  allColumnNames,
  onQuickEdit,
  onOpenDetail,
  children,
}: {
  card: CardModel
  colIdx: number
  cardIdx: number
  boardId: string
  allColumnNames: string[]
  onQuickEdit: () => void
  onOpenDetail: () => void
  children: ReactNode
}): JSX.Element {
  const mutation = useBoardMutation(boardId)
  const otherColumns = allColumnNames.filter((_, i) => i !== colIdx)

  return (
    <ContextMenu.Root>
      <ContextMenu.Trigger asChild>{children}</ContextMenu.Trigger>
      <ContextMenu.Portal>
        <ContextMenu.Content
          className="z-50 min-w-44 rounded-md bg-white p-1 shadow-lg ring-1 ring-slate-200 dark:bg-slate-800 dark:ring-slate-700 dark:text-slate-100"
        >
          <ContextMenu.Item className={itemCls} onSelect={onQuickEdit}>
            <Pencil size={15} className={iconCls} aria-hidden />
            Quick edit
          </ContextMenu.Item>
          <ContextMenu.Item className={itemCls} onSelect={onOpenDetail}>
            <Maximize2 size={15} className={iconCls} aria-hidden />
            Open details
          </ContextMenu.Item>
          <ContextMenu.Item
            className={itemCls}
            onSelect={() =>
              mutation.mutate({ type: 'complete_card', col_idx: colIdx, card_idx: cardIdx })
            }
          >
            {card.completed ? (
              <Square size={15} className={iconCls} aria-hidden />
            ) : (
              <CheckSquare size={15} className={iconCls} aria-hidden />
            )}
            {card.completed ? 'Mark incomplete' : 'Mark complete'}
          </ContextMenu.Item>

          <ContextMenu.Separator className="my-1 h-px bg-slate-200 dark:bg-slate-700" />

          <ContextMenu.Sub>
            <ContextMenu.SubTrigger
              disabled={otherColumns.length === 0}
              className={itemCls + ' justify-between'}
            >
              <span className="flex items-center gap-2">
                <ArrowRightLeft size={15} className={iconCls} aria-hidden />
                Move to column
              </span>
              <span aria-hidden>▸</span>
            </ContextMenu.SubTrigger>
            <ContextMenu.Portal>
              <ContextMenu.SubContent
                className="z-50 min-w-40 rounded-md bg-white p-1 shadow-lg ring-1 ring-slate-200 dark:bg-slate-800 dark:ring-slate-700 dark:text-slate-100"
              >
                {otherColumns.map((name) => (
                  <ContextMenu.Item
                    key={name}
                    className={itemCls}
                    onSelect={() =>
                      mutation.mutate({
                        type: 'move_card',
                        col_idx: colIdx,
                        card_idx: cardIdx,
                        target_column: name,
                      })
                    }
                  >
                    {name}
                  </ContextMenu.Item>
                ))}
              </ContextMenu.SubContent>
            </ContextMenu.Portal>
          </ContextMenu.Sub>

          <MoveToBoardSubmenu
            srcBoardId={boardId}
            colIdx={colIdx}
            cardIdx={cardIdx}
            triggerCls={itemCls + ' justify-between'}
            contentCls="z-50 min-w-40 rounded-md bg-white p-1 shadow-lg ring-1 ring-slate-200 dark:bg-slate-800 dark:ring-slate-700 dark:text-slate-100"
            itemCls={itemCls}
            triggerIcon={<FolderOutput size={15} className={iconCls} aria-hidden />}
          />

          <ContextMenu.Separator className="my-1 h-px bg-slate-200 dark:bg-slate-700" />

          <ContextMenu.Item
            className={itemCls + ' text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950'}
            onSelect={() =>
              stageDelete(
                () => mutation.mutate({ type: 'delete_card', col_idx: colIdx, card_idx: cardIdx }),
                card.title,
              )
            }
          >
            <Trash2 size={15} className="shrink-0" aria-hidden />
            Delete
          </ContextMenu.Item>
        </ContextMenu.Content>
      </ContextMenu.Portal>
    </ContextMenu.Root>
  )
}
