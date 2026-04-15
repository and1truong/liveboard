import { Client } from '../../../shared/src/client.js'
import { iframeTransport } from '../../../shared/src/transports/post-message.js'

const logEl = document.getElementById('log')!

function line(label: string, ok: boolean, detail = ''): void {
  const div = document.createElement('div')
  div.className = 'line'
  div.innerHTML = `<span class="${ok ? 'ok' : 'fail'}">${ok ? 'OK  ' : 'FAIL'}</span> — ${label} ${detail}`
  logEl.appendChild(div)
}

async function run(): Promise<void> {
  const transport = iframeTransport(window.location.origin)
  const client = new Client(transport, { rendererId: 'stub', rendererVersion: '0.0.1' })

  try {
    const w = await client.ready()
    line('handshake', true, `protocol=${w.protocol} caps=[${w.capabilities.join(',')}]`)
  } catch (e) {
    line('handshake', false, String(e))
    return
  }

  try {
    const list = await client.listBoards()
    line('board.list', list.length > 0, `${list.length} boards`)
  } catch (e) {
    line('board.list', false, String(e))
  }

  try {
    const ws = await client.workspaceInfo()
    line('workspace.info', true, ws.name)
  } catch (e) {
    line('workspace.info', false, String(e))
  }

  try {
    const b = await client.getBoard('welcome')
    line('board.get', (b.columns?.length ?? 0) > 0, `name=${b.name} v=${b.version}`)
  } catch (e) {
    line('board.get', false, String(e))
  }

  try {
    const s = await client.getSettings('welcome')
    line('settings.get', true, s.view_mode)
  } catch (e) {
    line('settings.get', false, String(e))
  }

  try {
    await client.putBoardSettings('welcome', { card_display_mode: 'compact' })
    line('settings.put', true)
  } catch (e) {
    line('settings.put', false, String(e))
  }

  let observedVersion = -1
  client.on('board.updated', (d) => {
    observedVersion = d.version
  })
  try {
    await client.subscribe('welcome')
    line('subscribe', true)
  } catch (e) {
    line('subscribe', false, String(e))
  }

  try {
    const before = await client.getBoard('welcome')
    const after = await client.mutateBoard(
      'welcome',
      before.version ?? 0,
      { type: 'add_card', column: 'Todo', title: 'stub inserted' },
    )
    line('board.mutate', (after.version ?? 0) > (before.version ?? 0), `v=${after.version}`)
  } catch (e) {
    line('board.mutate', false, String(e))
  }

  await new Promise((r) => setTimeout(r, 50))
  line('event: board.updated received', observedVersion > 0, `v=${observedVersion}`)

  try {
    await client.mutateBoard('welcome', 0, { type: 'add_card', column: 'Todo', title: 'stale' })
    line('board.mutate stale→error', false, 'expected rejection')
  } catch (e) {
    const code = (e as { code?: string }).code
    line('board.mutate stale→error', code === 'VERSION_CONFLICT', `code=${code}`)
  }

  try {
    await client.unsubscribe('welcome')
    line('unsubscribe', true)
  } catch (e) {
    line('unsubscribe', false, String(e))
  }
}

void run()
