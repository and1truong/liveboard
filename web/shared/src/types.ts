// Hand-maintained mirror of pkg/models/models.go and internal/api/v1/mutations.go.
// Field names MUST match Go JSON tags. Vector tests catch drift.

export interface Board {
  version?: number
  name?: string
  description?: string
  icon?: string
  tags?: string[]
  tag_colors?: Record<string, string>
  members?: string[]
  list_collapse?: boolean[]
  settings?: BoardSettings
  columns?: Column[]
  file_path?: string
  created_at?: string
  updated_at?: string
}

export interface Column {
  name: string
  cards: Card[]
  collapsed?: boolean
}

export interface Card {
  title: string
  completed?: boolean
  no_checkbox?: boolean
  tags?: string[]
  inline_tags?: string[]
  assignee?: string
  priority?: string
  due?: string
  metadata?: Record<string, string>
  body?: string
}

export interface BoardSettings {
  show_checkbox?: boolean | null
  card_position?: string | null
  expand_columns?: boolean | null
  view_mode?: string | null
  card_display_mode?: string | null
  week_start?: string | null
}

// Tagged union — discriminator is `type`.
export type MutationOp =
  | { type: 'add_card'; column: string; title: string; prepend?: boolean }
  | { type: 'move_card'; col_idx: number; card_idx: number; target_column: string }
  | {
      type: 'reorder_card'
      col_idx: number
      card_idx: number
      before_idx: number
      target_column: string
    }
  | {
      type: 'edit_card'
      col_idx: number
      card_idx: number
      title: string
      body: string
      tags: string[]
      priority: string
      due: string
      assignee: string
    }
  | { type: 'delete_card'; col_idx: number; card_idx: number }
  | { type: 'complete_card'; col_idx: number; card_idx: number }
  | { type: 'tag_card'; col_idx: number; card_idx: number; tags: string[] }
  | { type: 'add_column'; name: string }
  | { type: 'rename_column'; old_name: string; new_name: string }
  | { type: 'delete_column'; name: string }
  | { type: 'move_column'; name: string; after_col: string }
  | { type: 'sort_column'; col_idx: number; sort_by: string }
  | { type: 'toggle_column_collapse'; col_idx: number }
  | { type: 'update_board_meta'; name: string; description: string; tags: string[] }
  | { type: 'update_board_members'; members: string[] }
  | { type: 'update_board_icon'; icon: string }
  | { type: 'update_board_settings'; settings: BoardSettings }

// Canonical error codes. Thrown by applyOp as OpError instances.
export type ErrorCode = 'NOT_FOUND' | 'OUT_OF_RANGE' | 'INVALID' | 'ALREADY_EXISTS' | 'INTERNAL'

export class OpError extends Error {
  constructor(public code: ErrorCode, message: string) {
    super(message)
    this.name = 'OpError'
  }
}
