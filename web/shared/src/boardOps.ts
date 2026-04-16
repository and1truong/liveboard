import type { Board, Column, Card, MutationOp } from './types.js'
import { OpError } from './types.js'
import { newCardId } from './util/cardid.js'

function ensureCardId(c: Card): void {
  if (!c.id) c.id = newCardId()
}

// applyOp returns a new board with op applied. Input is not mutated.
// Mirrors internal/api/v1.Apply semantics. Shared parity vectors guard drift.
export function applyOp(board: Board, op: MutationOp): Board {
  const b: Board = structuredClone(board)

  switch (op.type) {
    case 'add_card': {
      const col = colByName(b, op.column)
      if (!col) throw new OpError('NOT_FOUND', `column ${op.column}`)
      const card: Card = { title: op.title }
      ensureCardId(card)
      col.cards = op.prepend ? [card, ...(col.cards ?? [])] : [...(col.cards ?? []), card]
      return b
    }
    case 'move_card': {
      const src = colAt(b, op.col_idx)
      cardAt(src, op.card_idx, op.col_idx)
      const card = src.cards[op.card_idx]!
      ensureCardId(card)
      src.cards = src.cards.filter((_, i) => i !== op.card_idx)
      const dst = colByName(b, op.target_column)
      if (!dst) throw new OpError('NOT_FOUND', `target column ${op.target_column}`)
      dst.cards = [...(dst.cards ?? []), card]
      return b
    }
    case 'reorder_card': {
      const src = colAt(b, op.col_idx)
      cardAt(src, op.card_idx, op.col_idx)
      const card = src.cards[op.card_idx]!
      ensureCardId(card)
      src.cards = src.cards.filter((_, i) => i !== op.card_idx)
      const dst = colByName(b, op.target_column)
      if (!dst) throw new OpError('NOT_FOUND', `target column ${op.target_column}`)
      const cards = dst.cards ?? []
      if (op.before_idx < 0 || op.before_idx >= cards.length) {
        dst.cards = [...cards, card]
      } else {
        dst.cards = [...cards.slice(0, op.before_idx), card, ...cards.slice(op.before_idx)]
      }
      return b
    }
    case 'edit_card': {
      const col = colAt(b, op.col_idx)
      cardAt(col, op.card_idx, op.col_idx)
      const card = col.cards[op.card_idx]!
      ensureCardId(card)
      if (op.title !== '') card.title = op.title
      card.body = op.body
      card.tags = op.tags
      card.links = op.links
      card.priority = op.priority
      card.due = op.due
      card.assignee = op.assignee
      return b
    }
    case 'delete_card': {
      const col = colAt(b, op.col_idx)
      cardAt(col, op.card_idx, op.col_idx)
      col.cards = col.cards.filter((_, i) => i !== op.card_idx)
      return b
    }
    case 'complete_card': {
      const col = colAt(b, op.col_idx)
      cardAt(col, op.card_idx, op.col_idx)
      const card = col.cards[op.card_idx]!
      ensureCardId(card)
      card.completed = !card.completed
      return b
    }
    case 'tag_card': {
      const col = colAt(b, op.col_idx)
      cardAt(col, op.card_idx, op.col_idx)
      const card = col.cards[op.card_idx]!
      ensureCardId(card)
      const existing = new Set(card.tags ?? [])
      const merged = [...(card.tags ?? [])]
      for (const t of op.tags) {
        if (!existing.has(t)) {
          merged.push(t)
          existing.add(t)
        }
      }
      card.tags = merged
      return b
    }
    case 'add_column': {
      b.columns = [...(b.columns ?? []), { name: op.name, cards: [] }]
      return b
    }
    case 'delete_column': {
      const cols = b.columns ?? []
      const idx = cols.findIndex((c) => c.name === op.name)
      if (idx < 0) return b // idempotent
      b.columns = cols.filter((_, i) => i !== idx)
      if (b.list_collapse && idx < b.list_collapse.length) {
        b.list_collapse = [
          ...b.list_collapse.slice(0, idx),
          ...b.list_collapse.slice(idx + 1),
        ]
      }
      return b
    }
    case 'rename_column': {
      const cols = b.columns ?? []
      let found = false
      for (const c of cols) {
        if (c.name === op.old_name) {
          c.name = op.new_name
          found = true
        }
      }
      if (!found) throw new OpError('NOT_FOUND', `column ${op.old_name}`)
      return b
    }
    case 'move_column': {
      const cols = b.columns ?? []
      // Align list_collapse length.
      const collapse = b.list_collapse ?? []
      while (collapse.length < cols.length) collapse.push(false)
      // Collapse map keyed by column name.
      const collapseByName = new Map<string, boolean>()
      for (let i = 0; i < cols.length; i++) {
        collapseByName.set(cols[i]!.name, collapse[i] ?? false)
      }
      const movingIdx = cols.findIndex((c) => c.name === op.name)
      if (movingIdx < 0) throw new OpError('NOT_FOUND', `column ${op.name}`)
      const moving = cols[movingIdx]!
      const remaining = cols.filter((_, i) => i !== movingIdx)
      let reordered: Column[]
      if (op.after_col === '') {
        reordered = [moving, ...remaining]
      } else {
        reordered = []
        for (const c of remaining) {
          reordered.push(c)
          if (c.name === op.after_col) reordered.push(moving)
        }
      }
      b.columns = reordered
      b.list_collapse = reordered.map((c) => collapseByName.get(c.name) ?? false)
      return b
    }
    case 'sort_column': {
      const col = colAt(b, op.col_idx)
      const cards = [...(col.cards ?? [])]
      switch (op.sort_by) {
        case 'name':
          cards.sort((a, c) => stableCompare(a.title.toLowerCase(), c.title.toLowerCase()))
          break
        case 'priority':
          cards.sort((a, c) => priorityRank(c.priority) - priorityRank(a.priority))
          break
        case 'due':
          cards.sort((a, c) => dueCompare(a.due ?? '', c.due ?? ''))
          break
        default:
          throw new OpError('INTERNAL', `unknown sort key ${op.sort_by}`)
      }
      col.cards = cards
      return b
    }
    case 'toggle_column_collapse': {
      const cols = b.columns ?? []
      if (op.col_idx < 0 || op.col_idx >= cols.length) {
        throw new OpError('OUT_OF_RANGE', `column index ${op.col_idx}`)
      }
      const lc = b.list_collapse ?? []
      while (lc.length < cols.length) lc.push(false)
      lc[op.col_idx] = !lc[op.col_idx]
      b.list_collapse = lc
      return b
    }
    case 'update_board_meta': {
      if (op.name !== '') b.name = op.name
      b.description = op.description
      b.tags = op.tags
      return b
    }
    case 'update_board_members': {
      b.members = op.members
      return b
    }
    case 'update_board_icon': {
      b.icon = op.icon
      return b
    }
    case 'update_board_settings': {
      b.settings = op.settings
      return b
    }
    case 'update_tag_colors': {
      b.tag_colors = Object.keys(op.tag_colors).length > 0 ? op.tag_colors : undefined
      return b
    }
    case 'move_card_to_board': {
      // Source-side optimistic apply. Destination board is updated by a
      // separate write and observed via SSE / query invalidation.
      const src = colAt(b, op.col_idx)
      cardAt(src, op.card_idx, op.col_idx)
      src.cards = src.cards.filter((_, i) => i !== op.card_idx)
      return b
    }
    default:
      throw new OpError('INTERNAL', `unimplemented op: ${(op as MutationOp).type}`)
  }
}

function priorityRank(p: string | undefined): number {
  switch ((p ?? '').toLowerCase()) {
    case 'critical':
      return 4
    case 'high':
      return 3
    case 'medium':
      return 2
    case 'low':
      return 1
    default:
      return 0
  }
}

function stableCompare(a: string, b: string): number {
  return a < b ? -1 : a > b ? 1 : 0
}

function dueCompare(a: string, b: string): number {
  if (a === '' && b === '') return 0
  if (a === '') return 1
  if (b === '') return -1
  return stableCompare(a, b)
}

function colByName(b: Board, name: string): Column | undefined {
  return (b.columns ?? []).find((c) => c.name === name)
}

function colAt(b: Board, idx: number): Column {
  const cols = b.columns ?? []
  if (idx < 0 || idx >= cols.length) {
    throw new OpError('OUT_OF_RANGE', `column index ${idx}`)
  }
  return cols[idx]!
}

function cardAt(col: Column, idx: number, colIdx: number): void {
  const cards = col.cards ?? []
  if (idx < 0 || idx >= cards.length) {
    throw new OpError('OUT_OF_RANGE', `card index ${idx} in column ${colIdx}`)
  }
}
