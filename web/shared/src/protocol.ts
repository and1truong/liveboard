// Wire format for iframe ↔ shell postMessage communication.
// Tagged unions — discriminator `kind`. Requests and responses correlate by `id`.

import type { AppSettings, MutationOp, BoardSettings } from './types.js'

export const PROTOCOL_VERSION = 1 as const

// Iframe → Shell
export type Request =
  | { id: string; kind: 'request'; method: 'board.list'; params?: undefined }
  | { id: string; kind: 'request'; method: 'board.get'; params: { boardId: string } }
  | {
      id: string
      kind: 'request'
      method: 'board.mutate'
      params: { boardId: string; clientVersion: number; op: MutationOp }
    }
  | { id: string; kind: 'request'; method: 'workspace.info'; params?: undefined }
  | { id: string; kind: 'request'; method: 'settings.get'; params: { boardId: string } }
  | {
      id: string
      kind: 'request'
      method: 'settings.put'
      params: { boardId: string; patch: Partial<BoardSettings> }
    }
  | { id: string; kind: 'request'; method: 'subscribe'; params: { boardId: string } }
  | { id: string; kind: 'request'; method: 'unsubscribe'; params: { boardId: string } }
  | { id: string; kind: 'request'; method: 'board.create'; params: { name: string } }
  | { id: string; kind: 'request'; method: 'board.rename'; params: { boardId: string; newName: string } }
  | { id: string; kind: 'request'; method: 'board.delete'; params: { boardId: string } }
  | { id: string; kind: 'request'; method: 'search'; params: { query: string; limit?: number } }
  | { id: string; kind: 'request'; method: 'backlinks'; params: { cardId: string } }
  | { id: string; kind: 'request'; method: 'board.listLite'; params?: undefined }
  | { id: string; kind: 'request'; method: 'appSettings.get'; params?: undefined }
  | {
      id: string
      kind: 'request'
      method: 'appSettings.put'
      params: { patch: Partial<AppSettings> }
    }
  | { id: string; kind: 'request'; method: 'board.pin'; params: { boardId: string } }

export type ErrorCode =
  | 'NOT_FOUND'
  | 'OUT_OF_RANGE'
  | 'INVALID'
  | 'ALREADY_EXISTS'
  | 'INTERNAL'
  | 'VERSION_CONFLICT'
  | 'PROTOCOL_UNSUPPORTED'

export type Response =
  | { id: string; kind: 'response'; ok: true; data: unknown }
  | { id: string; kind: 'response'; ok: false; error: { code: ErrorCode; message: string } }

export type Event =
  | { kind: 'event'; type: 'board.updated'; data: { boardId: string; version: number } }
  | { kind: 'event'; type: 'settings.updated'; data: { boardId: string } }
  | { kind: 'event'; type: 'connection.status'; data: { online: boolean } }
  | { kind: 'event'; type: 'board.list.updated'; data: null }
  | { kind: 'event'; type: 'active.changed'; data: { boardId: string | null; cardPos?: { colIdx: number; cardIdx: number } | null; focusedColumn?: string | null } }
  | { kind: 'event'; type: 'active.set'; data: { boardId: string | null; cardPos?: { colIdx: number; cardIdx: number } | null; focusedColumn?: string | null } }
  | { kind: 'event'; type: 'title.changed'; data: { title: string; icon?: string | null } }
  | { kind: 'event'; type: 'key.forward'; data: { key: string; code: string; keyCode: number; charCode: number; metaKey: boolean; ctrlKey: boolean; altKey: boolean; shiftKey: boolean; repeat: boolean } }

export interface Hello {
  kind: 'hello'
  protocols: number[]
  rendererId: string
  rendererVersion: string
}

export interface Welcome {
  kind: 'welcome'
  protocol: number
  shellVersion: string
  capabilities: string[]
  initialBoardId?: string | null
  initialCardPos?: { colIdx: number; cardIdx: number } | null
  initialFocusedColumn?: string | null
}

export interface HandshakeError {
  kind: 'welcome-error'
  error: { code: 'PROTOCOL_UNSUPPORTED'; minSupported: number; maxSupported: number }
}

export type Message = Request | Response | Event | Hello | Welcome | HandshakeError

export class ProtocolError extends Error {
  constructor(public code: ErrorCode, message: string) {
    super(message)
    this.name = 'ProtocolError'
  }
}

export type { Board, BoardSettings, MutationOp } from './types.js'
