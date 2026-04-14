import type {
  BackendAdapter,
  BoardSummary,
  BoardUpdateHandler,
  ResolvedSettings,
  Subscription,
  WorkspaceInfo,
} from '../adapter.js'
import type { Board, BoardSettings, MutationOp } from '../types.js'
import { OpError } from '../types.js'
import { ProtocolError } from '../protocol.js'
import { applyOp } from '../boardOps.js'
import type { StorageDriver } from './local-storage-driver.js'
import { WELCOME_BOARD, WORKSPACE_NAME } from './local-seed.js'

const KEY_PREFIX = 'liveboard:v1:'
const boardKey = (id: string): string => `${KEY_PREFIX}board:${id}`
const workspaceKey = (): string => `${KEY_PREFIX}workspace`

interface StoredWorkspace {
  name: string
  boardIds: string[]
}

export class LocalAdapter implements BackendAdapter {
  constructor(private readonly storage: StorageDriver) {
    this.seedIfEmpty()
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
    return ws.boardIds.map((id) => {
      const b = this.loadBoard(id)
      return {
        id,
        name: b.name ?? id,
        icon: b.icon,
        version: b.version ?? 0,
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

  private publishUpdate(_boardId: string, _version: number): void {
    // BroadcastChannel wiring lands in Task 6.
  }

  async getSettings(_boardId: string): Promise<ResolvedSettings> {
    throw new ProtocolError('INTERNAL', 'getSettings not yet implemented')
  }

  async putBoardSettings(_boardId: string, _patch: Partial<BoardSettings>): Promise<void> {
    throw new ProtocolError('INTERNAL', 'putBoardSettings not yet implemented')
  }

  subscribe(_boardId: string, _onUpdate: BoardUpdateHandler): Subscription {
    return { close: () => {} }
  }
}
