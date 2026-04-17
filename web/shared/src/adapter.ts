import type { AppSettings, Board, BoardSettings, MutationOp } from './types.js'

export interface BoardSummary {
  id: string
  name: string
  description?: string
  icon?: string
  version: number
  tags?: string[]
  updatedAgo?: string
  cardCount?: number
  doneCount?: number
  pinned?: boolean
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

export type ExportFormat = 'html' | 'markdown'

export interface BoardListLiteEntry {
  slug: string
  name: string
  columns: string[]
}

export interface BackendAdapter {
  listBoards(): Promise<BoardSummary[]>
  listBoardsLite(): Promise<BoardListLiteEntry[]>
  getWorkspaceInfo(): Promise<WorkspaceInfo>
  getBoard(boardId: string): Promise<Board>
  mutateBoard(boardId: string, clientVersion: number, op: MutationOp): Promise<Board>
  getSettings(boardId: string): Promise<ResolvedSettings>
  putBoardSettings(boardId: string, patch: Partial<BoardSettings>): Promise<void>
  getAppSettings(): Promise<AppSettings>
  putAppSettings(patch: Partial<AppSettings>): Promise<void>
  subscribe(boardId: string, onUpdate: BoardUpdateHandler): Subscription
  createBoard(name: string): Promise<BoardSummary>
  renameBoard(boardId: string, newName: string): Promise<BoardSummary>
  deleteBoard(boardId: string): Promise<void>
  togglePin(boardId: string): Promise<void>
  onBoardListUpdate(handler: () => void): Subscription
  search(query: string, limit?: number): Promise<SearchHit[]>
  backlinks(cardId: string): Promise<BacklinkHit[]>
  // Absolute/same-origin URL that, when navigated to, downloads the workspace as a ZIP.
  // Returns null when the adapter has no backing server to produce an export.
  getExportUrl(format: ExportFormat): string | null
  // Feature flags the adapter supports. Surfaced through the welcome handshake
  // so the renderer can enable/disable UI affordances up front. Known values:
  //   'local-storage', 'realtime', 'export:html', 'export:markdown'
  capabilities(): string[]
}

export interface SearchHit {
  boardId: string
  boardName: string
  colIdx: number
  cardIdx: number
  cardId: string
  cardTitle: string
  snippet: string
}

export interface BacklinkHit {
  boardId: string
  boardName: string
  colIdx: number
  cardIdx: number
  cardTitle: string
}
