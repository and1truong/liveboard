import type { AppSettings, Board, BoardSettings } from './types.js'
import type { BacklinkHit, BoardListLiteEntry, BoardSummary, ExportFormat, ResolvedSettings, SearchHit, WorkspaceInfo } from './adapter.js'
import type { MutationOp } from './types.js'
import type { Event as ProtoEvent, Message, Request, Welcome } from './protocol.js'
import { ProtocolError, PROTOCOL_VERSION } from './protocol.js'
import type { Transport } from './transport.js'

type EventType = ProtoEvent['type']
type EventHandler<T extends EventType> = (
  data: Extract<ProtoEvent, { type: T }>['data'],
) => void

interface Pending {
  resolve: (data: unknown) => void
  reject: (err: Error) => void
}

export interface ClientOptions {
  rendererId: string
  rendererVersion: string
}

export class Client {
  private readonly pending = new Map<string, Pending>()
  private readonly handlers = new Map<EventType, Set<EventHandler<EventType>>>()
  private nextId = 1
  private welcome: Welcome | null = null
  private welcomePromise: Promise<Welcome>
  private welcomeResolvers!: { check: () => void; reject: (e: Error) => void }

  constructor(private readonly transport: Transport, opts: ClientOptions) {
    this.transport.onMessage((m) => this.handle(m))
    this.welcomePromise = new Promise<Welcome>((resolve, reject) => {
      const timer = setTimeout(() => reject(new ProtocolError('INTERNAL', 'welcome timeout')), 5000)
      const check = (): void => {
        if (this.welcome) {
          clearTimeout(timer)
          resolve(this.welcome)
        }
      }
      this.welcomeResolvers = {
        check,
        reject: (e) => {
          clearTimeout(timer)
          reject(e)
        },
      }
    })
    this.transport.send({
      kind: 'hello',
      protocols: [PROTOCOL_VERSION],
      rendererId: opts.rendererId,
      rendererVersion: opts.rendererVersion,
    })
  }

  ready(): Promise<Welcome> {
    return this.welcomePromise
  }

  // Synchronous snapshot of capabilities negotiated in the welcome handshake.
  // Returns an empty list before the handshake completes — callers that need
  // it up-front should await ready() first.
  capabilities(): string[] {
    return this.welcome?.capabilities ?? []
  }

  hasCapability(name: string): boolean {
    return this.capabilities().includes(name)
  }

  // attachmentsBaseURL returns the prefix the renderer can use to build
  // attachment download URLs locally. Returns null in local mode.
  // Available only after the welcome handshake completes (await ready()).
  attachmentsBaseURL(): string | null {
    return this.welcome?.attachmentsBaseURL ?? null
  }

  // attachmentURL builds a stable download URL. Returns the
  // `attachment:<hash>` sentinel when no base URL is available — caller
  // must resolve via async lookup in that case.
  attachmentURL(att: { h: string; n: string }): string {
    const base = this.attachmentsBaseURL()
    if (!base) return `attachment:${att.h}`
    return `${base}/${att.h}/${encodeURIComponent(att.n)}`
  }

  // attachmentThumbURL appends ?thumb=1 in server mode. In local mode
  // returns the sentinel — caller must resolve via async lookup.
  attachmentThumbURL(att: { h: string; n: string }): string {
    const base = this.attachmentsBaseURL()
    if (!base) return `attachment:${att.h}`
    return `${base}/${att.h}/${encodeURIComponent(att.n)}?thumb=1`
  }

  // uploadAttachment sends the file to the active adapter's storage and
  // returns the descriptor to embed in a subsequent add_attachments mutation.
  uploadAttachment(file: File): Promise<import('./types.js').Attachment> {
    return this.request({ kind: 'request', method: 'attachment.upload', params: { file } })
  }

  private handle(msg: Message): void {
    switch (msg.kind) {
      case 'welcome':
        this.welcome = msg
        this.welcomeResolvers.check()
        return
      case 'welcome-error':
        this.welcomeResolvers.reject(
          new ProtocolError(
            'PROTOCOL_UNSUPPORTED',
            `server supports ${msg.error.minSupported}..${msg.error.maxSupported}`,
          ),
        )
        return
      case 'response': {
        const p = this.pending.get(msg.id)
        if (!p) return
        this.pending.delete(msg.id)
        if (msg.ok) p.resolve(msg.data)
        else p.reject(new ProtocolError(msg.error.code, msg.error.message))
        return
      }
      case 'event': {
        const set = this.handlers.get(msg.type) as Set<EventHandler<typeof msg.type>> | undefined
        if (!set) return
        for (const h of set) h(msg.data)
        return
      }
    }
  }

