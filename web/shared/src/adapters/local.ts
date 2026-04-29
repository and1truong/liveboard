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
import type { AppSettings, Attachment, Board, BoardSettings, MutationOp } from '../types.js'
import { OpError } from '../types.js'
import { ProtocolError } from '../protocol.js'
import { applyOp } from '../boardOps.js'
import type { StorageDriver } from './local-storage-driver.js'
import { WELCOME_BOARD, WORKSPACE_NAME } from './local-seed.js'
import { slugify } from '../util/slug.js'
import { putBlob } from './local-attachments.js'

const KEY_PREFIX = 'liveboard:v1:'
const boardKey = (id: string): string => `${KEY_PREFIX}board:${id}`
const workspaceKey = (): string => `${KEY_PREFIX}workspace`

interface StoredWorkspace {
  name: string
  boardIds: string[]
  // Folders that exist in the workspace even if empty. Populated on folder
  // CRUD and kept in sync when boards move/rename.
  folders?: string[]
}

function parseFolder(id: string): { folder: string; name: string } {
  const i = id.lastIndexOf('/')
  if (i < 0) return { folder: '', name: id }
  return { folder: id.slice(0, i), name: id.slice(i + 1) }
}

export class LocalAdapter implements BackendAdapter {
  private readonly channel: BroadcastChannel | null
  private readonly handlers = new Map<string, Set<BoardUpdateHandler>>()
  private readonly boardListHandlers = new Set<() => void>()

  constructor(private readonly storage: StorageDriver, channelName = 'liveboard') {
    this.seedIfEmpty()
    this.channel =
      typeof BroadcastChannel !== 'undefined' ? new BroadcastChannel(channelName) : null
    if (this.channel) {
      this.channel.onmessage = (ev: MessageEvent) => {
        const data = ev.data as { type?: string; boardId?: string; version?: number }
        if (data?.type === 'board.updated' && data.boardId) {
          this.fanOut(data.boardId, data.version ?? 0)
        } else if (data?.type === 'board.list.updated') {
          this.fanOutBoardList()
        }
      }
    }
  }

  private seedIfEmpty(): void {
    if (this.storage.get(workspaceKey()) !== null) return
    const ws: StoredWorkspace = { name: WORKSPACE_NAME, boardIds: ['welcome'] }
    this.storage.set(workspaceKey(), JSON.stringify(ws))
    this.storage.set(boardKey('welcome'), JSON.stringify(WELCOME_BOARD))
  }

  private loadWorkspace(): StoredWorkspace {
    const raw = this.storage.get(workspaceKey())
    if (raw === null) throw new ProtocolError('INTERNAL', 'workspace missing')
    return JSON.parse(raw) as StoredWorkspace
  }

  private loadBoard(id: string): Board {
    const raw = this.storage.get(boardKey(id))
    if (raw === null) throw new ProtocolError('NOT_FOUND', `board ${id}`)
    return JSON.parse(raw) as Board
  }

  async listBoards(): Promise<BoardSummary[]> {
    const ws = this.loadWorkspace()
    const settings = await this.getAppSettings()
    const pins = settings.pinned_boards
    const pinnedIdx = new Map(pins.map((id, i) => [id, i]))

    const summaries: BoardSummary[] = ws.boardIds.map((id) => {
      const b = this.loadBoard(id)
      const { folder } = parseFolder(id)
      return {
        id,
        folder: folder || undefined,
        name: b.name ?? id,
        icon: b.icon,
        version: b.version ?? 0,
        pinned: pinnedIdx.has(id) || undefined,
      }
    })

    summaries.sort((a, b) => {
      const pi = pinnedIdx.get(a.id)
      const pj = pinnedIdx.get(b.id)
      if (pi !== undefined && pj !== undefined) return pi - pj
      if (pi !== undefined) return -1
      if (pj !== undefined) return 1
      const fa = a.folder ?? ''
      const fb = b.folder ?? ''
      if (fa !== fb) return fa.localeCompare(fb)
      return a.name.localeCompare(b.name)
    })

    return summaries
  }

  async listBoardsLite(): Promise<BoardListLiteEntry[]> {
    const ws = this.loadWorkspace()
    return ws.boardIds.map((id) => {
      const b = this.loadBoard(id)
      return {
        slug: id,
        name: b.name ?? id,
        columns: (b.columns ?? []).map((c) => c.name),
      }
    })
  }

  async getWorkspaceInfo(): Promise<WorkspaceInfo> {
    const ws = this.loadWorkspace()
    return { name: ws.name, boardCount: ws.boardIds.length }
  }

  async getBoard(boardId: string): Promise<Board> {
    return this.loadBoard(boardId)
  }

