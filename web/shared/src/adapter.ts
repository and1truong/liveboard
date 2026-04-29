import type { AppSettings, Attachment, Board, BoardSettings, MutationOp } from './types.js'

export interface BoardSummary {
  id: string
  // folder is the first path segment of `id` for boards nested one level deep,
  // or empty/undefined for root-level boards. The server populates it; the
  // sidebar groups by this field.
  folder?: string
  name: string
  description?: string
  icon?: string
  icon_color?: string
  version: number
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
  // createBoard / renameBoard accept an optional `folder` so boards can be
  // created or moved into a first-level subdirectory. Empty / undefined means
  // workspace root.
  createBoard(name: string, folder?: string): Promise<BoardSummary>
  renameBoard(boardId: string, newName: string, folder?: string): Promise<BoardSummary>
  deleteBoard(boardId: string): Promise<void>
  togglePin(boardId: string): Promise<void>
  listFolders(): Promise<string[]>
  createFolder(name: string): Promise<void>
  renameFolder(oldName: string, newName: string): Promise<void>
  deleteFolder(name: string): Promise<void>
  onBoardListUpdate(handler: () => void): Subscription
  search(query: string, limit?: number): Promise<SearchHit[]>
  backlinks(cardId: string): Promise<BacklinkHit[]>
  // Absolute/same-origin URL that, when navigated to, downloads the workspace as a ZIP.
  // Returns null when the adapter has no backing server to produce an export.
  getExportUrl(format: ExportFormat, opts?: { includeAttachments?: boolean }): string | null
  // Feature flags the adapter supports. Surfaced through the welcome handshake
  // so the renderer can enable/disable UI affordances up front. Known values:
  //   'local-storage', 'realtime', 'export:html', 'export:markdown'
  capabilities(): string[]

  // attachmentsBaseURL returns the URL prefix the renderer can use to
  // construct download URLs locally (e.g. "/api/v1/attachments"). Returns
  // null when this adapter has no URL backing (LocalAdapter).
  attachmentsBaseURL(): string | null
  // uploadAttachment stores the bytes (HTTP for server, IndexedDB for local)
  // and returns the descriptor to embed in a subsequent add_attachments mutation.
  uploadAttachment(file: File): Promise<Attachment>
  // attachmentURL resolves a descriptor to a URL the renderer can pass to
  // <a href> / <img src>. ServerAdapter returns an HTTP path; LocalAdapter
  // returns an `attachment:<hash>` sentinel that the renderer's body-markdown
  // rewrite plugin resolves to a blob: URL on demand.
  attachmentURL(att: Pick<Attachment, 'h' | 'n'>): string
  // Optional thumb URL. ServerAdapter appends `?thumb=1`; LocalAdapter may
  // omit (renderer falls back to attachmentURL).
  attachmentThumbURL?(att: Pick<Attachment, 'h' | 'n'>): string
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
