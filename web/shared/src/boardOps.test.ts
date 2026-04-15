import { describe, expect, it, afterEach } from 'bun:test'
import { readFileSync, readdirSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { applyOp } from './boardOps.js'
import type { Board, MutationOp, ErrorCode } from './types.js'
import { OpError } from './types.js'
import { _setGenerator, _resetGenerator } from './util/cardid.js'

interface Vector {
  name: string
  description?: string
  board_before: Board
  op: MutationOp
  board_after?: Board
  expected_error?: ErrorCode
}

// Vectors live at repo root /testdata/mutations. Tests run from web/shared/.
const vectorDir = resolve(process.cwd(), '..', '..', 'testdata', 'mutations')

// Recursively drop null map values and null array elements so that null /
// missing / [] normalize to the same shape across runners.
function stripNulls(v: unknown): unknown {
  if (v === null || v === undefined) return undefined
  if (Array.isArray(v)) {
    return v.filter((x) => x !== null && x !== undefined).map(stripNulls)
  }
  if (typeof v === 'object') {
    const out: Record<string, unknown> = {}
    for (const [k, val] of Object.entries(v as Record<string, unknown>)) {
      if (k === 'id') continue
      if (val === null || val === undefined) continue
      const stripped = stripNulls(val)
      if (Array.isArray(stripped) && stripped.length === 0) continue
      out[k] = stripped
    }
    return out
  }
  return v
}
const vectorFiles = readdirSync(vectorDir).filter((f) => f.endsWith('.json'))

describe('mutation vectors', () => {
  if (vectorFiles.length === 0) {
    it('finds vectors', () => {
      throw new Error(`no vectors in ${vectorDir}`)
    })
    return
  }

  for (const file of vectorFiles) {
    it(file, () => {
      const raw = readFileSync(join(vectorDir, file), 'utf8')
      const vec: Vector = JSON.parse(raw)

      if (vec.expected_error) {
        let thrown: unknown
        try {
          applyOp(vec.board_before, vec.op)
        } catch (e) {
          thrown = e
        }
        expect(thrown).toBeInstanceOf(OpError)
        expect((thrown as OpError).code).toBe(vec.expected_error)
        return
      }

      const got = applyOp(vec.board_before, vec.op)
      expect(stripNulls(JSON.parse(JSON.stringify(got)))).toEqual(
        stripNulls(JSON.parse(JSON.stringify(vec.board_after))),
      )
    })
  }
})

describe('applyOp card id assignment', () => {
  afterEach(() => _resetGenerator())

  it('add_card assigns id', () => {
    _setGenerator(() => 'OPTID00001')
    const b: Board = { columns: [{ name: 'Todo', cards: [] }] }
    const out = applyOp(b, { type: 'add_card', column: 'Todo', title: 'x' })
    expect(out.columns![0]!.cards[0]!.id).toBe('OPTID00001')
  })

  it('edit_card assigns id to id-less card', () => {
    _setGenerator(() => 'OPTID00002')
    const b: Board = { columns: [{ name: 'Todo', cards: [{ title: 'x' }] }] }
    const out = applyOp(b, {
      type: 'edit_card', col_idx: 0, card_idx: 0,
      title: 'x', body: '', tags: [], priority: '', due: '', assignee: '',
    })
    expect(out.columns![0]!.cards[0]!.id).toBe('OPTID00002')
  })

  it('preserves existing id on edit_card', () => {
    _setGenerator(() => 'SHOULD_NOT_USE')
    const b: Board = { columns: [{ name: 'Todo', cards: [{ id: 'KEEPME1234', title: 'x' }] }] }
    const out = applyOp(b, {
      type: 'edit_card', col_idx: 0, card_idx: 0,
      title: 'x', body: '', tags: [], priority: '', due: '', assignee: '',
    })
    expect(out.columns![0]!.cards[0]!.id).toBe('KEEPME1234')
  })
})
