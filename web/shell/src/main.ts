import { Broker } from '../../shared/src/broker.js'
import { LocalAdapter } from '../../shared/src/adapters/local.js'
import { ServerAdapter } from '../../shared/src/adapters/server.js'
import { BrowserStorage } from '../../shared/src/adapters/local-storage-driver.js'
import { shellTransport } from '../../shared/src/transports/post-message.js'
import type { BackendAdapter } from '../../shared/src/adapter.js'

interface LiveboardConfig {
  adapter: 'local' | 'server'
  baseUrl?: string
}

const SHELL_VERSION = '0.0.1'

function readConfig(): LiveboardConfig {
  const raw = (window as unknown as { __LIVEBOARD_CONFIG__?: LiveboardConfig }).__LIVEBOARD_CONFIG__
  if (raw && (raw.adapter === 'local' || raw.adapter === 'server')) return raw
  return { adapter: 'local' }
}

function makeAdapter(cfg: LiveboardConfig): BackendAdapter {
  if (cfg.adapter === 'server') {
    return new ServerAdapter({ baseUrl: cfg.baseUrl ?? '/api/v1' })
  }
  return new LocalAdapter(new BrowserStorage())
}

const BOARD_PATH_RE = /^\/app\/b\/([^/]+?)(?:\/c\/(\d+)-(\d+))?(?:\/|$)/

interface ParsedUrl {
  boardId: string | null
  cardPos: { colIdx: number; cardIdx: number } | null
}

function parseUrl(): ParsedUrl {
  const m = window.location.pathname.match(BOARD_PATH_RE)
  if (!m) return { boardId: null, cardPos: null }
  const boardId = m[1]
  const cardPos = m[2] != null ? { colIdx: Number(m[2]), cardIdx: Number(m[3]) } : null
  return { boardId, cardPos }
}

function buildUrl(boardId: string | null, cardPos?: { colIdx: number; cardIdx: number } | null): string {
  if (!boardId) return '/app/'
  let url = `/app/b/${boardId}`
  if (cardPos) url += `/c/${cardPos.colIdx}-${cardPos.cardIdx}`
  return url
}

function bootstrap(): void {
  const iframe = document.getElementById('renderer') as HTMLIFrameElement | null
  if (!iframe) throw new Error('renderer iframe not found')

  const params = new URLSearchParams(window.location.search)
  const mode = params.get('renderer') ?? 'default'
  iframe.src = mode === 'stub' ? '/app/renderer-stub/' : '/app/renderer/default/'

  const adapter = makeAdapter(readConfig())
  const transport = shellTransport(iframe, window.location.origin)
  const initial = parseUrl()
  const broker = new Broker(transport, adapter, {
    shellVersion: SHELL_VERSION,
    initialBoardId: initial.boardId,
    initialCardPos: initial.cardPos,
  })

  broker.onEvent('active.changed', ({ boardId, cardPos }) => {
    const url = buildUrl(boardId, cardPos)
    if (window.location.pathname !== url) {
      window.history.pushState({ boardId, cardPos }, '', url)
    }
  })

  window.addEventListener('popstate', () => {
    const { boardId, cardPos } = parseUrl()
    broker.emit('active.set', { boardId, cardPos })
  })

  window.addEventListener('beforeunload', () => broker.close())
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', bootstrap)
} else {
  bootstrap()
}
