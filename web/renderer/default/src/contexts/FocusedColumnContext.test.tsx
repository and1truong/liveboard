import { describe, expect, it } from 'bun:test'
import { act, renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import type { Column } from '@shared/types.js'
import { ClientProvider } from '../queries.js'
import { ActiveBoardProvider, useActiveBoard } from './ActiveBoardContext.js'
import {
  FocusedColumnProvider,
  useFocusedColumn,
} from './FocusedColumnContext.js'

const cols: Column[] = [
  { name: 'Todo', cards: [] },
  { name: 'Doing', cards: [] },
  { name: 'Done', cards: [] },
]

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return { client, qc }
}

function makeWrapper(client: Client, qc: QueryClient, columns: Column[], initialBoardId = 'b1') {
  return function Wrap({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={qc}>
        <ClientProvider client={client}>
          <ActiveBoardProvider initialBoardId={initialBoardId}>
            <FocusedColumnProvider columns={columns}>
              {children}
            </FocusedColumnProvider>
          </ActiveBoardProvider>
        </ClientProvider>
      </QueryClientProvider>
    )
  }
}

describe('FocusedColumnContext', () => {
  it('starts with focused=null', async () => {
    const { client, qc } = await setup()
    const { result } = renderHook(() => useFocusedColumn(), {
      wrapper: makeWrapper(client, qc, cols),
    })
    expect(result.current.focused).toBeNull()
  })

  it('setFocused updates the value', async () => {
    const { client, qc } = await setup()
    const { result } = renderHook(() => useFocusedColumn(), {
      wrapper: makeWrapper(client, qc, cols),
    })
    act(() => result.current.setFocused('Todo'))
    expect(result.current.focused).toBe('Todo')
    act(() => result.current.setFocused(null))
    expect(result.current.focused).toBeNull()
  })

  it('clears focused when active board changes', async () => {
    const { client, qc } = await setup()
    const { result } = renderHook(
      () => ({ focused: useFocusedColumn(), active: useActiveBoard() }),
      { wrapper: makeWrapper(client, qc, cols, 'b1') },
    )
    act(() => result.current.focused.setFocused('Todo'))
    expect(result.current.focused.focused).toBe('Todo')
    act(() => result.current.active.setActive('b2'))
    expect(result.current.focused.focused).toBeNull()
  })

  it('clears focused when the focused column is removed', async () => {
    const { client, qc } = await setup()
    let columns: Column[] = cols
    const { result, rerender } = renderHook(() => useFocusedColumn(), {
      wrapper: ({ children }: { children: React.ReactNode }) => (
        <QueryClientProvider client={qc}>
          <ClientProvider client={client}>
            <ActiveBoardProvider initialBoardId="b1">
              <FocusedColumnProvider columns={columns}>
                {children}
              </FocusedColumnProvider>
            </ActiveBoardProvider>
          </ClientProvider>
        </QueryClientProvider>
      ),
    })
    act(() => result.current.setFocused('Doing'))
    expect(result.current.focused).toBe('Doing')
    columns = [cols[0]!, cols[2]!]
    rerender()
    expect(result.current.focused).toBeNull()
  })

  it('clears focused when the focused column is renamed', async () => {
    const { client, qc } = await setup()
    let columns: Column[] = cols
    const { result, rerender } = renderHook(() => useFocusedColumn(), {
      wrapper: ({ children }: { children: React.ReactNode }) => (
        <QueryClientProvider client={qc}>
          <ClientProvider client={client}>
            <ActiveBoardProvider initialBoardId="b1">
              <FocusedColumnProvider columns={columns}>
                {children}
              </FocusedColumnProvider>
            </ActiveBoardProvider>
          </ClientProvider>
        </QueryClientProvider>
      ),
    })
    act(() => result.current.setFocused('Doing'))
    columns = [cols[0]!, { name: 'In Progress', cards: [] }, cols[2]!]
    rerender()
    expect(result.current.focused).toBeNull()
  })

  it('Escape keydown clears focused', async () => {
    const { client, qc } = await setup()
    const { result } = renderHook(() => useFocusedColumn(), {
      wrapper: makeWrapper(client, qc, cols),
    })
    act(() => result.current.setFocused('Todo'))
    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    })
    expect(result.current.focused).toBeNull()
  })

  it('Escape is ignored while an input is focused', async () => {
    const input = document.createElement('input')
    document.body.appendChild(input)
    input.focus()
    try {
      const { client, qc } = await setup()
      const { result } = renderHook(() => useFocusedColumn(), {
        wrapper: makeWrapper(client, qc, cols),
      })
      act(() => result.current.setFocused('Todo'))
      act(() => {
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
      })
      expect(result.current.focused).toBe('Todo')
    } finally {
      input.remove()
    }
  })

  it('Escape is ignored while a select is focused', async () => {
    const select = document.createElement('select')
    document.body.appendChild(select)
    select.focus()
    try {
      const { client, qc } = await setup()
      const { result } = renderHook(() => useFocusedColumn(), {
        wrapper: makeWrapper(client, qc, cols),
      })
      act(() => result.current.setFocused('Todo'))
      act(() => {
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
      })
      expect(result.current.focused).toBe('Todo')
    } finally {
      document.body.removeChild(select)
    }
  })

  it('Escape is ignored while a Radix dialog is open', async () => {
    const dialog = document.createElement('div')
    dialog.setAttribute('role', 'dialog')
    dialog.setAttribute('data-state', 'open')
    document.body.appendChild(dialog)
    try {
      const { client, qc } = await setup()
      const { result } = renderHook(() => useFocusedColumn(), {
        wrapper: makeWrapper(client, qc, cols),
      })
      act(() => result.current.setFocused('Todo'))
      act(() => {
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
      })
      expect(result.current.focused).toBe('Todo')
    } finally {
      dialog.remove()
    }
  })

  it('does not handle Escape when no column is focused', async () => {
    const { client, qc } = await setup()
    const { result } = renderHook(() => useFocusedColumn(), {
      wrapper: makeWrapper(client, qc, cols),
    })
    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    })
    expect(result.current.focused).toBeNull()
  })
})
