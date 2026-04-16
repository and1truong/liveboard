import { describe, expect, it } from 'bun:test'
import { act, fireEvent, render } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import type { Column } from '@shared/types.js'
import { ClientProvider } from '../queries.js'
import { ActiveBoardProvider } from '../contexts/ActiveBoardContext.js'
import {
  FocusedColumnProvider,
  useFocusedColumn,
} from '../contexts/FocusedColumnContext.js'
import { FocusExitBar } from './FocusExitBar.js'

const cols: Column[] = [{ name: 'Todo', cards: [] }]

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return { client, qc }
}

function Wrapper({ client, qc, children }: { client: Client; qc: QueryClient; children: React.ReactNode }): JSX.Element {
  return (
    <QueryClientProvider client={qc}>
      <ClientProvider client={client}>
        <ActiveBoardProvider initialBoardId="b1">
          <FocusedColumnProvider columns={cols}>
            {children}
          </FocusedColumnProvider>
        </ActiveBoardProvider>
      </ClientProvider>
    </QueryClientProvider>
  )
}

describe('FocusExitBar', () => {
  it('renders with the focused column name', async () => {
    const { client, qc } = await setup()
    let setFocusedRef: ((v: string | null) => void) | null = null

    function TestHost(): JSX.Element {
      const { setFocused } = useFocusedColumn()
      setFocusedRef = setFocused
      return <FocusExitBar />
    }

    const { findByText } = render(
      <Wrapper client={client} qc={qc}>
        <TestHost />
      </Wrapper>,
    )

    act(() => {
      setFocusedRef!('Todo')
    })

    expect(await findByText(/Focusing:/)).toBeDefined()
    expect(await findByText('Todo')).toBeDefined()
  })

  it('renders nothing when no column is focused', async () => {
    const { client, qc } = await setup()
    const { container } = render(
      <Wrapper client={client} qc={qc}>
        <FocusExitBar />
      </Wrapper>,
    )
    expect(container.firstChild).toBeNull()
  })

  it('exit button clears focused', async () => {
    const { client, qc } = await setup()
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
      <Wrapper client={client} qc={qc}>
        <Host />
      </Wrapper>,
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
