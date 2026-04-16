import { describe, expect, it } from 'bun:test'
import { act, fireEvent, render } from '@testing-library/react'
import type { Column } from '@shared/types.js'
import {
  FocusedColumnProvider,
  useFocusedColumn,
} from '../contexts/FocusedColumnContext.js'
import { FocusExitBar } from './FocusExitBar.js'

const cols: Column[] = [{ name: 'Todo', cards: [] }]

describe('FocusExitBar', () => {
  it('renders with the focused column name', async () => {
    let setFocusedRef: ((v: string | null) => void) | null = null

    function TestHost(): JSX.Element {
      const { setFocused } = useFocusedColumn()
      setFocusedRef = setFocused
      return <FocusExitBar />
    }

    const { findByText } = render(
      <FocusedColumnProvider columns={cols} active="b1">
        <TestHost />
      </FocusedColumnProvider>,
    )

    act(() => {
      setFocusedRef!('Todo')
    })

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
    let setFocusedRef: ((v: string | null) => void) | null = null

    function Host(): JSX.Element {
      const { focused, setFocused } = useFocusedColumn()
      setFocusedRef = setFocused
      return (
        <>
          <FocusExitBar />
          <span data-testid="state">{focused ?? 'null'}</span>
        </>
      )
    }
    const { findByText, getByTestId } = render(
      <FocusedColumnProvider columns={cols} active="b1">
        <Host />
      </FocusedColumnProvider>,
    )

    act(() => {
      setFocusedRef!('Todo')
    })

    const btn = await findByText(/Exit Focus/)
    act(() => {
      fireEvent.click(btn)
    })
    expect(getByTestId('state').textContent).toBe('null')
  })
})
