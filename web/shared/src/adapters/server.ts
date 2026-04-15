import type { Board, BoardSettings, MutationOp } from '../types.js'
import type {
  BackendAdapter,
  BoardSummary,
  BoardUpdateHandler,
  ResolvedSettings,
  Subscription,
  WorkspaceInfo,
} from '../adapter.js'
import { ProtocolError, type ErrorCode } from '../protocol.js'

export interface ServerAdapterOptions {
  baseUrl: string
  fetch?: typeof globalThis.fetch
}

export class ServerAdapter implements BackendAdapter {
  private readonly baseUrl: string
  private readonly fetchFn: typeof globalThis.fetch
  private es: EventSource | null = null
  private readonly perBoard = new Map<string, Set<BoardUpdateHandler>>()
  private readonly listHandlers = new Set<() => void>()

  constructor(opts: ServerAdapterOptions) {
    this.baseUrl = opts.baseUrl.replace(/\/$/, '')
    this.fetchFn = opts.fetch ?? globalThis.fetch.bind(globalThis)
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

  // === BackendAdapter — stubbed; filled in Tasks 2–5 ===
  listBoards(): Promise<BoardSummary[]> {
    return this.getJSON<BoardSummary[]>('/boards')
  }
  createBoard(_name: string): Promise<BoardSummary> { throw new Error('not implemented') }
  renameBoard(_boardId: string, _newName: string): Promise<BoardSummary> { throw new Error('not implemented') }
  deleteBoard(_boardId: string): Promise<void> { throw new Error('not implemented') }
  getBoard(_boardId: string): Promise<Board> { throw new Error('not implemented') }
  mutateBoard(_boardId: string, _clientVersion: number, _op: MutationOp): Promise<Board> { throw new Error('not implemented') }
  getSettings(_boardId: string): Promise<ResolvedSettings> { throw new Error('not implemented') }
  putBoardSettings(_boardId: string, _patch: Partial<BoardSettings>): Promise<void> { throw new Error('not implemented') }
  async getWorkspaceInfo(): Promise<WorkspaceInfo> {
    const raw = await this.getJSON<{ name: string; board_count: number }>('/workspace')
    return { name: raw.name, boardCount: raw.board_count }
  }
  subscribe(_boardId: string, _onUpdate: BoardUpdateHandler): Subscription { throw new Error('not implemented') }
  onBoardListUpdate(_handler: () => void): Subscription { throw new Error('not implemented') }
}
