import { describe, expect, it, mock } from 'bun:test'
import { act, renderHook } from '@testing-library/react'
import type { ReactNode } from 'react'
import { ActiveBoardProvider, useActiveBoard } from './ActiveBoardContext.js'
import { ClientProvider } from '../queries.js'

const fakeClient = {
  emit: mock(() => {}),
  on: mock(() => () => {}),
} as unknown as import('@shared/client.js').Client

function Wrapper({ children }: { children: ReactNode }): JSX.Element {
  return (
    <ClientProvider client={fakeClient}>
      <ActiveBoardProvider>{children}</ActiveBoardProvider>
    </ClientProvider>
  )
}

describe('ActiveBoardContext', () => {
  it('starts with active=null', () => {
    const { result } = renderHook(() => useActiveBoard(), { wrapper: Wrapper })
    expect(result.current.active).toBeNull()
  })

  it('setActive updates the context and emits active.changed', () => {
    const { result } = renderHook(() => useActiveBoard(), { wrapper: Wrapper })
    act(() => result.current.setActive('foo'))
    expect(result.current.active).toBe('foo')
    expect(fakeClient.emit).toHaveBeenCalledWith('active.changed', { boardId: 'foo', cardPos: null })
    act(() => result.current.setActive(null))
    expect(result.current.active).toBeNull()
  })

  it('throws when used outside provider', () => {
    expect(() => renderHook(() => useActiveBoard())).toThrow()
  })
})
