// Hand-maintained mirror of pkg/models/models.go and internal/api/v1/mutations.go.
// Field names MUST match Go JSON tags. Vector tests catch drift.

export interface Board {
  version?: number
  name?: string
  description?: string
  icon?: string
  icon_color?: string
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
  id?: string
  title: string
  completed?: boolean
  no_checkbox?: boolean
  tags?: string[]
  inline_tags?: string[]
  links?: string[]
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

// Mirrors internal/web.AppSettings — concrete workspace-level preferences.
export interface AppSettings {
  site_name: string
  theme: string
  color_theme: string
  font_family: string
  column_width: number
  sidebar_position: string
  default_columns: string[]
  show_checkbox: boolean
  newline_trigger: string
  card_position: string
  card_display_mode: string
  keyboard_shortcuts: boolean
  week_start: string
  pinned_boards: string[]
  tags: string[]
  tag_colors: Record<string, string>
  folder_collapse?: Record<string, boolean>
}

// MutationOp is generated from internal/board/mutation.go via
// `make codegen` (cmd/gen-ts-mutations). The discriminator is `type`.
export type { MutationOp } from './mutations.gen.js'

// Canonical error codes. Thrown by applyOp as OpError instances.
export type ErrorCode = 'NOT_FOUND' | 'OUT_OF_RANGE' | 'INVALID' | 'ALREADY_EXISTS' | 'INTERNAL'

export class OpError extends Error {
  constructor(public code: ErrorCode, message: string) {
    super(message)
    this.name = 'OpError'
  }
}
