import { describe, expect, it } from 'bun:test'
import { readFileSync, readdirSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { applyOp } from './boardOps.js'
import type { Board, MutationOp, ErrorCode } from './types.js'
import { OpError } from './types.js'

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
      expect(JSON.parse(JSON.stringify(got))).toEqual(
        JSON.parse(JSON.stringify(vec.board_after)),
      )
    })
  }
})