  async mutateBoard(boardId: string, clientVersion: number, op: MutationOp): Promise<Board> {
    const board = this.loadBoard(boardId)
    const currentVersion = board.version ?? 0
    if (clientVersion >= 0 && clientVersion !== currentVersion) {
      throw new ProtocolError('VERSION_CONFLICT', `expected version ${clientVersion}, have ${currentVersion}`)
    }
    try {
      if (op.type === 'move_card_to_board') {
        const src = board.columns?.[op.col_idx]
        const card = src?.cards?.[op.card_idx]
        if (!card) throw new OpError('OUT_OF_RANGE', `card at ${op.col_idx}/${op.card_idx}`)
        const dst = this.loadBoard(op.dst_board)
        const dstCol = (dst.columns ?? []).find((c) => c.name === op.dst_column)
        if (!dstCol) throw new OpError('NOT_FOUND', `target column ${op.dst_column}`)
        dstCol.cards = [card, ...(dstCol.cards ?? [])]
        dst.version = (dst.version ?? 0) + 1
        this.storage.set(boardKey(op.dst_board), JSON.stringify(dst))
        this.publishUpdate(op.dst_board, dst.version)
      }
      const next = applyOp(board, op)
      next.version = currentVersion + 1
      this.storage.set(boardKey(boardId), JSON.stringify(next))
      this.publishUpdate(boardId, next.version)
      return next
    } catch (e) {
      if (e instanceof OpError) throw new ProtocolError(e.code, e.message)
      throw e
    }
  }

  subscribe(boardId: string, onUpdate: BoardUpdateHandler): Subscription {
    let set = this.handlers.get(boardId)
    if (!set) {
      set = new Set()
      this.handlers.set(boardId, set)
    }
    set.add(onUpdate)
    return {
      close: () => {
        this.handlers.get(boardId)?.delete(onUpdate)
      },
    }
  }

  private fanOut(boardId: string, version: number): void {
    const set = this.handlers.get(boardId)
    if (!set) return
    for (const h of set) h({ boardId, version })
  }

  private publishUpdate(boardId: string, version: number): void {
    this.fanOut(boardId, version)
    this.channel?.postMessage({ type: 'board.updated', boardId, version })
  }

  async getSettings(boardId: string): Promise<ResolvedSettings> {
    const board = this.loadBoard(boardId)
    return {
      show_checkbox: board.settings?.show_checkbox ?? true,
      card_position:
        (board.settings?.card_position as ResolvedSettings['card_position']) ?? 'bottom',
      expand_columns: board.settings?.expand_columns ?? false,
      view_mode: (board.settings?.view_mode as ResolvedSettings['view_mode']) ?? 'board',
      card_display_mode:
        (board.settings?.card_display_mode as ResolvedSettings['card_display_mode']) ?? 'normal',
      week_start: (board.settings?.week_start as ResolvedSettings['week_start']) ?? 'monday',
    }
  }

  async putBoardSettings(boardId: string, patch: Partial<BoardSettings>): Promise<void> {
    const board = this.loadBoard(boardId)
    board.settings = { ...(board.settings ?? {}), ...patch }
    board.version = (board.version ?? 0) + 1
    this.storage.set(boardKey(boardId), JSON.stringify(board))
    this.publishUpdate(boardId, board.version)
  }

  async getAppSettings(): Promise<AppSettings> {
    const raw = this.storage.get('liveboard:app_settings')
    const saved = raw ? (JSON.parse(raw) as Partial<AppSettings>) : {}
    return {
      site_name: saved.site_name ?? 'LiveBoard',
      theme: saved.theme ?? 'system',
      color_theme: saved.color_theme ?? 'aqua',
      font_family: saved.font_family ?? 'system',
      column_width: saved.column_width ?? 280,
      sidebar_position: saved.sidebar_position ?? 'left',
      default_columns: saved.default_columns ?? ['not now', 'maybe?', 'done'],
      show_checkbox: saved.show_checkbox ?? true,
      newline_trigger: saved.newline_trigger ?? 'shift-enter',
      card_position: saved.card_position ?? 'append',
      card_display_mode: saved.card_display_mode ?? 'full',
      keyboard_shortcuts: saved.keyboard_shortcuts ?? false,
      week_start: saved.week_start ?? 'sunday',
      pinned_boards: saved.pinned_boards ?? [],
      tags: saved.tags ?? [],
      tag_colors: saved.tag_colors ?? {},
    }
  }

  async putAppSettings(patch: Partial<AppSettings>): Promise<void> {
    const current = await this.getAppSettings()
    this.storage.set('liveboard:app_settings', JSON.stringify({ ...current, ...patch }))
  }

  async togglePin(boardId: string): Promise<void> {
    const settings = await this.getAppSettings()
    const pins = settings.pinned_boards
    const idx = pins.indexOf(boardId)
    const next = idx >= 0 ? pins.filter((_, i) => i !== idx) : [...pins, boardId]
    await this.putAppSettings({ pinned_boards: next })
    this.publishBoardListUpdate()
  }

