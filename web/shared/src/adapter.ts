import type { Board, BoardSettings, MutationOp } from './types.js'

export interface BoardSummary {
  id: string
  name: string
  icon?: string
  version: number
}

export interface WorkspaceInfo {
  name: string
  boardCount: number
}

// Mirrors internal/web.ResolvedSettings — concrete (non-nullable) values.
export interface ResolvedSettings {
  show_checkbox: boolean
  card_position: string
  expand_columns: boolean
  view_mode: string
  card_display_mode: string
  week_start: string
}

export interface Subscription {
  close(): void
}

export type BoardUpdateHandler = (payload: { boardId: string; version: number }) => void

export interface BackendAdapter {
  listBoards(): Promise<BoardSummary[]>
  getWorkspaceInfo(): Promise<WorkspaceInfo>
  getBoard(boardId: string): Promise<Board>
  mutateBoard(boardId: string, clientVersion: number, op: MutationOp): Promise<Board>
  getSettings(boardId: string): Promise<ResolvedSettings>
  putBoardSettings(boardId: string, patch: Partial<BoardSettings>): Promise<void>
  subscribe(boardId: string, onUpdate: BoardUpdateHandler): Subscription
}
