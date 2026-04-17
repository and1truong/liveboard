import type { AppSettings, Board, BoardSettings, MutationOp } from '../types.js'
import type {
  BackendAdapter,
  BacklinkHit,
  BoardListLiteEntry,
  BoardSummary,
  BoardUpdateHandler,
  ExportFormat,
  ResolvedSettings,
  SearchHit,
  Subscription,
  WorkspaceInfo,
} from '../adapter.js'
import { ProtocolError, type ErrorCode } from '../protocol.js'

export interface ServerAdapterOptions {
  baseUrl: string
  fetch?: typeof globalThis.fetch
  exportPath?: string
}

export class ServerAdapter implements BackendAdapter {
  private readonly baseUrl: string
  private readonly fetchFn: typeof globalThis.fetch
  private readonly exportPath: string
  private es: EventSource | null = null
  private readonly perBoard = new Map<string, Set<BoardUpdateHandler>>()
  private readonly listHandlers = new Set<() => void>()

  constructor(opts: ServerAdapterOptions) {
    this.baseUrl = opts.baseUrl.replace(/\/$/, '')
    this.fetchFn = opts.fetch ?? globalThis.fetch.bind(globalThis)
    this.exportPath = opts.exportPath ?? '/api/export'
  }

  getExportUrl(format: ExportFormat): string | null {
    const qs = format === 'markdown' ? '?format=md' : '?format=html'
    return this.exportPath + qs
  }

  capabilities(): string[] {
    return ['realtime', 'export:html', 'export:markdown']
  }

  private async request(
    method: string,
    path: string,
    body?: unknown,
  ): Promise<Response> {
    let res: Response
    try {
      res = await this.fetchFn(`${this.baseUrl}${path}`, {
        method,
        headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
        body: body !== undefined ? JSON.stringify(body) : undefined,
      })
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      throw new ProtocolError('INTERNAL', msg)
    }
    if (!res.ok) throw await this.decodeError(res)
    return res
  }

  private async decodeError(res: Response): Promise<ProtocolError> {
    let code: ErrorCode = 'INTERNAL'
    let message = `${res.status} ${res.statusText}`
    try {
      const body = (await res.json()) as { error?: { code?: string; message?: string } }
      if (body.error?.code) code = body.error.code as ErrorCode
      if (body.error?.message) message = body.error.message
    } catch {
      // non-JSON body — keep defaults
    }
    return new ProtocolError(code, message)
  }

  private async getJSON<T>(path: string): Promise<T> {
    const res = await this.request('GET', path)
    return (await res.json()) as T
  }

  private async postJSON<T>(path: string, body: unknown): Promise<T> {
    const res = await this.request('POST', path, body)
    return (await res.json()) as T
  }

  private async patchJSON<T>(path: string, body: unknown): Promise<T> {
    const res = await this.request('PATCH', path, body)
    return (await res.json()) as T
  }

  private async putEmpty(path: string, body: unknown): Promise<void> {
    await this.request('PUT', path, body)
  }

  private async deleteEmpty(path: string): Promise<void> {
    await this.request('DELETE', path)
  }

  // Board ids may contain a single "/" (folder/name). The server routes
  // per-board endpoints under action-prefixed catch-alls (see v1/router.go).
  private boardPath(id: string): string {
    return id.split('/').map(encodeURIComponent).join('/')
  }