  onBoardListUpdate(handler: () => void): Subscription {
    this.boardListHandlers.add(handler)
    return {
      close: () => {
        this.boardListHandlers.delete(handler)
      },
    }
  }

  private fanOutBoardList(): void {
    for (const h of this.boardListHandlers) h()
  }

  private publishBoardListUpdate(): void {
    this.fanOutBoardList()
    this.channel?.postMessage({ type: 'board.list.updated' })
  }

  async createBoard(name: string, folder?: string): Promise<BoardSummary> {
    const trimmed = name.trim()
    if (!trimmed) throw new ProtocolError('INVALID', 'name required')
    const slug = slugify(trimmed)
    if (!slug) throw new ProtocolError('INVALID', 'name has no usable characters')
    const normFolder = (folder ?? '').trim().replace(/^\/+|\/+$/g, '')
    const id = normFolder ? `${normFolder}/${slug}` : slug
    const ws = this.loadWorkspace()
    if (ws.boardIds.includes(id)) {
      throw new ProtocolError('ALREADY_EXISTS', `board ${id} exists`)
    }
    const board: Board = {
      name: trimmed,
      version: 1,
      columns: [{ name: 'Todo', cards: [] }],
    }
    this.storage.set(boardKey(id), JSON.stringify(board))
    ws.boardIds.push(id)
    if (normFolder && !(ws.folders ?? []).includes(normFolder)) {
      ws.folders = [...(ws.folders ?? []), normFolder]
    }
    this.storage.set(workspaceKey(), JSON.stringify(ws))
    this.publishBoardListUpdate()
    return { id, folder: normFolder || undefined, name: trimmed, version: 1 }
  }

  async renameBoard(boardId: string, newName: string, folder?: string): Promise<BoardSummary> {
    const trimmed = newName.trim()
    if (!trimmed) throw new ProtocolError('INVALID', 'name required')
    const slug = slugify(trimmed)
    if (!slug) throw new ProtocolError('INVALID', 'name has no usable characters')
    const currentFolder = parseFolder(boardId).folder
    const normFolder =
      folder !== undefined ? folder.trim().replace(/^\/+|\/+$/g, '') : currentFolder
    const newId = normFolder ? `${normFolder}/${slug}` : slug
    const board = this.loadBoard(boardId)
    const ws = this.loadWorkspace()
    if (newId !== boardId && ws.boardIds.includes(newId)) {
      throw new ProtocolError('ALREADY_EXISTS', `board ${newId} exists`)
    }
    board.name = trimmed
    board.version = (board.version ?? 0) + 1
    if (newId === boardId) {
      this.storage.set(boardKey(boardId), JSON.stringify(board))
    } else {
      this.storage.set(boardKey(newId), JSON.stringify(board))
      this.storage.remove(boardKey(boardId))
      const idx = ws.boardIds.indexOf(boardId)
      if (idx >= 0) ws.boardIds[idx] = newId
      if (normFolder && !(ws.folders ?? []).includes(normFolder)) {
        ws.folders = [...(ws.folders ?? []), normFolder]
      }
      this.storage.set(workspaceKey(), JSON.stringify(ws))
    }
    this.publishBoardListUpdate()
    return { id: newId, folder: normFolder || undefined, name: trimmed, version: board.version }
  }

  async listFolders(): Promise<string[]> {
    const ws = this.loadWorkspace()
    // Union of registered folders and any folder present in a boardId.
    const set = new Set<string>(ws.folders ?? [])
    for (const id of ws.boardIds) {
      const { folder } = parseFolder(id)
      if (folder) set.add(folder)
    }
    return [...set].sort()
  }

  async createFolder(name: string): Promise<void> {
    const trimmed = name.trim()
    if (!trimmed) throw new ProtocolError('INVALID', 'name required')
    if (trimmed.includes('/')) throw new ProtocolError('INVALID', 'folder name cannot contain /')
    const ws = this.loadWorkspace()
    const existing = await this.listFolders()
    if (existing.includes(trimmed)) {
      throw new ProtocolError('ALREADY_EXISTS', `folder ${trimmed} exists`)
    }
    ws.folders = [...(ws.folders ?? []), trimmed]
    this.storage.set(workspaceKey(), JSON.stringify(ws))
    this.publishBoardListUpdate()
  }

