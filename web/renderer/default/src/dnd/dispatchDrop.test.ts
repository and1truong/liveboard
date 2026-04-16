import { describe, expect, it } from 'bun:test'
import type { Board } from '@shared/types.js'
import { dispatchDrop } from './dispatchDrop.js'

const board: Board = {
  name: 'b',
  version: 1,
  columns: [
    { name: 'A', cards: [{ title: 'a0' }, { title: 'a1' }, { title: 'a2' }] },
    { name: 'B', cards: [{ title: 'b0' }] },
    { name: 'C', cards: [] },
  ],
}

function active(id: string, type: 'card' | 'column', data: Record<string, unknown>) {
  return { id, data: { current: { type, ...data } } }
}
function over(id: string, type: 'card' | 'column', data: Record<string, unknown>) {
  return { id, data: { current: { type, ...data } } }
}

describe('dispatchDrop', () => {
  it('returns null for null over', () => {
    expect(
      dispatchDrop(active('card:0:0', 'card', { col_idx: 0, card_idx: 0 }), null, board),
    ).toBeNull()
  })

  it('returns null when card dropped on itself', () => {
    expect(
      dispatchDrop(
        active('card:0:1', 'card', { col_idx: 0, card_idx: 1 }),
        over('card:0:1', 'card', { col_idx: 0, card_idx: 1 }),
        board,
      ),
    ).toBeNull()
  })

  it('reorders card within same column (down)', () => {
    expect(
      dispatchDrop(
        active('card:0:0', 'card', { col_idx: 0, card_idx: 0 }),
        over('card:0:2', 'card', { col_idx: 0, card_idx: 2 }),
        board,
      ),
    ).toEqual({
      type: 'reorder_card',
      col_idx: 0,
      card_idx: 0,
      before_idx: 2,
      target_column: 'A',
    })
  })

  it('reorders card within same column (up)', () => {
    expect(
      dispatchDrop(
        active('card:0:2', 'card', { col_idx: 0, card_idx: 2 }),
        over('card:0:0', 'card', { col_idx: 0, card_idx: 0 }),
        board,
      ),
    ).toEqual({
      type: 'reorder_card',
      col_idx: 0,
      card_idx: 2,
      before_idx: 0,
      target_column: 'A',
    })
  })

  it('moves card across columns at index', () => {
    expect(
      dispatchDrop(
        active('card:0:1', 'card', { col_idx: 0, card_idx: 1 }),
        over('card:1:0', 'card', { col_idx: 1, card_idx: 0 }),
        board,
      ),
    ).toEqual({
      type: 'reorder_card',
      col_idx: 0,
      card_idx: 1,
      before_idx: 0,
      target_column: 'B',
    })
  })

  it('moves card to empty column (over column id)', () => {
    expect(
      dispatchDrop(
        active('card:0:0', 'card', { col_idx: 0, card_idx: 0 }),
        over('column:C', 'column', { name: 'C', col_idx: 2 }),
        board,
      ),
    ).toEqual({
      type: 'move_card',
      col_idx: 0,
      card_idx: 0,
      target_column: 'C',
    })
  })

  it('moves card to non-empty column header (append to end)', () => {
    expect(
      dispatchDrop(
        active('card:0:0', 'card', { col_idx: 0, card_idx: 0 }),
        over('column:B', 'column', { name: 'B', col_idx: 1 }),
        board,
      ),
    ).toEqual({
      type: 'move_card',
      col_idx: 0,
      card_idx: 0,
      target_column: 'B',
    })
  })

  it('appends card to end via column-end drop zone', () => {
    expect(
      dispatchDrop(
        active('card:0:0', 'card', { col_idx: 0, card_idx: 0 }),
        over('colend:B', 'column-end', { name: 'B' }),
        board,
      ),
    ).toEqual({
      type: 'move_card',
      col_idx: 0,
      card_idx: 0,
      target_column: 'B',
    })
  })

  it('returns null when column dropped on itself', () => {
    expect(
      dispatchDrop(
        active('column:A', 'column', { name: 'A', col_idx: 0 }),
        over('column:A', 'column', { name: 'A', col_idx: 0 }),
        board,
      ),
    ).toBeNull()
  })

  it('moves column right (drops on column at higher index)', () => {
    expect(
      dispatchDrop(
        active('column:A', 'column', { name: 'A', col_idx: 0 }),
        over('column:C', 'column', { name: 'C', col_idx: 2 }),
        board,
      ),
    ).toEqual({ type: 'move_column', name: 'A', after_col: 'C' })
  })

  it('moves column left (drops on column at lower index, lands at first slot)', () => {
    expect(
      dispatchDrop(
        active('column:C', 'column', { name: 'C', col_idx: 2 }),
        over('column:A', 'column', { name: 'A', col_idx: 0 }),
        board,
      ),
    ).toEqual({ type: 'move_column', name: 'C', after_col: '' })
  })

  it('moves column left into middle (drops on middle column from right)', () => {
    expect(
      dispatchDrop(
        active('column:C', 'column', { name: 'C', col_idx: 2 }),
        over('column:B', 'column', { name: 'B', col_idx: 1 }),
        board,
      ),
    ).toEqual({ type: 'move_column', name: 'C', after_col: 'A' })
  })

  it('returns null when types do not match (card over nothing)', () => {
    expect(
      dispatchDrop(
        active('card:0:0', 'card', { col_idx: 0, card_idx: 0 }),
        null,
        board,
      ),
    ).toBeNull()
  })
})
