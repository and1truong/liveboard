// AUTO-GENERATED FROM internal/board/mutation.go.
// Run `make codegen` to regenerate. Do not edit by hand.

import type { BoardSettings } from './types.js'

export interface AddCardOp {
  type: 'add_card'
  column: string
  title: string
  prepend?: boolean
}

export interface AddColumnOp {
  type: 'add_column'
  name: string
}

export interface CompleteCardOp {
  type: 'complete_card'
  col_idx: number
  card_idx: number
}

export interface DeleteCardOp {
  type: 'delete_card'
  col_idx: number
  card_idx: number
}

export interface DeleteColumnOp {
  type: 'delete_column'
  name: string
}

export interface EditCardOp {
  type: 'edit_card'
  col_idx: number
  card_idx: number
  title: string
  body: string
  tags: string[]
  links: string[]
  priority: string
  due: string
  assignee: string
}

export interface MoveCardOp {
  type: 'move_card'
  col_idx: number
  card_idx: number
  target_column: string
}

export interface MoveCardToBoardOp {
  type: 'move_card_to_board'
  col_idx: number
  card_idx: number
  dst_board: string
  dst_column: string
}

export interface MoveColumnOp {
  type: 'move_column'
  name: string
  after_col: string
}

export interface RenameColumnOp {
  type: 'rename_column'
  old_name: string
  new_name: string
}

export interface ReorderCardOp {
  type: 'reorder_card'
  col_idx: number
  card_idx: number
  before_idx: number
  target_column: string
}

export interface SortColumnOp {
  type: 'sort_column'
  col_idx: number
  sort_by: string
}

export interface TagCardOp {
  type: 'tag_card'
  col_idx: number
  card_idx: number
  tags: string[]
}

export interface ToggleColumnCollapseOp {
  type: 'toggle_column_collapse'
  col_idx: number
}

export interface UpdateBoardIconOp {
  type: 'update_board_icon'
  icon?: string | null
  icon_color?: string | null
}

export interface UpdateBoardMembersOp {
  type: 'update_board_members'
  members: string[]
}

export interface UpdateBoardMetaOp {
  type: 'update_board_meta'
  name: string
  description: string
}

export interface UpdateBoardSettingsOp {
  type: 'update_board_settings'
  settings: BoardSettings
}

export type MutationOp =
  | AddCardOp
  | AddColumnOp
  | CompleteCardOp
  | DeleteCardOp
  | DeleteColumnOp
  | EditCardOp
  | MoveCardOp
  | MoveCardToBoardOp
  | MoveColumnOp
  | RenameColumnOp
  | ReorderCardOp
  | SortColumnOp
  | TagCardOp
  | ToggleColumnCollapseOp
  | UpdateBoardIconOp
  | UpdateBoardMembersOp
  | UpdateBoardMetaOp
  | UpdateBoardSettingsOp