  async renameFolder(oldName: string, newName: string): Promise<void> {
    const from = oldName.trim()
    const to = newName.trim()
    if (!from || !to) throw new ProtocolError('INVALID', 'names required')
    if (to.includes('/')) throw new ProtocolError('INVALID', 'folder name cannot contain /')
    const ws = this.loadWorkspace()
    const prefix = from + '/'
    const folders = new Set(ws.folders ?? [])
    if (!folders.has(from) && !ws.boardIds.some((id) => id.startsWith(prefix))) {
      throw new ProtocolError('NOT_FOUND', `folder ${from}`)
    }
    const existingFolders = await this.listFolders()
    if (existingFolders.includes(to)) {
      throw new ProtocolError('ALREADY_EXISTS', `folder ${to} exists`)
    }
    // Rewrite every board id.
    ws.boardIds = ws.boardIds.map((id) => {
      if (!id.startsWith(prefix)) return id
      const next = to + '/' + id.slice(prefix.length)
      const raw = this.storage.get(boardKey(id))
      if (raw !== null) {
        this.storage.set(boardKey(next), raw)
        this.storage.remove(boardKey(id))
      }
      return next
    })
    folders.delete(from)
    folders.add(to)
    ws.folders = [...folders]
    this.storage.set(workspaceKey(), JSON.stringify(ws))
    // Rewrite pin ids pointing into the old folder.
    const settings = await this.getAppSettings()
    const pins = settings.pinned_boards.map((p) =>
      p.startsWith(prefix) ? to + '/' + p.slice(prefix.length) : p,
    )
    await this.putAppSettings({ pinned_boards: pins })
    this.publishBoardListUpdate()
  }

  async deleteFolder(name: string): Promise<void> {
    const trimmed = name.trim()
    if (!trimmed) throw new ProtocolError('INVALID', 'name required')
    const ws = this.loadWorkspace()
    const prefix = trimmed + '/'
    if (ws.boardIds.some((id) => id.startsWith(prefix))) {
      throw new ProtocolError('INVALID', 'folder not empty')
    }
    ws.folders = (ws.folders ?? []).filter((f) => f !== trimmed)
    this.storage.set(workspaceKey(), JSON.stringify(ws))
    this.publishBoardListUpdate()
  }

  async deleteBoard(boardId: string): Promise<void> {
    this.loadBoard(boardId)
    this.storage.remove(boardKey(boardId))
    const ws = this.loadWorkspace()
    ws.boardIds = ws.boardIds.filter((x) => x !== boardId)
    this.storage.set(workspaceKey(), JSON.stringify(ws))
    this.publishBoardListUpdate()
  }

  async search(query: string, limit = 20): Promise<SearchHit[]> {
    const q = query.trim().toLowerCase()
    if (!q) return []
    const ws = this.loadWorkspace()
    const hits: SearchHit[] = []
    for (const id of ws.boardIds) {
      const board = this.loadBoard(id)
      const boardName = board.name ?? id
      const cols = board.columns ?? []
      for (let colIdx = 0; colIdx < cols.length; colIdx++) {
        const cards = cols[colIdx]?.cards ?? []
        for (let cardIdx = 0; cardIdx < cards.length; cardIdx++) {
          const c = cards[cardIdx]!
          const haystack = `${c.title ?? ''} ${c.body ?? ''} ${(c.tags ?? []).join(' ')}`.toLowerCase()
          if (haystack.includes(q)) {
            hits.push({
              boardId: id,
              boardName,
              colIdx,
              cardIdx,
              cardId: c.id ?? '',
              cardTitle: c.title ?? '',
              snippet: c.title ?? '',
            })
            if (hits.length >= limit) return hits
          }
        }
      }
    }
    return hits
  }

  getExportUrl(_format: ExportFormat): string | null {
    return null
  }

  capabilities(): string[] {
    return ['local-storage', 'realtime']
  }

  attachmentsBaseURL(): string | null {
    return null
  }

  async uploadAttachment(file: File): Promise<Attachment> {
    return putBlob(file, file.name)
  }

  attachmentURL(att: Pick<Attachment, 'h' | 'n'>): string {
    // Returns a sentinel URL that the renderer's body-markdown rewrite
    // resolves to a `blob:` URL via getBlob() at view time. A direct
    // synchronous URL is impossible because IndexedDB lookups are async.
    return `attachment:${att.h}`
  }

  async backlinks(cardId: string): Promise<BacklinkHit[]> {
    if (!cardId) return []
    const ws = this.loadWorkspace()
    const target = ':' + cardId
    const out: BacklinkHit[] = []
    for (const id of ws.boardIds) {
      const board = this.loadBoard(id)
      const cols = board.columns ?? []
      for (let c = 0; c < cols.length; c++) {
        const cards = cols[c]?.cards ?? []
        for (let k = 0; k < cards.length; k++) {
          const links = cards[k]!.links ?? []
          if (links.some((l) => l.endsWith(target))) {
            out.push({
              boardId: id,
              boardName: board.name ?? id,
              colIdx: c,
              cardIdx: k,
              cardTitle: cards[k]!.title ?? '',
            })
          }
        }
      }
    }
    return out
  }
}
