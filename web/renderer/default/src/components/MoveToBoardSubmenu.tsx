import * as ContextMenu from '@radix-ui/react-context-menu'
import { useBoardsLite } from '../queries/useBoardsLite.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'

export function MoveToBoardSubmenu({
  srcBoardId,
  colIdx,
  cardIdx,
  triggerCls,
  contentCls,
  itemCls,
}: {
  srcBoardId: string
  colIdx: number
  cardIdx: number
  triggerCls: string
  contentCls: string
  itemCls: string
}): JSX.Element {
  const { data: boards } = useBoardsLite()
  const mutation = useBoardMutation(srcBoardId)
  const others = (boards ?? []).filter((b) => b.slug !== srcBoardId)

  return (
    <ContextMenu.Sub>
      <ContextMenu.SubTrigger disabled={others.length === 0} className={triggerCls}>
        Move to board<span aria-hidden>▸</span>
      </ContextMenu.SubTrigger>
      <ContextMenu.Portal>
        <ContextMenu.SubContent className={contentCls}>
          {others.map((b) => (
            <ContextMenu.Sub key={b.slug}>
              <ContextMenu.SubTrigger
                disabled={b.columns.length === 0}
                className={triggerCls}
              >
                {b.name}<span aria-hidden>▸</span>
              </ContextMenu.SubTrigger>
              <ContextMenu.Portal>
                <ContextMenu.SubContent className={contentCls}>
                  {b.columns.map((col) => (
                    <ContextMenu.Item
                      key={col}
                      className={itemCls}
                      onSelect={() =>
                        mutation.mutate({
                          type: 'move_card_to_board',
                          col_idx: colIdx,
                          card_idx: cardIdx,
                          dst_board: b.slug,
                          dst_column: col,
                        })
                      }
                    >
                      {col}
                    </ContextMenu.Item>
                  ))}
                </ContextMenu.SubContent>
              </ContextMenu.Portal>
            </ContextMenu.Sub>
          ))}
        </ContextMenu.SubContent>
      </ContextMenu.Portal>
    </ContextMenu.Sub>
  )
}
