import { describe, expect, it } from 'bun:test'
import { act, fireEvent, render } from '@testing-library/react'
import type { Column } from '@shared/types.js'
import {
  FocusedColumnProvider,
  useFocusedColumn,
} from '../contexts/FocusedColumnContext.js'
import { FocusExitBar } from './FocusExitBar.js'

const cols: Column[] = [{ name: 'Todo', cards: [] }]

function Probe({ initial }: { initial: string }): JSX.Element {
  const { focused, setFocused } = useFocusedColumn()
  // Set once on first render.
  if (focused === null && initial) {
    queueMicrotask(() => setFocused(initial))
  }
  return <FocusExitBar />
}

describe('FocusExitBar', () => {
  it('renders with the focused column name', async () => {
    const { findByText } = render(
      <FocusedColumnProvider columns={cols} active="b1">
        <Probe initial="Todo" />
      </FocusedColumnProvider>,
    )
    expect(await findByText(/Focusing:/)).toBeDefined()
    expect(await findByText('Todo')).toBeDefined()
  })

  it('renders nothing when no column is focused', () => {
    const { container } = render(
      <FocusedColumnProvider columns={cols} active="b1">
        <FocusExitBar />
      </FocusedColumnProvider>,
    )
    expect(container.firstChild).toBeNull()
  })

  it('exit button clears focused', async () => {
    function Host(): JSX.Element {
      const { focused } = useFocusedColumn()
      return (
        <>
          <Probe initial="Todo" />
          <span data-testid="state">{focused ?? 'null'}</span>
        </>
      )
    }
    const { findByText, getByTestId } = render(
      <FocusedColumnProvider columns={cols} active="b1">
        <Host />
      </FocusedColumnProvider>,
    )
    const btn = await findByText(/Exit Focus/)
    act(() => {
      fireEvent.click(btn)
    })
    expect(getByTestId('state').textContent).toBe('null')
  })
})
