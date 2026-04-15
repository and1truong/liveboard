import type { BackendAdapter, Subscription } from './adapter.js'
import type { Message, Request, Response, ErrorCode } from './protocol.js'
import { PROTOCOL_VERSION } from './protocol.js'
import type { Transport } from './transport.js'

export interface BrokerOptions {
  shellVersion: string
  capabilities?: string[]
}

export class Broker {
  private readonly subs = new Map<string, Subscription>()
  private readonly boardListSub: Subscription

  constructor(
    private readonly transport: Transport,
    private readonly adapter: BackendAdapter,
    private readonly opts: BrokerOptions,
  ) {
    this.transport.onMessage((m) => {
      void this.route(m)
    })
    this.boardListSub = this.adapter.onBoardListUpdate(() => {
      this.transport.send({ kind: 'event', type: 'board.list.updated', data: null })
    })
  }

  private async route(msg: Message): Promise<void> {
    if (msg.kind === 'hello') {
      if (!msg.protocols.includes(PROTOCOL_VERSION)) {
        this.transport.send({
          kind: 'welcome-error',
          error: {
            code: 'PROTOCOL_UNSUPPORTED',
            minSupported: PROTOCOL_VERSION,
            maxSupported: PROTOCOL_VERSION,
          },
        })
        return
      }
      this.transport.send({
        kind: 'welcome',
        protocol: PROTOCOL_VERSION,
        shellVersion: this.opts.shellVersion,
        capabilities: this.opts.capabilities ?? ['local-storage', 'realtime'],
      })
      return
    }
    if (msg.kind !== 'request') return

    try {
      const data = await this.handle(msg)
      const resp: Response = { id: msg.id, kind: 'response', ok: true, data }
      this.transport.send(resp)
    } catch (e) {
      const rawCode =
        e && typeof e === 'object' && 'code' in e && typeof (e as { code?: unknown }).code === 'string'
          ? (e as { code: string }).code
          : 'INTERNAL'
      const message = e instanceof Error ? e.message : String(e)
      this.transport.send({
        id: msg.id,
        kind: 'response',
        ok: false,
        error: { code: rawCode as ErrorCode, message },
      })
    }
  }

  private async handle(req: Request): Promise<unknown> {
    switch (req.method) {
      case 'board.list':
        return this.adapter.listBoards()
      case 'board.get':
        return this.adapter.getBoard(req.params.boardId)
      case 'board.mutate':
        return this.adapter.mutateBoard(
          req.params.boardId,
          req.params.clientVersion,
          req.params.op,
        )
      case 'workspace.info':
        return this.adapter.getWorkspaceInfo()
      case 'settings.get':
        return this.adapter.getSettings(req.params.boardId)
      case 'settings.put':
        await this.adapter.putBoardSettings(req.params.boardId, req.params.patch)
        return null
      case 'subscribe': {
        const { boardId } = req.params
        this.subs.get(boardId)?.close()
        const sub = this.adapter.subscribe(boardId, ({ version }) => {
          this.transport.send({
            kind: 'event',
            type: 'board.updated',
            data: { boardId, version },
          })
        })
        this.subs.set(boardId, sub)
        return null
      }
      case 'unsubscribe':
        this.subs.get(req.params.boardId)?.close()
        this.subs.delete(req.params.boardId)
        return null
      case 'board.create':
        return this.adapter.createBoard(req.params.name)
      case 'board.rename':
        return this.adapter.renameBoard(req.params.boardId, req.params.newName)
      case 'board.delete':
        await this.adapter.deleteBoard(req.params.boardId)
        return null
    }
  }

  close(): void {
    this.boardListSub.close()
    for (const s of this.subs.values()) s.close()
    this.subs.clear()
    this.transport.close()
  }
}
