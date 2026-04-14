import { describe, expect, it } from 'bun:test'
import { PROTOCOL_VERSION, ProtocolError } from './protocol.js'
import type { Request, Response, Event } from './protocol.js'

describe('protocol', () => {
  it('exports a stable version integer', () => {
    expect(PROTOCOL_VERSION).toBe(1)
  })

  it('ProtocolError carries a code', () => {
    const e = new ProtocolError('NOT_FOUND', 'no board')
    expect(e.code).toBe('NOT_FOUND')
    expect(e.message).toBe('no board')
  })

  it('Request discriminator narrows via method', () => {
    const r: Request = { id: 'x', kind: 'request', method: 'board.list' }
    expect(r.method).toBe('board.list')
  })

  it('Response ok=true carries data; ok=false carries error', () => {
    const ok: Response = { id: 'x', kind: 'response', ok: true, data: null }
    const err: Response = {
      id: 'x',
      kind: 'response',
      ok: false,
      error: { code: 'INTERNAL', message: 'boom' },
    }
    expect(ok.ok).toBe(true)
    expect(err.ok).toBe(false)
  })

  it('Event types are distinguishable by type field', () => {
    const e: Event = { kind: 'event', type: 'board.updated', data: { boardId: 'x', version: 1 } }
    expect(e.type).toBe('board.updated')
  })
})
