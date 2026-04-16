import { describe, expect, it, beforeEach } from 'bun:test'
import { useState } from 'react'
import { fireEvent, render } from '@testing-library/react'
import type { Board } from '@shared/types.js'
import { BoardFilterProvider, useBoardFilter } from '../contexts/BoardFilterContext.js'
import { FilterPopover } from './FilterPopover.js'

const board: Board = {
  name: 'Test',
  tags: ['frontend', 'backend', 'urgent'],
  tag_colors: { urgent: '#e05252' },
  columns: [],
}

function Harness({ board }: { board: Board }): JSX.Element {
  const [open, setOpen] = useState(false)
  return (
    <BoardFilterProvider boardId="test" availableTags={board.tags ?? []}>
      <FilterPopover
        board={board}
        availableTags={board.tags ?? []}
        open={open}
        onOpenChange={setOpen}
      />
      <FilterReadout />
      <button type="button" onClick={() => setOpen(true)}>force-open</button>
    </BoardFilterProvider>
  )
}

function FilterReadout(): JSX.Element {
  const { filter } = useBoardFilter()
  return (
    <div data-testid="readout">
      q={filter.query}|tags={filter.tags.join(',')}|hide={String(filter.hideCompleted)}
    </div>
  )
}

describe('FilterPopover', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('toggles a tag selection on click', () => {
    const { getByText, getByRole, getByTestId } = render(<Harness board={board} />)
    fireEvent.click(getByText('force-open'))
    const frontend = getByRole('checkbox', { name: 'frontend' })
    expect(frontend.getAttribute('aria-checked')).toBe('false')
    fireEvent.click(frontend)
    expect(getByTestId('readout').textContent).toContain('tags=frontend')
    fireEvent.click(frontend)
    expect(getByTestId('readout').textContent).toContain('tags=|')
  })

  it('reset clears all filters', () => {
    const { getByText, getByRole, getByTestId } = render(<Harness board={board} />)
    fireEvent.click(getByText('force-open'))
    fireEvent.click(getByRole('checkbox', { name: 'urgent' }))
    fireEvent.click(getByRole('checkbox', { name: 'frontend' }))
    fireEvent.click(getByRole('switch', { name: /Hide completed/i }))

    const readout = getByTestId('readout')
    expect(readout.textContent).toBe('q=|tags=urgent,frontend|hide=true')

    fireEvent.click(getByRole('button', { name: 'Reset' }))
    expect(readout.textContent).toBe('q=|tags=|hide=false')
  })

  it('shows trigger badge with active count', () => {
    const { getByText, getByRole, getByLabelText } = render(<Harness board={board} />)
    fireEvent.click(getByText('force-open'))
    fireEvent.click(getByRole('checkbox', { name: 'frontend' }))
    fireEvent.click(getByRole('checkbox', { name: 'backend' }))
    expect(getByLabelText(/Filter \(2 active\)/)).toBeTruthy()
  })

  it('shows empty-state message when board has no tags', () => {
    const empty: Board = { ...board, tags: [], tag_colors: {} }
    const { getByText } = render(<Harness board={empty} />)
    fireEvent.click(getByText('force-open'))
    expect(getByText('No tags on this board yet.')).toBeTruthy()
  })
})