  on<T extends EventType>(type: T, handler: EventHandler<T>): () => void {
    let set = this.handlers.get(type) as Set<EventHandler<T>> | undefined
    if (!set) {
      set = new Set()
      this.handlers.set(type, set as unknown as Set<EventHandler<EventType>>)
    }
    set.add(handler)
    return () => set!.delete(handler)
  }

  emit<T extends EventType>(type: T, data: Extract<ProtoEvent, { type: T }>['data']): void {
    this.transport.send({ kind: 'event', type, data } as ProtoEvent)
  }

  private request<T>(req: Omit<Request, 'id'>): Promise<T> {
    const id = `r${this.nextId++}`
    return new Promise<T>((resolve, reject) => {
      this.pending.set(id, { resolve: resolve as (d: unknown) => void, reject })
      this.transport.send({ ...req, id } as Request)
    })
  }

  listBoards(): Promise<BoardSummary[]> {
    return this.request({ kind: 'request', method: 'board.list' })
  }
  listBoardsLite(): Promise<BoardListLiteEntry[]> {
    return this.request({ kind: 'request', method: 'board.listLite' })
  }
  getBoard(boardId: string): Promise<Board> {
    return this.request({ kind: 'request', method: 'board.get', params: { boardId } })
  }
  mutateBoard(boardId: string, clientVersion: number, op: MutationOp): Promise<Board> {
    return this.request({
      kind: 'request',
      method: 'board.mutate',
      params: { boardId, clientVersion, op },
    })
  }
  workspaceInfo(): Promise<WorkspaceInfo> {
    return this.request({ kind: 'request', method: 'workspace.info' })
  }
  getSettings(boardId: string): Promise<ResolvedSettings> {
    return this.request({ kind: 'request', method: 'settings.get', params: { boardId } })
  }
  putBoardSettings(boardId: string, patch: Partial<BoardSettings>): Promise<void> {
    return this.request({
      kind: 'request',
      method: 'settings.put',
      params: { boardId, patch },
    })
  }
  getAppSettings(): Promise<AppSettings> {
    return this.request({ kind: 'request', method: 'appSettings.get' })
  }
  putAppSettings(patch: Partial<AppSettings>): Promise<void> {
    return this.request({
      kind: 'request',
      method: 'appSettings.put',
      params: { patch },
    })
  }
  subscribe(boardId: string): Promise<void> {
    return this.request({ kind: 'request', method: 'subscribe', params: { boardId } })
  }
  unsubscribe(boardId: string): Promise<void> {
    return this.request({ kind: 'request', method: 'unsubscribe', params: { boardId } })
  }
  createBoard(name: string, folder?: string): Promise<BoardSummary> {
    return this.request({ kind: 'request', method: 'board.create', params: { name, folder } })
  }
  renameBoard(boardId: string, newName: string, folder?: string): Promise<BoardSummary> {
    return this.request({
      kind: 'request',
      method: 'board.rename',
      params: { boardId, newName, folder },
    })
  }
  deleteBoard(boardId: string): Promise<void> {
    return this.request({ kind: 'request', method: 'board.delete', params: { boardId } })
  }
  togglePin(boardId: string): Promise<void> {
    return this.request({ kind: 'request', method: 'board.pin', params: { boardId } })
  }
  listFolders(): Promise<string[]> {
    return this.request({ kind: 'request', method: 'folder.list' })
  }
  createFolder(name: string): Promise<void> {
    return this.request({ kind: 'request', method: 'folder.create', params: { name } })
  }
  renameFolder(oldName: string, newName: string): Promise<void> {
    return this.request({ kind: 'request', method: 'folder.rename', params: { oldName, newName } })
  }
  deleteFolder(name: string): Promise<void> {
    return this.request({ kind: 'request', method: 'folder.delete', params: { name } })
  }
  search(query: string, limit?: number): Promise<SearchHit[]> {
    return this.request({ kind: 'request', method: 'search', params: { query, limit } })
  }
  backlinks(cardId: string): Promise<BacklinkHit[]> {
    return this.request({ kind: 'request', method: 'backlinks', params: { cardId } })
  }
  getExportUrl(
    format: ExportFormat,
    opts?: { includeAttachments?: boolean },
  ): Promise<{ url: string | null }> {
    return this.request({
      kind: 'request',
      method: 'workspace.exportUrl',
      params: { format, includeAttachments: opts?.includeAttachments },
    })
  }
}