  listBoards(): Promise<BoardSummary[]> {
    return this.getJSON<BoardSummary[]>('/boards')
  }
  listBoardsLite(): Promise<BoardListLiteEntry[]> {
    return this.getJSON<BoardListLiteEntry[]>('/boards/list-lite')
  }
  createBoard(name: string, folder?: string): Promise<BoardSummary> {
    return this.postJSON<BoardSummary>('/boards', { name, folder })
  }
  renameBoard(boardId: string, newName: string, folder?: string): Promise<BoardSummary> {
    const body: { new_name: string; folder?: string } = { new_name: newName }
    if (folder !== undefined) body.folder = folder
    return this.patchJSON<BoardSummary>(`/boards/board/${this.boardPath(boardId)}`, body)
  }
  deleteBoard(boardId: string): Promise<void> {
    return this.deleteEmpty(`/boards/board/${this.boardPath(boardId)}`)
  }
  async togglePin(boardId: string): Promise<void> {
    await this.request('POST', `/boards/pin/${this.boardPath(boardId)}`)
  }
  getBoard(boardId: string): Promise<Board> {
    return this.getJSON<Board>(`/boards/board/${this.boardPath(boardId)}`)
  }
  mutateBoard(boardId: string, clientVersion: number, op: MutationOp): Promise<Board> {
    return this.postJSON<Board>(
      `/boards/mutate/${this.boardPath(boardId)}`,
      { client_version: clientVersion, op },
    )
  }
  getSettings(boardId: string): Promise<ResolvedSettings> {
    return this.getJSON<ResolvedSettings>(`/boards/settings/${this.boardPath(boardId)}`)
  }
  putBoardSettings(boardId: string, patch: Partial<BoardSettings>): Promise<void> {
    return this.putEmpty(`/boards/settings/${this.boardPath(boardId)}`, patch)
  }
  listFolders(): Promise<string[]> {
    return this.getJSON<string[]>('/boards/folders')
  }
  async createFolder(name: string): Promise<void> {
    await this.request('POST', '/boards/folders', { name })
  }
  async renameFolder(oldName: string, newName: string): Promise<void> {
    await this.request('PATCH', `/boards/folders/${encodeURIComponent(oldName)}`, { new_name: newName })
  }
  async deleteFolder(name: string): Promise<void> {
    await this.request('DELETE', `/boards/folders/${encodeURIComponent(name)}`)
  }
  getAppSettings(): Promise<AppSettings> {
    return this.getJSON<AppSettings>('/settings')
  }
  putAppSettings(patch: Partial<AppSettings>): Promise<void> {
    return this.putEmpty('/settings', patch)
  }
  async search(query: string, limit = 20): Promise<SearchHit[]> {
    const params = new URLSearchParams({ q: query, limit: String(limit) })
    const raw = await this.getJSON<Array<{
      board_id: string
      board_name: string
      col_idx: number
      card_idx: number
      card_id: string
      card_title: string
      snippet: string
    }>>(`/search?${params}`)
    return raw.map((d) => ({
      boardId: d.board_id,
      boardName: d.board_name,
      colIdx: d.col_idx,
      cardIdx: d.card_idx,
      cardId: d.card_id,
      cardTitle: d.card_title,
      snippet: d.snippet,
    }))
  }

  async backlinks(cardId: string): Promise<BacklinkHit[]> {
    if (!cardId) return []
    const raw = await this.getJSON<Array<{
      board_id: string
      board_name: string
      col_idx: number
      card_idx: number
      card_title: string
    }>>(`/cards/${encodeURIComponent(cardId)}/backlinks`)
    return raw.map((d) => ({
      boardId: d.board_id,
      boardName: d.board_name,
      colIdx: d.col_idx,
      cardIdx: d.card_idx,
      cardTitle: d.card_title,
    }))
  }
  async getWorkspaceInfo(): Promise<WorkspaceInfo> {
    const raw = await this.getJSON<{ name: string; board_count: number }>('/workspace')
    return { name: raw.name, boardCount: raw.board_count }
  }
  subscribe(boardId: string, onUpdate: BoardUpdateHandler): Subscription {
    let set = this.perBoard.get(boardId)
    if (!set) {
      set = new Set()
      this.perBoard.set(boardId, set)
    }
    set.add(onUpdate)
    this.ensureEventSource()
    return {
      close: () => {
        const s = this.perBoard.get(boardId)
        if (!s) return
        s.delete(onUpdate)
        if (s.size === 0) this.perBoard.delete(boardId)
        this.closeIfIdle()
      },
    }
  }

  onBoardListUpdate(handler: () => void): Subscription {
    this.listHandlers.add(handler)
    this.ensureEventSource()
    return {
      close: () => {
        this.listHandlers.delete(handler)
        this.closeIfIdle()
      },
    }
  }

  private ensureEventSource(): void {
    if (this.es) return
    if (typeof EventSource === 'undefined') return // Test env / SSR — handlers stay registered but never fire.
    const es = new EventSource(`${this.baseUrl}/events`)
    es.addEventListener('board.updated', (ev) => {
      try {
        const data = JSON.parse((ev as MessageEvent).data) as { board_id: string; version: number }
        const set = this.perBoard.get(data.board_id)
        if (set) for (const h of set) h({ boardId: data.board_id, version: data.version })
      } catch {
        // ignore malformed payload
      }
    })
    es.addEventListener('board.list.updated', () => {
      for (const h of this.listHandlers) h()
    })
    this.es = es
  }

  private closeIfIdle(): void {
    if (this.perBoard.size === 0 && this.listHandlers.size === 0 && this.es) {
      this.es.close()
      this.es = null
    }
  }
}
